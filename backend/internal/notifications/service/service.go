// Package service là use case của notifications: hộp thư, luồng Server-Sent Events, và
// phản ứng với event của content, moderation, penalties (tạo thông báo).
package service

import (
	"context"
	"log/slog"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/logger"
	"github.com/google/uuid"

	content "sunshine/internal/content/contract"
	identity "sunshine/internal/identity/contract"
	moderation "sunshine/internal/moderation/contract"
	"sunshine/internal/notifications/domain"
	penalties "sunshine/internal/penalties/contract"
)

// Đường dẫn trong ứng dụng (frontend) mà thông báo trỏ tới
const (
	linkRequests = "/yeu-cau"
	linkThread   = "/t/"
)

type Deps struct {
	Repo     domain.Repository
	Content  content.Reader
	Accounts identity.AccountReader
	Broker   *Broker
	// PurgeAfterDays: thông báo cũ hơn bị xoá
	PurgeAfterDays int
}

type Service struct {
	repo      domain.Repository
	content   content.Reader
	accounts  identity.AccountReader
	broker    *Broker
	purgeDays int
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, content: d.Content, accounts: d.Accounts, broker: d.Broker, purgeDays: d.PurgeAfterDays}
}

func (s *Service) List(ctx context.Context) ([]domain.Item, int, error) {
	actor, err := auth.Require(ctx, domain.PermRead)
	if err != nil {
		return nil, 0, err
	}
	return s.repo.List(ctx, actor.AccountID, 50)
}

func (s *Service) MarkAllRead(ctx context.Context) error {
	actor, err := auth.Require(ctx, domain.PermRead)
	if err != nil {
		return err
	}
	return s.repo.MarkAllRead(ctx, actor.AccountID)
}

// Subscribe cho handler SSE: kênh nhận tín hiệu khi tài khoản đang đăng nhập có thông báo mới
func (s *Service) Subscribe(ctx context.Context) (<-chan struct{}, func(), error) {
	actor, err := auth.Require(ctx, domain.PermRead)
	if err != nil {
		return nil, nil, err
	}
	ch, cancel := s.broker.Subscribe(actor.AccountID)
	return ch, cancel, nil
}

// Listen nối NOTIFY của DB với broker; cmd/server chạy trong goroutine
func (s *Service) Listen(ctx context.Context) error { return s.repo.Listen(ctx, s.broker.Publish) }

func (s *Service) Purge(ctx context.Context) error {
	n, err := s.repo.Purge(ctx, s.purgeDays)
	if err == nil && n > 0 {
		logger.FromContext(ctx).InfoContext(ctx, "purged notifications", slog.Int64("rows", n))
	}
	return err
}

// --- subscriber ---

func (s *Service) channelName(ctx context.Context, id uuid.UUID) string {
	c, err := s.content.Channel(ctx, id)
	if err != nil {
		return "diễn đàn"
	}
	return c.Name
}

// OnDecision: lời nhắn nêu quy tắc khi nội dung bị từ chối, gỡ hay tự xoá (FR-4.5)
func (s *Service) OnDecision(ctx context.Context, d moderation.DecisionRecorded) error {
	if !d.Violation() {
		return nil
	}
	return s.repo.Insert(ctx, []domain.NewItem{{
		AccountID: d.AuthorID, Kind: domain.KindModeration, DedupeKey: "decision:" + d.DecisionID.String(),
		Body: domain.DecisionMessage(d.Action, s.channelName(ctx, d.ChannelID), d.Rule, d.AppealDeadline),
		Link: linkRequests,
	}})
}

func (s *Service) OnAppeal(ctx context.Context, a moderation.AppealDecided) error {
	body := domain.AppealDeclinedMessage(a.Rule)
	if a.Accepted {
		body = domain.AppealAcceptedMessage
	}
	return s.repo.Insert(ctx, []domain.NewItem{{
		AccountID: a.AuthorID, Kind: domain.KindAppeal, DedupeKey: "appeal:" + a.AppealID.String(), Body: body, Link: linkRequests,
	}})
}

func (s *Service) OnReadOnly(ctx context.Context, e penalties.ReadOnlyStarted) error {
	return s.repo.Insert(ctx, []domain.NewItem{{
		AccountID: e.AccountID, Kind: domain.KindReadOnly, DedupeKey: "timeout:" + e.TimeoutID.String(),
		Body: domain.ReadOnlyMessage(e.EndsAt, e.Rule), Link: linkRequests,
	}})
}

// OnPublished: báo bình luận mới cho tác giả bài, trả lời cho tác giả bình luận, và tin
// chính thức cho mọi cư dân (FR-2.2). Không báo cho chính người viết, không báo lại khi
// nội dung được khôi phục.
func (s *Service) OnPublished(ctx context.Context, p content.Published) error {
	if !p.FirstTime {
		return nil
	}
	items, err := s.content.Items(ctx, []content.Ref{p.Ref})
	if err != nil {
		return err
	}
	it, ok := items[p.Ref]
	if !ok {
		return nil
	}
	key := "published:" + p.Ref.Type + ":" + p.Ref.ID.String()
	link := linkThread + p.PostID.String()
	var out []domain.NewItem
	if p.Ref.Type == content.TypePost {
		if !p.Official {
			return nil
		}
		residents, err := s.accounts.ActiveResidentIDs(ctx)
		if err != nil {
			return err
		}
		for _, id := range residents {
			if id != p.AuthorID {
				out = append(out, domain.NewItem{AccountID: id, Kind: domain.KindAnnounce, DedupeKey: key,
					Body: domain.AnnouncementMessage(it.PostTitle), Link: link})
			}
		}
	} else {
		authors, err := s.accounts.GetAccounts(ctx, []uuid.UUID{p.AuthorID})
		if err != nil {
			return err
		}
		who := authors[p.AuthorID].Name
		if p.ParentAuthorID != nil && *p.ParentAuthorID != p.AuthorID {
			out = append(out, domain.NewItem{AccountID: *p.ParentAuthorID, Kind: domain.KindReply, DedupeKey: key,
				Body: domain.ReplyMessage(who, it.PostTitle), Link: link + "#c-" + p.Ref.ID.String()})
		}
		if p.PostAuthorID != p.AuthorID && (p.ParentAuthorID == nil || *p.ParentAuthorID != p.PostAuthorID) {
			out = append(out, domain.NewItem{AccountID: p.PostAuthorID, Kind: domain.KindComment, DedupeKey: key,
				Body: domain.CommentMessage(who, it.PostTitle), Link: link + "#c-" + p.Ref.ID.String()})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return s.repo.Insert(ctx, out)
}
