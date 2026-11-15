// Package repository cài đặt domain.Repository của penalties bằng sqlc.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/penalties/contract"
	"sunshine/internal/penalties/domain"
	"sunshine/internal/penalties/repository/db"
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

func toSettings(r db.PenaltiesSetting) domain.Settings {
	return domain.Settings{StrikeLimit: int(r.StrikeLimit), StrikeWindowDays: int(r.StrikeWindowDays),
		TimeoutDays: int(r.TimeoutDays), UpdatedAt: r.UpdatedAt}
}

func (r *Repository) Settings(ctx context.Context) (domain.Settings, error) {
	row, err := r.q.GetSettings(ctx)
	return toSettings(row), err
}

func (r *Repository) UpdateSettings(ctx context.Context, s domain.Settings, changes []contract.Change) (domain.Settings, error) {
	var out domain.Settings
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		row, err := r.q.WithTx(tx).UpdateSettings(ctx, db.UpdateSettingsParams{
			StrikeLimit: int32(s.StrikeLimit), StrikeWindowDays: int32(s.StrikeWindowDays), TimeoutDays: int32(s.TimeoutDays),
		})
		if err != nil {
			return err
		}
		out = toSettings(row)
		if len(changes) == 0 {
			return nil
		}
		e, err := events.New(contract.EventSettingsChanged, contract.AggregateSettings, contract.SettingsAggregateID,
			contract.SettingsChanged{Changes: changes})
		if err != nil {
			return err
		}
		return r.outbox.Append(ctx, tx, e)
	})
	return out, err
}

func (r *Repository) ActiveTimeouts(ctx context.Context) ([]domain.Timeout, error) {
	rows, err := r.q.ListActiveTimeouts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Timeout, len(rows))
	for i, t := range rows {
		out[i] = domain.Timeout{AccountID: t.AccountID, Rule: int(t.LastRule), EndsAt: t.EndsAt}
	}
	return out, nil
}

func (r *Repository) emit(ctx context.Context, tx pgx.Tx, typ string, account uuid.UUID, payload any) error {
	e, err := events.New(typ, contract.AggregateAccount, account, payload)
	if err != nil {
		return err
	}
	return r.outbox.Append(ctx, tx, e)
}

func (r *Repository) RecordStrike(ctx context.Context, account, decision uuid.UUID, rule int, s domain.Settings) (*domain.Timeout, error) {
	var started *domain.Timeout
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		n, err := q.InsertStrike(ctx, db.InsertStrikeParams{ID: id, AccountID: account, DecisionID: decision, Rule: int32(rule)})
		if err != nil || n == 0 {
			return err // đã ghi ở lần giao trước
		}
		count, err := q.CountActiveStrikes(ctx, db.CountActiveStrikesParams{AccountID: account, WindowDays: int32(s.StrikeWindowDays)})
		if err != nil {
			return err
		}
		if int(count) < s.StrikeLimit {
			return nil
		}
		tid, err := uuid.NewV7()
		if err != nil {
			return err
		}
		ends := time.Now().Add(time.Duration(s.TimeoutDays) * 24 * time.Hour)
		if _, err := q.InsertTimeout(ctx, db.InsertTimeoutParams{
			ID: tid, AccountID: account, DecisionID: decision, LastRule: int32(rule), EndsAt: ends,
		}); err != nil {
			return err
		}
		started = &domain.Timeout{ID: tid, AccountID: account, Rule: rule, EndsAt: ends}
		return r.emit(ctx, tx, contract.EventReadOnlyStarted, account, contract.ReadOnlyStarted{
			AccountID: account, TimeoutID: tid, Rule: rule, EndsAt: ends,
		})
	})
	return started, err
}

func (r *Repository) RevokeStrike(ctx context.Context, decision uuid.UUID, s domain.Settings) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		account, err := q.RevokeStrike(ctx, decision)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		count, err := q.CountActiveStrikes(ctx, db.CountActiveStrikesParams{AccountID: account, WindowDays: int32(s.StrikeWindowDays)})
		if err != nil || int(count) >= s.StrikeLimit {
			return err
		}
		n, err := q.LiftTimeouts(ctx, account)
		if err != nil || n == 0 {
			return err
		}
		return r.emit(ctx, tx, contract.EventReadOnlyLifted, account, contract.ReadOnlyLifted{AccountID: account})
	})
}

func (r *Repository) ActiveTimeout(ctx context.Context, account uuid.UUID) (*domain.Timeout, error) {
	row, err := r.q.ActiveTimeout(ctx, account)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.Timeout{ID: row.ID, AccountID: row.AccountID, Rule: int(row.LastRule), EndsAt: row.EndsAt}, nil
}

func (r *Repository) ActiveStrikes(ctx context.Context, account uuid.UUID, windowDays int) (int, error) {
	n, err := r.q.CountActiveStrikes(ctx, db.CountActiveStrikesParams{AccountID: account, WindowDays: int32(windowDays)})
	return int(n), err
}

func (r *Repository) Stats(ctx context.Context, windowDays int) (int64, int64, error) {
	row, err := r.q.Stats(ctx, int32(windowDays))
	return row.ActiveStrikes, row.ReadOnly, err
}
