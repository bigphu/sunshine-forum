// Package penalties lo vi phạm đã xác nhận và tạm chỉ đọc (FR-1.3): vượt giới hạn trong
// khoảng thời gian trượt thì tài khoản chỉ được đọc tới hết hạn. content hỏi Standing
// trước mỗi lần viết, nên tạm chỉ đọc có hiệu lực ngay ở request kế tiếp.
//
//	m, err := penalties.New(penalties.Deps{Pool: pool, Outbox: outbox})
//	m.Subscribe(registry)   // cmd/server và cmd/worker
//	err = m.Mount(r)        // cmd/server
package penalties

import (
	"fmt"

	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/penalties/contract"
	"sunshine/internal/penalties/handler"
	"sunshine/internal/penalties/repository"
	"sunshine/internal/penalties/service"
	"sunshine/internal/penalties/subscriber"
)

type Deps struct {
	Pool   *pgxpool.Pool
	Outbox *events.Outbox
}

type Module struct {
	svc     *service.Service
	handler *handler.Handler
}

func New(d Deps) (*Module, error) {
	if d.Pool == nil || d.Outbox == nil {
		return nil, fmt.Errorf("penalties: Pool and Outbox are required")
	}
	svc := service.New(repository.New(d.Pool, d.Outbox))
	return &Module{svc: svc, handler: handler.New(svc)}, nil
}

func (m *Module) Mount(r chi.Router) error { return m.handler.Mount(r) }

func (m *Module) Subscribe(r *events.Registry) { subscriber.Register(r, m.svc) }

func (m *Module) Standing() contract.StandingReader { return m.svc }

func (m *Module) StatsReader() contract.StatsReader { return m.svc.StatsReader() }

func (m *Module) APIDoc() (web.APIDoc, error) {
	spec, err := handler.Spec()
	if err != nil {
		return web.APIDoc{}, fmt.Errorf("penalties: load openapi spec: %w", err)
	}
	return web.APIDoc{Name: "penalties", Spec: spec}, nil
}
