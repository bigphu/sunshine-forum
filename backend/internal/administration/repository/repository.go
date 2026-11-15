// Package repository cài đặt domain.Repository của administration bằng sqlc.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/administration/domain"
	"sunshine/internal/administration/repository/db"
)

type Repository struct{ q *db.Queries }

var _ domain.Repository = (*Repository)(nil)

func New(pool *pgxpool.Pool) *Repository { return &Repository{q: db.New(pool)} }

func (r *Repository) Create(ctx context.Context, e domain.Export, data []byte) (domain.Export, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Export{}, err
	}
	row, err := r.q.InsertExport(ctx, db.InsertExportParams{
		ID: id, CreatedBy: e.CreatedBy, ItemCount: int32(e.ItemCount), Purpose: e.Purpose, Data: data,
		PeriodFrom: e.PeriodFrom, PeriodTo: e.PeriodTo, DeleteAfter: e.DeleteAfter,
	})
	if err != nil {
		return domain.Export{}, err
	}
	return domain.Export{ID: row.ID, CreatedBy: row.CreatedBy, ItemCount: int(row.ItemCount), Purpose: row.Purpose, PeriodFrom: row.PeriodFrom,
		PeriodTo: row.PeriodTo, DeleteAfter: row.DeleteAfter, CreatedAt: row.CreatedAt}, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Export, error) {
	rows, err := r.q.ListExports(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Export, len(rows))
	for i, x := range rows {
		out[i] = domain.Export{ID: x.ID, CreatedBy: x.CreatedBy, ItemCount: int(x.ItemCount), Purpose: x.Purpose, PeriodFrom: x.PeriodFrom,
			PeriodTo: x.PeriodTo, DeleteAfter: x.DeleteAfter, CreatedAt: x.CreatedAt}
	}
	return out, nil
}

func (r *Repository) Data(ctx context.Context, id uuid.UUID) ([]byte, time.Time, error) {
	row, err := r.q.GetExportData(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, time.Time{}, domain.ErrExportNotFound
	}
	return row.Data, row.CreatedAt, err
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	n, err := r.q.DeleteExport(ctx, id)
	if err == nil && n == 0 {
		return domain.ErrExportNotFound
	}
	return err
}

func (r *Repository) Purge(ctx context.Context) (int64, error) { return r.q.PurgeExports(ctx) }
