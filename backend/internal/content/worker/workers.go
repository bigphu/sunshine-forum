// Package worker chạy job River của content trong cmd/worker.
package worker

import (
	"context"

	"github.com/riverqueue/river"

	"sunshine/internal/content/job"
)

type jobsService interface {
	Rescan(ctx context.Context) error
	Purge(ctx context.Context) error
}

type Rescan struct {
	river.WorkerDefaults[job.RescanArgs]
	svc jobsService
}

func NewRescan(svc jobsService) *Rescan { return &Rescan{svc: svc} }

func (w *Rescan) Work(ctx context.Context, _ *river.Job[job.RescanArgs]) error {
	return w.svc.Rescan(ctx)
}

type Purge struct {
	river.WorkerDefaults[job.PurgeArgs]
	svc jobsService
}

func NewPurge(svc jobsService) *Purge { return &Purge{svc: svc} }

func (w *Purge) Work(ctx context.Context, _ *river.Job[job.PurgeArgs]) error { return w.svc.Purge(ctx) }
