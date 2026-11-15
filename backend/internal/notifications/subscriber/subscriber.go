// Package subscriber: notifications tạo thông báo từ event của content, moderation và
// penalties. Idempotent nhờ dedupe_key.
package subscriber

import (
	"context"

	"github.com/bigphu/goat/events"

	content "sunshine/internal/content/contract"
	moderation "sunshine/internal/moderation/contract"
	penalties "sunshine/internal/penalties/contract"
)

type handler interface {
	OnDecision(ctx context.Context, d moderation.DecisionRecorded) error
	OnAppeal(ctx context.Context, a moderation.AppealDecided) error
	OnReadOnly(ctx context.Context, e penalties.ReadOnlyStarted) error
	OnPublished(ctx context.Context, p content.Published) error
}

const Name = "notifications.notify"

func Register(r *events.Registry, h handler) {
	r.On(Name, func(ctx context.Context, e events.Event) error {
		switch e.Type {
		case moderation.EventDecisionRecorded:
			p, err := events.Decode[moderation.DecisionRecorded](e)
			if err != nil {
				return err
			}
			return h.OnDecision(ctx, p)
		case moderation.EventAppealDecided:
			p, err := events.Decode[moderation.AppealDecided](e)
			if err != nil {
				return err
			}
			return h.OnAppeal(ctx, p)
		case penalties.EventReadOnlyStarted:
			p, err := events.Decode[penalties.ReadOnlyStarted](e)
			if err != nil {
				return err
			}
			return h.OnReadOnly(ctx, p)
		case content.EventPublished:
			p, err := events.Decode[content.Published](e)
			if err != nil {
				return err
			}
			return h.OnPublished(ctx, p)
		}
		return nil
	}, moderation.EventDecisionRecorded, moderation.EventAppealDecided, penalties.EventReadOnlyStarted, content.EventPublished)
}
