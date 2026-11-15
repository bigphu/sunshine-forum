package handler

import (
	"context"

	"github.com/bigphu/goat/auth"

	"sunshine/internal/identity/domain"
	"sunshine/internal/identity/handler/api"
)

func toMe(a domain.Account, perms []string, csrf string) api.Me {
	if perms == nil {
		perms = []string{}
	}
	return api.Me{
		Id: a.ID, Email: a.Email, Name: a.Name, Unit: a.Unit, Role: api.Role(a.Role),
		Permissions: perms, CsrfToken: csrf,
	}
}

func (h *Handler) Login(ctx context.Context, req api.LoginRequestObject) (api.LoginResponseObject, error) {
	in, err := h.svc.Login(ctx, req.Body.Email, req.Body.Password)
	if err != nil {
		return nil, err
	}
	return api.Login200JSONResponse{
		Body:    toMe(in.Account, h.svc.Permissions(in.Account.Role), in.CSRF),
		Headers: api.Login200ResponseHeaders{SetCookie: cookieHeader(h.cookie.set(in.Token))},
	}, nil
}

func (h *Handler) Logout(ctx context.Context, req api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	token := ""
	if req.Params.SunshineSession != nil {
		token = *req.Params.SunshineSession
	}
	if err := h.svc.Logout(ctx, token); err != nil {
		return nil, err
	}
	return api.Logout204Response{Headers: api.Logout204ResponseHeaders{SetCookie: cookieHeader(h.cookie.clear())}}, nil
}

func (h *Handler) ForgotPassword(ctx context.Context, req api.ForgotPasswordRequestObject) (api.ForgotPasswordResponseObject, error) {
	if err := h.svc.ForgotPassword(ctx, req.Body.Email); err != nil {
		return nil, err
	}
	return api.ForgotPassword202Response{}, nil
}

func (h *Handler) InspectPasswordToken(ctx context.Context, req api.InspectPasswordTokenRequestObject) (api.InspectPasswordTokenResponseObject, error) {
	info, err := h.svc.PasswordTokenInfo(ctx, req.Body.Token)
	if err != nil {
		return nil, err
	}
	return api.InspectPasswordToken200JSONResponse{
		Name: info.Name, Email: info.Email, Purpose: api.PasswordTokenInfoPurpose(info.Purpose),
	}, nil
}

func (h *Handler) SetPassword(ctx context.Context, req api.SetPasswordRequestObject) (api.SetPasswordResponseObject, error) {
	in, err := h.svc.SetPasswordWithToken(ctx, req.Body.Token, req.Body.Password)
	if err != nil {
		return nil, err
	}
	return api.SetPassword200JSONResponse{
		Body:    toMe(in.Account, h.svc.Permissions(in.Account.Role), in.CSRF),
		Headers: api.SetPassword200ResponseHeaders{SetCookie: cookieHeader(h.cookie.set(in.Token))},
	}, nil
}

func (h *Handler) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	a, perms, err := h.svc.Me(ctx)
	if err != nil {
		return nil, err
	}
	sess, _ := sessionFrom(ctx)
	return api.GetMe200JSONResponse(toMe(a, perms, sess.CSRFToken)), nil
}

func (h *Handler) ChangePassword(ctx context.Context, req api.ChangePasswordRequestObject) (api.ChangePasswordResponseObject, error) {
	sess, ok := sessionFrom(ctx)
	if !ok {
		return nil, auth.ErrUnauthenticated
	}
	if err := h.svc.ChangePassword(ctx, sess, req.Body.CurrentPassword, req.Body.NewPassword); err != nil {
		return nil, err
	}
	return api.ChangePassword204Response{}, nil
}
