// Package handler nối API content (strict server sinh từ openapi.yaml) với service.
package handler

//go:generate go tool oapi-codegen -config oapi.yaml openapi.yaml

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/bigphu/goat/errs"
	"github.com/bigphu/goat/web"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"sunshine/internal/content/domain"
	"sunshine/internal/content/handler/api"
	"sunshine/internal/content/service"
	identity "sunshine/internal/identity/contract"
)

const baseURL = "/api/v1"

type Handler struct{ svc *service.Service }

var _ api.StrictServerInterface = (*Handler)(nil)

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router) error {
	spec, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("content: load openapi spec: %w", err)
	}
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  web.RequestError,
		ResponseErrorHandlerFunc: web.WriteProblem,
	})
	api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter: r, BaseURL: baseURL,
		Middlewares:      []api.MiddlewareFunc{web.ValidateRequests(spec, baseURL)},
		ErrorHandlerFunc: web.RequestError,
	})
	return nil
}

func Spec() (*openapi3.T, error) { return api.GetSwagger() }

// --- chuyển đổi ---

func toChannel(c domain.Channel) api.Channel {
	return api.Channel{Id: c.ID, Slug: c.Slug, Name: c.Name, Description: c.Description, OfficialOnly: c.OfficialOnly}
}

func toAuthor(a identity.Account) api.Author {
	return api.Author{Id: a.ID, Name: a.Name, Unit: a.Unit, Staff: a.Staff}
}

func toPost(v service.PostView) api.Post {
	return api.Post{
		Id: v.ID, Channel: toChannel(v.Channel), Author: toAuthor(v.Author), Official: v.Official,
		Title: v.Title, Body: v.Body, Status: api.ContentStatus(v.Status), Pinned: v.Pinned, Edited: v.Edited,
		CommentCount: v.CommentCount, OfficialReplied: v.OfficialReplied, CreatedAt: v.CreatedAt, Mine: v.Mine, ImitatesOfficial: v.ImitatesOfficial,
	}
}

func toPosts(vs []service.PostView) []api.Post {
	out := make([]api.Post, len(vs))
	for i, v := range vs {
		out[i] = toPost(v)
	}
	return out
}

func toComment(v service.CommentView) api.Comment {
	c := api.Comment{
		Id: v.ID, ParentId: v.ParentID, Body: v.Body, Status: api.ContentStatus(v.Status), Edited: v.Edited,
		Official: v.Official, CreatedAt: v.CreatedAt, Mine: v.Mine, Placeholder: v.Placeholder,
	}
	if !v.Placeholder {
		a := toAuthor(v.Author)
		c.Author = &a
	}
	return c
}

func toSubmitted(s service.Submitted) api.Submitted {
	return api.Submitted{PostId: s.PostID, CommentId: s.CommentID, Status: api.ContentStatus(s.Status)}
}

// Cursor dạng base64url("<unix nano>.<uuid>"); client chỉ chuyển lại, không đọc
func encodeCursor(c *domain.Cursor) *string {
	if c == nil {
		return nil
	}
	s := base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil, "%d.%s", c.At.UnixNano(), c.ID))
	return &s
}

var errBadCursor = errs.BadRequest("Trang không hợp lệ. Hãy tải lại.",
	errs.WithType("/errors/invalid-cursor"), errs.WithTitle("Invalid cursor"))

func decodeCursor(s *string) (*domain.Cursor, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(*s)
	if err != nil {
		return nil, errBadCursor
	}
	at, id, ok := strings.Cut(string(raw), ".")
	if !ok {
		return nil, errBadCursor
	}
	var nanos int64
	if _, err := fmt.Sscan(at, &nanos); err != nil {
		return nil, errBadCursor
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, errBadCursor
	}
	return &domain.Cursor{At: time.Unix(0, nanos), ID: uid}, nil
}

// --- route ---

func (h *Handler) ListChannels(ctx context.Context, _ api.ListChannelsRequestObject) (api.ListChannelsResponseObject, error) {
	chans, err := h.svc.Channels(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListChannels200JSONResponse, len(chans))
	for i, c := range chans {
		out[i] = toChannel(c)
	}
	return out, nil
}

func (h *Handler) ListPosts(ctx context.Context, req api.ListPostsRequestObject) (api.ListPostsResponseObject, error) {
	before, err := decodeCursor(req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	slug := ""
	if req.Params.Channel != nil {
		slug = *req.Params.Channel
	}
	f, err := h.svc.Feed(ctx, slug, before)
	if err != nil {
		return nil, err
	}
	return api.ListPosts200JSONResponse{Pinned: toPosts(f.Pinned), Items: toPosts(f.Items), NextCursor: encodeCursor(f.Next)}, nil
}

func (h *Handler) CreatePost(ctx context.Context, req api.CreatePostRequestObject) (api.CreatePostResponseObject, error) {
	s, err := h.svc.CreatePost(ctx, req.Body.ChannelId, req.Body.Title, req.Body.Body)
	if err != nil {
		return nil, err
	}
	return api.CreatePost201JSONResponse(toSubmitted(s)), nil
}

func (h *Handler) GetThread(ctx context.Context, req api.GetThreadRequestObject) (api.GetThreadResponseObject, error) {
	p, comments, err := h.svc.Thread(ctx, req.PostId)
	if err != nil {
		return nil, err
	}
	out := api.GetThread200JSONResponse{Post: toPost(p), Comments: make([]api.Comment, len(comments))}
	for i, c := range comments {
		out.Comments[i] = toComment(c)
	}
	return out, nil
}

func (h *Handler) UpdatePost(ctx context.Context, req api.UpdatePostRequestObject) (api.UpdatePostResponseObject, error) {
	s, err := h.svc.UpdatePost(ctx, req.PostId, req.Body.Title, req.Body.Body)
	if err != nil {
		return nil, err
	}
	return api.UpdatePost200JSONResponse(toSubmitted(s)), nil
}

func (h *Handler) DeletePost(ctx context.Context, req api.DeletePostRequestObject) (api.DeletePostResponseObject, error) {
	if err := h.svc.DeletePost(ctx, req.PostId); err != nil {
		return nil, err
	}
	return api.DeletePost204Response{}, nil
}

func (h *Handler) SetPostPinned(ctx context.Context, req api.SetPostPinnedRequestObject) (api.SetPostPinnedResponseObject, error) {
	if err := h.svc.SetPinned(ctx, req.PostId, req.Body.Pinned); err != nil {
		return nil, err
	}
	return api.SetPostPinned204Response{}, nil
}

func (h *Handler) CreateComment(ctx context.Context, req api.CreateCommentRequestObject) (api.CreateCommentResponseObject, error) {
	s, err := h.svc.CreateComment(ctx, req.PostId, req.Body.ParentId, req.Body.Body)
	if err != nil {
		return nil, err
	}
	return api.CreateComment201JSONResponse(toSubmitted(s)), nil
}

func (h *Handler) UpdateComment(ctx context.Context, req api.UpdateCommentRequestObject) (api.UpdateCommentResponseObject, error) {
	s, err := h.svc.UpdateComment(ctx, req.CommentId, req.Body.Body)
	if err != nil {
		return nil, err
	}
	return api.UpdateComment200JSONResponse(toSubmitted(s)), nil
}

func (h *Handler) DeleteComment(ctx context.Context, req api.DeleteCommentRequestObject) (api.DeleteCommentResponseObject, error) {
	if err := h.svc.DeleteComment(ctx, req.CommentId); err != nil {
		return nil, err
	}
	return api.DeleteComment204Response{}, nil
}

func (h *Handler) SearchPosts(ctx context.Context, req api.SearchPostsRequestObject) (api.SearchPostsResponseObject, error) {
	posts, err := h.svc.Search(ctx, req.Params.Q)
	if err != nil {
		return nil, err
	}
	return api.SearchPosts200JSONResponse(toPosts(posts)), nil
}

func (h *Handler) ListMyPending(ctx context.Context, _ api.ListMyPendingRequestObject) (api.ListMyPendingResponseObject, error) {
	posts, comments, parents, err := h.svc.MyPending(ctx)
	if err != nil {
		return nil, err
	}
	out := api.ListMyPending200JSONResponse{Posts: toPosts(posts), Comments: make([]api.PendingComment, len(comments))}
	for i, c := range comments {
		out.Comments[i] = api.PendingComment{Comment: toComment(c), Post: toPost(parents[c.PostID])}
	}
	return out, nil
}
