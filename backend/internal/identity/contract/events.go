package contract

import "github.com/google/uuid"

// Event của identity, aggregate "identity.account". Payload chỉ có ID và mã, không có
// email hay tên (event bị dọn sau 90 ngày, nhưng không được giữ dữ liệu cá nhân).
const (
	AggregateAccount = "identity.account"

	EventAccountInvited     = "identity.account_invited"
	EventAccountActivated   = "identity.account_activated"
	EventAccountDeactivated = "identity.account_deactivated"
	EventRoleChanged        = "identity.role_changed"
)

type AccountInvited struct {
	AccountID uuid.UUID `json:"account_id"`
	Role      string    `json:"role"`
}

type AccountActivated struct {
	AccountID uuid.UUID `json:"account_id"`
}

type AccountDeactivated struct {
	AccountID uuid.UUID `json:"account_id"`
}

type RoleChanged struct {
	AccountID uuid.UUID `json:"account_id"`
	Role      string    `json:"role"`
}
