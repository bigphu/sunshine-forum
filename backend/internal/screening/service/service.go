// Package service là use case của screening: sàng lọc khi gửi (Screener cho content),
// kiểm tra bản nháp, và cài đặt của Admin. Cấu hình được cache và làm mới khi DB phát
// NOTIFY, nên đổi ngưỡng có hiệu lực ngay mà không cần khởi động lại (FR-5.1).
package service

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/logger"
	"github.com/google/uuid"

	"sunshine/internal/screening/contract"
	"sunshine/internal/screening/domain"
	"sunshine/internal/screening/scorer"
)

// ModelTimeout: NFR-1.2 cho mô hình 200 ms, gồm cả đường mạng hai chiều
const ModelTimeout = 200 * time.Millisecond

// cacheTTL: lưới an toàn nếu NOTIFY bị lỡ
const cacheTTL = 30 * time.Second

type Service struct {
	repo    domain.Repository
	scorer  scorer.Scorer
	standIn bool

	mu       sync.Mutex
	snap     domain.Snapshot
	loadedAt time.Time
}

// New: standIn cho biết scorer là StandIn (bật nút giả lập sự cố)
func New(repo domain.Repository, sc scorer.Scorer, standIn bool) *Service {
	return &Service{repo: repo, scorer: sc, standIn: standIn}
}

var _ contract.Screener = (*Service)(nil)

func (s *Service) snapshot(ctx context.Context) (domain.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.loadedAt.IsZero() && time.Since(s.loadedAt) < cacheTTL {
		return s.snap, nil
	}
	snap, err := s.repo.Snapshot(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	s.snap, s.loadedAt = snap, time.Now()
	return snap, nil
}

func (s *Service) invalidate() {
	s.mu.Lock()
	s.loadedAt = time.Time{}
	s.mu.Unlock()
}

// Watch chạy tới khi ctx bị huỷ; cmd/server và cmd/worker gọi trong goroutine
func (s *Service) Watch(ctx context.Context) error { return s.repo.Watch(ctx, s.invalidate) }

// StandInOffline cho StandIn: đọc cờ giả lập sự cố từ cấu hình đã cache
func (s *Service) StandInOffline(ctx context.Context) bool {
	snap, err := s.snapshot(ctx)
	return err == nil && snap.Settings.StandInOffline
}

func (s *Service) score(ctx context.Context, text string) (scorer.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, ModelTimeout)
	defer cancel()
	return s.scorer.Score(ctx, text)
}

// Screen: blocklist, kiểm tra ngôn ngữ, mô hình, rồi ngưỡng (contract.Screener)
func (s *Service) Screen(ctx context.Context, text string, t contract.Target) (contract.Result, error) {
	snap, err := s.snapshot(ctx)
	if err != nil {
		return contract.Result{}, err
	}
	// 1. Blocklist: mỗi mục mang quy tắc riêng, khớp thì giữ lại chờ duyệt
	if b := snap.MatchBlock(text); b != nil {
		rule := b.Rule
		return contract.Result{Outcome: contract.OutcomePending, Reason: contract.ReasonBlocklist, Rule: &rule}, nil
	}
	// 2. Ngôn ngữ khác: mô hình chỉ hiểu tiếng Việt, nên giữ lại mà không chấm (FR-3.4).
	// Không nêu quy tắc và không tính vi phạm.
	if !domain.IsVietnamese(text) {
		return contract.Result{Outcome: contract.OutcomePending, Reason: contract.ReasonLanguage}, nil
	}
	// 3. Mô hình; lỗi hoặc quá 200 ms thì áp chính sách sự cố (NFR-4.2)
	res, err := s.score(ctx, text)
	if err != nil {
		logger.FromContext(ctx).WarnContext(ctx, "model unavailable, applying outage policy",
			slog.String("policy", snap.Settings.OutagePolicy), slog.String("err", err.Error()))
		out := contract.Result{Outcome: contract.OutcomePending, Reason: contract.ReasonOutage, NeedsRescan: true}
		if snap.Settings.OutagePolicy == domain.PolicyFailOpen && !t.AnnouncementComment {
			out.Outcome = contract.OutcomePublished
		}
		return out, nil
	}
	return domain.Decide(res, snap.Settings, snap.HoldFor(t.ChannelID), snap.AutoDeleteFor(t.ChannelID)), nil
}

func (s *Service) Worse(a, b contract.Result) contract.Result { return domain.Worse(a, b) }

// healthChecker: scorer có điểm kiểm tra sức khoẻ riêng (scorer.HTTP gọi /health). Scorer
// không có thì chấm thử một câu ngắn.
type healthChecker interface {
	Health(ctx context.Context) (string, error)
}

// probe trả về phiên bản mô hình nếu mô hình đang phục vụ được
func (s *Service) probe(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, ModelTimeout)
	defer cancel()
	if h, ok := s.scorer.(healthChecker); ok {
		return h.Health(ctx)
	}
	res, err := s.scorer.Score(ctx, "kiểm tra")
	return res.ModelVersion, err
}

func (s *Service) ModelOnline(ctx context.Context) bool {
	_, err := s.probe(ctx)
	return err == nil
}

// Check là kiểm tra bản nháp trước khi gửi (FR-3.1). Bản nháp không được lưu; cảnh báo
// không tính là vi phạm.
type CheckResult struct {
	Warnings     []domain.Warning
	ModelOffline bool
}

func (s *Service) Check(ctx context.Context, text string, channel uuid.UUID) (CheckResult, error) {
	if _, err := auth.Require(ctx, domain.PermCheck); err != nil {
		return CheckResult{}, err
	}
	snap, err := s.snapshot(ctx)
	if err != nil {
		return CheckResult{}, err
	}
	out := CheckResult{}
	if b := snap.MatchBlock(text); b != nil {
		if w, ok := domain.RuleWarning(b.Rule); ok {
			out.Warnings = append(out.Warnings, w)
		}
	}
	if domain.IsVietnamese(text) {
		res, err := s.score(ctx, text)
		if err != nil {
			out.ModelOffline = true
		} else {
			hold := snap.HoldFor(channel)
			hate, off := res.Labels["hate"], res.Labels["offensive"]
			switch {
			case hate >= hold && hate >= off:
				out.Warnings = append(out.Warnings, domain.HateWarning(filter(res.Sentences, "hate")))
			case off >= hold:
				out.Warnings = append(out.Warnings, domain.OffensiveWarning(filter(res.Sentences, "offensive")))
			}
		}
	}
	if domain.HasPhoneNumber(text) {
		w, _ := domain.RuleWarning(5)
		out.Warnings = append(out.Warnings, w)
	}
	return out, nil
}

func filter(ss []contract.Sentence, label string) []contract.Sentence {
	var out []contract.Sentence
	for _, x := range ss {
		if x.TopLabel == label {
			out = append(out, x)
		}
	}
	return out
}

// Blocklist cho trình duyệt: danh sách từ khoá chạy ngay khi gõ (FR-3.1)
func (s *Service) Blocklist(ctx context.Context) ([]domain.BlockEntry, error) {
	if _, err := auth.Require(ctx, domain.PermCheck); err != nil {
		return nil, err
	}
	snap, err := s.snapshot(ctx)
	return snap.Block, err
}

// Status cho composer: báo cho cư dân biết việc kiểm tra đang chậm
type Status struct {
	ModelOnline  bool
	ModelVersion string
	StandIn      bool
}

func (s *Service) Status(ctx context.Context) (Status, error) {
	if _, err := auth.Require(ctx, domain.PermCheck); err != nil {
		return Status{}, err
	}
	version, err := s.probe(ctx)
	return Status{ModelOnline: err == nil, ModelVersion: version, StandIn: s.standIn}, nil
}

// --- Cài đặt của Admin ---

func (s *Service) Settings(ctx context.Context) (domain.Snapshot, error) {
	if _, err := auth.Require(ctx, domain.PermSettingsManage); err != nil {
		return domain.Snapshot{}, err
	}
	return s.repo.Snapshot(ctx)
}

func (s *Service) UpdateSettings(ctx context.Context, in domain.Settings) (domain.Settings, error) {
	if _, err := auth.Require(ctx, domain.PermSettingsManage); err != nil {
		return domain.Settings{}, err
	}
	if err := in.Validate(); err != nil {
		return domain.Settings{}, err
	}
	cur, err := s.repo.Snapshot(ctx)
	if err != nil {
		return domain.Settings{}, err
	}
	old := cur.Settings
	var changes []contract.Change
	add := func(field, o, n string) {
		if o != n {
			changes = append(changes, contract.Change{Field: field, Old: o, New: n})
		}
	}
	add("auto_delete_enabled", fmt.Sprint(old.AutoDeleteEnabled), fmt.Sprint(in.AutoDeleteEnabled))
	add("hold_threshold", score(old.HoldThreshold), score(in.HoldThreshold))
	add("auto_delete_threshold", score(old.AutoDeleteThreshold), score(in.AutoDeleteThreshold))
	add("outage_policy", old.OutagePolicy, in.OutagePolicy)
	out, err := s.repo.UpdateSettings(ctx, in, changes)
	s.invalidate()
	return out, err
}

// score: ngưỡng trên thang /100 như giao diện hiển thị
func score(v float64) string { return strconv.Itoa(int(math.Round(v * 100))) }

func scoreP(v *float64) string {
	if v == nil {
		return "chung"
	}
	return score(*v)
}

// SetChannelThreshold đặt ngưỡng riêng của kênh; t nil là bỏ, dùng lại ngưỡng chung
func (s *Service) SetChannelThreshold(ctx context.Context, channel uuid.UUID, t *domain.ChannelThreshold) error {
	if _, err := auth.Require(ctx, domain.PermSettingsManage); err != nil {
		return err
	}
	if t != nil {
		if err := t.Validate(); err != nil {
			return err
		}
	}
	cur, err := s.repo.Snapshot(ctx)
	if err != nil {
		return err
	}
	oldHold, newHold := cur.HoldFor(channel), cur.Settings.HoldThreshold
	var oldDel, newDel *float64
	if o, ok := cur.Channels[channel]; ok {
		oldDel = o.AutoDelete
	}
	if t != nil {
		newHold, newDel = t.Hold, t.AutoDelete
	}
	var changes []contract.Change
	if score(oldHold) != score(newHold) {
		changes = append(changes, contract.Change{Field: "channel_hold", ChannelID: &channel, Old: score(oldHold), New: score(newHold)})
	}
	if scoreP(oldDel) != scoreP(newDel) {
		changes = append(changes, contract.Change{Field: "channel_auto_delete", ChannelID: &channel, Old: scoreP(oldDel), New: scoreP(newDel)})
	}
	err = s.repo.SetChannelThreshold(ctx, channel, t, changes)
	s.invalidate()
	return err
}

func (s *Service) AddBlock(ctx context.Context, term string, rule int) (domain.BlockEntry, error) {
	actor, err := auth.Require(ctx, domain.PermSettingsManage)
	if err != nil {
		return domain.BlockEntry{}, err
	}
	term = strings.Join(strings.Fields(term), " ")
	if domain.Fold(term) == "" || !contract.ValidRule(rule) {
		return domain.BlockEntry{}, domain.ErrInvalidTerm
	}
	b, err := s.repo.AddBlock(ctx, term, rule, actor.AccountID)
	s.invalidate()
	return b, err
}

func (s *Service) DeleteBlock(ctx context.Context, id uuid.UUID) error {
	if _, err := auth.Require(ctx, domain.PermSettingsManage); err != nil {
		return err
	}
	err := s.repo.DeleteBlock(ctx, id)
	s.invalidate()
	return err
}

// SetStandInOffline giả lập sự cố mô hình (chỉ với StandIn), để thử chính sách sự cố
func (s *Service) SetStandInOffline(ctx context.Context, offline bool) error {
	if _, err := auth.Require(ctx, domain.PermSettingsManage); err != nil {
		return err
	}
	if !s.standIn {
		return domain.ErrStandInOnly
	}
	err := s.repo.SetStandInOffline(ctx, offline)
	s.invalidate()
	return err
}

// IsStandIn: scorer đang dùng là StandIn (điểm mẫu)
func (s *Service) IsStandIn() bool { return s.standIn }
