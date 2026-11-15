package service

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	htmltemplate "html/template"
	"log/slog"
	texttemplate "text/template"

	"github.com/bigphu/goat/logger"
	"github.com/bigphu/goat/mail"

	"sunshine/internal/identity/domain"
	"sunshine/internal/identity/job"
)

//go:embed templates
var templateFS embed.FS

// Template parse một lần lúc khởi động; hỏng là lỗi lập trình nên Must
var (
	textTemplates = texttemplate.Must(texttemplate.ParseFS(templateFS, "templates/*.txt"))
	// Mỗi loại thư một bộ template riêng, vì cả hai cùng định nghĩa khối "body"
	htmlTemplates = map[domain.TokenPurpose]*htmltemplate.Template{
		domain.PurposeInvite: htmltemplate.Must(htmltemplate.ParseFS(templateFS, "templates/layout.html", "templates/invite.html")),
		domain.PurposeReset:  htmltemplate.Must(htmltemplate.ParseFS(templateFS, "templates/layout.html", "templates/reset.html")),
	}
)

var accountEmail = map[domain.TokenPurpose]struct {
	subject   string
	expiresIn string
}{
	domain.PurposeInvite: {"Lời mời tham gia diễn đàn Sunshine Apartment", "3 ngày"},
	domain.PurposeReset:  {"Đặt lại mật khẩu diễn đàn Sunshine Apartment", "1 ngày"},
}

type emailData struct {
	Name      string
	Link      string
	ExpiresIn string
}

// SendAccountEmail là việc của job identity.send_account_email. Link đã chết (dùng rồi,
// bị thay, hết hạn, tài khoản bị khoá) thì không gửi và trả nil. Lỗi gửi được bọc bằng
// %w để mail.IsPermanent nhìn xuyên qua.
func (s *Service) SendAccountEmail(ctx context.Context, args job.SendAccountEmailArgs) error {
	if s.mail == nil {
		return errors.New("identity: no mail sender configured")
	}
	log := logger.FromContext(ctx)
	info, err := s.PasswordTokenInfo(ctx, args.Token)
	if errors.Is(err, domain.ErrInvalidPasswordToken) {
		log.InfoContext(ctx, "account email skipped, link no longer valid", slog.String("purpose", args.Purpose))
		return nil
	}
	if err != nil {
		return err
	}
	kind := accountEmail[info.Purpose]
	data := emailData{
		Name:      info.Name,
		Link:      s.appURL + "/dat-mat-khau#token=" + args.Token,
		ExpiresIn: kind.expiresIn,
	}
	var text, html bytes.Buffer
	if err := textTemplates.ExecuteTemplate(&text, string(info.Purpose)+".txt", data); err != nil {
		return fmt.Errorf("identity: render email: %w", err)
	}
	if err := htmlTemplates[info.Purpose].ExecuteTemplate(&html, string(info.Purpose)+".html", data); err != nil {
		return fmt.Errorf("identity: render email: %w", err)
	}
	err = s.mail.Send(ctx, mail.Message{
		To:      mail.Address{Name: info.Name, Email: info.Email},
		Subject: kind.subject, Text: text.String(), HTML: html.String(),
		IdempotencyKey: "identity-" + args.Purpose + "-" + fmt.Sprintf("%x", HashToken(args.Token)[:8]),
	})
	if err != nil {
		return fmt.Errorf("identity: send %s email: %w", info.Purpose, err)
	}
	return nil
}

// Maintenance là việc của job định kỳ identity.maintenance
func (s *Service) Maintenance(ctx context.Context) error {
	log := logger.FromContext(ctx)
	sessions, err := s.sessions.Prune(ctx)
	if err != nil {
		return err
	}
	tokens, err := s.tokens.Prune(ctx)
	if err != nil {
		return err
	}
	purged, err := s.accounts.PurgeDeactivated(ctx, s.cfg.PurgeAfterDays)
	if err != nil {
		return err
	}
	log.InfoContext(ctx, "identity maintenance done",
		slog.Int64("sessions", sessions), slog.Int64("tokens", tokens), slog.Int64("accounts_purged", purged))
	return nil
}

// SeedAccount tạo tài khoản đã có mật khẩu (chỉ cmd/seed dùng); email đã có thì bỏ qua
func (s *Service) SeedAccount(ctx context.Context, in Invitation, password string) (domain.Account, error) {
	email, err := domain.NormalizeEmail(in.Email)
	if err != nil {
		return domain.Account{}, err
	}
	if a, err := s.accounts.GetByEmail(ctx, email); err == nil {
		return a, nil
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.Account{}, err
	}
	return s.accounts.CreateActive(ctx, domain.NewAccount{Email: email, Name: in.Name, Unit: in.Unit, Role: in.Role}, hash)
}

// Bootstrap tạo Quản trị viên đầu tiên khi DB chưa có tài khoản nào
func (s *Service) Bootstrap(ctx context.Context, email, password string) (bool, error) {
	n, err := s.accounts.Count(ctx)
	if err != nil || n > 0 {
		return false, err
	}
	if err := domain.CheckPassword(password, email); err != nil {
		return false, err
	}
	_, err = s.SeedAccount(ctx, Invitation{Email: email, Name: "Quản trị viên", Role: domain.RoleAdmin}, password)
	return err == nil, err
}
