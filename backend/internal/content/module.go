// Package content lo kênh, bài viết, bình luận hai cấp, ghim, tìm kiếm và dấu "Chính
// thức". Mọi nội dung đi qua screening trước khi lưu; quyết định kiểm duyệt đổi trạng
// thái qua event (subscriber), không gọi thẳng.
//
//	m, err := content.New(content.Deps{Pool: pool, Outbox: outbox, Screener: scr, Accounts: ar, Standing: pr, Config: cfg.Content})
//	m.Subscribe(registry)          // cmd/server và cmd/worker, trước khi dùng outbox
//	err = m.Mount(r)               // cmd/server
//	m.RegisterWorkers(workers)     // cmd/worker, cùng content.PeriodicJobs()
package content

import (
	"fmt"
	"time"

	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"sunshine/internal/content/contract"
	"sunshine/internal/content/handler"
	"sunshine/internal/content/job"
	"sunshine/internal/content/repository"
	"sunshine/internal/content/service"
	"sunshine/internal/content/subscriber"
	"sunshine/internal/content/worker"
	identity "sunshine/internal/identity/contract"
	penalties "sunshine/internal/penalties/contract"
	screening "sunshine/internal/screening/contract"
)

// Config của content, đọc bằng config.Load
type Config struct {
	// Nội dung bị xoá, từ chối hay gỡ được xoá hẳn sau ngần này ngày (NFR-3.3)
	PurgeAfterDays int `env:"CONTENT_PURGE_AFTER_DAYS" envDefault:"90"`
}

func (c Config) Validate() error {
	if c.PurgeAfterDays < 0 {
		return fmt.Errorf("content: CONTENT_PURGE_AFTER_DAYS must not be negative")
	}
	return nil
}

type Deps struct {
	Pool     *pgxpool.Pool
	Outbox   *events.Outbox
	Screener screening.Screener
	Accounts identity.AccountReader
	Standing penalties.StandingReader
	Config   Config
}

type Module struct {
	repo    *repository.Repository
	svc     *service.Service
	handler *handler.Handler
}

func New(d Deps) (*Module, error) {
	if err := d.Config.Validate(); err != nil {
		return nil, err
	}
	if d.Pool == nil || d.Outbox == nil || d.Screener == nil || d.Accounts == nil || d.Standing == nil {
		return nil, fmt.Errorf("content: Pool, Outbox, Screener, Accounts and Standing are required")
	}
	days := d.Config.PurgeAfterDays
	if days == 0 {
		days = 90
	}
	repo := repository.New(d.Pool, d.Outbox)
	svc := service.New(service.Deps{
		Repo: repo, Screener: d.Screener, Accounts: d.Accounts, Standing: d.Standing, PurgeAfterDays: days,
	})
	return &Module{repo: repo, svc: svc, handler: handler.New(svc)}, nil
}

func (m *Module) Mount(r chi.Router) error { return m.handler.Mount(r) }

// Subscribe: đổi trạng thái nội dung theo quyết định kiểm duyệt
func (m *Module) Subscribe(r *events.Registry) { subscriber.Register(r, m.repo) }

func (m *Module) Reader() contract.Reader { return m.svc.Reader() }

func (m *Module) Service() *service.Service { return m.svc }

func (m *Module) APIDoc() (web.APIDoc, error) {
	spec, err := handler.Spec()
	if err != nil {
		return web.APIDoc{}, fmt.Errorf("content: load openapi spec: %w", err)
	}
	return web.APIDoc{Name: "content", Spec: spec}, nil
}

func (m *Module) RegisterWorkers(workers *river.Workers) {
	river.AddWorker(workers, worker.NewRescan(m.svc))
	river.AddWorker(workers, worker.NewPurge(m.svc))
}

// PeriodicJobs: quét lại sau sự cố mỗi 15 giây, dọn nội dung quá hạn mỗi ngày
func PeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(15*time.Second),
			func() (river.JobArgs, *river.InsertOpts) { return job.RescanArgs{}, nil }, nil),
		river.NewPeriodicJob(river.PeriodicInterval(24*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) { return job.PurgeArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true}),
	}
}
