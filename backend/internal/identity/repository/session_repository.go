package repository

import (
	"context"
	"errors"
	"time"

	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/jobs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/identity/domain"
	"sunshine/internal/identity/repository/db"
)

type SessionRepository struct{ q *db.Queries }

var _ domain.SessionRepository = (*SessionRepository)(nil)

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{q: db.New(pool)}
}

func (r *SessionRepository) Create(ctx context.Context, s domain.Session) error {
	return r.q.CreateSession(ctx, db.CreateSessionParams{
		TokenHash: s.TokenHash, AccountID: s.AccountID, CsrfToken: s.CSRFToken, ExpiresAt: s.ExpiresAt,
	})
}

func (r *SessionRepository) GetActive(ctx context.Context, tokenHash []byte) (domain.Session, error) {
	row, err := r.q.GetActiveSession(ctx, tokenHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Session{}, domain.ErrSessionNotFound
	}
	if err != nil {
		return domain.Session{}, err
	}
	return domain.Session{
		TokenHash: row.TokenHash, AccountID: row.AccountID, Role: domain.RoleFromID(row.RoleID),
		CSRFToken: row.CsrfToken, LastSeenAt: row.LastSeenAt, ExpiresAt: row.ExpiresAt,
	}, nil
}

func (r *SessionRepository) Extend(ctx context.Context, tokenHash []byte, expiresAt time.Time) error {
	return r.q.ExtendSession(ctx, db.ExtendSessionParams{TokenHash: tokenHash, ExpiresAt: expiresAt})
}

func (r *SessionRepository) Delete(ctx context.Context, tokenHash []byte) error {
	return r.q.DeleteSession(ctx, tokenHash)
}

func (r *SessionRepository) Prune(ctx context.Context) (int64, error) {
	return r.q.PruneSessions(ctx)
}

type TokenRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
	jobs jobs.Enqueuer
}

var _ domain.TokenRepository = (*TokenRepository)(nil)

func NewTokenRepository(pool *pgxpool.Pool, enq jobs.Enqueuer) *TokenRepository {
	return &TokenRepository{pool: pool, q: db.New(pool), jobs: enq}
}

func (r *TokenRepository) Issue(ctx context.Context, accountID uuid.UUID, t domain.IssuedToken) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		return issueToken(ctx, r.q.WithTx(tx), r.jobs, tx, accountID, t)
	})
}

func (r *TokenRepository) Valid(ctx context.Context, hash []byte) (uuid.UUID, domain.TokenPurpose, time.Time, error) {
	row, err := r.q.GetValidPasswordToken(ctx, hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", time.Time{}, domain.ErrInvalidPasswordToken
	}
	if err != nil {
		return uuid.Nil, "", time.Time{}, err
	}
	return row.AccountID, domain.TokenPurpose(row.Purpose), row.ExpiresAt, nil
}

func (r *TokenRepository) Use(ctx context.Context, hash []byte) (bool, error) {
	n, err := r.q.UsePasswordToken(ctx, hash)
	return n == 1, err
}

func (r *TokenRepository) PendingInviteExpiry(ctx context.Context, accountID uuid.UUID) (*time.Time, error) {
	t, err := r.q.PendingInviteExpiry(ctx, accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TokenRepository) Prune(ctx context.Context) (int64, error) {
	return r.q.PruneTokens(ctx)
}

type RoleRepository struct{ q *db.Queries }

var _ domain.RoleRepository = (*RoleRepository)(nil)

func NewRoleRepository(pool *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{q: db.New(pool)}
}

func (r *RoleRepository) Permissions(ctx context.Context) (map[domain.Role][]string, error) {
	roles, err := r.q.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[domain.Role][]string, len(roles))
	for _, role := range roles {
		perms, err := r.q.RolePermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		out[domain.Role(role.Code)] = perms
	}
	return out, nil
}
