// Package repository cài đặt domain.Repository của moderation. Quyết định, dòng nhật ký
// kiểm toán và event được ghi trong cùng một transaction (SYSTEM_DESIGN: audit log).
package repository

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	content "sunshine/internal/content/contract"
	"sunshine/internal/moderation/contract"
	"sunshine/internal/moderation/domain"
	"sunshine/internal/moderation/repository/db"
)

type Repository struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	outbox *events.Outbox
}

var _ domain.Repository = (*Repository)(nil)

func New(pool *pgxpool.Pool, outbox *events.Outbox) *Repository {
	return &Repository{pool: pool, q: db.New(pool), outbox: outbox}
}

func strp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func f64(v *float32) *float64 {
	if v == nil {
		return nil
	}
	x := math.Round(float64(*v)*10000) / 10000
	return &x
}

func f32(v *float64) *float32 {
	if v == nil {
		return nil
	}
	x := float32(*v)
	return &x
}

func toDecision(r db.ModerationDecision) domain.Decision {
	d := domain.Decision{
		ID: r.ID, CaseID: r.CaseID, Ref: content.Ref{Type: r.ContentType, ID: r.ContentID}, AuthorID: r.AuthorID,
		ChannelID: r.ChannelID, Action: r.Action, ModeratorID: r.ModeratorID, OverturnedAI: r.OverturnedAi,
		AI:             domain.AI{Label: deref(r.AiLabel), Score: f64(r.AiScore), ModelVersion: deref(r.ModelVersion)},
		AppealDeadline: r.AppealDeadline, CreatedAt: r.CreatedAt,
	}
	if r.Rule != nil {
		d.Rule = int(*r.Rule)
	}
	return d
}

func toAppeal(r db.ModerationAppeal) domain.Appeal {
	return domain.Appeal{
		ID: r.ID, DecisionID: r.DecisionID, AuthorID: r.AuthorID, Reason: r.Reason, Status: r.Status,
		ReviewerID: r.ReviewerID, CreatedAt: r.CreatedAt, DecidedAt: r.DecidedAt,
	}
}

func (r *Repository) emit(ctx context.Context, tx pgx.Tx, typ, aggregate string, id uuid.UUID, payload any) error {
	e, err := events.New(typ, aggregate, id, payload)
	if err != nil {
		return err
	}
	return r.outbox.Append(ctx, tx, e)
}

func insertAudit(ctx context.Context, q *db.Queries, a domain.AuditEntry) error {
	var rule *int32
	if a.Rule != nil {
		x := int32(*a.Rule)
		rule = &x
	}
	return q.InsertAudit(ctx, db.InsertAuditParams{
		Action: a.Action, ModeratorID: a.ModeratorID, Subject: a.Subject, ContentType: strp(a.ContentType),
		AiLabel: strp(a.AI.Label), AiScore: f32(a.AI.Score), ModelVersion: strp(a.AI.ModelVersion),
		Rule: rule, Outcome: a.Outcome, OverturnedAi: a.OverturnedAI,
	})
}

// --- case ---

func (r *Repository) OpenCase(ctx context.Context, ref content.Ref, author, channel uuid.UUID) error {
	id, err := uuid.NewV7()
	if err != nil {
		return err
	}
	return r.q.OpenCase(ctx, db.OpenCaseParams{ID: id, ContentType: ref.Type, ContentID: ref.ID, AuthorID: author, ChannelID: channel})
}

func (r *Repository) CloseCase(ctx context.Context, ref content.Ref, status string) error {
	_, err := r.q.CloseOpenCase(ctx, db.CloseOpenCaseParams{Status: status, ContentType: ref.Type, ContentID: ref.ID})
	return err
}

func (r *Repository) OpenCases(ctx context.Context) ([]domain.Case, error) {
	rows, err := r.q.ListOpenCases(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Case, len(rows))
	for i, c := range rows {
		out[i] = domain.Case{ID: c.ID, Ref: content.Ref{Type: c.ContentType, ID: c.ContentID}, AuthorID: c.AuthorID,
			ChannelID: c.ChannelID, CreatedAt: c.CreatedAt}
	}
	return out, nil
}

func (r *Repository) HasOpenCase(ctx context.Context, ref content.Ref) (bool, error) {
	_, err := r.q.LockOpenCase(ctx, db.LockOpenCaseParams{ContentType: ref.Type, ContentID: ref.ID})
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	return err == nil, err
}

// --- quyết định ---

func (r *Repository) Record(ctx context.Context, items []domain.NewDecision, requireOpenCase bool) (int, error) {
	done := 0
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		done = 0
		for _, it := range items {
			d := it.Decision
			var caseID *uuid.UUID
			c, err := q.LockOpenCase(ctx, db.LockOpenCaseParams{ContentType: d.Ref.Type, ContentID: d.Ref.ID})
			switch {
			case errors.Is(err, pgx.ErrNoRows):
				if requireOpenCase {
					continue // người khác vừa xử lý, hoặc tác giả đã rút
				}
			case err != nil:
				return err
			default:
				caseID = &c.ID
			}
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			var rule *int32
			if d.Rule > 0 {
				x := int32(d.Rule)
				rule = &x
			}
			row, err := q.InsertDecision(ctx, db.InsertDecisionParams{
				ID: id, CaseID: caseID, ContentType: d.Ref.Type, ContentID: d.Ref.ID, AuthorID: d.AuthorID,
				ChannelID: d.ChannelID, Action: d.Action, Rule: rule, ModeratorID: d.ModeratorID,
				AiLabel: strp(d.AI.Label), AiScore: f32(d.AI.Score), ModelVersion: strp(d.AI.ModelVersion),
				OverturnedAi: d.OverturnedAI, AppealDeadline: d.AppealDeadline,
			})
			if errors.Is(err, pgx.ErrNoRows) {
				continue // tự xoá đã ghi (ON CONFLICT DO NOTHING)
			}
			if err != nil {
				return err
			}
			if caseID != nil && it.CloseCase {
				if _, err := q.CloseOpenCase(ctx, db.CloseOpenCaseParams{
					Status: domain.CaseDecided, ContentType: d.Ref.Type, ContentID: d.Ref.ID,
				}); err != nil {
					return err
				}
			}
			if err := insertAudit(ctx, q, it.Audit); err != nil {
				return err
			}
			if err := r.emit(ctx, tx, contract.EventDecisionRecorded, contract.AggregateDecision, row.ID, contract.DecisionRecorded{
				DecisionID: row.ID, ContentType: row.ContentType, ContentID: row.ContentID, AuthorID: row.AuthorID,
				ChannelID: row.ChannelID, Action: row.Action, Rule: d.Rule, AppealDeadline: row.AppealDeadline,
			}); err != nil {
				return err
			}
			done++
		}
		return nil
	})
	return done, err
}

func (r *Repository) Decision(ctx context.Context, id uuid.UUID) (domain.Decision, error) {
	row, err := r.q.GetDecision(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Decision{}, domain.ErrDecisionNotFound
	}
	return toDecision(row), err
}

func (r *Repository) Decisions(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.Decision, error) {
	rows, err := r.q.GetDecisionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := map[uuid.UUID]domain.Decision{}
	for _, row := range rows {
		out[row.ID] = toDecision(row)
	}
	return out, nil
}

func (r *Repository) DecisionsByAuthor(ctx context.Context, author uuid.UUID) ([]domain.DecisionWithAppeal, error) {
	rows, err := r.q.ListDecisionsByAuthor(ctx, author)
	if err != nil {
		return nil, err
	}
	out := make([]domain.DecisionWithAppeal, len(rows))
	for i, x := range rows {
		out[i].Decision = toDecision(db.ModerationDecision{
			ID: x.ID, CaseID: x.CaseID, ContentType: x.ContentType, ContentID: x.ContentID, AuthorID: x.AuthorID,
			ChannelID: x.ChannelID, Action: x.Action, Rule: x.Rule, ModeratorID: x.ModeratorID, AiLabel: x.AiLabel,
			AiScore: x.AiScore, ModelVersion: x.ModelVersion, OverturnedAi: x.OverturnedAi,
			AppealDeadline: x.AppealDeadline, CreatedAt: x.CreatedAt,
		})
		if x.AppealID != nil {
			out[i].Appeal = &domain.Appeal{
				ID: *x.AppealID, DecisionID: x.ID, AuthorID: x.AuthorID, Reason: deref(x.AppealReason),
				Status: deref(x.AppealStatus), CreatedAt: *x.AppealCreatedAt, DecidedAt: x.AppealDecidedAt,
			}
		}
	}
	return out, nil
}

// --- xem xét lại ---

func (r *Repository) CreateAppeal(ctx context.Context, a domain.Appeal) (domain.Appeal, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Appeal{}, err
	}
	row, err := r.q.InsertAppeal(ctx, db.InsertAppealParams{ID: id, DecisionID: a.DecisionID, AuthorID: a.AuthorID, Reason: a.Reason})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return domain.Appeal{}, domain.ErrAlreadyAppealed
	}
	if err != nil {
		return domain.Appeal{}, err
	}
	return toAppeal(row), nil
}

func (r *Repository) PendingAppeals(ctx context.Context) ([]domain.Appeal, error) {
	rows, err := r.q.ListPendingAppeals(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Appeal, len(rows))
	for i, a := range rows {
		out[i] = toAppeal(a)
	}
	return out, nil
}

func (r *Repository) DecideAppeal(ctx context.Context, appealID uuid.UUID, accepted bool, reviewer uuid.UUID, audit func(domain.Decision) domain.AuditEntry) (domain.Appeal, domain.Decision, error) {
	var appeal domain.Appeal
	var decision domain.Decision
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.LockAppeal(ctx, appealID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAppealNotFound
		}
		if err != nil {
			return err
		}
		if row.Status != domain.AppealPending {
			return domain.ErrAppealClosed
		}
		d, err := q.GetDecision(ctx, row.DecisionID)
		if err != nil {
			return err
		}
		decision = toDecision(d)
		if decision.ModeratorID != nil && *decision.ModeratorID == reviewer {
			return domain.ErrSameModerator
		}
		status := domain.AppealDeclined
		if accepted {
			status = domain.AppealAccepted
		}
		if err := q.DecideAppeal(ctx, db.DecideAppealParams{ID: appealID, Status: status, ReviewerID: &reviewer}); err != nil {
			return err
		}
		appeal = toAppeal(row)
		appeal.Status, appeal.ReviewerID = status, &reviewer
		if err := insertAudit(ctx, q, audit(decision)); err != nil {
			return err
		}
		return r.emit(ctx, tx, contract.EventAppealDecided, contract.AggregateAppeal, appealID, contract.AppealDecided{
			AppealID: appealID, DecisionID: d.ID, ContentType: d.ContentType, ContentID: d.ContentID,
			AuthorID: d.AuthorID, ChannelID: d.ChannelID, Accepted: accepted, Rule: decision.Rule,
		})
	})
	return appeal, decision, err
}

// --- nhật ký, thống kê ---

func (r *Repository) AuditConfig(ctx context.Context, eventID uuid.UUID, actor *uuid.UUID, outcome string) error {
	return r.q.InsertAuditFromEvent(ctx, db.InsertAuditFromEventParams{
		Action: "config_change", ModeratorID: actor, Subject: "—", Outcome: outcome, EventID: &eventID,
	})
}

// Audit: filter "" (tất cả), "overturned" (lật quyết định AI) hoặc "config" (đổi cấu hình)
func (r *Repository) Audit(ctx context.Context, beforeID *int64, filter string, limit int) ([]domain.AuditEntry, error) {
	rows, err := r.q.ListAudit(ctx, db.ListAuditParams{
		BeforeID: beforeID, OverturnedOnly: filter == "overturned", ConfigOnly: filter == "config", Lim: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.AuditEntry, len(rows))
	for i, a := range rows {
		out[i] = domain.AuditEntry{
			ID: a.ID, Action: a.Action, ModeratorID: a.ModeratorID, Subject: a.Subject, ContentType: deref(a.ContentType),
			AI:      domain.AI{Label: deref(a.AiLabel), Score: f64(a.AiScore), ModelVersion: deref(a.ModelVersion)},
			Outcome: a.Outcome, OverturnedAI: a.OverturnedAi, CreatedAt: a.CreatedAt,
		}
		if a.Rule != nil {
			x := int(*a.Rule)
			out[i].Rule = &x
		}
	}
	return out, nil
}

func (r *Repository) Stats(ctx context.Context, since time.Time) (contract.Stats, error) {
	d, err := r.q.DecisionStats(ctx, since)
	if err != nil {
		return contract.Stats{}, err
	}
	a, err := r.q.AppealStats(ctx, since)
	if err != nil {
		return contract.Stats{}, err
	}
	return contract.Stats{
		Approved: d.Approved, Rejected: d.Rejected, Removed: d.Removed, AutoDeleted: d.AutoDeleted,
		OverturnedAI: d.OverturnedAi, AppealsPending: a.Pending, AppealsAccepted: a.Accepted, AppealsDeclined: a.Declined,
		HeldWrong: d.HeldWrong, Missed: d.Missed, DeletedWrong: a.DeletedWrong, Agreed: d.Agreed,
	}, nil
}

func (r *Repository) Corrections(ctx context.Context, from, to time.Time) ([]domain.DecisionWithAppeal, error) {
	rows, err := r.q.ListCorrections(ctx, db.ListCorrectionsParams{PeriodFrom: from, PeriodTo: to})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DecisionWithAppeal, len(rows))
	for i, x := range rows {
		out[i].Decision = toDecision(db.ModerationDecision{
			ID: x.ID, CaseID: x.CaseID, ContentType: x.ContentType, ContentID: x.ContentID, AuthorID: x.AuthorID,
			ChannelID: x.ChannelID, Action: x.Action, Rule: x.Rule, ModeratorID: x.ModeratorID, AiLabel: x.AiLabel,
			AiScore: x.AiScore, ModelVersion: x.ModelVersion, OverturnedAi: x.OverturnedAi,
			AppealDeadline: x.AppealDeadline, CreatedAt: x.CreatedAt,
		})
		if x.AppealStatus != nil {
			out[i].Appeal = &domain.Appeal{DecisionID: x.ID, Status: *x.AppealStatus}
		}
	}
	return out, nil
}
