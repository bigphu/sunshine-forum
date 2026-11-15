// Package job là args của job River do identity xếp. Tách khỏi worker để repository
// (xếp job) không phải import code chạy job.
package job

import (
	"github.com/bigphu/goat/jobs"
	"github.com/riverqueue/river"
)

// SendAccountEmailArgs: gửi thư chứa link đặt mật khẩu (lời mời hoặc đặt lại).
// Repository xếp job này trong cùng transaction phát token. Token là token thô vì DB
// chỉ lưu hash; nó nằm trong river_job.args tới khi River dọn job đã xong (24 giờ).
type SendAccountEmailArgs struct {
	Token   string `json:"token"`
	Purpose string `json:"purpose"`
}

func (SendAccountEmailArgs) Kind() string { return "identity.send_account_email" }

func (SendAccountEmailArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault}
}

// MaintenanceArgs: job định kỳ dọn phiên hết hạn, link cũ và xoá dữ liệu cá nhân của
// tài khoản đã khoá quá 90 ngày (NFR-3.3).
type MaintenanceArgs struct{}

func (MaintenanceArgs) Kind() string { return "identity.maintenance" }

func (MaintenanceArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: jobs.QueueDefault}
}
