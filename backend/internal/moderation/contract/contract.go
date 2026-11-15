// Package contract là phần công khai của moderation: tên và payload event (content,
// penalties, notifications phản ứng với quyết định), và các reader cho administration.
package contract

import (
	"context"
	"time"

	"github.com/google/uuid"
)

const (
	ActionApprove    = "approve"
	ActionReject     = "reject"
	ActionRemove     = "remove"
	ActionAutoDelete = "auto_delete"
)

const (
	AggregateDecision = "moderation.decision"
	AggregateAppeal   = "moderation.appeal"

	// Quyết định đã ghi (cùng transaction với nhật ký kiểm toán). content đổi trạng thái,
	// penalties tính vi phạm, notifications báo cho tác giả.
	EventDecisionRecorded = "moderation.decision_recorded"
	// Yêu cầu xem xét lại đã có kết quả cuối cùng
	EventAppealDecided = "moderation.appeal_decided"
)

// DecisionRecorded chỉ chứa ID và mã (event bị dọn sau 90 ngày, không giữ nội dung)
type DecisionRecorded struct {
	DecisionID     uuid.UUID  `json:"decision_id"`
	ContentType    string     `json:"content_type"`
	ContentID      uuid.UUID  `json:"content_id"`
	AuthorID       uuid.UUID  `json:"author_id"`
	ChannelID      uuid.UUID  `json:"channel_id"`
	Action         string     `json:"action"`
	Rule           int        `json:"rule"`
	AppealDeadline *time.Time `json:"appeal_deadline,omitempty"`
}

// Violation: quyết định này có tính là một vi phạm đã xác nhận không
func (d DecisionRecorded) Violation() bool {
	return d.Action == ActionReject || d.Action == ActionRemove || d.Action == ActionAutoDelete
}

type AppealDecided struct {
	AppealID    uuid.UUID `json:"appeal_id"`
	DecisionID  uuid.UUID `json:"decision_id"`
	ContentType string    `json:"content_type"`
	ContentID   uuid.UUID `json:"content_id"`
	AuthorID    uuid.UUID `json:"author_id"`
	ChannelID   uuid.UUID `json:"channel_id"`
	Accepted    bool      `json:"accepted"`
	Rule        int       `json:"rule"`
}

// --- Cho administration ---

type Stats struct {
	Approved, Rejected, Removed, AutoDeleted int64
	AppealsPending, AppealsAccepted          int64
	AppealsDeclined                          int64
	// OverturnedAI: quyết định của quản lý đảo kết luận của bộ lọc
	OverturnedAI int64
	// Lật quyết định AI theo loại: giữ nhầm (duyệt nội dung bị giữ), bỏ sót (gỡ nội dung đã
	// đăng), xoá nhầm (xem xét lại thành công của nội dung bị tự xoá)
	HeldWrong, Missed, DeletedWrong int64
	// Agreed: từ chối nội dung mô hình đã gắn nhãn (quản lý đồng ý với AI)
	Agreed int64
}

// Correction là một quyết định đã chốt đảo kết luận của AI, cho bản xuất huấn luyện
type Correction struct {
	DecisionID   uuid.UUID
	ContentType  string
	ContentID    uuid.UUID
	Action       string // approve (AI giữ nhầm), remove (AI bỏ sót), reject sau khi AI giữ đúng...
	FinalLabel   string // clean | offensive | hate
	Rule         int
	AIScore      *float64
	ModelVersion string
	DecidedAt    time.Time
}

type Reader interface {
	Stats(ctx context.Context, since time.Time) (Stats, error)
	// Corrections: quyết định có ai_label, đã qua hạn xem xét lại, trong khoảng [from, to]
	Corrections(ctx context.Context, from, to time.Time) ([]Correction, error)
}
