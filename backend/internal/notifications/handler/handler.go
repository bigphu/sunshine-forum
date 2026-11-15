// Package handler nối API notifications với service: route sinh từ openapi.yaml và luồng
// Server-Sent Events viết tay (stream.go).
package handler

//go:generate go tool oapi-codegen -config oapi.yaml openapi.yaml

import (
	"context"
	"fmt"

	"github.com/bigphu/goat/web"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"

	"sunshine/internal/notifications/handler/api"
	"sunshine/internal/notifications/service"
)

const baseURL = "/api/v1"

type Handler struct{ svc *service.Service }

var _ api.StrictServerInterface = (*Handler)(nil)

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router) error {
	spec, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("notifications: load openapi spec: %w", err)
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
	r.Get(baseURL+"/notifications/stream", h.Stream)
	return nil
}

func Spec() (*openapi3.T, error) { return api.GetSwagger() }

func (h *Handler) ListNotifications(ctx context.Context, _ api.ListNotificationsRequestObject) (api.ListNotificationsResponseObject, error) {
	items, unread, err := h.svc.List(ctx)
	if err != nil {
		return nil, err
	}
	out := api.ListNotifications200JSONResponse{Items: make([]api.Notification, len(items)), Unread: unread}
	for i, it := range items {
		out.Items[i] = api.Notification{
			Id: it.ID, Kind: api.NotificationKind(it.Kind), Body: it.Body, Link: it.Link,
			ReadAt: it.ReadAt, CreatedAt: it.CreatedAt,
		}
	}
	return out, nil
}

func (h *Handler) MarkNotificationsRead(ctx context.Context, _ api.MarkNotificationsReadRequestObject) (api.MarkNotificationsReadResponseObject, error) {
	if err := h.svc.MarkAllRead(ctx); err != nil {
		return nil, err
	}
	return api.MarkNotificationsRead204Response{}, nil
}
