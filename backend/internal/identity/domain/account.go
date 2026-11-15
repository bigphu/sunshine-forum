// Package domain là thực thể, quy tắc và lỗi của identity. Không biết HTTP hay SQL.
package domain

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Role là một trong ba vai trò cố định (FR-1.2). Không có vai trò Owner riêng.
type Role string

const (
	RoleResident Role = "resident"
	RoleManager  Role = "manager"
	RoleAdmin    Role = "admin"
)

func (r Role) Valid() bool { return r == RoleResident || r == RoleManager || r == RoleAdmin }

// IsStaff: Quản lý và Quản trị viên (trả lời chính thức, kiểm duyệt)
func (r Role) IsStaff() bool { return r == RoleManager || r == RoleAdmin }

// ID cố định của vai trò, seed ở migration 00002
var roleIDs = map[Role]uuid.UUID{
	RoleResident: uuid.MustParse("00000000-0000-7000-8000-000000000001"),
	RoleManager:  uuid.MustParse("00000000-0000-7000-8000-000000000002"),
	RoleAdmin:    uuid.MustParse("00000000-0000-7000-8000-000000000003"),
}

func (r Role) ID() uuid.UUID { return roleIDs[r] }

func RoleFromID(id uuid.UUID) Role {
	for r, rid := range roleIDs {
		if rid == id {
			return r
		}
	}
	return ""
}

type Status string

const (
	StatusInvited     Status = "invited"
	StatusActive      Status = "active"
	StatusDeactivated Status = "deactivated"
	// Đã xoá dữ liệu cá nhân sau 90 ngày bị khoá (NFR-3.3)
	StatusPurged Status = "purged"
)

// Account là một người có tên trong danh sách cư dân (hoặc ban quản lý). Email, tên và
// căn hộ đã được giải mã; repository lo mã hoá khi ghi.
type Account struct {
	ID            uuid.UUID
	Email         string
	Name          string
	Unit          string
	Role          Role
	Status        Status
	PasswordHash  string
	LastSignInAt  *time.Time
	DeactivatedAt *time.Time
	CreatedAt     time.Time
}

// CanSignIn: tài khoản chưa nhận lời mời trả ErrBadCredentials, không lộ ra email có tồn tại
func (a Account) CanSignIn() error {
	if a.Status != StatusActive || a.PasswordHash == "" {
		return ErrBadCredentials
	}
	return nil
}

// NormalizeEmail bỏ khoảng trắng, viết thường và kiểm tra dạng local@domain
func NormalizeEmail(s string) (string, error) {
	e := strings.ToLower(strings.TrimSpace(s))
	local, host, ok := strings.Cut(e, "@")
	if !ok || local == "" || host == "" || strings.Contains(host, "@") ||
		strings.ContainsAny(e, " \t\r\n") || len(e) > 254 {
		return "", ErrInvalidEmail
	}
	return e, nil
}

func CleanName(s string) (string, error) {
	n := strings.Join(strings.Fields(s), " ")
	if n == "" || utf8.RuneCountInString(n) > 100 {
		return "", ErrInvalidName
	}
	return n, nil
}

func CleanUnit(s string) (string, error) {
	u := strings.ToUpper(strings.TrimSpace(s))
	if utf8.RuneCountInString(u) > 20 {
		return "", ErrInvalidUnit
	}
	return u, nil
}

// CheckPassword: ít nhất 8 ký tự, không trùng email, tối đa 72 byte (giới hạn của bcrypt)
func CheckPassword(pw, email string) error {
	if utf8.RuneCountInString(pw) < 8 || len(pw) > 72 {
		return ErrWeakPassword
	}
	if email != "" && strings.EqualFold(pw, email) {
		return ErrPasswordIsEmail
	}
	return nil
}
