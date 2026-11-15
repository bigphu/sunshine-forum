package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"time"

	"github.com/bigphu/goat/auth"

	"sunshine/internal/identity/domain"
)

// SignIn là kết quả đăng nhập: token thô đặt vào cookie, CSRF token trả cho client
type SignIn struct {
	Account domain.Account
	Token   string
	CSRF    string
}

// Login kiểm tra email và mật khẩu rồi mở phiên mới.
func (s *Service) Login(ctx context.Context, email, password string) (SignIn, error) {
	e, err := domain.NormalizeEmail(email)
	if err != nil {
		return SignIn{}, domain.ErrBadCredentials
	}
	a, err := s.accounts.GetByEmail(ctx, e)
	if errors.Is(err, domain.ErrAccountNotFound) {
		// Vẫn băm để thời gian trả lời không lộ email có tồn tại hay không
		_ = s.hasher.Verify("$argon2id$v=19$m=19456,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$aGFzaGhhc2hoYXNoaGFzaGhhc2hoYXNoaGFzaGhhc2g", password)
		return SignIn{}, domain.ErrBadCredentials
	}
	if err != nil {
		return SignIn{}, err
	}
	if err := a.CanSignIn(); err != nil {
		return SignIn{}, err
	}
	if !s.hasher.Verify(a.PasswordHash, password) {
		return SignIn{}, domain.ErrBadCredentials
	}
	out, err := s.openSession(ctx, a)
	if err != nil {
		return SignIn{}, err
	}
	_ = s.accounts.TouchSignIn(ctx, a.ID)
	return out, nil
}

func (s *Service) openSession(ctx context.Context, a domain.Account) (SignIn, error) {
	raw, hash := newToken()
	csrf, _ := newToken()
	err := s.sessions.Create(ctx, domain.Session{
		TokenHash: hash, AccountID: a.ID, Role: a.Role, CSRFToken: csrf,
		ExpiresAt: s.now().Add(s.cfg.SessionTTL),
	})
	if err != nil {
		return SignIn{}, err
	}
	return SignIn{Account: a, Token: raw, CSRF: csrf}, nil
}

// Logout xoá phiên của cookie hiện tại; không có phiên cũng không lỗi
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	if rawToken == "" {
		return nil
	}
	return s.sessions.Delete(ctx, HashToken(rawToken))
}

// Authenticate đọc phiên ở mỗi request và dựng Actor với quyền của vai trò hiện tại.
// Phiên được gia hạn trượt, nhưng chỉ ghi DB khi lần ghi trước đã quá một phút.
func (s *Service) Authenticate(ctx context.Context, rawToken string) (domain.Session, auth.Actor, error) {
	sess, err := s.sessions.GetActive(ctx, HashToken(rawToken))
	if err != nil {
		return domain.Session{}, auth.Actor{}, err
	}
	if now := s.now(); now.Sub(sess.LastSeenAt) > time.Minute {
		if err := s.sessions.Extend(ctx, sess.TokenHash, now.Add(s.cfg.SessionTTL)); err != nil {
			return domain.Session{}, auth.Actor{}, err
		}
	}
	return sess, auth.Actor{AccountID: sess.AccountID, Permissions: s.perms[sess.Role]}, nil
}

// CheckCSRF so header X-CSRF-Token với token của phiên
func CheckCSRF(sess domain.Session, header string) error {
	if header == "" || subtle.ConstantTimeCompare([]byte(header), []byte(sess.CSRFToken)) != 1 {
		return domain.ErrCSRF
	}
	return nil
}

// Me là tài khoản đang đăng nhập
func (s *Service) Me(ctx context.Context) (domain.Account, []string, error) {
	actor, ok := auth.FromContext(ctx)
	if !ok || actor.IsSystem() {
		return domain.Account{}, nil, auth.ErrUnauthenticated
	}
	a, err := s.accounts.Get(ctx, actor.AccountID)
	if err != nil {
		return domain.Account{}, nil, err
	}
	return a, actor.Permissions, nil
}

// ChangePassword đổi mật khẩu của chính mình; các thiết bị khác phải đăng nhập lại
func (s *Service) ChangePassword(ctx context.Context, sess domain.Session, current, next string) error {
	a, _, err := s.Me(ctx)
	if err != nil {
		return err
	}
	if !s.hasher.Verify(a.PasswordHash, current) {
		return domain.ErrWrongPassword
	}
	if err := domain.CheckPassword(next, a.Email); err != nil {
		return err
	}
	hash, err := s.hasher.Hash(next)
	if err != nil {
		return err
	}
	_, err = s.accounts.SetPassword(ctx, a.ID, hash, sess.TokenHash)
	return err
}

// ForgotPassword gửi link đặt lại nếu email thuộc tài khoản đang hoạt động. Luôn trả
// nil để không lộ email nào có tài khoản.
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	e, err := domain.NormalizeEmail(email)
	if err != nil {
		return nil
	}
	a, err := s.accounts.GetByEmail(ctx, e)
	if err != nil || a.Status != domain.StatusActive {
		return nil
	}
	raw, hash := newToken()
	return s.tokens.Issue(ctx, a.ID, domain.IssuedToken{
		Raw: raw, Hash: hash, Purpose: domain.PurposeReset, ExpiresAt: s.now().Add(s.cfg.ResetTTL),
	})
}

// TokenInfo cho trang đặt mật khẩu: tên và email của người nhận link
type TokenInfo struct {
	Name    string
	Email   string
	Purpose domain.TokenPurpose
}

func (s *Service) PasswordTokenInfo(ctx context.Context, raw string) (TokenInfo, error) {
	accountID, purpose, _, err := s.tokens.Valid(ctx, HashToken(raw))
	if err != nil {
		return TokenInfo{}, err
	}
	a, err := s.accounts.Get(ctx, accountID)
	if err != nil {
		return TokenInfo{}, domain.ErrInvalidPasswordToken
	}
	if a.Status != domain.StatusInvited && a.Status != domain.StatusActive {
		return TokenInfo{}, domain.ErrInvalidPasswordToken
	}
	return TokenInfo{Name: a.Name, Email: a.Email, Purpose: purpose}, nil
}

// SetPasswordWithToken nhận lời mời hoặc đặt lại mật khẩu, rồi mở phiên mới. Mọi phiên
// cũ bị thu hồi.
func (s *Service) SetPasswordWithToken(ctx context.Context, raw, password string) (SignIn, error) {
	hash := HashToken(raw)
	accountID, _, _, err := s.tokens.Valid(ctx, hash)
	if err != nil {
		return SignIn{}, err
	}
	a, err := s.accounts.Get(ctx, accountID)
	if err != nil {
		return SignIn{}, domain.ErrInvalidPasswordToken
	}
	if err := domain.CheckPassword(password, a.Email); err != nil {
		return SignIn{}, err
	}
	pwHash, err := s.hasher.Hash(password)
	if err != nil {
		return SignIn{}, err
	}
	ok, err := s.tokens.Use(ctx, hash)
	if err != nil {
		return SignIn{}, err
	}
	if !ok {
		return SignIn{}, domain.ErrInvalidPasswordToken
	}
	a, err = s.accounts.SetPassword(ctx, a.ID, pwHash, nil)
	if err != nil {
		return SignIn{}, err
	}
	return s.openSession(ctx, a)
}
