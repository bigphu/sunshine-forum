// Package subscriber: content phản ứng với quyết định kiểm duyệt bằng cách đổi trạng
// thái nội dung. Handler idempotent: chỉ đổi khi trạng thái hiện tại phù hợp.
package subscriber

import (
	"context"

	"github.com/bigphu/goat/events"

	"sunshine/internal/content/contract"
	moderation "sunshine/internal/moderation/contract"
)

// statusSetter là phần service subscriber cần
type statusSetter interface {
	SetStatus(ctx context.Context, ref contract.Ref, to string, from []string, firstTime bool) (bool, error)
}

const Name = "content.apply_moderation"

// Register đăng ký subscriber vào registry dùng chung của API và worker
func Register(r *events.Registry, repo statusSetter) {
	r.On(Name, func(ctx context.Context, e events.Event) error {
		switch e.Type {
		case moderation.EventDecisionRecorded:
			d, err := events.Decode[moderation.DecisionRecorded](e)
			if err != nil {
				return err
			}
			return onDecision(ctx, repo, d)
		case moderation.EventAppealDecided:
			a, err := events.Decode[moderation.AppealDecided](e)
			if err != nil {
				return err
			}
			if !a.Accepted {
				return nil
			}
			// Xem xét lại thành công: khôi phục nội dung, không báo lại cho người theo dõi
			_, err = repo.SetStatus(ctx, contract.Ref{Type: a.ContentType, ID: a.ContentID}, contract.StatusPublished,
				[]string{contract.StatusRejected, contract.StatusRemoved, contract.StatusAutoDeleted}, false)
			return err
		}
		return nil
	}, moderation.EventDecisionRecorded, moderation.EventAppealDecided)
}

func onDecision(ctx context.Context, repo statusSetter, d moderation.DecisionRecorded) error {
	ref := contract.Ref{Type: d.ContentType, ID: d.ContentID}
	var err error
	switch d.Action {
	case moderation.ActionApprove:
		_, err = repo.SetStatus(ctx, ref, contract.StatusPublished, []string{contract.StatusPending}, true)
	case moderation.ActionReject:
		_, err = repo.SetStatus(ctx, ref, contract.StatusRejected, []string{contract.StatusPending}, false)
	case moderation.ActionRemove:
		_, err = repo.SetStatus(ctx, ref, contract.StatusRemoved, []string{contract.StatusPublished}, false)
	}
	return err
}
