// Package handler nối API administration (strict server sinh từ openapi.yaml) với service.
package handler

//go:generate go tool oapi-codegen -config oapi.yaml openapi.yaml

import (
	"bytes"
	"context"
	"fmt"

	"github.com/bigphu/goat/web"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"sunshine/internal/administration/domain"
	"sunshine/internal/administration/handler/api"
	"sunshine/internal/administration/service"
)

const baseURL = "/api/v1"

type Handler struct{ svc *service.Service }

var _ api.StrictServerInterface = (*Handler)(nil)

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Mount(r chi.Router) error {
	spec, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("administration: load openapi spec: %w", err)
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

func (h *Handler) GetAnalytics(ctx context.Context, req api.GetAnalyticsRequestObject) (api.GetAnalyticsResponseObject, error) {
	days := 30 // validator không áp default của schema
	if req.Params.Days != nil {
		days = int(*req.Params.Days)
	}
	a, err := h.svc.Analytics(ctx, days)
	if err != nil {
		return nil, err
	}
	c, m, p := a.Content, a.Moderation, a.Penalties
	out := api.GetAnalytics200JSONResponse{
		Days: a.Days, Checked: c.Submitted, Flagged: c.Flagged, Agreement: a.Agreement,
		Approved: m.Approved, Rejected: m.Rejected, Removed: m.Removed, AutoDeleted: m.AutoDeleted,
		AppealsPending: m.AppealsPending, AppealsAccepted: m.AppealsAccepted, AppealsDeclined: m.AppealsDeclined,
		ActiveStrikes: p.ActiveStrikes, ReadOnlyAccounts: p.ReadOnly, FlaggedByDay: []api.DayCount{},
	}
	out.ByLabel.Hate, out.ByLabel.Offensive, out.ByLabel.Blocklist, out.ByLabel.Language = c.Hate, c.Offensive, c.Blocklist, c.Language
	out.Overturned.HeldWrong, out.Overturned.Missed, out.Overturned.DeletedWrong = m.HeldWrong, m.Missed, m.DeletedWrong
	for _, d := range a.Content.FlaggedByDay {
		out.FlaggedByDay = append(out.FlaggedByDay, api.DayCount{Day: openapi_types.Date{Time: d.Day}, Count: d.Count})
	}
	return out, nil
}

func toExport(e domain.Export) api.Export {
	return api.Export{Id: e.ID, ItemCount: e.ItemCount, Purpose: api.ExportPurpose(e.Purpose), PeriodFrom: e.PeriodFrom, PeriodTo: e.PeriodTo,
		DeleteAfter: e.DeleteAfter, CreatedAt: e.CreatedAt}
}

func (h *Handler) ListExports(ctx context.Context, _ api.ListExportsRequestObject) (api.ListExportsResponseObject, error) {
	l, err := h.svc.Exports(ctx)
	if err != nil {
		return nil, err
	}
	out := api.ListExports200JSONResponse{Eligible: l.Eligible, PeriodFrom: l.PeriodFrom, PeriodTo: l.PeriodTo,
		Exports: make([]api.Export, len(l.Exports))}
	for i, e := range l.Exports {
		out.Exports[i] = toExport(e)
	}
	return out, nil
}

func (h *Handler) CreateExport(ctx context.Context, req api.CreateExportRequestObject) (api.CreateExportResponseObject, error) {
	e, err := h.svc.CreateExport(ctx, string(req.Body.Purpose))
	if err != nil {
		return nil, err
	}
	return api.CreateExport201JSONResponse(toExport(e)), nil
}

func (h *Handler) DownloadExport(ctx context.Context, req api.DownloadExportRequestObject) (api.DownloadExportResponseObject, error) {
	data, created, err := h.svc.ExportData(ctx, req.ExportId)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf(`attachment; filename="sunshine-retraining-%s.jsonl"`, created.Format("2006-01-02"))
	return api.DownloadExport200ApplicationxNdjsonResponse{
		Body: bytes.NewReader(data), ContentLength: int64(len(data)),
		Headers: api.DownloadExport200ResponseHeaders{ContentDisposition: &name},
	}, nil
}

func (h *Handler) DeleteExport(ctx context.Context, req api.DeleteExportRequestObject) (api.DeleteExportResponseObject, error) {
	if err := h.svc.DeleteExport(ctx, req.ExportId); err != nil {
		return nil, err
	}
	return api.DeleteExport204Response{}, nil
}
