// Package repository cài đặt domain.Repository của screening bằng sqlc (repository/db),
// và nghe NOTIFY screening_settings để service làm mới bản cache.
package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/screening/contract"
	"sunshine/internal/screening/domain"
	"sunshine/internal/screening/repository/db"
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

func (r *Repository) emitChanges(ctx context.Context, tx pgx.Tx, changes []contract.Change) error {
	if len(changes) == 0 {
		return nil
	}
	e, err := events.New(contract.EventSettingsChanged, contract.AggregateSettings, contract.SettingsAggregateID,
		contract.SettingsChanged{Changes: changes})
	if err != nil {
		return err
	}
	return r.outbox.Append(ctx, tx, e)
}

// real trong DB là float32: làm tròn để 0.95 không thành 0.949999988
func f64(v float32) float64 { return math.Round(float64(v)*10000) / 10000 }

func toSettings(row db.ScreeningSetting) domain.Settings {
	return domain.Settings{
		HoldThreshold: f64(row.HoldThreshold), AutoDeleteEnabled: row.AutoDeleteEnabled,
		AutoDeleteThreshold: f64(row.AutoDeleteThreshold), OutagePolicy: row.OutagePolicy,
		StandInOffline: row.StandInOffline, UpdatedAt: row.UpdatedAt,
	}
}

func (r *Repository) Snapshot(ctx context.Context) (domain.Snapshot, error) {
	row, err := r.q.GetSettings(ctx)
	if err != nil {
		return domain.Snapshot{}, fmt.Errorf("screening: load settings: %w", err)
	}
	chans, err := r.q.ListChannelThresholds(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	block, err := r.q.ListBlocklist(ctx)
	if err != nil {
		return domain.Snapshot{}, err
	}
	s := domain.Snapshot{Settings: toSettings(row), Channels: map[uuid.UUID]domain.ChannelThreshold{}}
	for _, c := range chans {
		t := domain.ChannelThreshold{Hold: f64(c.HoldThreshold)}
		if c.AutoDeleteThreshold != nil {
			v := f64(*c.AutoDeleteThreshold)
			t.AutoDelete = &v
		}
		s.Channels[c.ChannelID] = t
	}
	for _, b := range block {
		s.Block = append(s.Block, domain.BlockEntry{ID: b.ID, Term: b.Term, Rule: int(b.Rule), CreatedBy: b.CreatedBy, CreatedAt: b.CreatedAt})
	}
	return s, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, s domain.Settings, changes []contract.Change) (domain.Settings, error) {
	var out domain.Settings
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		row, err := r.q.WithTx(tx).UpdateSettings(ctx, db.UpdateSettingsParams{
			HoldThreshold: float32(s.HoldThreshold), AutoDeleteEnabled: s.AutoDeleteEnabled,
			AutoDeleteThreshold: float32(s.AutoDeleteThreshold), OutagePolicy: s.OutagePolicy,
		})
		if err != nil {
			return err
		}
		out = toSettings(row)
		return r.emitChanges(ctx, tx, changes)
	})
	return out, err
}

func (r *Repository) SetStandInOffline(ctx context.Context, offline bool) error {
	return r.q.SetStandInOffline(ctx, offline)
}

func (r *Repository) SetChannelThreshold(ctx context.Context, channel uuid.UUID, t *domain.ChannelThreshold, changes []contract.Change) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		var err error
		if t == nil {
			err = q.DeleteChannelThreshold(ctx, channel)
		} else {
			var del *float32
			if t.AutoDelete != nil {
				x := float32(*t.AutoDelete)
				del = &x
			}
			err = q.UpsertChannelThreshold(ctx, db.UpsertChannelThresholdParams{
				ChannelID: channel, HoldThreshold: float32(t.Hold), AutoDeleteThreshold: del,
			})
		}
		if err != nil {
			return err
		}
		return r.emitChanges(ctx, tx, changes)
	})
}

func (r *Repository) AddBlock(ctx context.Context, term string, rule int, by uuid.UUID) (domain.BlockEntry, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.BlockEntry{}, err
	}
	row, err := r.q.InsertBlock(ctx, db.InsertBlockParams{ID: id, Term: term, TermFolded: domain.Fold(term), Rule: int32(rule), CreatedBy: &by})
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return domain.BlockEntry{}, domain.ErrTermExists
	}
	if err != nil {
		return domain.BlockEntry{}, err
	}
	return domain.BlockEntry{ID: row.ID, Term: row.Term, Rule: int(row.Rule), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt}, nil
}

func (r *Repository) DeleteBlock(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteBlock(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrTermNotFound
	}
	return nil
}

// Watch giữ một kết nối riêng LISTEN screening_settings; mất kết nối thì thử lại sau 5 giây
// và gọi onChange để không bỏ lỡ thay đổi trong lúc mất.
func (r *Repository) Watch(ctx context.Context, onChange func()) error {
	for ctx.Err() == nil {
		err := r.listen(ctx, onChange)
		if ctx.Err() != nil {
			return nil
		}
		_ = err
		onChange()
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
	}
	return nil
}

func (r *Repository) listen(ctx context.Context, onChange func()) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "LISTEN screening_settings"); err != nil {
		return err
	}
	for {
		if _, err := conn.Conn().WaitForNotification(ctx); err != nil {
			return err
		}
		onChange()
	}
}
