// Package repository cài đặt interface repository của identity bằng sqlc (repository/db).
// Mã hoá email, tên, căn hộ khi ghi và giải mã khi đọc; transaction, outbox và job
// gửi thư nằm ở đây, service không thấy.
package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/identity/contract"
	"sunshine/internal/identity/domain"
	"sunshine/internal/identity/job"
	"sunshine/internal/identity/repository/db"
	"sunshine/internal/privacy"
)

type AccountRepository struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	outbox *events.Outbox
	jobs   jobs.Enqueuer
	cipher *privacy.Cipher
}

var _ domain.AccountRepository = (*AccountRepository)(nil)

func NewAccountRepository(pool *pgxpool.Pool, outbox *events.Outbox, enq jobs.Enqueuer, c *privacy.Cipher) *AccountRepository {
	return &AccountRepository{pool: pool, q: db.New(pool), outbox: outbox, jobs: enq, cipher: c}
}

func isUniqueViolation(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "23505"
}

func (r *AccountRepository) toAccount(row db.IdentityAccount) (domain.Account, error) {
	a := domain.Account{
		ID: row.ID, Role: domain.RoleFromID(row.RoleID), Status: domain.Status(row.Status),
		LastSignInAt: row.LastSignInAt, DeactivatedAt: row.DeactivatedAt, CreatedAt: row.CreatedAt,
	}
	if row.PasswordHash != nil {
		a.PasswordHash = *row.PasswordHash
	}
	if a.Status == domain.StatusPurged {
		a.Name = "Cựu cư dân"
		return a, nil
	}
	var err error
	if a.Email, err = r.cipher.Decrypt(row.EmailEnc); err != nil {
		return a, fmt.Errorf("identity: decrypt email of %s: %w", row.ID, err)
	}
	if a.Name, err = r.cipher.Decrypt(row.NameEnc); err != nil {
		return a, fmt.Errorf("identity: decrypt name of %s: %w", row.ID, err)
	}
	if a.Unit, err = r.cipher.Decrypt(row.UnitEnc); err != nil {
		return a, fmt.Errorf("identity: decrypt unit of %s: %w", row.ID, err)
	}
	return a, nil
}

func (r *AccountRepository) toAccounts(rows []db.IdentityAccount) ([]domain.Account, error) {
	out := make([]domain.Account, 0, len(rows))
	for _, row := range rows {
		a, err := r.toAccount(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func (r *AccountRepository) append(ctx context.Context, tx pgx.Tx, typ string, id uuid.UUID, payload any) error {
	e, err := events.New(typ, contract.AggregateAccount, id, payload)
	if err != nil {
		return err
	}
	return r.outbox.Append(ctx, tx, e)
}

func (r *AccountRepository) insert(ctx context.Context, tx pgx.Tx, in domain.NewAccount, status domain.Status) (db.IdentityAccount, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return db.IdentityAccount{}, err
	}
	row, err := r.q.WithTx(tx).CreateAccount(ctx, db.CreateAccountParams{
		ID: id, EmailHash: r.cipher.Hash(in.Email), EmailEnc: r.cipher.Encrypt(in.Email),
		NameEnc: r.cipher.Encrypt(in.Name), UnitEnc: r.cipher.Encrypt(in.Unit),
		RoleID: in.Role.ID(), Status: string(status),
	})
	if isUniqueViolation(err) {
		return row, domain.ErrEmailTaken
	}
	return row, err
}

func (r *AccountRepository) Create(ctx context.Context, in domain.NewAccount) (domain.Account, error) {
	var out domain.Account
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		row, err := r.insert(ctx, tx, in, domain.StatusInvited)
		if err != nil {
			return err
		}
		if err := issueToken(ctx, r.q.WithTx(tx), r.jobs, tx, row.ID, in.Invite); err != nil {
			return err
		}
		if out, err = r.toAccount(row); err != nil {
			return err
		}
		return r.append(ctx, tx, contract.EventAccountInvited, row.ID, contract.AccountInvited{AccountID: row.ID, Role: string(in.Role)})
	})
	return out, err
}

func (r *AccountRepository) CreateActive(ctx context.Context, in domain.NewAccount, passwordHash string) (domain.Account, error) {
	var out domain.Account
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		row, err := r.insert(ctx, tx, in, domain.StatusInvited)
		if err != nil {
			return err
		}
		row, err = r.q.WithTx(tx).SetPassword(ctx, db.SetPasswordParams{ID: row.ID, PasswordHash: &passwordHash})
		if err != nil {
			return err
		}
		out, err = r.toAccount(row)
		return err
	})
	return out, err
}

func (r *AccountRepository) Get(ctx context.Context, id uuid.UUID) (domain.Account, error) {
	row, err := r.q.GetAccount(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, domain.ErrAccountNotFound
	}
	if err != nil {
		return domain.Account{}, err
	}
	return r.toAccount(row)
}

func (r *AccountRepository) GetByEmail(ctx context.Context, email string) (domain.Account, error) {
	row, err := r.q.GetAccountByEmailHash(ctx, r.cipher.Hash(email))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Account{}, domain.ErrAccountNotFound
	}
	if err != nil {
		return domain.Account{}, err
	}
	return r.toAccount(row)
}

func (r *AccountRepository) GetMany(ctx context.Context, ids []uuid.UUID) ([]domain.Account, error) {
	rows, err := r.q.GetAccountsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return r.toAccounts(rows)
}

func (r *AccountRepository) List(ctx context.Context) ([]domain.Account, error) {
	rows, err := r.q.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	return r.toAccounts(rows)
}

func (r *AccountRepository) ActiveIDsByRole(ctx context.Context, roles []domain.Role) ([]uuid.UUID, error) {
	codes := make([]string, len(roles))
	for i, role := range roles {
		codes[i] = string(role)
	}
	return r.q.ListActiveAccountIDsByRole(ctx, codes)
}

func (r *AccountRepository) Count(ctx context.Context) (int64, error) {
	return r.q.CountAccounts(ctx)
}

func (r *AccountRepository) SetPassword(ctx context.Context, id uuid.UUID, hash string, keepSession []byte) (domain.Account, error) {
	var out domain.Account
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		before, err := q.GetAccount(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		row, err := q.SetPassword(ctx, db.SetPasswordParams{ID: id, PasswordHash: &hash})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		if keepSession == nil {
			err = q.DeleteAccountSessions(ctx, id)
		} else {
			err = q.DeleteOtherSessions(ctx, db.DeleteOtherSessionsParams{AccountID: id, Keep: keepSession})
		}
		if err != nil {
			return err
		}
		if out, err = r.toAccount(row); err != nil {
			return err
		}
		if before.Status == string(domain.StatusInvited) {
			return r.append(ctx, tx, contract.EventAccountActivated, id, contract.AccountActivated{AccountID: id})
		}
		return nil
	})
	return out, err
}

func (r *AccountRepository) SetRole(ctx context.Context, id uuid.UUID, role domain.Role) (domain.Account, error) {
	var out domain.Account
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		row, err := r.q.WithTx(tx).SetRole(ctx, db.SetRoleParams{ID: id, RoleID: role.ID()})
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		if out, err = r.toAccount(row); err != nil {
			return err
		}
		return r.append(ctx, tx, contract.EventRoleChanged, id, contract.RoleChanged{AccountID: id, Role: string(role)})
	})
	return out, err
}

func (r *AccountRepository) Deactivate(ctx context.Context, id uuid.UUID) (domain.Account, error) {
	var out domain.Account
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		row, err := q.Deactivate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrAccountNotFound
		}
		if err != nil {
			return err
		}
		if err := q.DeleteAccountSessions(ctx, id); err != nil {
			return err
		}
		if out, err = r.toAccount(row); err != nil {
			return err
		}
		return r.append(ctx, tx, contract.EventAccountDeactivated, id, contract.AccountDeactivated{AccountID: id})
	})
	return out, err
}

func (r *AccountRepository) TouchSignIn(ctx context.Context, id uuid.UUID) error {
	return r.q.TouchSignIn(ctx, id)
}

func (r *AccountRepository) PurgeDeactivated(ctx context.Context, days int) (int64, error) {
	return r.q.PurgeDeactivated(ctx, int32(days))
}

// issueToken huỷ link cùng mục đích cũ, ghi link mới và xếp job gửi thư, trong tx
func issueToken(ctx context.Context, q *db.Queries, enq jobs.Enqueuer, tx pgx.Tx, accountID uuid.UUID, t domain.IssuedToken) error {
	if err := q.RevokePasswordTokens(ctx, db.RevokePasswordTokensParams{AccountID: accountID, Purpose: string(t.Purpose)}); err != nil {
		return err
	}
	if err := q.CreatePasswordToken(ctx, db.CreatePasswordTokenParams{
		TokenHash: t.Hash, AccountID: accountID, Purpose: string(t.Purpose), ExpiresAt: t.ExpiresAt,
	}); err != nil {
		return err
	}
	return enq.EnqueueTx(ctx, tx, job.SendAccountEmailArgs{Token: t.Raw, Purpose: string(t.Purpose)})
}
