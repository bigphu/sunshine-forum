// Package service là use case của identity: đăng nhập, phiên, lời mời, đặt lại mật
// khẩu và quản lý tài khoản. Use case quản trị bắt đầu bằng auth.Require; service không
// thấy HTTP, SQL hay transaction.
package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/mail"
	"github.com/google/uuid"

	"sunshine/internal/identity/domain"
)

type Settings struct {
	// Phiên trượt: không dùng tới trong khoảng này thì hết hạn
	SessionTTL time.Duration
	// Hạn của link mời (3 ngày) và link đặt lại mật khẩu (1 ngày)
	InviteTTL time.Duration
	ResetTTL  time.Duration
	// Xoá dữ liệu cá nhân của tài khoản đã khoá sau ngần này ngày (NFR-3.3)
	PurgeAfterDays int
}

type Deps struct {
	Accounts domain.AccountRepository
	Sessions domain.SessionRepository
	Tokens   domain.TokenRepository
	Roles    domain.RoleRepository
	Hasher   Hasher
	// Mail chỉ cần cho SendAccountEmail (worker); API để nil
	Mail mail.Sender
	// AppURL là gốc của frontend, link trong thư trỏ về đây
	AppURL   string
	Settings Settings
	Now      func() time.Time
}

type Service struct {
	accounts domain.AccountRepository
	sessions domain.SessionRepository
	tokens   domain.TokenRepository
	roles    domain.RoleRepository
	hasher   Hasher
	mail     mail.Sender
	appURL   string
	cfg      Settings
	now      func() time.Time
	perms    map[domain.Role][]string
}

func New(d Deps) *Service {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{
		accounts: d.Accounts, sessions: d.Sessions, tokens: d.Tokens, roles: d.Roles,
		hasher: d.Hasher, mail: d.Mail, appURL: d.AppURL, cfg: d.Settings, now: now,
	}
}

// LoadPermissions nạp quyền của ba vai trò; gọi một lần lúc khởi động (vai trò cố định,
// quyền chỉ đổi qua migration)
func (s *Service) LoadPermissions(ctx context.Context) error {
	perms, err := s.roles.Permissions(ctx)
	if err != nil {
		return fmt.Errorf("identity: load permissions: %w", err)
	}
	s.perms = perms
	return nil
}

func (s *Service) Permissions(role domain.Role) []string { return s.perms[role] }

// LoadActor là jobs.ActorLoader: quyền hiện tại của tài khoản đã gây ra event/job.
// Tài khoản bị khoá chạy với không quyền nào.
func (s *Service) LoadActor(ctx context.Context, id uuid.UUID) (auth.Actor, error) {
	a, err := s.accounts.Get(ctx, id)
	if err != nil {
		return auth.Actor{}, err
	}
	actor := auth.Actor{AccountID: id}
	if a.Status == domain.StatusActive {
		actor.Permissions = s.perms[a.Role]
	}
	return actor, nil
}

// newToken trả token ngẫu nhiên 32 byte (base64url) và SHA-256 của nó
func newToken() (string, []byte) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	raw := base64.RawURLEncoding.EncodeToString(b)
	return raw, HashToken(raw)
}

func HashToken(raw string) []byte {
	h := sha256.Sum256([]byte(raw))
	return h[:]
}
