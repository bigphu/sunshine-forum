// cmd/migrate chạy migration goose (migrations/) rồi migration của River, sau đó thoát.
// Compose chỉ start app và worker khi migrate thoát 0.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/bigphu/goat/config"
	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/logger"

	"sunshine/internal/dbmigrate"
)

type migrateConfig struct {
	Log logger.Config
	DB  database.Config
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var cfg migrateConfig
	if err := config.Load(&cfg); err != nil {
		return err
	}
	log := logger.New(os.Stdout, cfg.Log)
	slog.SetDefault(log)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer database.Close(pool)
	if err := dbmigrate.Up(ctx, pool, log); err != nil {
		return err
	}
	log.Info("migrations done")
	return nil
}
