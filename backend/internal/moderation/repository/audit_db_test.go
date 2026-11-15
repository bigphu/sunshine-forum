package repository_test

import (
	"context"
	"testing"

	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"
	"github.com/google/uuid"

	content "sunshine/internal/content/contract"
	"sunshine/internal/moderation/contract"
	"sunshine/internal/moderation/domain"
	"sunshine/internal/moderation/repository"
	"sunshine/internal/testdb"
)

func newRepo(t *testing.T) *repository.Repository {
	t.Helper()
	pool := testdb.Pool(t)
	client, err := jobs.NewInsertClient(pool, nil)
	if err != nil {
		t.Fatal(err)
	}
	return repository.New(pool, events.NewOutbox(events.NewRegistry(), client))
}

// Quyết định và dòng nhật ký ghi cùng nhau; nhật ký không sửa, không xoá được (FR-5.3)
func TestRecordWritesAuditAndAuditIsAppendOnly(t *testing.T) {
	repo, ctx := newRepo(t), context.Background()
	pool := testdb.Pool(t)
	ref := content.Ref{Type: content.TypePost, ID: uuid.New()}
	author, channel, mod := uuid.New(), uuid.New(), uuid.New()
	if err := repo.OpenCase(ctx, ref, author, channel); err != nil {
		t.Fatal(err)
	}
	// Mở lại là no-op (event giao hai lần)
	if err := repo.OpenCase(ctx, ref, author, channel); err != nil {
		t.Fatal(err)
	}
	rule := 1
	n, err := repo.Record(ctx, []domain.NewDecision{{
		Decision:  domain.Decision{Ref: ref, AuthorID: author, ChannelID: channel, Action: contract.ActionReject, Rule: 1, ModeratorID: &mod},
		Audit:     domain.AuditEntry{Action: "reject", ModeratorID: &mod, Subject: "u-test", ContentType: "post", Rule: &rule, Outcome: "rejected"},
		CloseCase: true,
	}}, true)
	if err != nil || n != 1 {
		t.Fatalf("Record = %d, %v", n, err)
	}
	// Case đã đóng: quyết định lần hai (người khác bấm cùng lúc) bị bỏ qua
	n, err = repo.Record(ctx, []domain.NewDecision{{
		Decision: domain.Decision{Ref: ref, AuthorID: author, ChannelID: channel, Action: contract.ActionApprove, ModeratorID: &mod},
		Audit:    domain.AuditEntry{Action: "approve", Subject: "u-test", Outcome: "published"}, CloseCase: true,
	}}, true)
	if err != nil || n != 0 {
		t.Fatalf("second Record = %d, %v; want 0 (case already decided)", n, err)
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM moderation.audit_log WHERE subject = 'u-test'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("audit rows = %d, %v", count, err)
	}
	if _, err := pool.Exec(ctx, `UPDATE moderation.audit_log SET outcome = 'changed' WHERE subject = 'u-test'`); err == nil {
		t.Fatal("audit log update must fail")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM moderation.audit_log WHERE subject = 'u-test'`); err == nil {
		t.Fatal("audit log delete must fail")
	}
}
