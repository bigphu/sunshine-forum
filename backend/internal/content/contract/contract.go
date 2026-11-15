// Package contract là phần công khai của content: moderation đọc nội dung bị giữ để hiện
// trong hàng chờ, administration đọc văn bản cho bản xuất huấn luyện, notifications đọc
// tên kênh và tiêu đề bài. Không kiểm tra quyền: module gọi tự quyết định.
package contract

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	TypePost    = "post"
	TypeComment = "comment"
)

const (
	StatusPublished   = "published"
	StatusPending     = "pending"
	StatusRejected    = "rejected"
	StatusRemoved     = "removed"
	StatusAutoDeleted = "auto_deleted"
	StatusDeleted     = "deleted"
)

// Ref trỏ tới một bài hoặc một bình luận
type Ref struct {
	Type string    `json:"type"`
	ID   uuid.UUID `json:"id"`
}

// Screening là kết quả sàng lọc gần nhất của nội dung
type Screening struct {
	Reason       string // model | blocklist | language | outage
	Label        string // clean | offensive | hate
	Score        *float64
	ModelVersion string
	Rule         *int
	// Sentences: đoạn bị gắn cờ, JSON [{start,end,top_label,score}] theo vị trí ký tự
	Sentences json.RawMessage
}

// Item là một bài hoặc bình luận với đủ thông tin cho hàng chờ
type Item struct {
	Ref         Ref
	PostID      uuid.UUID
	PostTitle   string // bài chứa bình luận, hoặc chính bài đó
	Title       string // rỗng với bình luận
	Body        string
	ChannelID   uuid.UUID
	ChannelName string
	AuthorID    uuid.UUID
	Status      string
	Screening   Screening
	CreatedAt   time.Time
}

type Channel struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	OfficialOnly bool
}

type Stats struct {
	Submitted int64
	Flagged   int64
	// Theo nhãn: mô hình (hate, offensive), từ khoá chặn, ngôn ngữ khác
	Hate, Offensive, Blocklist, Language int64
	// FlaggedByDay: số nội dung bị bộ lọc giữ lại mỗi ngày (giờ Việt Nam), cũ nhất trước
	FlaggedByDay []DayCount
}

type DayCount struct {
	Day   time.Time
	Count int64
}

type Reader interface {
	// Items trả các nội dung tìm thấy, bỏ qua ref không tồn tại
	Items(ctx context.Context, refs []Ref) (map[Ref]Item, error)
	Channel(ctx context.Context, id uuid.UUID) (Channel, error)
	Stats(ctx context.Context, since time.Time, days int) (Stats, error)
}
