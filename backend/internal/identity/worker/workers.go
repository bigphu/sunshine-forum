// Package worker chạy job River của identity trong cmd/worker.
package worker

import (
	"context"

	"github.com/bigphu/goat/mail"
	"github.com/riverqueue/river"

	"sunshine/internal/identity/job"
)

// accountMailer là phần service worker cần; *service.Service thoả interface này
type accountMailer interface {
	SendAccountEmail(ctx context.Context, args job.SendAccountEmailArgs) error
	Maintenance(ctx context.Context) error
}

type SendAccountEmail struct {
	river.WorkerDefaults[job.SendAccountEmailArgs]
	svc accountMailer
}

func NewSendAccountEmail(svc accountMailer) *SendAccountEmail { return &SendAccountEmail{svc: svc} }

// Work gửi thư; lỗi vĩnh viễn (địa chỉ hỏng, SMTP 5xx) thì huỷ job thay vì thử lại
// hàng giờ. Không cần actor: link trong job đã là quyền hạn của việc này.
func (w *SendAccountEmail) Work(ctx context.Context, j *river.Job[job.SendAccountEmailArgs]) error {
	err := w.svc.SendAccountEmail(ctx, j.Args)
	if mail.IsPermanent(err) {
		return river.JobCancel(err)
	}
	return err
}

type Maintenance struct {
	river.WorkerDefaults[job.MaintenanceArgs]
	svc accountMailer
}

func NewMaintenance(svc accountMailer) *Maintenance { return &Maintenance{svc: svc} }

func (w *Maintenance) Work(ctx context.Context, _ *river.Job[job.MaintenanceArgs]) error {
	return w.svc.Maintenance(ctx)
}
