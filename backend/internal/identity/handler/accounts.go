package handler

import (
	"context"

	"sunshine/internal/identity/domain"
	"sunshine/internal/identity/handler/api"
	"sunshine/internal/identity/service"
)

func toAPIAccount(v service.AccountView) api.Account {
	a := v.Account
	return api.Account{
		Id: a.ID, Email: a.Email, Name: a.Name, Unit: a.Unit, Role: api.Role(a.Role),
		Status: api.AccountStatus(a.Status), LastSignInAt: a.LastSignInAt, DeactivatedAt: a.DeactivatedAt,
		InviteExpiresAt: v.InviteExpiresAt, CreatedAt: a.CreatedAt,
	}
}

func (h *Handler) ListAccounts(ctx context.Context, _ api.ListAccountsRequestObject) (api.ListAccountsResponseObject, error) {
	list, err := h.svc.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make(api.ListAccounts200JSONResponse, len(list))
	for i, v := range list {
		out[i] = toAPIAccount(v)
	}
	return out, nil
}

func (h *Handler) InviteAccount(ctx context.Context, req api.InviteAccountRequestObject) (api.InviteAccountResponseObject, error) {
	unit := ""
	if req.Body.Unit != nil {
		unit = *req.Body.Unit
	}
	v, err := h.svc.Invite(ctx, service.Invitation{
		Email: req.Body.Email, Name: req.Body.Name, Unit: unit, Role: domain.Role(req.Body.Role),
	})
	if err != nil {
		return nil, err
	}
	return api.InviteAccount201JSONResponse(toAPIAccount(v)), nil
}

func (h *Handler) ReinviteAccount(ctx context.Context, req api.ReinviteAccountRequestObject) (api.ReinviteAccountResponseObject, error) {
	v, err := h.svc.Reinvite(ctx, req.AccountId)
	if err != nil {
		return nil, err
	}
	return api.ReinviteAccount200JSONResponse(toAPIAccount(v)), nil
}

func (h *Handler) DeactivateAccount(ctx context.Context, req api.DeactivateAccountRequestObject) (api.DeactivateAccountResponseObject, error) {
	v, err := h.svc.Deactivate(ctx, req.AccountId)
	if err != nil {
		return nil, err
	}
	return api.DeactivateAccount200JSONResponse(toAPIAccount(v)), nil
}

func (h *Handler) SetAccountRole(ctx context.Context, req api.SetAccountRoleRequestObject) (api.SetAccountRoleResponseObject, error) {
	v, err := h.svc.SetRole(ctx, req.AccountId, domain.Role(req.Body.Role))
	if err != nil {
		return nil, err
	}
	return api.SetAccountRole200JSONResponse(toAPIAccount(v)), nil
}

func (h *Handler) ImportAccounts(ctx context.Context, req api.ImportAccountsRequestObject) (api.ImportAccountsResponseObject, error) {
	people := make([]service.Invitation, len(req.Body.People))
	for i, p := range req.Body.People {
		unit := ""
		if p.Unit != nil {
			unit = *p.Unit
		}
		people[i] = service.Invitation{Email: p.Email, Name: p.Name, Unit: unit}
	}
	n, skipped, err := h.svc.Import(ctx, people)
	if err != nil {
		return nil, err
	}
	out := api.ImportAccounts200JSONResponse{Invited: n}
	for _, s := range skipped {
		out.Skipped = append(out.Skipped, struct {
			Email  string `json:"email"`
			Reason string `json:"reason"`
			Row    int    `json:"row"`
		}{Email: s.Email, Reason: s.Reason, Row: s.Row})
	}
	if out.Skipped == nil {
		out.Skipped = []struct {
			Email  string `json:"email"`
			Reason string `json:"reason"`
			Row    int    `json:"row"`
		}{}
	}
	return out, nil
}
