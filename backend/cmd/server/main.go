// cmd/server chạy HTTP API.
//
//	go run ./cmd/server   // hoặc make run-api; cần .env.local và Postgres đã migrate
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	_ "time/tzdata" // múi giờ Asia/Ho_Chi_Minh kể cả khi image không có zoneinfo

	"github.com/bigphu/goat/config"
	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/logger"
	"github.com/bigphu/goat/server"
	"github.com/bigphu/goat/web"
	"github.com/go-chi/chi/v5"

	"sunshine/internal/administration"
	"sunshine/internal/content"
	"sunshine/internal/identity"
	"sunshine/internal/moderation"
	"sunshine/internal/notifications"
	"sunshine/internal/penalties"
	"sunshine/internal/privacy"
	"sunshine/internal/screening"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var cfg serverConfig
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
	cipher, err := privacy.New(cfg.Privacy)
	if err != nil {
		return err
	}
	insertClient, err := jobs.NewInsertClient(pool, log)
	if err != nil {
		return err
	}
	// API và worker dựng cùng registry: Append cần biết xếp job cho subscriber nào
	registry := events.NewRegistry()
	outbox := events.NewOutbox(registry, insertClient)
	enq := jobs.NewRiver(insertClient)

	identityMod, err := identity.New(ctx, identity.Deps{Pool: pool, Outbox: outbox, Jobs: enq, Cipher: cipher, Config: cfg.Identity})
	if err != nil {
		return err
	}
	if err := identityMod.Bootstrap(ctx); err != nil {
		return err
	}
	screeningMod, err := screening.New(screening.Deps{Pool: pool, Outbox: outbox, Config: cfg.Screening})
	if err != nil {
		return err
	}
	penaltiesMod, err := penalties.New(penalties.Deps{Pool: pool, Outbox: outbox})
	if err != nil {
		return err
	}
	contentMod, err := content.New(content.Deps{
		Pool: pool, Outbox: outbox, Screener: screeningMod.Screener(), Accounts: identityMod.AccountReader(),
		Standing: penaltiesMod.Standing(), Config: cfg.Content,
	})
	if err != nil {
		return err
	}
	moderationMod, err := moderation.New(moderation.Deps{
		Pool: pool, Outbox: outbox, Content: contentMod.Reader(), Accounts: identityMod.AccountReader(), Standing: penaltiesMod.Standing(), Cipher: cipher,
	})
	if err != nil {
		return err
	}
	notificationsMod, err := notifications.New(notifications.Deps{
		Pool: pool, Content: contentMod.Reader(), Accounts: identityMod.AccountReader(), Config: cfg.Notifications,
	})
	if err != nil {
		return err
	}
	adminMod, err := administration.New(administration.Deps{
		Pool: pool, Content: contentMod.Reader(), Moderation: moderationMod.Reader(), Penalties: penaltiesMod.StatsReader(),
	})
	if err != nil {
		return err
	}
	contentMod.Subscribe(registry)
	moderationMod.Subscribe(registry)
	penaltiesMod.Subscribe(registry)
	notificationsMod.Subscribe(registry)

	// Cache cài đặt sàng lọc và luồng thông báo nghe NOTIFY của Postgres
	go func() { _ = screeningMod.Watch(ctx) }()
	go func() { _ = notificationsMod.Listen(ctx) }()

	srv := server.New(cfg.HTTP, log)
	r := srv.Router()
	// Phiên đăng nhập và CSRF cho mọi route, trước khi module nào Mount
	r.Use(identityMod.SessionMiddleware())
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			_ = web.Text(w, http.StatusServiceUnavailable, "database unavailable")
			return
		}
		_ = web.Text(w, http.StatusOK, "ok")
	})
	type mountable interface {
		Mount(r chi.Router) error
		APIDoc() (web.APIDoc, error)
	}
	mods := []mountable{identityMod, screeningMod, contentMod, moderationMod, penaltiesMod, notificationsMod, adminMod}
	for _, m := range mods {
		if err := m.Mount(r); err != nil {
			return err
		}
	}
	// Swagger UI ở /api/docs, chỉ khi dev (HTTP_API_DOCS=true): trang công khai
	if cfg.HTTP.APIDocs {
		var docs []web.APIDoc
		for _, m := range mods {
			doc, err := m.APIDoc()
			if err != nil {
				return err
			}
			docs = append(docs, doc)
		}
		if err := web.MountDocs(r, docs...); err != nil {
			return err
		}
		log.Info("API docs enabled", slog.String("path", "/api/docs"))
	}
	return srv.Run(ctx)
}
