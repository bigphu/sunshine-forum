// Package moderation lo hàng chờ kiểm duyệt, quyết định (cả hàng loạt), gỡ nội dung đã
// đăng, yêu cầu xem xét lại và nhật ký kiểm toán chỉ ghi thêm.
//
// Yêu cầu xem xét lại nằm trong module này (SYSTEM_DESIGN tách "appeals" riêng) vì quyết
// định xem xét lại phải ghi nhật ký kiểm toán trong cùng transaction, và nhật ký thuộc
// moderation.
//
//	m, err := moderation.New(moderation.Deps{Pool: pool, Outbox: outbox, Content: cr, Accounts: ar, Cipher: c})
//	m.Subscribe(registry)   // cmd/server và cmd/worker
//	err = m.Mount(r)        // cmd/server
package moderation

import (
	"fmt"

	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/web"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	content "sunshine/internal/content/contract"
	identity "sunshine/internal/identity/contract"
	"sunshine/internal/moderation/contract"
	"sunshine/internal/moderation/handler"
	"sunshine/internal/moderation/repository"
	"sunshine/internal/moderation/service"
	"sunshine/internal/moderation/subscriber"
	penalties "sunshine/internal/penalties/contract"
	"sunshine/internal/privacy"
)

type Deps struct {
	Pool     *pgxpool.Pool
	Outbox   *events.Outbox
	Content  content.Reader
	Accounts identity.AccountReader
	Standing penalties.StandingReader
	Cipher   *privacy.Cipher
}

type Module struct {
	svc     *service.Service
	handler *handler.Handler
}

func New(d Deps) (*Module, error) {
	if d.Pool == nil || d.Outbox == nil || d.Content == nil || d.Accounts == nil || d.Standing == nil || d.Cipher == nil {
		return nil, fmt.Errorf("moderation: Pool, Outbox, Content, Accounts, Standing and Cipher are required")
	}
	svc := service.New(service.Deps{
		Repo: repository.New(d.Pool, d.Outbox), Content: d.Content, Accounts: d.Accounts, Standing: d.Standing, Cipher: d.Cipher,
	})
	return &Module{svc: svc, handler: handler.New(svc)}, nil
}

func (m *Module) Mount(r chi.Router) error { return m.handler.Mount(r) }

// Subscribe: mở / đóng case theo event của content, ghi quyết định tự xoá
func (m *Module) Subscribe(r *events.Registry) {
	subscriber.Register(r, m.svc)
	subscriber.RegisterConfig(r, m.svc)
}

func (m *Module) Reader() contract.Reader { return m.svc.Reader() }

func (m *Module) Service() *service.Service { return m.svc }

func (m *Module) APIDoc() (web.APIDoc, error) {
	spec, err := handler.Spec()
	if err != nil {
		return web.APIDoc{}, fmt.Errorf("moderation: load openapi spec: %w", err)
	}
	return web.APIDoc{Name: "moderation", Spec: spec}, nil
}
