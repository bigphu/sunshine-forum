// Package domain là thực thể và quy tắc của content: kênh, bài viết, bình luận hai cấp.
// Không biết HTTP hay SQL.
package domain

import (
	"context"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bigphu/goat/errs"
	"github.com/google/uuid"

	"sunshine/internal/content/contract"
	screening "sunshine/internal/screening/contract"
)

const (
	PermRead     = "content.read"
	PermWrite    = "content.write"
	PermAnnounce = "content.announce"
	PermPin      = "content.pin"
)

const MaxTitle = 200

type Channel struct {
	ID           uuid.UUID
	Slug         string
	Name         string
	Description  string
	OfficialOnly bool
}

// Screening là kết quả sàng lọc lưu cùng nội dung
type Screening struct {
	Reason       string
	Label        string
	Score        *float64
	ModelVersion string
	Rule         *int
	Sentences    []screening.Sentence
	NeedsRescan  bool
}

func ScreeningFrom(r screening.Result) Screening {
	return Screening{
		Reason: r.Reason, Label: r.Label, Score: r.Score, ModelVersion: r.ModelVersion,
		Rule: r.Rule, Sentences: r.Sentences, NeedsRescan: r.NeedsRescan,
	}
}

type Post struct {
	ID           uuid.UUID
	ChannelID    uuid.UUID
	AuthorID     uuid.UUID
	Official     bool
	Title        string
	Body         string
	Status       string
	Pinned       bool
	Edited       bool
	CommentCount int
	// OfficialReplied: ban quản lý đã trả lời trong bài
	OfficialReplied bool
	Screening       Screening
	CreatedAt       time.Time
}

type Comment struct {
	ID        uuid.UUID
	PostID    uuid.UUID
	ParentID  *uuid.UUID
	AuthorID  uuid.UUID
	Official  bool
	Body      string
	Status    string
	Edited    bool
	Screening Screening
	CreatedAt time.Time
}

// Editable: tác giả sửa hay xoá được nội dung đang hiển thị hoặc đang chờ duyệt
func Editable(status string) bool {
	return status == contract.StatusPublished || status == contract.StatusPending
}

func CleanTitle(s string) (string, error) {
	t := strings.Join(strings.Fields(s), " ")
	if t == "" || utf8.RuneCountInString(t) > MaxTitle {
		return "", ErrInvalidTitle
	}
	return t, nil
}

// CleanBody: tối đa 5.000 ký tự, đếm theo ký tự chứ không theo byte (FR-2.7)
func CleanBody(s string) (string, error) {
	b := strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
	if b == "" {
		return "", ErrEmptyBody
	}
	if utf8.RuneCountInString(b) > screening.MaxChars {
		return "", ErrBodyTooLong
	}
	return b, nil
}

var officialLike = regexp.MustCompile(`(?i)^\s*(\[?\s*(thông báo|thong bao|tb)\b|ban quản lý|ban quan ly|bql\b)`)

// ImitatesOfficial: bài của cư dân mở đầu như tin của ban quản lý. Giao diện thêm dòng
// "Bài của cư dân, không phải tin chính thức của ban quản lý." (DESIGN.md)
func ImitatesOfficial(official bool, title, body string) bool {
	return !official && (officialLike.MatchString(title) || officialLike.MatchString(body))
}

// Cursor phân trang bảng tin theo (created_at, id)
type Cursor struct {
	At time.Time
	ID uuid.UUID
}

type NewPost struct {
	ChannelID uuid.UUID
	AuthorID  uuid.UUID
	Official  bool
	Title     string
	Body      string
	Status    string
	Screening Screening
}

type NewComment struct {
	PostID    uuid.UUID
	ParentID  *uuid.UUID
	AuthorID  uuid.UUID
	Official  bool
	Body      string
	Status    string
	Screening Screening
}

// Repository của content. Method ghi tự mở transaction và phát event theo trạng thái
// mới trong cùng transaction (held / published / auto_deleted / withdrawn).
type Repository interface {
	Channels(ctx context.Context) ([]Channel, error)
	Channel(ctx context.Context, id uuid.UUID) (Channel, error)
	ChannelBySlug(ctx context.Context, slug string) (Channel, error)

	CreatePost(ctx context.Context, in NewPost) (Post, error)
	GetPost(ctx context.Context, id uuid.UUID) (Post, error)
	// UpdatePost: chỉ tác giả, chỉ khi Editable; ErrNotEditable
	UpdatePost(ctx context.Context, id, author uuid.UUID, title, body, status string, s Screening) (Post, error)
	DeletePost(ctx context.Context, id, author uuid.UUID) error
	SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error
	Feed(ctx context.Context, channel *uuid.UUID, before *Cursor, limit int) ([]Post, error)
	Pinned(ctx context.Context, channel *uuid.UUID) ([]Post, error)
	Search(ctx context.Context, folded string, limit int) ([]Post, error)

	CreateComment(ctx context.Context, in NewComment) (Comment, error)
	GetComment(ctx context.Context, id uuid.UUID) (Comment, error)
	UpdateComment(ctx context.Context, id, author uuid.UUID, body, status string, s Screening) (Comment, error)
	DeleteComment(ctx context.Context, id, author uuid.UUID) error
	Thread(ctx context.Context, postID uuid.UUID) ([]Comment, error)

	PendingByAuthor(ctx context.Context, author uuid.UUID) ([]Post, []Comment, error)

	// SetStatus đổi trạng thái theo quyết định kiểm duyệt nếu trạng thái hiện tại nằm
	// trong from; phát content.published khi nội dung hiện lại. false: không đổi gì.
	SetStatus(ctx context.Context, ref contract.Ref, to string, from []string, firstTime bool) (bool, error)

	ToRescan(ctx context.Context, limit int) ([]contract.Ref, error)
	// ApplyRescan ghi kết quả quét lại; sau sự cố không tự xoá, chỉ giữ lại chờ duyệt
	ApplyRescan(ctx context.Context, ref contract.Ref, status string, s Screening) error

	Items(ctx context.Context, refs []contract.Ref) (map[contract.Ref]contract.Item, error)
	Stats(ctx context.Context, since time.Time) (contract.Stats, map[string]int64, error)
	Purge(ctx context.Context, days int) (int64, error)
}

var (
	ErrInvalidTitle = errs.Unprocessable("Tiêu đề cần từ 1 đến 200 ký tự.",
		errs.WithType("/errors/invalid-title"), errs.WithTitle("Invalid title"),
		errs.WithFields(errs.FieldInvalid("title", "Tiêu đề cần từ 1 đến 200 ký tự.")))
	ErrEmptyBody = errs.Unprocessable("Nội dung không được để trống.",
		errs.WithType("/errors/empty-body"), errs.WithTitle("Empty body"),
		errs.WithFields(errs.FieldInvalid("body", "Nội dung không được để trống.")))
	ErrBodyTooLong = errs.Unprocessable("Nội dung dài quá 5.000 ký tự. Hãy rút gọn trước khi gửi.",
		errs.WithType("/errors/body-too-long"), errs.WithTitle("Body too long"),
		errs.WithFields(errs.FieldInvalid("body", "Nội dung dài quá 5.000 ký tự.")))
	ErrChannelNotFound = errs.NotFound("Kênh không tồn tại.",
		errs.WithType("/errors/channel-not-found"), errs.WithTitle("Channel not found"))
	ErrPostNotFound = errs.NotFound("Không tìm thấy bài viết.",
		errs.WithType("/errors/post-not-found"), errs.WithTitle("Post not found"))
	ErrCommentNotFound = errs.NotFound("Không tìm thấy bình luận.",
		errs.WithType("/errors/comment-not-found"), errs.WithTitle("Comment not found"))
	ErrOfficialOnly = errs.Forbidden("Chỉ ban quản lý được đăng bài trong kênh này. Bạn vẫn có thể bình luận.",
		errs.WithType("/errors/official-only-channel"), errs.WithTitle("Official channel"))
	ErrNotEditable = errs.Conflict("Không thể sửa hoặc xoá nội dung này.",
		errs.WithType("/errors/not-editable"), errs.WithTitle("Content cannot be changed"))
	ErrPostClosed = errs.Conflict("Bài viết này không còn nhận bình luận.",
		errs.WithType("/errors/post-closed"), errs.WithTitle("Post not open for comments"))
	ErrInvalidParent = errs.Unprocessable("Bình luận được trả lời không tồn tại.",
		errs.WithType("/errors/invalid-parent"), errs.WithTitle("Invalid parent comment"))
)

// ReadOnly: tài khoản đang tạm chỉ đọc (FR-1.3)
func ErrReadOnly(until time.Time) error {
	vn := time.FixedZone("ICT", 7*3600)
	return errs.Forbidden("Tài khoản của bạn tạm thời chỉ được đọc đến "+until.In(vn).Format("15:04 02/01/2006")+".",
		errs.WithType("/errors/read-only"), errs.WithTitle("Account is read-only"))
}
