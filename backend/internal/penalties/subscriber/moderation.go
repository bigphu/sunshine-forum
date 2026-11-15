// Package subscriber: penalties tính vi phạm khi có quyết định kiểm duyệt và thu hồi khi
// yêu cầu xem xét lại thành công. Idempotent theo decision_id.
package subscriber

import (
	"context"

	"github.com/bigphu/goat/events"

	moderation "sunshine/internal/moderation/contract"
)

type handler interface {
	OnDecision(ctx context.Context, d moderation.DecisionRecorded) error
	OnAppeal(ctx context.Context, a moderation.AppealDecided) error
}

const Name = "penalties.count_strikes"

func Register(r *events.Registry, h handler) {
	r.On(Name, func(ctx context.Context, e events.Event) error {
		switch e.Type {
		case moderation.EventDecisionRecorded:
			d, err := events.Decode[moderation.DecisionRecorded](e)
			if err != nil {
				return err
			}
			return h.OnDecision(ctx, d)
		case moderation.EventAppealDecided:
			a, err := events.Decode[moderation.AppealDecided](e)
			if err != nil {
				return err
			}
			return h.OnAppeal(ctx, a)
		}
		return nil
	}, moderation.EventDecisionRecorded, moderation.EventAppealDecided)
}
