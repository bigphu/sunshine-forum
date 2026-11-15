// Package handler nối API penalties (strict server sinh từ openapi.yaml) với service.
package handler

//go:generate go tool oapi-codegen -config oapi.yaml openapi.yaml

import (
	"context"
	"fmt"

	"github.com/bigphu/goat/web"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"

	"sunshine/internal/penalties/domain"
	"sunshine/internal/penalties/handler/api"
	"sunshine/internal/penalties/service"
)

const baseURL = "/api/v1"

type Handler struct{ svc *service.Service }

var _ api.StrictServerInterface = (*Handler)(nil)

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router) error {
	spec, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("penalties: load openapi spec: %w", err)
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

func (h *Handler) GetMyStanding(ctx context.Context, _ api.GetMyStandingRequestObject) (api.GetMyStandingResponseObject, error) {
	st, cfg, err := h.svc.MyStanding(ctx)
	if err != nil {
		return nil, err
	}
	out := api.GetMyStanding200JSONResponse{
		ReadOnlyUntil: st.ReadOnlyUntil, ActiveStrikes: st.ActiveStrikes,
		StrikeLimit: cfg.StrikeLimit, StrikeWindowDays: cfg.StrikeWindowDays,
	}
	if st.ReadOnlyUntil != nil {
		out.LastRule = &st.LastRule
	}
	return out, nil
}

func toAPI(s domain.Settings) api.PenaltySettings {
	return api.PenaltySettings{StrikeLimit: s.StrikeLimit, StrikeWindowDays: s.StrikeWindowDays, TimeoutDays: s.TimeoutDays}
}

func (h *Handler) GetPenaltySettings(ctx context.Context, _ api.GetPenaltySettingsRequestObject) (api.GetPenaltySettingsResponseObject, error) {
	s, err := h.svc.Settings(ctx)
	if err != nil {
		return nil, err
	}
	return api.GetPenaltySettings200JSONResponse(toAPI(s)), nil
}

func (h *Handler) UpdatePenaltySettings(ctx context.Context, req api.UpdatePenaltySettingsRequestObject) (api.UpdatePenaltySettingsResponseObject, error) {
	s, err := h.svc.UpdateSettings(ctx, domain.Settings{
		StrikeLimit: req.Body.StrikeLimit, StrikeWindowDays: req.Body.StrikeWindowDays, TimeoutDays: req.Body.TimeoutDays,
	})
	if err != nil {
		return nil, err
	}
	return api.UpdatePenaltySettings200JSONResponse(toAPI(s)), nil
}

func (h *Handler) ListReadOnlyAccounts(ctx context.Context, _ api.ListReadOnlyAccountsRequestObject) (api.ListReadOnlyAccountsResponseObject, error) {
	list, err := h.svc.ReadOnlyAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListReadOnlyAccounts200JSONResponse, len(list))
	for i, t := range list {
		out[i] = api.ReadOnlyAccount{AccountId: t.AccountID, EndsAt: t.EndsAt, LastRule: t.Rule}
	}
	return out, nil
}
