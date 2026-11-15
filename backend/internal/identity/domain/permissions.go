package domain

// Quyền của identity. Mã quyền nằm trong bảng identity.permissions (migration 00002);
// module khác thêm mã của mình bằng migration riêng.
const (
	PermAccountManage = "identity.account.manage"
)
