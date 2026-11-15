// Package domain là thống kê bộ lọc và bản xuất dữ liệu huấn luyện của administration.
// Không biết HTTP hay SQL.
package domain

import (
	"context"
	"regexp"
	"time"

	"github.com/bigphu/goat/errs"
	"github.com/google/uuid"
)

const (
	PermAnalyticsRead = "administration.analytics.read"
	PermExportManage  = "administration.export.manage"
)

// Cửa sổ của bản xuất: quyết định đã qua hạn xem xét lại (30 ngày) nhưng bản gốc chưa
// tới hạn xoá (90 ngày). SYSTEM_DESIGN §3, "Retraining export".
const (
	ExportMinAge = 30 * 24 * time.Hour
	ExportMaxAge = 89 * 24 * time.Hour
	RetainFor    = 90 * 24 * time.Hour
)

type Export struct {
	ID          uuid.UUID
	CreatedBy   uuid.UUID
	ItemCount   int
	Purpose     string // training | evaluation
	PeriodFrom  time.Time
	PeriodTo    time.Time
	DeleteAfter time.Time
	CreatedAt   time.Time
}

// Sample là một dòng của bản xuất: không có mã người dùng hay mã bài viết
type Sample struct {
	Text         string   `json:"text"`
	Label        string   `json:"label"`
	Rule         int      `json:"rule"`
	AIScore      *float64 `json:"ai_score"`
	ModelVersion string   `json:"model_version"`
}

var (
	phonePattern = regexp.MustCompile(`(?:\+84|0)[\s.\-]?\d{2,3}[\s.\-]?\d{3}[\s.\-]?\d{3,4}`)
	emailPattern = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
	plate        = regexp.MustCompile(`\b\d{2}[A-Z]{1,2}[\s\-]?\d{3}[.\s]?\d{2}\b`)
)

// Scrub thay số điện thoại, email và biển số xe trước khi văn bản rời DB
func Scrub(s string) string {
	s = emailPattern.ReplaceAllString(s, "[email]")
	s = phonePattern.ReplaceAllString(s, "[số điện thoại]")
	return plate.ReplaceAllString(s, "[biển số]")
}

type Repository interface {
	Create(ctx context.Context, e Export, data []byte) (Export, error)
	List(ctx context.Context) ([]Export, error)
	Data(ctx context.Context, id uuid.UUID) ([]byte, time.Time, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Purge(ctx context.Context) (int64, error)
}

var (
	ErrNothingToExport = errs.Conflict("Chưa có sửa đổi nào đủ điều kiện để xuất.",
		errs.WithType("/errors/nothing-to-export"), errs.WithTitle("Nothing to export"))
	ErrExportNotFound = errs.NotFound("Bản xuất không còn (đã bị xoá hoặc tự xoá).",
		errs.WithType("/errors/export-not-found"), errs.WithTitle("Export not found"))
)
