// Package repository cài đặt domain.Repository của content bằng sqlc (repository/db).
// Mỗi lần ghi phát event theo trạng thái mới, trong cùng transaction với thay đổi.
package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/bigphu/goat/database"
	"github.com/bigphu/goat/events"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"sunshine/internal/content/contract"
	"sunshine/internal/content/domain"
	"sunshine/internal/content/repository/db"
	screening "sunshine/internal/screening/contract"
	"sunshine/internal/vntext"
)

type Repository struct {
	pool   *pgxpool.Pool
	q      *db.Queries
	outbox *events.Outbox
}

var _ domain.Repository = (*Repository)(nil)

func New(pool *pgxpool.Pool, outbox *events.Outbox) *Repository {
	return &Repository{pool: pool, q: db.New(pool), outbox: outbox}
}

// --- chuyển đổi ---

func strp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func f64(v *float32) *float64 {
	if v == nil {
		return nil
	}
	x := math.Round(float64(*v)*10000) / 10000
	return &x
}

func f32(v *float64) *float32 {
	if v == nil {
		return nil
	}
	x := float32(*v)
	return &x
}

func i32(v *int) *int32 {
	if v == nil {
		return nil
	}
	x := int32(*v)
	return &x
}

func intp(v *int32) *int {
	if v == nil {
		return nil
	}
	x := int(*v)
	return &x
}

func sentencesJSON(s []screening.Sentence) json.RawMessage {
	if len(s) == 0 {
		return nil
	}
	b, _ := json.Marshal(s)
	return b
}

func toScreening(reason, label, model *string, score *float32, rule *int32, sentences json.RawMessage, rescan bool) domain.Screening {
	s := domain.Screening{
		Reason: deref(reason), Label: deref(label), Score: f64(score), ModelVersion: deref(model),
		Rule: intp(rule), NeedsRescan: rescan,
	}
	if len(sentences) > 0 {
		_ = json.Unmarshal(sentences, &s.Sentences)
	}
	return s
}

func toPost(r db.ContentPost) domain.Post {
	return domain.Post{
		ID: r.ID, ChannelID: r.ChannelID, AuthorID: r.AuthorID, Official: r.Official, Title: r.Title, Body: r.Body,
		Status: r.Status, Pinned: r.Pinned, Edited: r.Edited, CommentCount: int(r.CommentCount), OfficialReplied: r.OfficialReplied, CreatedAt: r.CreatedAt,
		Screening: toScreening(r.ScreenReason, r.AiLabel, r.ModelVersion, r.AiScore, r.FlagRule, r.AiSentences, r.NeedsRescan),
	}
}

func toPosts(rows []db.ContentPost) []domain.Post {
	out := make([]domain.Post, len(rows))
	for i, r := range rows {
		out[i] = toPost(r)
	}
	return out
}

func toComment(r db.ContentComment) domain.Comment {
	return domain.Comment{
		ID: r.ID, PostID: r.PostID, ParentID: r.ParentID, AuthorID: r.AuthorID, Official: r.Official, Body: r.Body,
		Status: r.Status, Edited: r.Edited, CreatedAt: r.CreatedAt,
		Screening: toScreening(r.ScreenReason, r.AiLabel, r.ModelVersion, r.AiScore, r.FlagRule, r.AiSentences, r.NeedsRescan),
	}
}

func toChannel(c db.ContentChannel) domain.Channel {
	return domain.Channel{ID: c.ID, Slug: c.Slug, Name: c.Name, Description: c.Description, OfficialOnly: c.OfficialOnly}
}

func searchText(title, body string) string { return vntext.Fold(title + " " + body) }

// --- event ---

func (r *Repository) emit(ctx context.Context, tx pgx.Tx, postID uuid.UUID, typ string, payload any) error {
	e, err := events.New(typ, contract.AggregatePost, postID, payload)
	if err != nil {
		return err
	}
	return r.outbox.Append(ctx, tx, e)
}

// subject là những gì cần để phát event cho một bài hoặc bình luận
type subject struct {
	ref            contract.Ref
	postID         uuid.UUID
	channelID      uuid.UUID
	authorID       uuid.UUID
	official       bool
	postAuthorID   uuid.UUID
	parentAuthorID *uuid.UUID
	rule           *int
}

// emitStatus phát event ứng với trạng thái mới. prev rỗng là nội dung mới.
func (r *Repository) emitStatus(ctx context.Context, tx pgx.Tx, s subject, prev, next string, firstTime bool) error {
	if prev == next && prev != contract.StatusPending {
		return nil
	}
	switch next {
	case contract.StatusPublished:
		return r.emit(ctx, tx, s.postID, contract.EventPublished, contract.Published{
			Ref: s.ref, PostID: s.postID, ChannelID: s.channelID, AuthorID: s.authorID, Official: s.official,
			PostAuthorID: s.postAuthorID, ParentAuthorID: s.parentAuthorID, FirstTime: firstTime,
		})
	case contract.StatusPending:
		return r.emit(ctx, tx, s.postID, contract.EventHeld, contract.Held{Ref: s.ref, ChannelID: s.channelID, AuthorID: s.authorID})
	case contract.StatusAutoDeleted:
		rule := 4
		if s.rule != nil {
			rule = *s.rule
		}
		return r.emit(ctx, tx, s.postID, contract.EventAutoDeleted, contract.AutoDeleted{
			Ref: s.ref, ChannelID: s.channelID, AuthorID: s.authorID, Rule: rule,
		})
	case contract.StatusDeleted:
		return r.emit(ctx, tx, s.postID, contract.EventWithdrawn, contract.Withdrawn{Ref: s.ref})
	}
	return nil
}

func postSubject(p db.ContentPost, ch db.ContentChannel) subject {
	return subject{
		ref: contract.Ref{Type: contract.TypePost, ID: p.ID}, postID: p.ID, channelID: p.ChannelID,
		authorID: p.AuthorID, official: p.Official && ch.OfficialOnly, postAuthorID: p.AuthorID, rule: intp(p.FlagRule),
	}
}

func (r *Repository) commentSubject(ctx context.Context, q *db.Queries, c db.ContentComment) (subject, error) {
	p, err := q.GetPost(ctx, c.PostID)
	if err != nil {
		return subject{}, err
	}
	s := subject{
		ref: contract.Ref{Type: contract.TypeComment, ID: c.ID}, postID: c.PostID, channelID: p.ChannelID,
		authorID: c.AuthorID, postAuthorID: p.AuthorID, rule: intp(c.FlagRule),
	}
	if c.ParentID != nil {
		parent, err := q.GetComment(ctx, *c.ParentID)
		if err != nil {
			return subject{}, err
		}
		s.parentAuthorID = &parent.AuthorID
	}
	return s, nil
}

// --- kênh ---

func (r *Repository) Channels(ctx context.Context) ([]domain.Channel, error) {
	rows, err := r.q.ListChannels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Channel, len(rows))
	for i, c := range rows {
		out[i] = toChannel(c)
	}
	return out, nil
}

func (r *Repository) Channel(ctx context.Context, id uuid.UUID) (domain.Channel, error) {
	c, err := r.q.GetChannel(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Channel{}, domain.ErrChannelNotFound
	}
	return toChannel(c), err
}

func (r *Repository) ChannelBySlug(ctx context.Context, slug string) (domain.Channel, error) {
	c, err := r.q.GetChannelBySlug(ctx, slug)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Channel{}, domain.ErrChannelNotFound
	}
	return toChannel(c), err
}

// --- bài viết ---

func (r *Repository) CreatePost(ctx context.Context, in domain.NewPost) (domain.Post, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Post{}, err
	}
	var out domain.Post
	err = database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		ch, err := q.GetChannel(ctx, in.ChannelID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrChannelNotFound
		}
		if err != nil {
			return err
		}
		s := in.Screening
		row, err := q.InsertPost(ctx, db.InsertPostParams{
			ID: id, ChannelID: in.ChannelID, AuthorID: in.AuthorID, Official: in.Official, Title: in.Title, Body: in.Body,
			SearchText: searchText(in.Title, in.Body), Status: in.Status,
			ScreenReason: strp(s.Reason), AiLabel: strp(s.Label), AiScore: f32(s.Score), ModelVersion: strp(s.ModelVersion),
			FlagRule: i32(s.Rule), AiSentences: sentencesJSON(s.Sentences), NeedsRescan: s.NeedsRescan,
		})
		if err != nil {
			return err
		}
		out = toPost(row)
		return r.emitStatus(ctx, tx, postSubject(row, ch), "", row.Status, true)
	})
	return out, err
}

func (r *Repository) GetPost(ctx context.Context, id uuid.UUID) (domain.Post, error) {
	row, err := r.q.GetPost(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Post{}, domain.ErrPostNotFound
	}
	return toPost(row), err
}

func (r *Repository) UpdatePost(ctx context.Context, id, author uuid.UUID, title, body, status string, s domain.Screening) (domain.Post, error) {
	var out domain.Post
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, err := q.GetPostForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && cur.AuthorID != author) {
			return domain.ErrPostNotFound
		}
		if err != nil {
			return err
		}
		if !domain.Editable(cur.Status) {
			return domain.ErrNotEditable
		}
		row, err := q.UpdatePostContent(ctx, db.UpdatePostContentParams{
			ID: id, Title: title, Body: body, SearchText: searchText(title, body), Status: status,
			ScreenReason: strp(s.Reason), AiLabel: strp(s.Label), AiScore: f32(s.Score), ModelVersion: strp(s.ModelVersion),
			FlagRule: i32(s.Rule), AiSentences: sentencesJSON(s.Sentences), NeedsRescan: s.NeedsRescan,
		})
		if err != nil {
			return err
		}
		ch, err := q.GetChannel(ctx, row.ChannelID)
		if err != nil {
			return err
		}
		out = toPost(row)
		// Bài đã đăng được sửa vẫn sạch: không báo lại cho ai
		return r.emitStatus(ctx, tx, postSubject(row, ch), cur.Status, row.Status, cur.Status != contract.StatusPublished)
	})
	return out, err
}

func (r *Repository) DeletePost(ctx context.Context, id, author uuid.UUID) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, err := q.GetPostForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && cur.AuthorID != author) {
			return domain.ErrPostNotFound
		}
		if err != nil {
			return err
		}
		if !domain.Editable(cur.Status) {
			return domain.ErrNotEditable
		}
		row, err := q.SetPostStatus(ctx, db.SetPostStatusParams{ID: id, Status: contract.StatusDeleted, FromStatuses: []string{cur.Status}})
		if err != nil {
			return err
		}
		ch, err := q.GetChannel(ctx, row.ChannelID)
		if err != nil {
			return err
		}
		return r.emitStatus(ctx, tx, postSubject(row, ch), cur.Status, contract.StatusDeleted, false)
	})
}

func (r *Repository) SetPinned(ctx context.Context, id uuid.UUID, pinned bool) error {
	n, err := r.q.SetPostPinned(ctx, db.SetPostPinnedParams{ID: id, Pinned: pinned})
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrPostNotFound
	}
	return nil
}

func (r *Repository) Feed(ctx context.Context, channel *uuid.UUID, before *domain.Cursor, limit int) ([]domain.Post, error) {
	p := db.ListFeedParams{ChannelID: channel, Lim: int32(limit)}
	if before != nil {
		p.BeforeAt, p.BeforeID = &before.At, &before.ID
	}
	rows, err := r.q.ListFeed(ctx, p)
	return toPosts(rows), err
}

func (r *Repository) Pinned(ctx context.Context, channel *uuid.UUID) ([]domain.Post, error) {
	rows, err := r.q.ListPinned(ctx, channel)
	return toPosts(rows), err
}

func (r *Repository) Search(ctx context.Context, folded string, limit int) ([]domain.Post, error) {
	// Thoát ký tự đặc biệt của LIKE; folded đã bỏ dấu câu nên chỉ còn chữ, số, khoảng trắng
	pattern := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(folded)
	rows, err := r.q.SearchPosts(ctx, db.SearchPostsParams{Pattern: pattern, Lim: int32(limit)})
	return toPosts(rows), err
}

// --- bình luận ---

func (r *Repository) CreateComment(ctx context.Context, in domain.NewComment) (domain.Comment, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return domain.Comment{}, err
	}
	var out domain.Comment
	err = database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		post, err := q.GetPostForUpdate(ctx, in.PostID)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrPostNotFound
		}
		if err != nil {
			return err
		}
		if post.Status != contract.StatusPublished {
			return domain.ErrPostClosed
		}
		s := in.Screening
		row, err := q.InsertComment(ctx, db.InsertCommentParams{
			ID: id, PostID: in.PostID, ParentID: in.ParentID, AuthorID: in.AuthorID, Official: in.Official,
			Body: in.Body, Status: in.Status,
			ScreenReason: strp(s.Reason), AiLabel: strp(s.Label), AiScore: f32(s.Score), ModelVersion: strp(s.ModelVersion),
			FlagRule: i32(s.Rule), AiSentences: sentencesJSON(s.Sentences), NeedsRescan: s.NeedsRescan,
		})
		if err != nil {
			return err
		}
		if err := q.RecountComments(ctx, in.PostID); err != nil {
			return err
		}
		subj, err := r.commentSubject(ctx, q, row)
		if err != nil {
			return err
		}
		out = toComment(row)
		return r.emitStatus(ctx, tx, subj, "", row.Status, true)
	})
	return out, err
}

func (r *Repository) GetComment(ctx context.Context, id uuid.UUID) (domain.Comment, error) {
	row, err := r.q.GetComment(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Comment{}, domain.ErrCommentNotFound
	}
	return toComment(row), err
}

func (r *Repository) UpdateComment(ctx context.Context, id, author uuid.UUID, body, status string, s domain.Screening) (domain.Comment, error) {
	var out domain.Comment
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, err := q.GetCommentForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && cur.AuthorID != author) {
			return domain.ErrCommentNotFound
		}
		if err != nil {
			return err
		}
		if !domain.Editable(cur.Status) {
			return domain.ErrNotEditable
		}
		row, err := q.UpdateCommentContent(ctx, db.UpdateCommentContentParams{
			ID: id, Body: body, Status: status,
			ScreenReason: strp(s.Reason), AiLabel: strp(s.Label), AiScore: f32(s.Score), ModelVersion: strp(s.ModelVersion),
			FlagRule: i32(s.Rule), AiSentences: sentencesJSON(s.Sentences), NeedsRescan: s.NeedsRescan,
		})
		if err != nil {
			return err
		}
		if err := q.RecountComments(ctx, row.PostID); err != nil {
			return err
		}
		subj, err := r.commentSubject(ctx, q, row)
		if err != nil {
			return err
		}
		out = toComment(row)
		return r.emitStatus(ctx, tx, subj, cur.Status, row.Status, cur.Status != contract.StatusPublished)
	})
	return out, err
}

func (r *Repository) DeleteComment(ctx context.Context, id, author uuid.UUID) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		cur, err := q.GetCommentForUpdate(ctx, id)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && cur.AuthorID != author) {
			return domain.ErrCommentNotFound
		}
		if err != nil {
			return err
		}
		if !domain.Editable(cur.Status) {
			return domain.ErrNotEditable
		}
		row, err := q.SetCommentStatus(ctx, db.SetCommentStatusParams{ID: id, Status: contract.StatusDeleted, FromStatuses: []string{cur.Status}})
		if err != nil {
			return err
		}
		if err := q.RecountComments(ctx, row.PostID); err != nil {
			return err
		}
		subj, err := r.commentSubject(ctx, q, row)
		if err != nil {
			return err
		}
		return r.emitStatus(ctx, tx, subj, cur.Status, contract.StatusDeleted, false)
	})
}

func (r *Repository) Thread(ctx context.Context, postID uuid.UUID) ([]domain.Comment, error) {
	rows, err := r.q.ListThreadComments(ctx, postID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Comment, len(rows))
	for i, c := range rows {
		out[i] = toComment(c)
	}
	return out, nil
}

func (r *Repository) PendingByAuthor(ctx context.Context, author uuid.UUID) ([]domain.Post, []domain.Comment, error) {
	posts, err := r.q.ListPendingPostsByAuthor(ctx, author)
	if err != nil {
		return nil, nil, err
	}
	comments, err := r.q.ListPendingCommentsByAuthor(ctx, author)
	if err != nil {
		return nil, nil, err
	}
	out := make([]domain.Comment, len(comments))
	for i, c := range comments {
		out[i] = toComment(c)
	}
	return toPosts(posts), out, nil
}

// --- kiểm duyệt và quét lại ---

func (r *Repository) SetStatus(ctx context.Context, ref contract.Ref, to string, from []string, firstTime bool) (bool, error) {
	changed := false
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		switch ref.Type {
		case contract.TypePost:
			cur, err := q.GetPostForUpdate(ctx, ref.ID)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			row, err := q.SetPostStatus(ctx, db.SetPostStatusParams{ID: ref.ID, Status: to, FromStatuses: from})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			changed = true
			ch, err := q.GetChannel(ctx, row.ChannelID)
			if err != nil {
				return err
			}
			if to != contract.StatusPublished {
				return nil
			}
			return r.emitStatus(ctx, tx, postSubject(row, ch), cur.Status, to, firstTime)
		case contract.TypeComment:
			cur, err := q.GetCommentForUpdate(ctx, ref.ID)
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			row, err := q.SetCommentStatus(ctx, db.SetCommentStatusParams{ID: ref.ID, Status: to, FromStatuses: from})
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			if err != nil {
				return err
			}
			changed = true
			if err := q.RecountComments(ctx, row.PostID); err != nil {
				return err
			}
			if to != contract.StatusPublished {
				return nil
			}
			subj, err := r.commentSubject(ctx, q, row)
			if err != nil {
				return err
			}
			return r.emitStatus(ctx, tx, subj, cur.Status, to, firstTime)
		}
		return fmt.Errorf("content: unknown content type %q", ref.Type)
	})
	return changed, err
}

func (r *Repository) ToRescan(ctx context.Context, limit int) ([]contract.Ref, error) {
	posts, err := r.q.ListPostsToRescan(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	comments, err := r.q.ListCommentsToRescan(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	var out []contract.Ref
	for _, p := range posts {
		out = append(out, contract.Ref{Type: contract.TypePost, ID: p.ID})
	}
	for _, c := range comments {
		out = append(out, contract.Ref{Type: contract.TypeComment, ID: c.ID})
	}
	return out, nil
}

func (r *Repository) ApplyRescan(ctx context.Context, ref contract.Ref, status string, s domain.Screening) error {
	return database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		switch ref.Type {
		case contract.TypePost:
			cur, err := q.GetPostForUpdate(ctx, ref.ID)
			if err != nil || !cur.NeedsRescan || !domain.Editable(cur.Status) {
				return ignoreNoRows(err)
			}
			row, err := q.UpdatePostScreening(ctx, db.UpdatePostScreeningParams{
				ID: ref.ID, Status: status, ScreenReason: strp(s.Reason), AiLabel: strp(s.Label), AiScore: f32(s.Score),
				ModelVersion: strp(s.ModelVersion), FlagRule: i32(s.Rule), AiSentences: sentencesJSON(s.Sentences),
			})
			if err != nil {
				return ignoreNoRows(err)
			}
			ch, err := q.GetChannel(ctx, row.ChannelID)
			if err != nil {
				return err
			}
			return r.emitStatus(ctx, tx, postSubject(row, ch), cur.Status, row.Status, true)
		case contract.TypeComment:
			cur, err := q.GetCommentForUpdate(ctx, ref.ID)
			if err != nil || !cur.NeedsRescan || !domain.Editable(cur.Status) {
				return ignoreNoRows(err)
			}
			row, err := q.UpdateCommentScreening(ctx, db.UpdateCommentScreeningParams{
				ID: ref.ID, Status: status, ScreenReason: strp(s.Reason), AiLabel: strp(s.Label), AiScore: f32(s.Score),
				ModelVersion: strp(s.ModelVersion), FlagRule: i32(s.Rule), AiSentences: sentencesJSON(s.Sentences),
			})
			if err != nil {
				return ignoreNoRows(err)
			}
			if err := q.RecountComments(ctx, row.PostID); err != nil {
				return err
			}
			subj, err := r.commentSubject(ctx, q, row)
			if err != nil {
				return err
			}
			return r.emitStatus(ctx, tx, subj, cur.Status, row.Status, true)
		}
		return nil
	})
}

func ignoreNoRows(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// --- cho module khác ---

func (r *Repository) Items(ctx context.Context, refs []contract.Ref) (map[contract.Ref]contract.Item, error) {
	var postIDs, commentIDs []uuid.UUID
	for _, ref := range refs {
		if ref.Type == contract.TypePost {
			postIDs = append(postIDs, ref.ID)
		} else {
			commentIDs = append(commentIDs, ref.ID)
		}
	}
	comments, err := r.q.GetCommentsByIDs(ctx, commentIDs)
	if err != nil {
		return nil, err
	}
	for _, c := range comments {
		postIDs = append(postIDs, c.PostID)
	}
	posts, err := r.q.GetPostsByIDs(ctx, postIDs)
	if err != nil {
		return nil, err
	}
	chans, err := r.q.ListChannels(ctx)
	if err != nil {
		return nil, err
	}
	chanName := map[uuid.UUID]string{}
	for _, c := range chans {
		chanName[c.ID] = c.Name
	}
	byID := map[uuid.UUID]db.ContentPost{}
	for _, p := range posts {
		byID[p.ID] = p
	}
	out := map[contract.Ref]contract.Item{}
	want := map[contract.Ref]bool{}
	for _, ref := range refs {
		want[ref] = true
	}
	for _, p := range posts {
		ref := contract.Ref{Type: contract.TypePost, ID: p.ID}
		if !want[ref] {
			continue
		}
		out[ref] = contract.Item{
			Ref: ref, PostID: p.ID, PostTitle: p.Title, Title: p.Title, Body: p.Body, ChannelID: p.ChannelID,
			ChannelName: chanName[p.ChannelID], AuthorID: p.AuthorID, Status: p.Status, CreatedAt: p.CreatedAt,
			Screening: contract.Screening{Reason: deref(p.ScreenReason), Label: deref(p.AiLabel), Score: f64(p.AiScore),
				ModelVersion: deref(p.ModelVersion), Rule: intp(p.FlagRule), Sentences: p.AiSentences},
		}
	}
	for _, c := range comments {
		p := byID[c.PostID]
		ref := contract.Ref{Type: contract.TypeComment, ID: c.ID}
		out[ref] = contract.Item{
			Ref: ref, PostID: c.PostID, PostTitle: p.Title, Body: c.Body, ChannelID: p.ChannelID,
			ChannelName: chanName[p.ChannelID], AuthorID: c.AuthorID, Status: c.Status, CreatedAt: c.CreatedAt,
			Screening: contract.Screening{Reason: deref(c.ScreenReason), Label: deref(c.AiLabel), Score: f64(c.AiScore),
				ModelVersion: deref(c.ModelVersion), Rule: intp(c.FlagRule), Sentences: c.AiSentences},
		}
	}
	return out, nil
}

func (r *Repository) Stats(ctx context.Context, since time.Time) (contract.Stats, map[string]int64, error) {
	n, err := r.q.CountSubmittedSince(ctx, since)
	if err != nil {
		return contract.Stats{}, nil, err
	}
	lb, err := r.q.FlaggedByLabel(ctx, since)
	if err != nil {
		return contract.Stats{}, nil, err
	}
	rows, err := r.q.FlaggedByDay(ctx, since)
	if err != nil {
		return contract.Stats{}, nil, err
	}
	byDay := map[string]int64{}
	for _, row := range rows {
		byDay[row.Day] = row.N
	}
	return contract.Stats{Submitted: n, Hate: lb.Hate, Offensive: lb.Offensive, Blocklist: lb.Blocklist, Language: lb.Language}, byDay, nil
}

func (r *Repository) Purge(ctx context.Context, days int) (int64, error) {
	var total int64
	err := database.WithTx(ctx, r.pool, func(tx pgx.Tx) error {
		q := r.q.WithTx(tx)
		if _, err := q.BlankPurgeableComments(ctx, int32(days)); err != nil {
			return err
		}
		c, err := q.PurgeComments(ctx, int32(days))
		if err != nil {
			return err
		}
		p, err := q.PurgePosts(ctx, int32(days))
		total = c + p
		return err
	})
	return total, err
}
