// Package contract là phần công khai của identity cho module khác: đọc tên hiển thị
// của tác giả, danh sách người nhận thông báo, và tên các event.
package contract

import (
	"context"

	"github.com/google/uuid"
)

// Account là thông tin công khai của một tài khoản. Email không nằm ở đây: module
// khác không cần và không nên giữ.
type Account struct {
	ID    uuid.UUID
	Name  string
	Unit  string
	Role  string // resident | manager | admin
	Staff bool   // Quản lý hoặc Quản trị viên: bài của họ mang dấu "Chính thức"
}

// AccountReader không kiểm tra quyền: module gọi tự quyết định ai được xem gì.
type AccountReader interface {
	// GetAccounts trả các tài khoản tìm thấy, bỏ qua ID không tồn tại. Tài khoản đã
	// xoá dữ liệu có tên "Cựu cư dân".
	GetAccounts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Account, error)
	// ActiveResidentIDs: người nhận thông báo chính thức
	ActiveResidentIDs(ctx context.Context) ([]uuid.UUID, error)
}
