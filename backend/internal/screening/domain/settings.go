package domain

import (
	"context"
	"time"

	"github.com/bigphu/goat/errs"
	"github.com/google/uuid"

	"sunshine/internal/screening/contract"
	"sunshine/internal/screening/scorer"
)

const (
	PermCheck          = "screening.check"
	PermSettingsManage = "screening.settings.manage"
)

const (
	PolicyFailClosed = "fail_closed"
	PolicyFailOpen   = "fail_open"
)

// Settings là ngưỡng Admin chỉnh được (FR-5.1). Điểm 0–1.
type Settings struct {
	HoldThreshold       float64
	AutoDeleteEnabled   bool
	AutoDeleteThreshold float64
	OutagePolicy        string
	StandInOffline      bool
	UpdatedAt           time.Time
}

func (s Settings) Validate() error {
	if !validThreshold(s.HoldThreshold) || !validThreshold(s.AutoDeleteThreshold) {
		return ErrInvalidThreshold
	}
	if s.AutoDeleteThreshold <= s.HoldThreshold {
		return ErrAutoDeleteBelowHold
	}
	if s.OutagePolicy != PolicyFailClosed && s.OutagePolicy != PolicyFailOpen {
		return ErrInvalidPolicy
	}
	return nil
}

func validThreshold(v float64) bool { return v > 0 && v < 1 }

func ValidThreshold(v float64) error {
	if !validThreshold(v) {
		return ErrInvalidThreshold
	}
	return nil
}

func (t ChannelThreshold) Validate() error {
	if !validThreshold(t.Hold) || (t.AutoDelete != nil && !validThreshold(*t.AutoDelete)) {
		return ErrInvalidThreshold
	}
	if t.AutoDelete != nil && *t.AutoDelete <= t.Hold {
		return ErrAutoDeleteBelowHold
	}
	return nil
}

type BlockEntry struct {
	ID        uuid.UUID
	Term      string
	Rule      int
	CreatedBy *uuid.UUID
	CreatedAt time.Time
}

// ChannelThreshold là ngưỡng riêng của một kênh (FR-5.2); AutoDelete nil là dùng ngưỡng chung
type ChannelThreshold struct {
	Hold       float64
	AutoDelete *float64
}

// Snapshot là toàn bộ cấu hình sàng lọc, cache trong bộ nhớ và làm mới khi DB phát
// NOTIFY screening_settings
type Snapshot struct {
	Settings Settings
	Channels map[uuid.UUID]ChannelThreshold
	Block    []BlockEntry
}

func (s Snapshot) HoldFor(channel uuid.UUID) float64 {
	if v, ok := s.Channels[channel]; ok {
		return v.Hold
	}
	return s.Settings.HoldThreshold
}

func (s Snapshot) AutoDeleteFor(channel uuid.UUID) float64 {
	if v, ok := s.Channels[channel]; ok && v.AutoDelete != nil {
		return *v.AutoDelete
	}
	return s.Settings.AutoDeleteThreshold
}

// MatchBlock trả mục blocklist đầu tiên khớp (so khớp theo cụm từ, sau khi bỏ dấu)
func (s Snapshot) MatchBlock(text string) *BlockEntry {
	folded := Fold(text)
	for i := range s.Block {
		if ContainsPhrase(folded, Fold(s.Block[i].Term)) {
			return &s.Block[i]
		}
	}
	return nil
}

func ptr[T any](v T) *T { return &v }

// Decide áp ngưỡng lên điểm mô hình (COMMUNITY_RULES phần 3, "How a decision gets its
// rule"): hate trên ngưỡng luôn bị giữ để quản lý chọn Quy tắc 1–3 và được ưu tiên hơn
// offensive; offensive mang Quy tắc 4 và chỉ bị tự xoá khi Admin đã bật.
func Decide(r scorer.Result, s Settings, hold, autoDelete float64) contract.Result {
	off, hate := r.Labels["offensive"], r.Labels["hate"]
	out := contract.Result{
		Outcome: contract.OutcomePublished, Reason: contract.ReasonModel,
		ModelVersion: r.ModelVersion, Sentences: r.Sentences,
	}
	switch {
	case hate >= hold:
		out.Outcome, out.Label, out.Score = contract.OutcomePending, "hate", ptr(hate)
	case off >= hold:
		out.Outcome, out.Label, out.Score, out.Rule = contract.OutcomePending, "offensive", ptr(off), ptr(4)
		if s.AutoDeleteEnabled && off >= autoDelete {
			out.Outcome = contract.OutcomeAutoDeleted
		}
	default:
		out.Label, out.Score, out.Sentences = "clean", ptr(max(off, hate)), nil
	}
	return out
}

var severity = map[string]int{contract.OutcomePublished: 0, contract.OutcomePending: 1, contract.OutcomeAutoDeleted: 2}

// Worse chọn kết quả nặng hơn; bằng nhau thì chọn điểm cao hơn
func Worse(a, b contract.Result) contract.Result {
	if severity[b.Outcome] != severity[a.Outcome] {
		if severity[b.Outcome] > severity[a.Outcome] {
			return b
		}
		return a
	}
	if b.Score != nil && (a.Score == nil || *b.Score > *a.Score) {
		return b
	}
	return a
}

// Repository của screening. Ghi cài đặt phát NOTIFY (trigger trong DB).
type Repository interface {
	Snapshot(ctx context.Context) (Snapshot, error)
	// UpdateSettings và SetChannelThreshold ghi kèm event settings_changed (changes) trong
	// cùng transaction; changes rỗng thì không phát event
	UpdateSettings(ctx context.Context, s Settings, changes []contract.Change) (Settings, error)
	SetStandInOffline(ctx context.Context, offline bool) error
	SetChannelThreshold(ctx context.Context, channel uuid.UUID, t *ChannelThreshold, changes []contract.Change) error
	AddBlock(ctx context.Context, term string, rule int, by uuid.UUID) (BlockEntry, error)
	DeleteBlock(ctx context.Context, id uuid.UUID) error
	// Watch gọi onChange mỗi khi cấu hình đổi, tới khi ctx bị huỷ
	Watch(ctx context.Context, onChange func()) error
}

var (
	ErrInvalidThreshold = errs.Unprocessable("Ngưỡng phải nằm trong khoảng 1 đến 99.",
		errs.WithType("/errors/invalid-threshold"), errs.WithTitle("Invalid threshold"))
	ErrAutoDeleteBelowHold = errs.Unprocessable("Ngưỡng tự động xoá phải cao hơn ngưỡng giữ lại.",
		errs.WithType("/errors/auto-delete-below-hold"), errs.WithTitle("Auto-delete threshold too low"))
	ErrInvalidPolicy = errs.Unprocessable("Chính sách khi mô hình gặp sự cố không hợp lệ.",
		errs.WithType("/errors/invalid-outage-policy"), errs.WithTitle("Invalid outage policy"))
	ErrInvalidTerm = errs.Unprocessable("Cần nhập từ khoá và chọn quy tắc.",
		errs.WithType("/errors/invalid-blocklist-term"), errs.WithTitle("Invalid blocklist term"))
	ErrTermExists = errs.Conflict("Từ khoá này đã có trong danh sách.",
		errs.WithType("/errors/blocklist-term-exists"), errs.WithTitle("Term already listed"))
	ErrTermNotFound = errs.NotFound("Không tìm thấy từ khoá.",
		errs.WithType("/errors/blocklist-term-not-found"), errs.WithTitle("Term not found"))
	ErrStandInOnly = errs.Conflict("Chỉ giả lập sự cố được khi đang dùng dịch vụ chấm điểm tạm.",
		errs.WithType("/errors/stand-in-only"), errs.WithTitle("Only available with the stand-in scorer"))
)
