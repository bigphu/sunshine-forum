// Package notifications lo hộp thư trong ứng dụng và luồng Server-Sent Events (FR-2.2):
// lời nhắn kiểm duyệt nêu quy tắc (FR-4.5), bình luận và trả lời mới, tin chính thức.
// Email chỉ dùng cho lời mời và đặt lại mật khẩu (identity). Web Push chưa làm.
//
//	m, err := notifications.New(notifications.Deps{Pool: pool, Content: cr, Accounts: ar, Config: cfg.Notifications})
//	m.Subscribe(registry)   // cmd/server và cmd/worker
//	go m.Listen(ctx)        // cmd/server: NOTIFY -> SSE
//	err = m.Mount(r)        // cmd/server
//	m.RegisterWorkers(workers) // cmd/worker, cùng notifications.PeriodicJobs()
package notifications

import (
	"context"
	"fmt"
	"time"

	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	content "sunshine/internal/content/contract"
	identity "sunshine/internal/identity/contract"
	"sunshine/internal/notifications/handler"
	"sunshine/internal/notifications/repository"
	"sunshine/internal/notifications/service"
	"sunshine/internal/notifications/subscriber"
)

type Config struct {
	// Thông báo cũ hơn ngần này ngày bị xoá
	PurgeAfterDays int `env:"NOTIFICATIONS_PURGE_AFTER_DAYS" envDefault:"90"`
}

func (c Config) Validate() error {
	if c.PurgeAfterDays < 0 {
		return fmt.Errorf("notifications: NOTIFICATIONS_PURGE_AFTER_DAYS must not be negative")
	}
	return nil
}

type Deps struct {
	Pool     *pgxpool.Pool
	Content  content.Reader
	Accounts identity.AccountReader
	Config   Config
}

type Module struct {
	svc     *service.Service
	handler *handler.Handler
}

func New(d Deps) (*Module, error) {
	if err := d.Config.Validate(); err != nil {
		return nil, err
	}
	if d.Pool == nil || d.Content == nil || d.Accounts == nil {
		return nil, fmt.Errorf("notifications: Pool, Content and Accounts are required")
	}
	days := d.Config.PurgeAfterDays
	if days == 0 {
		days = 90
	}
	svc := service.New(service.Deps{
		Repo: repository.New(d.Pool), Content: d.Content, Accounts: d.Accounts, Broker: service.NewBroker(), PurgeAfterDays: days,
	})
	return &Module{svc: svc, handler: handler.New(svc)}, nil
}

func (m *Module) Mount(r chi.Router) error { return m.handler.Mount(r) }

func (m *Module) Subscribe(r *events.Registry) { subscriber.Register(r, m.svc) }

// Listen nối NOTIFY notifications với các kết nối SSE; chạy tới khi ctx bị huỷ
func (m *Module) Listen(ctx context.Context) error { return m.svc.Listen(ctx) }

func (m *Module) APIDoc() (web.APIDoc, error) {
	spec, err := handler.Spec()
	if err != nil {
		return web.APIDoc{}, fmt.Errorf("notifications: load openapi spec: %w", err)
	}
	return web.APIDoc{Name: "notifications", Spec: spec}, nil
}

// PurgeArgs: xoá thông báo cũ (NFR-3.3)
type PurgeArgs struct{}

func (PurgeArgs) Kind() string { return "notifications.purge" }

func (PurgeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault}
}

type purgeWorker struct {
	river.WorkerDefaults[PurgeArgs]
	svc *service.Service
}

func (w *purgeWorker) Work(ctx context.Context, _ *river.Job[PurgeArgs]) error {
	return w.svc.Purge(ctx)
}

func (m *Module) RegisterWorkers(workers *river.Workers) {
	river.AddWorker(workers, &purgeWorker{svc: m.svc})
}

func PeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(24*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) { return PurgeArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true}),
	}
}
