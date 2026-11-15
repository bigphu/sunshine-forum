// Package domain là thông báo trong ứng dụng và lời nhắn gửi cư dân (COMMUNITY_RULES.md,
// phần 2). Không biết HTTP hay SQL.
package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	screening "sunshine/internal/screening/contract"
)

const PermRead = "notifications.read"

const (
	KindModeration = "moderation"
	KindAppeal     = "appeal"
	KindReadOnly   = "read_only"
	KindComment    = "comment"
	KindReply      = "reply"
	KindAnnounce   = "announcement"
)

type Item struct {
	ID        uuid.UUID
	AccountID uuid.UUID
	Kind      string
	Body      string
	Link      string
	ReadAt    *time.Time
	CreatedAt time.Time
}

// NewItem: DedupeKey chống trùng khi event được giao lại
type NewItem struct {
	AccountID uuid.UUID
	Kind      string
	DedupeKey string
	Body      string
	Link      string
}

type Repository interface {
	Insert(ctx context.Context, items []NewItem) error
	List(ctx context.Context, account uuid.UUID, limit int) ([]Item, int, error)
	MarkAllRead(ctx context.Context, account uuid.UUID) error
	Purge(ctx context.Context, days int) (int64, error)
	// Listen gọi onNew(account) mỗi khi có thông báo mới, tới khi ctx bị huỷ
	Listen(ctx context.Context, onNew func(account uuid.UUID)) error
}

// --- Lời nhắn (FR-4.5): luôn nêu quy tắc cộng đồng, không nêu nhãn của mô hình ---

var vn = time.FixedZone("ICT", 7*3600)

func date(t time.Time) string     { return t.In(vn).Format("02/01/2006") }
func dateTime(t time.Time) string { return t.In(vn).Format("15:04 02/01/2006") }

var verbs = map[string]string{"reject": "từ chối", "remove": "gỡ", "auto_delete": "tự động xóa"}

func DecisionMessage(action, channel string, rule int, deadline *time.Time) string {
	msg := fmt.Sprintf("Nội dung của bạn trong kênh %s đã bị %s vì vi phạm Quy tắc %d: %s.",
		channel, verbs[action], rule, screening.RuleTitle(rule))
	if deadline != nil {
		msg += fmt.Sprintf(" Bạn có thể yêu cầu xem xét lại một lần trước ngày %s.", date(*deadline))
	}
	return msg
}

func ReadOnlyMessage(end time.Time, rule int) string {
	return fmt.Sprintf("Tài khoản của bạn tạm thời chỉ được đọc đến %s vì đã nhiều lần vi phạm quy tắc cộng đồng. Lần vi phạm gần nhất: Quy tắc %d: %s.",
		dateTime(end), rule, screening.RuleTitle(rule))
}

const AppealAcceptedMessage = "Yêu cầu xem xét lại của bạn đã được chấp nhận. Nội dung đã được khôi phục và lần vi phạm này đã được gỡ bỏ."

func AppealDeclinedMessage(rule int) string {
	return fmt.Sprintf("Yêu cầu xem xét lại của bạn không được chấp nhận. Nội dung vẫn bị từ chối vì vi phạm Quy tắc %d: %s. Đây là quyết định cuối cùng.",
		rule, screening.RuleTitle(rule))
}

func CommentMessage(who, title string) string {
	return fmt.Sprintf("%s đã bình luận trong bài \"%s\".", who, title)
}

func ReplyMessage(who, title string) string {
	return fmt.Sprintf("%s đã trả lời bình luận của bạn trong bài \"%s\".", who, title)
}

func AnnouncementMessage(title string) string {
	return fmt.Sprintf("Thông báo mới từ ban quản lý: \"%s\".", title)
}
