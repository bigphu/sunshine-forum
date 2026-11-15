// Package contract là phần công khai của penalties: module khác hỏi tài khoản có đang tạm
// chỉ đọc không (content chặn việc viết), và tên các event.
package contract

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Standing struct {
	// ReadOnlyUntil khác nil khi tài khoản đang tạm chỉ đọc
	ReadOnlyUntil *time.Time
	// LastRule: quy tắc của vi phạm gần nhất gây ra lần tạm chỉ đọc
	LastRule int
	// ActiveStrikes: vi phạm đã xác nhận còn được tính trong khoảng thời gian hiện tại
	ActiveStrikes int
}

type StandingReader interface {
	Standing(ctx context.Context, accountID uuid.UUID) (Standing, error)
}

// Thống kê cho administration
type Stats struct {
	ActiveStrikes int64
	ReadOnly      int64
}

type StatsReader interface {
	Stats(ctx context.Context) (Stats, error)
}

const (
	AggregateAccount = "penalties.account"

	EventReadOnlyStarted = "penalties.read_only_started"
	EventReadOnlyLifted  = "penalties.read_only_lifted"

	// Đổi giới hạn vi phạm được ghi vào nhật ký kiểm toán (moderation nghe event này)
	AggregateSettings    = "penalties.settings"
	EventSettingsChanged = "penalties.settings_changed"
)

// SettingsAggregateID: cài đặt chỉ có một bản
var SettingsAggregateID = uuid.MustParse("00000000-0000-7000-8000-0000000000a2")

// Change: Field là strike_limit, strike_window_days hoặc timeout_days
type Change struct {
	Field string `json:"field"`
	Old   string `json:"old"`
	New   string `json:"new"`
}

type SettingsChanged struct {
	Changes []Change `json:"changes"`
}

type ReadOnlyStarted struct {
	AccountID uuid.UUID `json:"account_id"`
	TimeoutID uuid.UUID `json:"timeout_id"`
	Rule      int       `json:"rule"`
	EndsAt    time.Time `json:"ends_at"`
}

type ReadOnlyLifted struct {
	AccountID uuid.UUID `json:"account_id"`
}
