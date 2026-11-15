// Package administration lo thống kê bộ lọc và bản xuất dữ liệu huấn luyện. Cài đặt ngưỡng
// thuộc screening, giới hạn vi phạm thuộc penalties, nhật ký thuộc moderation, tài khoản
// thuộc identity; trang Quản trị của frontend gọi thẳng các API đó.
//
//	m, err := administration.New(administration.Deps{Pool: pool, Content: cr, Moderation: mr, Penalties: pr})
//	err = m.Mount(r)           // cmd/server
//	m.RegisterWorkers(workers) // cmd/worker, cùng administration.PeriodicJobs()
package administration

import (
	"context"
	"fmt"
	"time"

	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"sunshine/internal/administration/handler"
	"sunshine/internal/administration/repository"
	"sunshine/internal/administration/service"
	content "sunshine/internal/content/contract"
	moderation "sunshine/internal/moderation/contract"
	penalties "sunshine/internal/penalties/contract"
)

type Deps struct {
	Pool       *pgxpool.Pool
	Content    content.Reader
	Moderation moderation.Reader
	Penalties  penalties.StatsReader
}

type Module struct {
	svc     *service.Service
	handler *handler.Handler
}

func New(d Deps) (*Module, error) {
	if d.Pool == nil || d.Content == nil || d.Moderation == nil || d.Penalties == nil {
		return nil, fmt.Errorf("administration: Pool, Content, Moderation and Penalties are required")
	}
	svc := service.New(service.Deps{Repo: repository.New(d.Pool), Content: d.Content, Moderation: d.Moderation, Penalties: d.Penalties})
	return &Module{svc: svc, handler: handler.New(svc)}, nil
}

func (m *Module) Mount(r chi.Router) error { return m.handler.Mount(r) }

func (m *Module) APIDoc() (web.APIDoc, error) {
	spec, err := handler.Spec()
	if err != nil {
		return web.APIDoc{}, fmt.Errorf("administration: load openapi spec: %w", err)
	}
	return web.APIDoc{Name: "administration", Spec: spec}, nil
}

// PurgeExportsArgs: xoá bản xuất tới hạn
type PurgeExportsArgs struct{}

func (PurgeExportsArgs) Kind() string { return "administration.purge_exports" }

func (PurgeExportsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault}
}

type purgeWorker struct {
	river.WorkerDefaults[PurgeExportsArgs]
	svc *service.Service
}

func (w *purgeWorker) Work(ctx context.Context, _ *river.Job[PurgeExportsArgs]) error {
	return w.svc.Purge(ctx)
}

func (m *Module) RegisterWorkers(workers *river.Workers) {
	river.AddWorker(workers, &purgeWorker{svc: m.svc})
}

func PeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(time.Hour),
			func() (river.JobArgs, *river.InsertOpts) { return PurgeExportsArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true}),
	}
}
