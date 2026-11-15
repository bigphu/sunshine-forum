// Package domain là quy tắc của moderation: case trong hàng chờ, quyết định, yêu cầu xem
// xét lại và nhật ký kiểm toán. Không biết HTTP hay SQL.
package domain

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bigphu/goat/errs"
	"github.com/google/uuid"

	content "sunshine/internal/content/contract"
	"sunshine/internal/moderation/contract"
	screening "sunshine/internal/screening/contract"
)

const (
	PermAppealCreate = "moderation.appeal.create"
	PermReview       = "moderation.review"
	PermRemove       = "moderation.remove"
	PermAppealReview = "moderation.appeal.review"
	PermAuditRead    = "moderation.audit.read"
)

// AppealWindow: yêu cầu xem xét lại trong vòng 30 ngày (FR-4.4)
const AppealWindow = 30 * 24 * time.Hour

const (
	CaseOpen      = "open"
	CaseDecided   = "decided"
	CaseWithdrawn = "withdrawn"

	AppealPending  = "pending"
	AppealAccepted = "accepted"
	AppealDeclined = "declined"
)

type Case struct {
	ID        uuid.UUID
	Ref       content.Ref
	AuthorID  uuid.UUID
	ChannelID uuid.UUID
	CreatedAt time.Time
}

// AI là kết luận của bộ lọc lưu cùng quyết định, để đo bộ lọc (FR-5.3)
type AI struct {
	Reason       string
	Label        string
	Score        *float64
	ModelVersion string
}

// Flagged: bộ lọc đã kết luận nội dung có vấn đề (mô hình gắn nhãn vi phạm, hoặc khớp blocklist).
// Nội dung bị giữ vì khác ngôn ngữ hay vì sự cố không phải kết luận của AI.
func (a AI) Flagged() bool {
	return a.Reason == screening.ReasonBlocklist || a.Label == "offensive" || a.Label == "hate"
}

func AIFrom(s content.Screening) AI {
	return AI{Reason: s.Reason, Label: s.Label, Score: s.Score, ModelVersion: s.ModelVersion}
}

type Decision struct {
	ID             uuid.UUID
	CaseID         *uuid.UUID
	Ref            content.Ref
	AuthorID       uuid.UUID
	ChannelID      uuid.UUID
	Action         string
	Rule           int
	ModeratorID    *uuid.UUID
	AI             AI
	OverturnedAI   bool
	AppealDeadline *time.Time
	CreatedAt      time.Time
}

// Appealable: chỉ nội dung bị từ chối, bị gỡ hay bị tự xoá, trong hạn, và chưa từng yêu cầu
func (d Decision) Appealable(now time.Time) bool {
	return d.Action != contract.ActionApprove && d.AppealDeadline != nil && now.Before(*d.AppealDeadline)
}

type Appeal struct {
	ID         uuid.UUID
	DecisionID uuid.UUID
	AuthorID   uuid.UUID
	Reason     string
	Status     string
	ReviewerID *uuid.UUID
	CreatedAt  time.Time
	DecidedAt  *time.Time
}

// CleanAppealReason: bắt buộc, 20 đến 2.000 ký tự (PRODUCT.md)
func CleanAppealReason(s string) (string, error) {
	r := strings.TrimSpace(s)
	if n := utf8.RuneCountInString(r); n < 20 || n > 2000 {
		return "", ErrAppealReason
	}
	return r, nil
}

// AuditEntry không chứa nội dung bài hay dữ liệu cá nhân; Subject là mã ẩn danh
type AuditEntry struct {
	ID           int64
	Action       string
	ModeratorID  *uuid.UUID
	Subject      string
	ContentType  string
	AI           AI
	Rule         *int
	Outcome      string
	OverturnedAI bool
	CreatedAt    time.Time
}

// NewDecision là một quyết định sắp ghi, kèm dòng nhật ký ghi cùng transaction
type NewDecision struct {
	Decision Decision
	Audit    AuditEntry
	// CloseCase: đóng case mở của nội dung (nếu có) với trạng thái decided
	CloseCase bool
}

type DecisionWithAppeal struct {
	Decision
	Appeal *Appeal
}

type Repository interface {
	OpenCase(ctx context.Context, ref content.Ref, author, channel uuid.UUID) error
	CloseCase(ctx context.Context, ref content.Ref, status string) error
	OpenCases(ctx context.Context) ([]Case, error)
	// HasOpenCase: nội dung đang chờ duyệt (khoá case tới hết transaction của Record)
	HasOpenCase(ctx context.Context, ref content.Ref) (bool, error)

	// Record ghi các quyết định, dòng nhật ký và event trong một transaction. Case đã
	// được người khác xử lý thì bỏ qua mục đó; trả số mục đã ghi.
	Record(ctx context.Context, items []NewDecision, requireOpenCase bool) (int, error)

	Decision(ctx context.Context, id uuid.UUID) (Decision, error)
	Decisions(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Decision, error)
	DecisionsByAuthor(ctx context.Context, author uuid.UUID) ([]DecisionWithAppeal, error)

	// CreateAppeal: ErrAlreadyAppealed nếu đã có
	CreateAppeal(ctx context.Context, a Appeal) (Appeal, error)
	PendingAppeals(ctx context.Context) ([]Appeal, error)
	// DecideAppeal cập nhật yêu cầu, ghi nhật ký và event trong một transaction.
	// ErrAppealClosed nếu đã được xem xét.
	DecideAppeal(ctx context.Context, appealID uuid.UUID, accepted bool, reviewer uuid.UUID, audit func(Decision) AuditEntry) (Appeal, Decision, error)

	Audit(ctx context.Context, beforeID *int64, filter string, limit int) ([]AuditEntry, error)
	// AuditConfig ghi một dòng đổi cấu hình từ event; eventID chống ghi trùng
	AuditConfig(ctx context.Context, eventID uuid.UUID, actor *uuid.UUID, outcome string) error
	Stats(ctx context.Context, since time.Time) (contract.Stats, error)
	Corrections(ctx context.Context, from, to time.Time) ([]DecisionWithAppeal, error)
}

var (
	ErrRuleRequired = errs.Unprocessable("Hãy chọn quy tắc bị vi phạm.",
		errs.WithType("/errors/rule-required"), errs.WithTitle("Rule required"),
		errs.WithFields(errs.FieldInvalid("rule", "Hãy chọn quy tắc bị vi phạm.")))
	ErrNothingSelected = errs.Unprocessable("Chưa chọn nội dung nào.",
		errs.WithType("/errors/nothing-selected"), errs.WithTitle("No items"))
	ErrNotPublished = errs.Conflict("Chỉ gỡ được nội dung đang hiển thị.",
		errs.WithType("/errors/not-published"), errs.WithTitle("Content is not published"))
	ErrContentNotFound = errs.NotFound("Không tìm thấy nội dung.",
		errs.WithType("/errors/content-not-found"), errs.WithTitle("Content not found"))
	ErrDecisionNotFound = errs.NotFound("Không tìm thấy quyết định.",
		errs.WithType("/errors/decision-not-found"), errs.WithTitle("Decision not found"))
	ErrAppealReason = errs.Unprocessable("Lý do cần từ 20 đến 2.000 ký tự.",
		errs.WithType("/errors/appeal-reason"), errs.WithTitle("Invalid appeal reason"),
		errs.WithFields(errs.FieldInvalid("reason", "Lý do cần từ 20 đến 2.000 ký tự.")))
	ErrNotAppealable = errs.Conflict("Quyết định này không còn yêu cầu xem xét lại được.",
		errs.WithType("/errors/not-appealable"), errs.WithTitle("Decision cannot be appealed"))
	ErrAlreadyAppealed = errs.Conflict("Mỗi quyết định chỉ được yêu cầu xem xét lại một lần.",
		errs.WithType("/errors/already-appealed"), errs.WithTitle("Already appealed"))
	ErrAppealNotFound = errs.NotFound("Không tìm thấy yêu cầu xem xét lại.",
		errs.WithType("/errors/appeal-not-found"), errs.WithTitle("Appeal not found"))
	ErrAppealClosed = errs.Conflict("Yêu cầu này đã được xem xét.",
		errs.WithType("/errors/appeal-closed"), errs.WithTitle("Appeal already decided"))
	// Người đã ra quyết định không tự xét lại quyết định của mình (PRODUCT.md)
	ErrSameModerator = errs.Forbidden("Bạn đã ra quyết định ban đầu. Một quản lý khác hoặc Quản trị viên cần xem xét yêu cầu này.",
		errs.WithType("/errors/same-moderator"), errs.WithTitle("Another moderator must review"))
)
