// Package service là use case của penalties: tình trạng của tài khoản, cài đặt của Admin,
// và phản ứng với quyết định kiểm duyệt (subscriber).
package service

import (
	"context"
	"strconv"

	"github.com/bigphu/goat/auth"
	"github.com/google/uuid"

	moderation "sunshine/internal/moderation/contract"
	"sunshine/internal/penalties/contract"
	"sunshine/internal/penalties/domain"
)

type Service struct{ repo domain.Repository }

func New(repo domain.Repository) *Service { return &Service{repo: repo} }

var _ contract.StandingReader = (*Service)(nil)

// Standing không kiểm tra quyền: module khác (content) hỏi trước khi cho viết
func (s *Service) Standing(ctx context.Context, account uuid.UUID) (contract.Standing, error) {
	cfg, err := s.repo.Settings(ctx)
	if err != nil {
		return contract.Standing{}, err
	}
	t, err := s.repo.ActiveTimeout(ctx, account)
	if err != nil {
		return contract.Standing{}, err
	}
	n, err := s.repo.ActiveStrikes(ctx, account, cfg.StrikeWindowDays)
	if err != nil {
		return contract.Standing{}, err
	}
	out := contract.Standing{ActiveStrikes: n}
	if t != nil {
		out.ReadOnlyUntil, out.LastRule = &t.EndsAt, t.Rule
	}
	return out, nil
}

// MyStanding cho trang "Yêu cầu của tôi" và thẻ tạm chỉ đọc
func (s *Service) MyStanding(ctx context.Context) (contract.Standing, domain.Settings, error) {
	actor, ok := auth.FromContext(ctx)
	if !ok || actor.IsSystem() {
		return contract.Standing{}, domain.Settings{}, auth.ErrUnauthenticated
	}
	st, err := s.Standing(ctx, actor.AccountID)
	if err != nil {
		return contract.Standing{}, domain.Settings{}, err
	}
	cfg, err := s.repo.Settings(ctx)
	return st, cfg, err
}

func (s *Service) Settings(ctx context.Context) (domain.Settings, error) {
	if _, err := auth.Require(ctx, domain.PermSettingsManage); err != nil {
		return domain.Settings{}, err
	}
	return s.repo.Settings(ctx)
}

func (s *Service) UpdateSettings(ctx context.Context, in domain.Settings) (domain.Settings, error) {
	if _, err := auth.Require(ctx, domain.PermSettingsManage); err != nil {
		return domain.Settings{}, err
	}
	if err := in.Validate(); err != nil {
		return domain.Settings{}, err
	}
	old, err := s.repo.Settings(ctx)
	if err != nil {
		return domain.Settings{}, err
	}
	var changes []contract.Change
	for _, c := range []contract.Change{
		{Field: "strike_limit", Old: strconv.Itoa(old.StrikeLimit), New: strconv.Itoa(in.StrikeLimit)},
		{Field: "strike_window_days", Old: strconv.Itoa(old.StrikeWindowDays), New: strconv.Itoa(in.StrikeWindowDays)},
		{Field: "timeout_days", Old: strconv.Itoa(old.TimeoutDays), New: strconv.Itoa(in.TimeoutDays)},
	} {
		if c.Old != c.New {
			changes = append(changes, c)
		}
	}
	return s.repo.UpdateSettings(ctx, in, changes)
}

// ReadOnlyAccounts: tài khoản đang tạm chỉ đọc (Admin)
func (s *Service) ReadOnlyAccounts(ctx context.Context) ([]domain.Timeout, error) {
	if _, err := auth.Require(ctx, domain.PermSettingsManage); err != nil {
		return nil, err
	}
	return s.repo.ActiveTimeouts(ctx)
}

// OnDecision: chỉ quyết định từ chối, gỡ và tự xoá tính là vi phạm đã xác nhận.
// Cảnh báo trước khi gửi và nội dung còn chờ duyệt không tính.
func (s *Service) OnDecision(ctx context.Context, d moderation.DecisionRecorded) error {
	if !d.Violation() {
		return nil
	}
	cfg, err := s.repo.Settings(ctx)
	if err != nil {
		return err
	}
	_, err = s.repo.RecordStrike(ctx, d.AuthorID, d.DecisionID, d.Rule, cfg)
	return err
}

// OnAppeal: xem xét lại thành công thì thu hồi vi phạm
func (s *Service) OnAppeal(ctx context.Context, a moderation.AppealDecided) error {
	if !a.Accepted {
		return nil
	}
	cfg, err := s.repo.Settings(ctx)
	if err != nil {
		return err
	}
	return s.repo.RevokeStrike(ctx, a.DecisionID, cfg)
}

type statsReader struct{ s *Service }

func (s *Service) StatsReader() contract.StatsReader { return statsReader{s} }

func (r statsReader) Stats(ctx context.Context) (contract.Stats, error) {
	cfg, err := r.s.repo.Settings(ctx)
	if err != nil {
		return contract.Stats{}, err
	}
	strikes, ro, err := r.s.repo.Stats(ctx, cfg.StrikeWindowDays)
	return contract.Stats{ActiveStrikes: strikes, ReadOnly: ro}, err
}
