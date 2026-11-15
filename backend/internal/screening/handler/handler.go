// Package handler nối API screening (strict server sinh từ openapi.yaml) với service.
package handler

//go:generate go tool oapi-codegen -config oapi.yaml openapi.yaml

import (
	"context"
	"fmt"
	"time"

	"github.com/bigphu/goat/middleware"
	"github.com/bigphu/goat/web"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"sunshine/internal/screening/contract"
	"sunshine/internal/screening/domain"
	"sunshine/internal/screening/handler/api"
	"sunshine/internal/screening/service"
)

const baseURL = "/api/v1"

type Handler struct{ svc *service.Service }

var _ api.StrictServerInterface = (*Handler)(nil)

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Mount: kiểm tra bản nháp bị giới hạn theo IP (SYSTEM_DESIGN, rate limits)
func (h *Handler) Mount(r chi.Router) error {
	spec, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("screening: load openapi spec: %w", err)
	}
	mws := []api.MiddlewareFunc{
		web.ValidateRequests(spec, baseURL),
		middleware.ForOperations(web.MustOperations(spec, baseURL, "POST /api/v1/screening/check"),
			middleware.RateLimit(200*time.Millisecond, 30)),
	}
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  web.RequestError,
		ResponseErrorHandlerFunc: web.WriteProblem,
	})
	api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter: r, BaseURL: baseURL, Middlewares: mws, ErrorHandlerFunc: web.RequestError,
	})
	return nil
}

func Spec() (*openapi3.T, error) { return api.GetSwagger() }

func (h *Handler) ListRules(context.Context, api.ListRulesRequestObject) (api.ListRulesResponseObject, error) {
	out := make(api.ListRules200JSONResponse, len(contract.Rules))
	for i, r := range contract.Rules {
		out[i] = api.Rule{Number: r.Number, Title: r.Title, Text: r.Text}
	}
	return out, nil
}

func toSentences(ss []contract.Sentence) []api.Sentence {
	out := make([]api.Sentence, len(ss))
	for i, s := range ss {
		out[i] = api.Sentence{Start: s.Start, End: s.End, TopLabel: s.TopLabel, Score: s.Score}
	}
	return out
}

func (h *Handler) CheckDraft(ctx context.Context, req api.CheckDraftRequestObject) (api.CheckDraftResponseObject, error) {
	var channel uuid.UUID
	if req.Body.ChannelId != nil {
		channel = *req.Body.ChannelId
	}
	res, err := h.svc.Check(ctx, req.Body.Text, channel)
	if err != nil {
		return nil, err
	}
	out := api.CheckDraft200JSONResponse{ModelOffline: res.ModelOffline, Warnings: []api.Warning{}}
	for _, w := range res.Warnings {
		out.Warnings = append(out.Warnings, api.Warning{Rule: w.Rule, Message: w.Message, Sentences: toSentences(w.Sentences)})
	}
	return out, nil
}

func (h *Handler) GetScreeningStatus(ctx context.Context, _ api.GetScreeningStatusRequestObject) (api.GetScreeningStatusResponseObject, error) {
	st, err := h.svc.Status(ctx)
	if err != nil {
		return nil, err
	}
	return api.GetScreeningStatus200JSONResponse{ModelOnline: st.ModelOnline, ModelVersion: st.ModelVersion, StandIn: st.StandIn}, nil
}

func toBlock(b domain.BlockEntry) api.BlockEntry {
	return api.BlockEntry{Id: b.ID, Term: b.Term, Rule: b.Rule, CreatedBy: b.CreatedBy, CreatedAt: b.CreatedAt}
}

func (h *Handler) ListBlocklist(ctx context.Context, _ api.ListBlocklistRequestObject) (api.ListBlocklistResponseObject, error) {
	list, err := h.svc.Blocklist(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListBlocklist200JSONResponse, len(list))
	for i, b := range list {
		out[i] = toBlock(b)
	}
	return out, nil
}

func (h *Handler) AddBlocklistTerm(ctx context.Context, req api.AddBlocklistTermRequestObject) (api.AddBlocklistTermResponseObject, error) {
	b, err := h.svc.AddBlock(ctx, req.Body.Term, req.Body.Rule)
	if err != nil {
		return nil, err
	}
	return api.AddBlocklistTerm201JSONResponse(toBlock(b)), nil
}

func (h *Handler) DeleteBlocklistTerm(ctx context.Context, req api.DeleteBlocklistTermRequestObject) (api.DeleteBlocklistTermResponseObject, error) {
	if err := h.svc.DeleteBlock(ctx, req.TermId); err != nil {
		return nil, err
	}
	return api.DeleteBlocklistTerm204Response{}, nil
}

func (h *Handler) settings(ctx context.Context) (api.ScreeningSettings, error) {
	snap, err := h.svc.Settings(ctx)
	if err != nil {
		return api.ScreeningSettings{}, err
	}
	s := snap.Settings
	out := api.ScreeningSettings{
		HoldThreshold: s.HoldThreshold, AutoDeleteEnabled: s.AutoDeleteEnabled, AutoDeleteThreshold: s.AutoDeleteThreshold,
		OutagePolicy: api.OutagePolicy(s.OutagePolicy), StandIn: h.svc.IsStandIn(), StandInOffline: s.StandInOffline,
		ChannelThresholds: []api.ChannelThreshold{}, UpdatedAt: s.UpdatedAt,
	}
	for id, v := range snap.Channels {
		out.ChannelThresholds = append(out.ChannelThresholds, api.ChannelThreshold{ChannelId: id, HoldThreshold: v.Hold, AutoDeleteThreshold: v.AutoDelete})
	}
	return out, nil
}

func (h *Handler) GetScreeningSettings(ctx context.Context, _ api.GetScreeningSettingsRequestObject) (api.GetScreeningSettingsResponseObject, error) {
	s, err := h.settings(ctx)
	if err != nil {
		return nil, err
	}
	return api.GetScreeningSettings200JSONResponse(s), nil
}

func (h *Handler) UpdateScreeningSettings(ctx context.Context, req api.UpdateScreeningSettingsRequestObject) (api.UpdateScreeningSettingsResponseObject, error) {
	b := req.Body
	if _, err := h.svc.UpdateSettings(ctx, domain.Settings{
		HoldThreshold: b.HoldThreshold, AutoDeleteEnabled: b.AutoDeleteEnabled,
		AutoDeleteThreshold: b.AutoDeleteThreshold, OutagePolicy: string(b.OutagePolicy),
	}); err != nil {
		return nil, err
	}
	s, err := h.settings(ctx)
	if err != nil {
		return nil, err
	}
	return api.UpdateScreeningSettings200JSONResponse(s), nil
}

func (h *Handler) SetChannelThreshold(ctx context.Context, req api.SetChannelThresholdRequestObject) (api.SetChannelThresholdResponseObject, error) {
	var t *domain.ChannelThreshold
	if req.Body.HoldThreshold != nil {
		t = &domain.ChannelThreshold{Hold: *req.Body.HoldThreshold, AutoDelete: req.Body.AutoDeleteThreshold}
	}
	if err := h.svc.SetChannelThreshold(ctx, req.ChannelId, t); err != nil {
		return nil, err
	}
	return api.SetChannelThreshold204Response{}, nil
}

func (h *Handler) SetStandInOffline(ctx context.Context, req api.SetStandInOfflineRequestObject) (api.SetStandInOfflineResponseObject, error) {
	if err := h.svc.SetStandInOffline(ctx, req.Body.Offline); err != nil {
		return nil, err
	}
	return api.SetStandInOffline204Response{}, nil
}
