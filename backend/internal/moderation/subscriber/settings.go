package subscriber

import (
	"context"

	"github.com/bigphu/goat/events"
	"github.com/google/uuid"

	"sunshine/internal/moderation/service"
	penalties "sunshine/internal/penalties/contract"
	screening "sunshine/internal/screening/contract"
)

type configHandler interface {
	OnConfigChanged(ctx context.Context, eventID uuid.UUID, changes []service.ConfigChange) error
}

const ConfigName = "moderation.audit_settings"

// RegisterConfig ghi thay đổi cài đặt bộ lọc và giới hạn vi phạm vào nhật ký kiểm toán
func RegisterConfig(r *events.Registry, h configHandler) {
	r.On(ConfigName, func(ctx context.Context, e events.Event) error {
		var changes []service.ConfigChange
		switch e.Type {
		case screening.EventSettingsChanged:
			p, err := events.Decode[screening.SettingsChanged](e)
			if err != nil {
				return err
			}
			for _, c := range p.Changes {
				changes = append(changes, service.ConfigChange{Field: c.Field, ChannelID: c.ChannelID, Old: c.Old, New: c.New})
			}
		case penalties.EventSettingsChanged:
			p, err := events.Decode[penalties.SettingsChanged](e)
			if err != nil {
				return err
			}
			for _, c := range p.Changes {
				changes = append(changes, service.ConfigChange{Field: c.Field, Old: c.Old, New: c.New})
			}
		}
		return h.OnConfigChanged(ctx, e.ID, changes)
	}, screening.EventSettingsChanged, penalties.EventSettingsChanged)
}
