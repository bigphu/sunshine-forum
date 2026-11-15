// Package repository cài đặt domain.Repository của notifications bằng sqlc, và nghe
// NOTIFY notifications (trigger khi thêm thông báo) để đẩy tới trình duyệt.
package repository

import (
	"context"
	"time"

	"github.com/bigphu/goat/database"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/notifications/domain"
	"sunshine/internal/notifications/repository/db"
)

type Repository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

var _ domain.Repository = (*Repository)(nil)

func New(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool, q: db.New(pool)} }

func (r *Repository) Insert(ctx context.Context, items []domain.NewItem) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		for _, it := range items {
			id, err := uuid.NewV7()
			if err != nil {
				return err
			}
			if _, err := q.InsertItem(ctx, db.InsertItemParams{
				ID: id, AccountID: it.AccountID, Kind: it.Kind, DedupeKey: it.DedupeKey, Body: it.Body, Link: it.Link,
			}); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) List(ctx context.Context, account uuid.UUID, limit int) ([]domain.Item, int, error) {
	rows, err := r.q.ListItems(ctx, db.ListItemsParams{AccountID: account, Lim: int32(limit)})
	if err != nil {
		return nil, 0, err
	}
	unread, err := r.q.CountUnread(ctx, account)
	if err != nil {
		return nil, 0, err
	}
	out := make([]domain.Item, len(rows))
	for i, it := range rows {
		out[i] = domain.Item{ID: it.ID, AccountID: it.AccountID, Kind: it.Kind, Body: it.Body, Link: it.Link,
			ReadAt: it.ReadAt, CreatedAt: it.CreatedAt}
	}
	return out, int(unread), nil
}

func (r *Repository) MarkAllRead(ctx context.Context, account uuid.UUID) error {
	return r.q.MarkAllRead(ctx, account)
}

func (r *Repository) Purge(ctx context.Context, days int) (int64, error) {
	return r.q.PurgeItems(ctx, int32(days))
}

// Listen giữ một kết nối LISTEN notifications; mất kết nối thì thử lại sau 5 giây
func (r *Repository) Listen(ctx context.Context, onNew func(uuid.UUID)) error {
	for ctx.Err() == nil {
		_ = r.listen(ctx, onNew)
		select {
		case <-ctx.Done():
		case <-time.After(5 * time.Second):
		}
	}
	return nil
}

func (r *Repository) listen(ctx context.Context, onNew func(uuid.UUID)) error {
	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "LISTEN notifications"); err != nil {
		return err
	}
	for {
		n, err := conn.Conn().WaitForNotification(ctx)
		if err != nil {
			return err
		}
		if id, err := uuid.Parse(n.Payload); err == nil {
			onNew(id)
		}
	}
}
