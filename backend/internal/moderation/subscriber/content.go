// Package subscriber: moderation phản ứng với event của content: mở case khi nội dung bị
// giữ lại, đóng case khi nội dung được đăng hoặc bị xoá, ghi quyết định tự xoá.
package subscriber

import (
	"context"

	"github.com/bigphu/goat/events"

	content "sunshine/internal/content/contract"
)

type handler interface {
	OnHeld(ctx context.Context, e content.Held) error
	OnGone(ctx context.Context, ref content.Ref) error
	OnAutoDeleted(ctx context.Context, e content.AutoDeleted) error
}

const Name = "moderation.track_content"

func Register(r *events.Registry, h handler) {
	r.On(Name, func(ctx context.Context, e events.Event) error {
		switch e.Type {
		case content.EventHeld:
			p, err := events.Decode[content.Held](e)
			if err != nil {
				return err
			}
			return h.OnHeld(ctx, p)
		case content.EventPublished:
			p, err := events.Decode[content.Published](e)
			if err != nil {
				return err
			}
			return h.OnGone(ctx, p.Ref)
		case content.EventWithdrawn:
			p, err := events.Decode[content.Withdrawn](e)
			if err != nil {
				return err
			}
			return h.OnGone(ctx, p.Ref)
		case content.EventAutoDeleted:
			p, err := events.Decode[content.AutoDeleted](e)
			if err != nil {
				return err
			}
			if err := h.OnGone(ctx, p.Ref); err != nil {
				return err
			}
			return h.OnAutoDeleted(ctx, p)
		}
		return nil
	}, content.EventHeld, content.EventPublished, content.EventWithdrawn, content.EventAutoDeleted)
}
