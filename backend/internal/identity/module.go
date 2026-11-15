// Package identity lo tài khoản và đăng nhập: lời mời theo danh sách cư dân, ba vai trò
// cố định, phiên lưu trong DB (cookie HttpOnly + CSRF token), đặt lại mật khẩu.
//
// main dựng module một lần và dùng các phần nó cần:
//
//	m, err := identity.New(ctx, identity.Deps{Pool: pool, Outbox: outbox, Jobs: enq, Cipher: c, Config: cfg.Identity})
//	r.Use(m.SessionMiddleware())               // cmd/server: trước mọi module
//	err = m.Bootstrap(ctx)                     // cmd/server: Quản trị viên đầu tiên
//	err = m.Mount(r)                           // cmd/server: route /api/v1/...
//	events.RegisterWorker(..., m.LoadActor)    // cmd/worker
//	m.RegisterWorkers(workers)                 // cmd/worker (cần Deps.Mail), cùng identity.PeriodicJobs()
//
// Module khác chỉ dùng identity qua identity/contract (m.AccountReader()).
package identity

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/events"
	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/logger"
	"github.com/bigphu/goat/mail"
	"github.com/bigphu/goat/web"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"sunshine/internal/identity/contract"
	"sunshine/internal/identity/domain"
	"sunshine/internal/identity/handler"
	"sunshine/internal/identity/job"
	"sunshine/internal/identity/repository"
	"sunshine/internal/identity/service"
	"sunshine/internal/identity/worker"
	"sunshine/internal/privacy"
)

type Deps struct {
	Pool   *pgxpool.Pool
	Outbox *events.Outbox
	// Jobs xếp job gửi thư trong transaction phát link đặt mật khẩu
	Jobs   jobs.Enqueuer
	Cipher *privacy.Cipher
	// Mail chỉ cmd/worker cần (RegisterWorkers)
	Mail   mail.Sender
	Config Config
}

type Module struct {
	cfg     Config
	mail    mail.Sender
	svc     *service.Service
	handler *handler.Handler
	cookie  handler.CookieSettings
}

// New dựng module và nạp quyền của ba vai trò từ DB
func New(ctx context.Context, d Deps) (*Module, error) {
	if err := d.Config.Validate(); err != nil {
		return nil, err
	}
	if d.Pool == nil || d.Outbox == nil || d.Jobs == nil || d.Cipher == nil {
		return nil, fmt.Errorf("identity: Pool, Outbox, Jobs and Cipher are required")
	}
	cfg := d.Config.withDefaults()
	svc := service.New(service.Deps{
		Accounts: repository.NewAccountRepository(d.Pool, d.Outbox, d.Jobs, d.Cipher),
		Sessions: repository.NewSessionRepository(d.Pool),
		Tokens:   repository.NewTokenRepository(d.Pool, d.Jobs),
		Roles:    repository.NewRoleRepository(d.Pool),
		Hasher:   service.NewArgon2id(),
		Mail:     d.Mail,
		AppURL:   cfg.AppURL,
		Settings: service.Settings{
			SessionTTL: cfg.SessionTTL, InviteTTL: cfg.InviteTTL, ResetTTL: cfg.ResetTTL, PurgeAfterDays: cfg.PurgeAfterDays,
		},
	})
	if err := svc.LoadPermissions(ctx); err != nil {
		return nil, err
	}
	cookie := handler.CookieSettings{Secure: cfg.CookieSecure, Domain: cfg.CookieDomain, MaxAge: cfg.SessionTTL}
	return &Module{cfg: cfg, mail: d.Mail, svc: svc, handler: handler.New(svc, cookie), cookie: cookie}, nil
}

// SessionMiddleware đặt Actor của phiên vào ctx và kiểm tra CSRF; gắn ở router gốc
// trước khi module nào Mount.
func (m *Module) SessionMiddleware() func(http.Handler) http.Handler {
	return handler.SessionMiddleware(m.svc, m.cookie)
}

// Mount gắn route /api/v1/... của identity
func (m *Module) Mount(r chi.Router) error { return m.handler.Mount(r) }

// Bootstrap tạo Quản trị viên đầu tiên từ ADMIN_EMAIL/ADMIN_PASSWORD nếu chưa có tài
// khoản nào. Không đặt hai biến đó thì bỏ qua.
func (m *Module) Bootstrap(ctx context.Context) error {
	if m.cfg.AdminEmail == "" {
		return nil
	}
	created, err := m.svc.Bootstrap(ctx, m.cfg.AdminEmail, m.cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("identity: bootstrap admin: %w", err)
	}
	if created {
		logger.FromContext(ctx).InfoContext(ctx, "created bootstrap administrator")
	}
	return nil
}

// SeedAccount tạo tài khoản demo có mật khẩu (chỉ cmd/seed dùng)
func (m *Module) SeedAccount(ctx context.Context, email, name, unit, role, password string) (uuid.UUID, error) {
	a, err := m.svc.SeedAccount(ctx, service.Invitation{Email: email, Name: name, Unit: unit, Role: domain.Role(role)}, password)
	return a.ID, err
}

// LoadActor là jobs.ActorLoader cho events.RegisterWorker và job của module khác
func (m *Module) LoadActor(ctx context.Context, id uuid.UUID) (auth.Actor, error) {
	return m.svc.LoadActor(ctx, id)
}

// APIDoc cho trang tài liệu API (web.MountDocs)
func (m *Module) APIDoc() (web.APIDoc, error) {
	spec, err := handler.Spec()
	if err != nil {
		return web.APIDoc{}, fmt.Errorf("identity: load openapi spec: %w", err)
	}
	return web.APIDoc{Name: "identity", Spec: spec}, nil
}

func (m *Module) AccountReader() contract.AccountReader { return m.svc.AccountReader() }

// Service cho test
func (m *Module) Service() *service.Service { return m.svc }

// RegisterWorkers thêm worker của identity vào cmd/worker. Cần Deps.Mail.
func (m *Module) RegisterWorkers(workers *river.Workers) error {
	if m.mail == nil {
		return fmt.Errorf("identity: RegisterWorkers needs Deps.Mail")
	}
	river.AddWorker(workers, worker.NewSendAccountEmail(m.svc))
	river.AddWorker(workers, worker.NewMaintenance(m.svc))
	return nil
}

// PeriodicJobs: dọn phiên, link cũ và tài khoản đã khoá quá hạn, mỗi giờ
func PeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(time.Hour),
			func() (river.JobArgs, *river.InsertOpts) { return job.MaintenanceArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true}),
	}
}
