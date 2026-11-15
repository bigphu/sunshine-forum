// Package domain là quy tắc của penalties: vi phạm đã xác nhận, số lần trong khoảng thời
// gian trượt, và tạm chỉ đọc (FR-1.3). Không dùng shadow ban.
package domain

import (
	"context"
	"time"

	"github.com/bigphu/goat/errs"
	"github.com/google/uuid"

	"sunshine/internal/penalties/contract"
)

const PermSettingsManage = "penalties.settings.manage"

type Settings struct {
	StrikeLimit      int
	StrikeWindowDays int
	TimeoutDays      int
	UpdatedAt        time.Time
}

func (s Settings) Validate() error {
	if s.StrikeLimit < 1 || s.StrikeLimit > 50 || s.StrikeWindowDays < 1 || s.StrikeWindowDays > 365 ||
		s.TimeoutDays < 1 || s.TimeoutDays > 365 {
		return ErrInvalidSettings
	}
	return nil
}

type Timeout struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	Rule      int
	EndsAt    time.Time
}

type Repository interface {
	Settings(ctx context.Context) (Settings, error)
	// UpdateSettings ghi kèm event settings_changed trong cùng transaction
	UpdateSettings(ctx context.Context, s Settings, changes []contract.Change) (Settings, error)
	ActiveTimeouts(ctx context.Context) ([]Timeout, error)
	// RecordStrike ghi vi phạm (idempotent theo decision) và, nếu vượt giới hạn, mở một lần
	// tạm chỉ đọc kèm event read_only_started, trong cùng transaction. Trả lần tạm chỉ đọc
	// vừa mở (nil nếu không).
	RecordStrike(ctx context.Context, account, decision uuid.UUID, rule int, s Settings) (*Timeout, error)
	// RevokeStrike thu hồi vi phạm của decision; số vi phạm còn dưới giới hạn thì gỡ tạm chỉ
	// đọc kèm event read_only_lifted.
	RevokeStrike(ctx context.Context, decision uuid.UUID, s Settings) error
	ActiveTimeout(ctx context.Context, account uuid.UUID) (*Timeout, error)
	ActiveStrikes(ctx context.Context, account uuid.UUID, windowDays int) (int, error)
	Stats(ctx context.Context, windowDays int) (strikes, readOnly int64, err error)
}

var ErrInvalidSettings = errs.Unprocessable("Số lần vi phạm, khoảng thời gian và thời hạn chỉ đọc phải từ 1 trở lên.",
	errs.WithType("/errors/invalid-penalty-settings"), errs.WithTitle("Invalid penalty settings"))
