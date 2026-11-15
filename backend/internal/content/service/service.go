// Package service là use case của content: bảng tin, chủ đề, viết, sửa, xoá, ghim, tìm
// kiếm. Mọi bài và bình luận (kể cả khi sửa) đi qua screening trước khi lưu (FR-3.2).
package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/logger"
	"github.com/google/uuid"

	"sunshine/internal/content/contract"
	"sunshine/internal/content/domain"
	identity "sunshine/internal/identity/contract"
	penalties "sunshine/internal/penalties/contract"
	screening "sunshine/internal/screening/contract"
	"sunshine/internal/vntext"
)

const (
	feedPageSize   = 20
	searchPageSize = 30
)

type Deps struct {
	Repo     domain.Repository
	Screener screening.Screener
	Accounts identity.AccountReader
	Standing penalties.StandingReader
	// PurgeAfterDays: nội dung bị xoá, từ chối hay gỡ được xoá hẳn sau ngần này ngày
	PurgeAfterDays int
}

type Service struct {
	repo      domain.Repository
	screener  screening.Screener
	accounts  identity.AccountReader
	standing  penalties.StandingReader
	purgeDays int
}

func New(d Deps) *Service {
	return &Service{repo: d.Repo, screener: d.Screener, accounts: d.Accounts, standing: d.Standing, purgeDays: d.PurgeAfterDays}
}

// --- xem ---

type PostView struct {
	domain.Post
	Channel          domain.Channel
	Author           identity.Account
	Mine             bool
	ImitatesOfficial bool
}

type CommentView struct {
	domain.Comment
	Author identity.Account
	Mine   bool
	// Placeholder: bị ẩn hay xoá nhưng còn trả lời bên dưới; không có nội dung
	Placeholder bool
}

type Feed struct {
	Pinned []PostView
	Items  []PostView
	Next   *domain.Cursor
}

func (s *Service) Channels(ctx context.Context) ([]domain.Channel, error) {
	if _, err := auth.Require(ctx, domain.PermRead); err != nil {
		return nil, err
	}
	return s.repo.Channels(ctx)
}

func (s *Service) views(ctx context.Context, me uuid.UUID, posts []domain.Post) ([]PostView, error) {
	chans, err := s.repo.Channels(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[uuid.UUID]domain.Channel{}
	for _, c := range chans {
		byID[c.ID] = c
	}
	ids := make([]uuid.UUID, len(posts))
	for i, p := range posts {
		ids[i] = p.AuthorID
	}
	authors, err := s.accounts.GetAccounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]PostView, len(posts))
	for i, p := range posts {
		out[i] = PostView{
			Post: p, Channel: byID[p.ChannelID], Author: authors[p.AuthorID], Mine: p.AuthorID == me,
			ImitatesOfficial: domain.ImitatesOfficial(p.Official, p.Title, p.Body),
		}
	}
	return out, nil
}

// Feed: slug rỗng là Bảng tin (mọi kênh). Bài ghim chỉ có ở trang đầu.
func (s *Service) Feed(ctx context.Context, slug string, before *domain.Cursor) (Feed, error) {
	actor, err := auth.Require(ctx, domain.PermRead)
	if err != nil {
		return Feed{}, err
	}
	var channel *uuid.UUID
	if slug != "" {
		c, err := s.repo.ChannelBySlug(ctx, slug)
		if err != nil {
			return Feed{}, err
		}
		channel = &c.ID
	}
	var pinned []domain.Post
	if before == nil {
		if pinned, err = s.repo.Pinned(ctx, channel); err != nil {
			return Feed{}, err
		}
	}
	items, err := s.repo.Feed(ctx, channel, before, feedPageSize+1)
	if err != nil {
		return Feed{}, err
	}
	var next *domain.Cursor
	if len(items) > feedPageSize {
		items = items[:feedPageSize]
		last := items[len(items)-1]
		next = &domain.Cursor{At: last.CreatedAt, ID: last.ID}
	}
	all, err := s.views(ctx, actor.AccountID, append(pinned, items...))
	if err != nil {
		return Feed{}, err
	}
	return Feed{Pinned: all[:len(pinned)], Items: all[len(pinned):], Next: next}, nil
}

// Thread: bài và bình luận. Bài chưa hiển thị chỉ tác giả xem được. Bình luận bị ẩn để
// lại chỗ trống nếu còn trả lời bên dưới (PRODUCT.md).
func (s *Service) Thread(ctx context.Context, id uuid.UUID) (PostView, []CommentView, error) {
	actor, err := auth.Require(ctx, domain.PermRead)
	if err != nil {
		return PostView{}, nil, err
	}
	p, err := s.repo.GetPost(ctx, id)
	if err != nil {
		return PostView{}, nil, err
	}
	me := actor.AccountID
	if p.Status != contract.StatusPublished && (p.AuthorID != me || p.Status == contract.StatusDeleted) {
		return PostView{}, nil, domain.ErrPostNotFound
	}
	pv, err := s.views(ctx, me, []domain.Post{p})
	if err != nil {
		return PostView{}, nil, err
	}
	all, err := s.repo.Thread(ctx, id)
	if err != nil {
		return PostView{}, nil, err
	}
	visible := func(c domain.Comment) bool {
		return c.Status == contract.StatusPublished || (c.AuthorID == me && c.Status != contract.StatusDeleted)
	}
	hasVisibleReply := map[uuid.UUID]bool{}
	ids := []uuid.UUID{}
	for _, c := range all {
		if c.ParentID != nil && visible(c) {
			hasVisibleReply[*c.ParentID] = true
		}
		ids = append(ids, c.AuthorID)
	}
	authors, err := s.accounts.GetAccounts(ctx, ids)
	if err != nil {
		return PostView{}, nil, err
	}
	var out []CommentView
	for _, c := range all {
		switch {
		case visible(c):
			out = append(out, CommentView{Comment: c, Author: authors[c.AuthorID], Mine: c.AuthorID == me})
		case hasVisibleReply[c.ID]:
			out = append(out, CommentView{
				Comment:     domain.Comment{ID: c.ID, PostID: c.PostID, Status: c.Status, CreatedAt: c.CreatedAt},
				Placeholder: true,
			})
		}
	}
	return pv[0], out, nil
}

// Search tìm trong tiêu đề và nội dung bài đã đăng; gõ không dấu vẫn khớp
func (s *Service) Search(ctx context.Context, q string) ([]PostView, error) {
	actor, err := auth.Require(ctx, domain.PermRead)
	if err != nil {
		return nil, err
	}
	folded := vntext.Fold(q)
	if len([]rune(folded)) < 2 {
		return []PostView{}, nil
	}
	posts, err := s.repo.Search(ctx, folded, searchPageSize)
	if err != nil {
		return nil, err
	}
	return s.views(ctx, actor.AccountID, posts)
}

// MyPending: nội dung của mình đang chờ duyệt (trang "Yêu cầu của tôi")
func (s *Service) MyPending(ctx context.Context) ([]PostView, []CommentView, map[uuid.UUID]PostView, error) {
	actor, err := auth.Require(ctx, domain.PermRead)
	if err != nil {
		return nil, nil, nil, err
	}
	posts, comments, err := s.repo.PendingByAuthor(ctx, actor.AccountID)
	if err != nil {
		return nil, nil, nil, err
	}
	pv, err := s.views(ctx, actor.AccountID, posts)
	if err != nil {
		return nil, nil, nil, err
	}
	parents := map[uuid.UUID]PostView{}
	cv := make([]CommentView, len(comments))
	for i, c := range comments {
		cv[i] = CommentView{Comment: c, Mine: true}
		if _, ok := parents[c.PostID]; !ok {
			p, err := s.repo.GetPost(ctx, c.PostID)
			if err != nil {
				return nil, nil, nil, err
			}
			v, err := s.views(ctx, actor.AccountID, []domain.Post{p})
			if err != nil {
				return nil, nil, nil, err
			}
			parents[c.PostID] = v[0]
		}
	}
	return pv, cv, parents, nil
}

// --- viết ---

// requireWriter: quyền viết và không đang tạm chỉ đọc (kiểm tra ở mỗi request)
func (s *Service) requireWriter(ctx context.Context) (auth.Actor, error) {
	actor, err := auth.Require(ctx, domain.PermWrite)
	if err != nil {
		return auth.Actor{}, err
	}
	st, err := s.standing.Standing(ctx, actor.AccountID)
	if err != nil {
		return auth.Actor{}, err
	}
	if st.ReadOnlyUntil != nil {
		return auth.Actor{}, domain.ErrReadOnly(*st.ReadOnlyUntil)
	}
	return actor, nil
}

// screenPost chấm tiêu đề và nội dung trong hai lần riêng (SYSTEM_DESIGN §5), lấy kết quả nặng hơn
func (s *Service) screenPost(ctx context.Context, title, body string, channel uuid.UUID) (screening.Result, error) {
	t := screening.Target{ChannelID: channel}
	a, err := s.screener.Screen(ctx, title, t)
	if err != nil {
		return a, err
	}
	b, err := s.screener.Screen(ctx, body, t)
	if err != nil {
		return b, err
	}
	return s.screener.Worse(a, b), nil
}

// Submitted cho người viết biết nội dung đã hiển thị hay đang chờ xem xét
type Submitted struct {
	PostID    uuid.UUID
	CommentID *uuid.UUID
	Status    string
}

func (s *Service) CreatePost(ctx context.Context, channelID uuid.UUID, title, body string) (Submitted, error) {
	actor, err := s.requireWriter(ctx)
	if err != nil {
		return Submitted{}, err
	}
	ch, err := s.repo.Channel(ctx, channelID)
	if err != nil {
		return Submitted{}, err
	}
	if ch.OfficialOnly && !actor.Can(domain.PermAnnounce) {
		return Submitted{}, domain.ErrOfficialOnly
	}
	if title, err = domain.CleanTitle(title); err != nil {
		return Submitted{}, err
	}
	if body, err = domain.CleanBody(body); err != nil {
		return Submitted{}, err
	}
	res, err := s.screenPost(ctx, title, body, channelID)
	if err != nil {
		return Submitted{}, err
	}
	p, err := s.repo.CreatePost(ctx, domain.NewPost{
		ChannelID: channelID, AuthorID: actor.AccountID, Official: actor.Can(domain.PermAnnounce),
		Title: title, Body: body, Status: res.Outcome, Screening: domain.ScreeningFrom(res),
	})
	if err != nil {
		return Submitted{}, err
	}
	return Submitted{PostID: p.ID, Status: p.Status}, nil
}

func (s *Service) UpdatePost(ctx context.Context, id uuid.UUID, title, body string) (Submitted, error) {
	actor, err := s.requireWriter(ctx)
	if err != nil {
		return Submitted{}, err
	}
	cur, err := s.repo.GetPost(ctx, id)
	if err != nil {
		return Submitted{}, err
	}
	if title, err = domain.CleanTitle(title); err != nil {
		return Submitted{}, err
	}
	if body, err = domain.CleanBody(body); err != nil {
		return Submitted{}, err
	}
	// Nội dung sửa được quét lại (US-3.3)
	res, err := s.screenPost(ctx, title, body, cur.ChannelID)
	if err != nil {
		return Submitted{}, err
	}
	p, err := s.repo.UpdatePost(ctx, id, actor.AccountID, title, body, res.Outcome, domain.ScreeningFrom(res))
	if err != nil {
		return Submitted{}, err
	}
	return Submitted{PostID: p.ID, Status: p.Status}, nil
}

// DeletePost: xoá mềm, giữ 90 ngày rồi xoá hẳn (NFR-3.3)
func (s *Service) DeletePost(ctx context.Context, id uuid.UUID) error {
	actor, err := auth.Require(ctx, domain.PermWrite)
	if err != nil {
		return err
	}
	return s.repo.DeletePost(ctx, id, actor.AccountID)
}

func (s *Service) SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error {
	if _, err := auth.Require(ctx, domain.PermPin); err != nil {
		return err
	}
	return s.repo.SetPinned(ctx, id, pinned)
}

func (s *Service) CreateComment(ctx context.Context, postID uuid.UUID, parentID *uuid.UUID, body string) (Submitted, error) {
	actor, err := s.requireWriter(ctx)
	if err != nil {
		return Submitted{}, err
	}
	post, err := s.repo.GetPost(ctx, postID)
	if err != nil {
		return Submitted{}, err
	}
	if post.Status != contract.StatusPublished {
		return Submitted{}, domain.ErrPostNotFound
	}
	// Chỉ hai cấp: trả lời một trả lời thì gắn vào bình luận gốc
	if parentID != nil {
		parent, err := s.repo.GetComment(ctx, *parentID)
		if err != nil || parent.PostID != postID {
			return Submitted{}, domain.ErrInvalidParent
		}
		if parent.ParentID != nil {
			parentID = parent.ParentID
		}
	}
	if body, err = domain.CleanBody(body); err != nil {
		return Submitted{}, err
	}
	ch, err := s.repo.Channel(ctx, post.ChannelID)
	if err != nil {
		return Submitted{}, err
	}
	res, err := s.screener.Screen(ctx, body, screening.Target{ChannelID: ch.ID, AnnouncementComment: ch.OfficialOnly})
	if err != nil {
		return Submitted{}, err
	}
	c, err := s.repo.CreateComment(ctx, domain.NewComment{
		PostID: postID, ParentID: parentID, AuthorID: actor.AccountID, Official: actor.Can(domain.PermAnnounce),
		Body: body, Status: res.Outcome, Screening: domain.ScreeningFrom(res),
	})
	if err != nil {
		return Submitted{}, err
	}
	return Submitted{PostID: postID, CommentID: &c.ID, Status: c.Status}, nil
}

func (s *Service) UpdateComment(ctx context.Context, id uuid.UUID, body string) (Submitted, error) {
	actor, err := s.requireWriter(ctx)
	if err != nil {
		return Submitted{}, err
	}
	cur, err := s.repo.GetComment(ctx, id)
	if err != nil {
		return Submitted{}, err
	}
	if body, err = domain.CleanBody(body); err != nil {
		return Submitted{}, err
	}
	post, err := s.repo.GetPost(ctx, cur.PostID)
	if err != nil {
		return Submitted{}, err
	}
	ch, err := s.repo.Channel(ctx, post.ChannelID)
	if err != nil {
		return Submitted{}, err
	}
	res, err := s.screener.Screen(ctx, body, screening.Target{ChannelID: ch.ID, AnnouncementComment: ch.OfficialOnly})
	if err != nil {
		return Submitted{}, err
	}
	c, err := s.repo.UpdateComment(ctx, id, actor.AccountID, body, res.Outcome, domain.ScreeningFrom(res))
	if err != nil {
		return Submitted{}, err
	}
	return Submitted{PostID: c.PostID, CommentID: &c.ID, Status: c.Status}, nil
}

func (s *Service) DeleteComment(ctx context.Context, id uuid.UUID) error {
	actor, err := auth.Require(ctx, domain.PermWrite)
	if err != nil {
		return err
	}
	return s.repo.DeleteComment(ctx, id, actor.AccountID)
}

// --- job ---

// Rescan chấm lại nội dung bị giữ hoặc được đăng trong lúc mô hình sự cố (NFR-4.2), để
// quản lý không phải tự xử lý tồn đọng. Sau sự cố không tự xoá: nội dung vi phạm chỉ bị
// giữ lại chờ duyệt.
func (s *Service) Rescan(ctx context.Context) error {
	if !s.screener.ModelOnline(ctx) {
		return nil
	}
	refs, err := s.repo.ToRescan(ctx, 100)
	if err != nil {
		return err
	}
	items, err := s.repo.Items(ctx, refs)
	if err != nil {
		return err
	}
	for _, ref := range refs {
		it, ok := items[ref]
		if !ok {
			continue
		}
		ch, err := s.repo.Channel(ctx, it.ChannelID)
		if err != nil {
			return err
		}
		var res screening.Result
		if ref.Type == contract.TypePost {
			res, err = s.screenPost(ctx, it.Title, it.Body, it.ChannelID)
		} else {
			res, err = s.screener.Screen(ctx, it.Body, screening.Target{ChannelID: it.ChannelID, AnnouncementComment: ch.OfficialOnly})
		}
		if err != nil {
			return err
		}
		if res.NeedsRescan {
			return nil // mô hình lại sự cố giữa chừng: lần sau làm tiếp
		}
		status := res.Outcome
		if status == screening.OutcomeAutoDeleted {
			status = contract.StatusPending
		}
		if err := s.repo.ApplyRescan(ctx, ref, status, domain.ScreeningFrom(res)); err != nil {
			return err
		}
		logger.FromContext(ctx).InfoContext(ctx, "rescanned after outage",
			slog.String("type", ref.Type), slog.String("id", ref.ID.String()), slog.String("status", status))
	}
	return nil
}

func (s *Service) Purge(ctx context.Context) error {
	n, err := s.repo.Purge(ctx, s.purgeDays)
	if err == nil && n > 0 {
		logger.FromContext(ctx).InfoContext(ctx, "purged content", slog.Int64("rows", n))
	}
	return err
}

// --- contract.Reader ---

type reader struct{ s *Service }

func (s *Service) Reader() contract.Reader { return reader{s} }

func (r reader) Items(ctx context.Context, refs []contract.Ref) (map[contract.Ref]contract.Item, error) {
	return r.s.repo.Items(ctx, refs)
}

func (r reader) Channel(ctx context.Context, id uuid.UUID) (contract.Channel, error) {
	c, err := r.s.repo.Channel(ctx, id)
	return contract.Channel{ID: c.ID, Slug: c.Slug, Name: c.Name, OfficialOnly: c.OfficialOnly}, err
}

var vn = time.FixedZone("ICT", 7*3600)

func (r reader) Stats(ctx context.Context, since time.Time, days int) (contract.Stats, error) {
	out, byDay, err := r.s.repo.Stats(ctx, since)
	if err != nil {
		return contract.Stats{}, err
	}
	today := time.Now().In(vn)
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, vn).AddDate(0, 0, -(days - 1))
	for i := range days {
		d := start.AddDate(0, 0, i)
		c := byDay[d.Format("2006-01-02")]
		out.FlaggedByDay = append(out.FlaggedByDay, contract.DayCount{Day: d, Count: c})
	}
	for _, c := range byDay {
		out.Flagged += c
	}
	return out, nil
}
