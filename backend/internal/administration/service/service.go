// Package service là use case của administration: thống kê bộ lọc (FR-5.4) và bản xuất
// dữ liệu huấn luyện tối thiểu (US-6.4). Dữ liệu của module khác chỉ đọc qua contract.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/bigphu/goat/auth"
	"github.com/google/uuid"

	"sunshine/internal/administration/domain"
	content "sunshine/internal/content/contract"
	moderation "sunshine/internal/moderation/contract"
	penalties "sunshine/internal/penalties/contract"
)

type Deps struct {
	Repo       domain.Repository
	Content    content.Reader
	Moderation moderation.Reader
	Penalties  penalties.StatsReader
	Now        func() time.Time
}

type Service struct {
	repo       domain.Repository
	content    content.Reader
	moderation moderation.Reader
	penalties  penalties.StatsReader
	now        func() time.Time
}

func New(d Deps) *Service {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{repo: d.Repo, content: d.Content, moderation: d.Moderation, penalties: d.Penalties, now: now}
}

type Analytics struct {
	Days       int
	Content    content.Stats
	Moderation moderation.Stats
	Penalties  penalties.Stats
	// Agreement: trong các quyết định về nội dung mô hình đã gắn nhãn, phần đồng ý với AI
	Agreement *float64
}

// Analytics: số liệu trong days ngày gần nhất; FlaggedByDay luôn 14 ngày
func (s *Service) Analytics(ctx context.Context, days int) (Analytics, error) {
	if _, err := auth.Require(ctx, domain.PermAnalyticsRead); err != nil {
		return Analytics{}, err
	}
	since := s.now().AddDate(0, 0, -days)
	c, err := s.content.Stats(ctx, since, days)
	if err != nil {
		return Analytics{}, err
	}
	m, err := s.moderation.Stats(ctx, since)
	if err != nil {
		return Analytics{}, err
	}
	p, err := s.penalties.Stats(ctx)
	if err != nil {
		return Analytics{}, err
	}
	out := Analytics{Days: days, Content: c, Moderation: m, Penalties: p}
	if total := m.Agreed + m.HeldWrong + m.Missed + m.DeletedWrong; total > 0 {
		v := float64(m.Agreed) / float64(total)
		out.Agreement = &v
	}
	return out, nil
}

// Window là khoảng quyết định đủ điều kiện xuất vào lúc now
func (s *Service) Window() (time.Time, time.Time) {
	now := s.now()
	return now.Add(-domain.ExportMaxAge), now.Add(-domain.ExportMinAge)
}

// samples dựng các dòng xuất: bỏ nội dung tác giả đã xoá hoặc đã bị xoá hẳn
func (s *Service) samples(ctx context.Context) ([]domain.Sample, time.Time, error) {
	from, to := s.Window()
	list, err := s.moderation.Corrections(ctx, from, to)
	if err != nil {
		return nil, time.Time{}, err
	}
	refs := make([]content.Ref, len(list))
	for i, c := range list {
		refs[i] = content.Ref{Type: c.ContentType, ID: c.ContentID}
	}
	items, err := s.content.Items(ctx, refs)
	if err != nil {
		return nil, time.Time{}, err
	}
	var out []domain.Sample
	oldest := to
	for _, c := range list {
		it, ok := items[content.Ref{Type: c.ContentType, ID: c.ContentID}]
		if !ok || it.Status == content.StatusDeleted || it.Body == "" {
			continue
		}
		text := it.Body
		if it.Title != "" {
			text = it.Title + "\n" + it.Body
		}
		out = append(out, domain.Sample{Text: domain.Scrub(text), Label: c.FinalLabel, Rule: c.Rule,
			AIScore: c.AIScore, ModelVersion: c.ModelVersion})
		if c.DecidedAt.Before(oldest) {
			oldest = c.DecidedAt
		}
	}
	return out, oldest, nil
}

type ExportList struct {
	Eligible   int
	PeriodFrom time.Time
	PeriodTo   time.Time
	Exports    []domain.Export
}

func (s *Service) Exports(ctx context.Context) (ExportList, error) {
	if _, err := auth.Require(ctx, domain.PermExportManage); err != nil {
		return ExportList{}, err
	}
	samples, _, err := s.samples(ctx)
	if err != nil {
		return ExportList{}, err
	}
	list, err := s.repo.List(ctx)
	if err != nil {
		return ExportList{}, err
	}
	from, to := s.Window()
	return ExportList{Eligible: len(samples), PeriodFrom: from, PeriodTo: to, Exports: list}, nil
}

// CreateExport ghi bản xuất JSON Lines; nó tự xoá trước khi bản gốc cũ nhất tới hạn 90 ngày
func (s *Service) CreateExport(ctx context.Context, purpose string) (domain.Export, error) {
	actor, err := auth.Require(ctx, domain.PermExportManage)
	if err != nil {
		return domain.Export{}, err
	}
	samples, oldest, err := s.samples(ctx)
	if err != nil {
		return domain.Export{}, err
	}
	if len(samples) == 0 {
		return domain.Export{}, domain.ErrNothingToExport
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for _, x := range samples {
		if err := enc.Encode(x); err != nil {
			return domain.Export{}, err
		}
	}
	from, to := s.Window()
	return s.repo.Create(ctx, domain.Export{
		CreatedBy: actor.AccountID, ItemCount: len(samples), Purpose: purpose, PeriodFrom: from, PeriodTo: to,
		DeleteAfter: oldest.Add(domain.RetainFor),
	}, buf.Bytes())
}

func (s *Service) ExportData(ctx context.Context, id uuid.UUID) ([]byte, time.Time, error) {
	if _, err := auth.Require(ctx, domain.PermExportManage); err != nil {
		return nil, time.Time{}, err
	}
	return s.repo.Data(ctx, id)
}

func (s *Service) DeleteExport(ctx context.Context, id uuid.UUID) error {
	if _, err := auth.Require(ctx, domain.PermExportManage); err != nil {
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) Purge(ctx context.Context) error {
	_, err := s.repo.Purge(ctx)
	return err
}
