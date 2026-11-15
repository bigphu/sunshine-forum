// Package screening sàng lọc nội dung trước khi đăng: blocklist, kiểm tra ngôn ngữ, gọi
// mô hình (tối đa 200 ms), ngưỡng theo kênh và chính sách khi mô hình gặp sự cố.
//
//	m := screening.New(screening.Deps{Pool: pool})
//	go m.Watch(ctx)            // cmd/server, cmd/worker: làm mới cache khi Admin đổi cài đặt
//	err = m.Mount(r)           // cmd/server
//	content.New(content.Deps{Screener: m.Screener(), ...})
//
// Mô hình thật chưa nối: Scorer mặc định là StandIn (điểm mẫu). Xem package scorer.
package screening

import (
	"context"
	"fmt"

	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/screening/contract"
	"sunshine/internal/screening/handler"
	"sunshine/internal/screening/repository"
	"sunshine/internal/screening/scorer"
	"sunshine/internal/screening/service"
)

type Deps struct {
	Pool   *pgxpool.Pool
	Outbox *events.Outbox
	// Scorer nil: dùng StandIn
	Scorer scorer.Scorer
}

type Module struct {
	svc     *service.Service
	handler *handler.Handler
}

func New(d Deps) (*Module, error) {
	if d.Pool == nil || d.Outbox == nil {
		return nil, fmt.Errorf("screening: Pool and Outbox are required")
	}
	repo := repository.New(d.Pool, d.Outbox)
	sc, standIn := d.Scorer, d.Scorer == nil
	var svc *service.Service
	if standIn {
		stand := &scorer.StandIn{}
		svc = service.New(repo, stand, true)
		stand.Offline = svc.StandInOffline
	} else {
		svc = service.New(repo, sc, false)
	}
	return &Module{svc: svc, handler: handler.New(svc)}, nil
}

func (m *Module) Mount(r chi.Router) error { return m.handler.Mount(r) }

// Watch nghe NOTIFY screening_settings tới khi ctx bị huỷ
func (m *Module) Watch(ctx context.Context) error { return m.svc.Watch(ctx) }

func (m *Module) Screener() contract.Screener { return m.svc }

// Service cho cmd/seed (từ khoá chặn mẫu)
func (m *Module) Service() *service.Service { return m.svc }

func (m *Module) APIDoc() (web.APIDoc, error) {
	spec, err := handler.Spec()
	if err != nil {
		return web.APIDoc{}, fmt.Errorf("screening: load openapi spec: %w", err)
	}
	return web.APIDoc{Name: "screening", Spec: spec}, nil
}
