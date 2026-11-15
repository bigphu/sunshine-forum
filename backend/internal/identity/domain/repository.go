package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Interface repository do service dùng, repository/ cài đặt. Method ghi tự mở
// transaction và ghi event, job trong cùng transaction; service không thấy transaction.

type TokenPurpose string

const (
	PurposeInvite TokenPurpose = "invite"
	PurposeReset  TokenPurpose = "reset"
)

// IssuedToken là link đặt mật khẩu vừa phát. Raw chỉ có ở đây và trong job gửi thư;
// DB giữ Hash.
type IssuedToken struct {
	Raw       string
	Hash      []byte
	Purpose   TokenPurpose
	ExpiresAt time.Time
}

type NewAccount struct {
	Email string // đã NormalizeEmail
	Name  string
	Unit  string
	Role  Role
	// Invite phát cùng transaction tạo account, kèm job gửi thư
	Invite IssuedToken
}

type AccountRepository interface {
	// Create: ErrEmailTaken; event account_invited
	Create(ctx context.Context, in NewAccount) (Account, error)
	// Get, GetByEmail: ErrAccountNotFound
	Get(ctx context.Context, id uuid.UUID) (Account, error)
	GetByEmail(ctx context.Context, email string) (Account, error)
	GetMany(ctx context.Context, ids []uuid.UUID) ([]Account, error)
	List(ctx context.Context) ([]Account, error)
	ActiveIDsByRole(ctx context.Context, roles []Role) ([]uuid.UUID, error)
	Count(ctx context.Context) (int64, error)
	// SetPassword kích hoạt tài khoản được mời; thu hồi mọi phiên trừ keepSession
	SetPassword(ctx context.Context, id uuid.UUID, hash string, keepSession []byte) (Account, error)
	// SetRole: event role_changed
	SetRole(ctx context.Context, id uuid.UUID, role Role) (Account, error)
	// Deactivate thu hồi mọi phiên; event account_deactivated
	Deactivate(ctx context.Context, id uuid.UUID) (Account, error)
	TouchSignIn(ctx context.Context, id uuid.UUID) error
	// PurgeDeactivated xoá dữ liệu cá nhân của tài khoản đã khoá quá days ngày
	PurgeDeactivated(ctx context.Context, days int) (int64, error)
	// CreateActive cho seed: tài khoản đã có mật khẩu
	CreateActive(ctx context.Context, in NewAccount, passwordHash string) (Account, error)
}

type Session struct {
	TokenHash  []byte
	AccountID  uuid.UUID
	Role       Role
	CSRFToken  string
	LastSeenAt time.Time
	ExpiresAt  time.Time
}

type SessionRepository interface {
	Create(ctx context.Context, s Session) error
	// GetActive: ErrSessionNotFound khi hết hạn, không có, hoặc tài khoản không còn hoạt động
	GetActive(ctx context.Context, tokenHash []byte) (Session, error)
	Extend(ctx context.Context, tokenHash []byte, expiresAt time.Time) error
	Delete(ctx context.Context, tokenHash []byte) error
	Prune(ctx context.Context) (int64, error)
}

type TokenRepository interface {
	// Issue thay link cùng mục đích cũ và xếp job gửi thư trong cùng transaction
	Issue(ctx context.Context, accountID uuid.UUID, t IssuedToken) error
	// Valid: ErrInvalidPasswordToken
	Valid(ctx context.Context, hash []byte) (accountID uuid.UUID, purpose TokenPurpose, expiresAt time.Time, err error)
	// Use đánh dấu đã dùng; false khi token vừa bị dùng hay hết hạn
	Use(ctx context.Context, hash []byte) (bool, error)
	PendingInviteExpiry(ctx context.Context, accountID uuid.UUID) (*time.Time, error)
	Prune(ctx context.Context) (int64, error)
}

type RoleRepository interface {
	// Permissions của mọi vai trò; vai trò cố định nên nạp một lần lúc khởi động
	Permissions(ctx context.Context) (map[Role][]string, error)
}
