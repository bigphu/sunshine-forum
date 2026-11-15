package privacy

import (
	"context"
	"log/slog"
	"time"

	"github.com/bigphu/goat/jobs"
	"github.com/bigphu/goat/logger"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

// Mỗi module tự xoá dữ liệu của mình sau 90 ngày (content, identity, notifications,
// administration). Job này xoá phần dùng chung: event của outbox (platform.events), vốn
// chỉ giữ ID nhưng vẫn không được giữ lâu hơn (SYSTEM_DESIGN, "90-day purge").

type PurgeEventsArgs struct{}

func (PurgeEventsArgs) Kind() string { return "privacy.purge_events" }

func (PurgeEventsArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault}
}

type purgeEvents struct {
	river.WorkerDefaults[PurgeEventsArgs]
	pool *pgxpool.Pool
	days int
}

func (w *purgeEvents) Work(ctx context.Context, _ *river.Job[PurgeEventsArgs]) error {
	tag, err := w.pool.Exec(ctx,
		`DELETE FROM platform.events WHERE occurred_at < now() - make_interval(days => $1)`, w.days)
	if err == nil && tag.RowsAffected() > 0 {
		logger.FromContext(ctx).InfoContext(ctx, "purged events", slog.Int64("rows", tag.RowsAffected()))
	}
	return err
}

// RegisterWorkers thêm job dọn event vào cmd/worker
func RegisterWorkers(workers *river.Workers, pool *pgxpool.Pool, days int) {
	if days <= 0 {
		days = 90
	}
	river.AddWorker(workers, &purgeEvents{pool: pool, days: days})
}

func PeriodicJobs() []*river.PeriodicJob {
	return []*river.PeriodicJob{
		river.NewPeriodicJob(river.PeriodicInterval(24*time.Hour),
			func() (river.JobArgs, *river.InsertOpts) { return PurgeEventsArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true}),
	}
}
