package contract

import "github.com/google/uuid"

// Đổi cài đặt bộ lọc được ghi vào nhật ký kiểm toán (moderation nghe event này), vì
// PRODUCT.md yêu cầu ghi lại thay đổi ngưỡng và chính sách sự cố.
const (
	AggregateSettings = "screening.settings"

	EventSettingsChanged = "screening.settings_changed"
)

// SettingsAggregateID: cài đặt chỉ có một bản, dùng một ID cố định làm aggregate
var SettingsAggregateID = uuid.MustParse("00000000-0000-7000-8000-0000000000a1")

// Change là một giá trị đã đổi. Field: hold_threshold, auto_delete_threshold,
// auto_delete_enabled, outage_policy, channel_hold, channel_auto_delete. Ngưỡng ghi theo
// thang /100.
type Change struct {
	Field     string     `json:"field"`
	ChannelID *uuid.UUID `json:"channel_id,omitempty"`
	Old       string     `json:"old"`
	New       string     `json:"new"`
}

type SettingsChanged struct {
	Changes []Change `json:"changes"`
}
