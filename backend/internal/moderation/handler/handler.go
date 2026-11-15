// Package handler nối API moderation (strict server sinh từ openapi.yaml) với service.
package handler

//go:generate go tool oapi-codegen -config oapi.yaml openapi.yaml

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/bigphu/goat/web"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"

	content "sunshine/internal/content/contract"
	identity "sunshine/internal/identity/contract"
	"sunshine/internal/moderation/domain"
	"sunshine/internal/moderation/handler/api"
	"sunshine/internal/moderation/service"
)

const baseURL = "/api/v1"

type Handler struct{ svc *service.Service }

var _ api.StrictServerInterface = (*Handler)(nil)

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router) error {
	spec, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("moderation: load openapi spec: %w", err)
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

func toPerson(a identity.Account) api.Person { return api.Person{Id: a.ID, Name: a.Name, Unit: a.Unit} }

func toRef(r content.Ref) api.ContentRef {
	return api.ContentRef{Type: api.ContentRefType(r.Type), Id: r.ID}
}

func fromRef(r api.ContentRef) content.Ref { return content.Ref{Type: string(r.Type), ID: r.Id} }

func toItem(it content.Item, author identity.Account, strikes int) api.ContentItem {
	s := api.Screening{
		Reason: it.Screening.Reason, Label: it.Screening.Label, Score: it.Screening.Score,
		ModelVersion: it.Screening.ModelVersion, Rule: it.Screening.Rule, Sentences: []api.Sentence{},
	}
	if len(it.Screening.Sentences) > 0 {
		_ = json.Unmarshal(it.Screening.Sentences, &s.Sentences)
	}
	return api.ContentItem{
		Ref: toRef(it.Ref), PostId: it.PostID, PostTitle: it.PostTitle, Title: it.Title, Body: it.Body,
		ChannelId: it.ChannelID, ChannelName: it.ChannelName, Author: toPerson(author), Status: it.Status,
		Screening: s, CreatedAt: it.CreatedAt, AuthorStrikes: &strikes,
	}
}

func toDecision(d domain.Decision, moderator *identity.Account) api.Decision {
	out := api.Decision{
		Id: d.ID, Action: api.DecisionAction(d.Action), Rule: d.Rule,
		AppealDeadline: d.AppealDeadline, CreatedAt: d.CreatedAt,
	}
	if moderator != nil {
		p := toPerson(*moderator)
		out.Moderator = &p
	}
	return out
}

func toAppeal(a domain.Appeal) api.Appeal {
	return api.Appeal{
		Id: a.ID, DecisionId: a.DecisionID, Reason: a.Reason, Status: api.AppealStatus(a.Status),
		CreatedAt: a.CreatedAt, DecidedAt: a.DecidedAt,
	}
}

// --- route ---

func (h *Handler) GetQueue(ctx context.Context, _ api.GetQueueRequestObject) (api.GetQueueResponseObject, error) {
	q, err := h.svc.Queue(ctx)
	if err != nil {
		return nil, err
	}
	out := api.GetQueue200JSONResponse{Items: []api.ContentItem{}, Appeals: []api.QueueAppeal{}}
	for _, it := range q.Items {
		out.Items = append(out.Items, toItem(it.Item, it.Author, it.AuthorStrikes))
	}
	for _, a := range q.Appeals {
		out.Appeals = append(out.Appeals, api.QueueAppeal{
			Appeal: toAppeal(a.Appeal), Decision: toDecision(a.Decision, a.Moderator),
			Item: toItem(a.Item, a.Author, a.AuthorStrikes), CanReview: a.CanReview,
		})
	}
	return out, nil
}

func (h *Handler) Decide(ctx context.Context, req api.DecideRequestObject) (api.DecideResponseObject, error) {
	refs := make([]content.Ref, len(req.Body.Items))
	for i, r := range req.Body.Items {
		refs[i] = fromRef(r)
	}
	rule := 0
	if req.Body.Rule != nil {
		rule = *req.Body.Rule
	}
	n, err := h.svc.Decide(ctx, refs, string(req.Body.Action), rule)
	if err != nil {
		return nil, err
	}
	return api.Decide200JSONResponse{Decided: n}, nil
}

func (h *Handler) RemoveContent(ctx context.Context, req api.RemoveContentRequestObject) (api.RemoveContentResponseObject, error) {
	if err := h.svc.Remove(ctx, fromRef(req.Body.Item), req.Body.Rule); err != nil {
		return nil, err
	}
	return api.RemoveContent204Response{}, nil
}

func (h *Handler) ReviewAppeal(ctx context.Context, req api.ReviewAppealRequestObject) (api.ReviewAppealResponseObject, error) {
	if err := h.svc.ReviewAppeal(ctx, req.AppealId, req.Body.Accept); err != nil {
		return nil, err
	}
	return api.ReviewAppeal204Response{}, nil
}

func (h *Handler) CreateAppeal(ctx context.Context, req api.CreateAppealRequestObject) (api.CreateAppealResponseObject, error) {
	a, err := h.svc.CreateAppeal(ctx, req.Body.DecisionId, req.Body.Reason)
	if err != nil {
		return nil, err
	}
	return api.CreateAppeal201JSONResponse(toAppeal(a)), nil
}

func (h *Handler) ListMyDecisions(ctx context.Context, _ api.ListMyDecisionsRequestObject) (api.ListMyDecisionsResponseObject, error) {
	list, err := h.svc.MyDecisions(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListMyDecisions200JSONResponse, len(list))
	for i, d := range list {
		m := api.MyDecision{
			Decision: toDecision(d.Decision, nil), ContentType: api.MyDecisionContentType(d.Ref.Type), CanAppeal: d.CanAppeal,
		}
		if d.Item != nil {
			m.PostId, m.ChannelName, m.Title, m.Body = &d.Item.PostID, d.Item.ChannelName, d.Item.PostTitle, d.Item.Body
		}
		if d.Appeal != nil {
			a := toAppeal(*d.Appeal)
			m.Appeal = &a
		}
		out[i] = m
	}
	return out, nil
}

func strp(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (h *Handler) ListAudit(ctx context.Context, req api.ListAuditRequestObject) (api.ListAuditResponseObject, error) {
	filter := ""
	if req.Params.Filter != nil {
		filter = string(*req.Params.Filter)
	}
	list, next, err := h.svc.Audit(ctx, req.Params.Before, filter)
	if err != nil {
		return nil, err
	}
	out := api.ListAudit200JSONResponse{Items: make([]api.AuditEntry, len(list)), NextBefore: next}
	for i, e := range list {
		a := api.AuditEntry{
			Id: e.ID, Action: e.Action, Subject: e.Subject, ContentType: strp(e.ContentType), AiLabel: strp(e.AI.Label),
			AiScore: e.AI.Score, ModelVersion: strp(e.AI.ModelVersion), Rule: e.Rule, Outcome: e.Outcome,
			OverturnedAi: e.OverturnedAI, CreatedAt: e.CreatedAt,
		}
		if e.Moderator != nil {
			a.Moderator = &e.Moderator.Name
		}
		out.Items[i] = a
	}
	return out, nil
}
