// cmd/worker chạy job nền trên River: giao event cho subscriber (đổi trạng thái nội dung,
// tính vi phạm, tạo thông báo), gửi thư mời / đặt lại mật khẩu, quét lại sau sự cố mô hình
// và dọn dữ liệu quá 90 ngày.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"github.com/bigphu/goat/config"
	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/logger"
	"github.com/bigphu/goat/mail"
	"github.com/riverqueue/river"

	"sunshine/internal/administration"
	"sunshine/internal/content"
	"sunshine/internal/identity"
	"sunshine/internal/moderation"
	"sunshine/internal/notifications"
	"sunshine/internal/penalties"
	"sunshine/internal/privacy"
	"sunshine/internal/screening"
)

// Thời gian cho job đang chạy hoàn tất khi tắt; compose cho 30s
const stopTimeout = 25 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	var cfg workerConfig
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
	// Subscriber và job có thể ghi thay đổi kèm event, nên worker cũng cần outbox
	insertClient, err := jobs.NewInsertClient(pool, log)
	if err != nil {
		return err
	}
	registry := events.NewRegistry()
	outbox := events.NewOutbox(registry, insertClient)
	enq := jobs.NewRiver(insertClient)
	sender, err := mail.New(cfg.Mail, log)
	if err != nil {
		return err
	}
	log.Info("mail transport ready", slog.String("transport", string(cfg.Mail.Transport)))

	identityMod, err := identity.New(ctx, identity.Deps{
		Pool: pool, Outbox: outbox, Jobs: enq, Cipher: cipher, Mail: sender, Config: cfg.Identity,
	})
	if err != nil {
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
	go func() { _ = screeningMod.Watch(ctx) }()

	workers := river.NewWorkers()
	events.RegisterWorker(workers, pool, registry, identityMod.LoadActor)
	if err := identityMod.RegisterWorkers(workers); err != nil {
		return err
	}
	contentMod.RegisterWorkers(workers)
	notificationsMod.RegisterWorkers(workers)
	adminMod.RegisterWorkers(workers)
	privacy.RegisterWorkers(workers, pool, cfg.EventsPurgeAfterDays)

	var periodic []*river.PeriodicJob
	for _, p := range [][]*river.PeriodicJob{
		identity.PeriodicJobs(), content.PeriodicJobs(), notifications.PeriodicJobs(),
		administration.PeriodicJobs(), privacy.PeriodicJobs(),
	} {
		periodic = append(periodic, p...)
	}
	client, err := jobs.NewWorkerClient(pool, log, cfg.Jobs, workers, periodic)
	if err != nil {
		return err
	}
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("worker: start: %w", err)
	}
	log.Info("worker started")
	<-ctx.Done()
	log.Info("worker stopping", slog.Duration("timeout", stopTimeout))
	stopCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	if err := client.Stop(stopCtx); err != nil {
		return fmt.Errorf("worker: stop: %w", err)
	}
	return nil
}
