// Package testdb dựng một Postgres thật (testcontainers) đã chạy migration của app, dùng
// chung cho mọi test DB trong một package (*_db_test.go). Không có Docker thì test bị bỏ
// qua, trừ khi CI=true.
package testdb

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"testing"

	"github.com/bigphu/goat/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"sunshine/internal/dbmigrate"
	"sunshine/internal/privacy"
)

const image = "postgres:17-alpine"

var (
	once    sync.Once
	pool    *pgxpool.Pool
	initErr error
)

// Pool trả pool tới DB test dùng chung của package
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if os.Getenv("CI") == "" {
		testcontainers.SkipIfProviderIsNotHealthy(t)
	}
	once.Do(func() { pool, initErr = start(context.Background()) })
	if initErr != nil {
		t.Fatalf("testdb: %v", initErr)
	}
	return pool
}

func start(ctx context.Context) (*pgxpool.Pool, error) {
	ctr, err := postgres.Run(ctx, image,
		postgres.WithDatabase("test"), postgres.WithUsername("app"), postgres.WithPassword("app_pw"),
		postgres.BasicWaitStrategies())
	if err != nil {
		return nil, fmt.Errorf("start postgres: %w", err)
	}
	url, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}
	p, err := database.Open(ctx, database.Config{URL: url})
	if err != nil {
		return nil, err
	}
	if err := dbmigrate.Up(ctx, p, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		p.Close()
		return nil, err
	}
	return p, nil
}

// Cipher với khoá cố định cho test
func Cipher(t *testing.T) *privacy.Cipher {
	t.Helper()
	enc := make([]byte, 32)
	mac := make([]byte, 32)
	for i := range enc {
		enc[i], mac[i] = byte(i), byte(255-i)
	}
	c, err := privacy.NewFromKeys(enc, mac)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
