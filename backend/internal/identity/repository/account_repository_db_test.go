package repository_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"

	"sunshine/internal/identity/domain"
	"sunshine/internal/identity/repository"
	"sunshine/internal/testdb"
)

func newRepo(t *testing.T) (*repository.AccountRepository, context.Context) {
	t.Helper()
	pool := testdb.Pool(t)
	client, err := jobs.NewInsertClient(pool, nil)
	if err != nil {
		t.Fatal(err)
	}
	outbox := events.NewOutbox(events.NewRegistry(), client)
	return repository.NewAccountRepository(pool, outbox, jobs.NewRiver(client), testdb.Cipher(t)), context.Background()
}

// Email, tên và căn hộ được mã hoá trong DB (NFR-3.1) nhưng vẫn tra được theo email
func TestAccountPersonalDataIsEncrypted(t *testing.T) {
	repo, ctx := newRepo(t)
	pool := testdb.Pool(t)
	a, err := repo.Create(ctx, domain.NewAccount{
		Email: "lan.encrypted@sunshine.test", Name: "Nguyễn Thị Lan", Unit: "A-1203", Role: domain.RoleResident,
		Invite: domain.IssuedToken{Raw: "raw", Hash: []byte("hash-encrypted"), Purpose: domain.PurposeInvite, ExpiresAt: time.Now().Add(time.Hour)},
	})
	if err != nil {
		t.Fatal(err)
	}
	var emailEnc, nameEnc []byte
	if err := pool.QueryRow(ctx, `SELECT email_enc, name_enc FROM identity.accounts WHERE id = $1`, a.ID).Scan(&emailEnc, &nameEnc); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(emailEnc, []byte("sunshine.test")) || bytes.Contains(nameEnc, []byte("Lan")) {
		t.Fatal("personal data stored in plain text")
	}
	got, err := repo.GetByEmail(ctx, "lan.encrypted@sunshine.test")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Nguyễn Thị Lan" || got.Unit != "A-1203" || got.Status != domain.StatusInvited {
		t.Fatalf("round trip = %+v", got)
	}
	_, err = repo.Create(ctx, domain.NewAccount{
		Email: "lan.encrypted@sunshine.test", Name: "Khác", Role: domain.RoleResident,
		Invite: domain.IssuedToken{Raw: "r2", Hash: []byte("hash-2"), Purpose: domain.PurposeInvite, ExpiresAt: time.Now().Add(time.Hour)},
	})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("duplicate email err = %v, want ErrEmailTaken", err)
	}
}

// Khoá tài khoản thu hồi mọi phiên ngay (đăng xuất ở request kế tiếp)
func TestDeactivateRevokesSessions(t *testing.T) {
	repo, ctx := newRepo(t)
	pool := testdb.Pool(t)
	a, err := repo.CreateActive(ctx, domain.NewAccount{Email: "minh.deact@sunshine.test", Name: "Minh", Role: domain.RoleResident}, "hash")
	if err != nil {
		t.Fatal(err)
	}
	sessions := repository.NewSessionRepository(pool)
	tok := []byte("session-token-hash-deact")
	if err := sessions.Create(ctx, domain.Session{TokenHash: tok, AccountID: a.ID, CSRFToken: "c", ExpiresAt: time.Now().Add(time.Hour)}); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.GetActive(ctx, tok); err != nil {
		t.Fatalf("session should be active: %v", err)
	}
	if _, err := repo.Deactivate(ctx, a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := sessions.GetActive(ctx, tok); !errors.Is(err, domain.ErrSessionNotFound) {
		t.Fatalf("session after deactivate err = %v", err)
	}
}
