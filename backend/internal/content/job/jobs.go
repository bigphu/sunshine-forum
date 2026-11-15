// Package job là args của job River do content chạy định kỳ.
package job

import (
	"github.com/bigphu/goat/jobs"
	"github.com/riverqueue/river"
)

// RescanArgs: chấm lại nội dung bị giữ hoặc được đăng trong lúc mô hình sự cố (NFR-4.2)
type RescanArgs struct{}

func (RescanArgs) Kind() string { return "content.rescan" }

func (RescanArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault}
}

// PurgeArgs: xoá hẳn nội dung bị xoá, từ chối hay gỡ quá 90 ngày (NFR-3.3)
type PurgeArgs struct{}

func (PurgeArgs) Kind() string { return "content.purge" }

func (PurgeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault}
}
