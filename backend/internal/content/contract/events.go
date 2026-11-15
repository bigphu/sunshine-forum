package contract

import "github.com/google/uuid"

// Event của content. Aggregate là bài viết (bình luận dùng aggregate của bài chứa nó).
// Payload chỉ có ID: event bị dọn sau 90 ngày và không được giữ nội dung.
const (
	AggregatePost = "content.post"

	// Nội dung đã hiển thị công khai (đăng ngay, được duyệt, hoặc được khôi phục).
	// notifications báo bình luận mới và tin chính thức.
	EventPublished = "content.published"
	// Nội dung bị bộ lọc giữ lại chờ duyệt. moderation mở case.
	EventHeld = "content.held"
	// Bộ lọc tự xoá (chỉ khi Admin đã bật). moderation ghi quyết định.
	EventAutoDeleted = "content.auto_deleted"
	// Tác giả xoá, hoặc sửa nội dung đang chờ thành nội dung sạch. moderation đóng case.
	EventWithdrawn = "content.withdrawn"
)

type Published struct {
	Ref       Ref       `json:"ref"`
	PostID    uuid.UUID `json:"post_id"`
	ChannelID uuid.UUID `json:"channel_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	// Official: bài của ban quản lý trong kênh chỉ dành cho ban quản lý (tin chính thức)
	Official bool `json:"official"`
	// PostAuthorID, ParentAuthorID: người nhận thông báo khi có bình luận / trả lời mới
	PostAuthorID   uuid.UUID  `json:"post_author_id"`
	ParentAuthorID *uuid.UUID `json:"parent_author_id,omitempty"`
	// FirstTime: false khi nội dung được khôi phục sau xem xét lại (không báo lại)
	FirstTime bool `json:"first_time"`
}

type Held struct {
	Ref       Ref       `json:"ref"`
	ChannelID uuid.UUID `json:"channel_id"`
	AuthorID  uuid.UUID `json:"author_id"`
}

type AutoDeleted struct {
	Ref       Ref       `json:"ref"`
	ChannelID uuid.UUID `json:"channel_id"`
	AuthorID  uuid.UUID `json:"author_id"`
	Rule      int       `json:"rule"`
}

type Withdrawn struct {
	Ref Ref `json:"ref"`
}
