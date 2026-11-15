// Package handler nối API identity (strict server sinh từ openapi.yaml) với service,
// và cung cấp middleware phiên dùng cho mọi module. Không kiểm tra quyền (service làm),
// không đụng DB.
package handler

//go:generate go tool oapi-codegen -config oapi.yaml openapi.yaml

import (
	"fmt"
	"time"

	"github.com/bigphu/goat/middleware"
	"github.com/bigphu/goat/web"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"

	"sunshine/internal/identity/handler/api"
	"sunshine/internal/identity/service"
)

const BaseURL = "/api/v1"

// Handler cài đặt api.StrictServerInterface
type Handler struct {
	svc    *service.Service
	cookie CookieSettings
}

var _ api.StrictServerInterface = (*Handler)(nil)

func New(svc *service.Service, cookie CookieSettings) *Handler {
	return &Handler{svc: svc, cookie: cookie}
}

// Giới hạn theo IP cho route công khai dễ bị dò (SYSTEM_DESIGN: đăng nhập, đặt lại mật khẩu)
var authLimits = []struct {
	op    string
	every time.Duration
	burst int
}{
	// Đoán mật khẩu: 10 lần liền, sau đó 10 lần mỗi phút
	{"POST /api/v1/auth/login", 6 * time.Second, 10},
	// Form công khai gửi thư: 5 lần liền, sau đó mỗi phút một lần
	{"POST /api/v1/auth/password/forgot", time.Minute, 5},
	{"POST /api/v1/auth/password/inspect", 6 * time.Second, 10},
	{"POST /api/v1/auth/password/set", 6 * time.Second, 10},
}

// Mount gắn route của identity lên r (router gốc). Phần tử cuối của Middlewares chạy
// trước: giới hạn lượt (rẻ, trước khi đọc body), rồi validator. Phiên đã được
// SessionMiddleware đặt vào ctx ở router gốc.
func (h *Handler) Mount(r chi.Router) error {
	spec, err := api.GetSwagger()
	if err != nil {
		return fmt.Errorf("identity: load openapi spec: %w", err)
	}
	mws := []api.MiddlewareFunc{
		web.ValidateRequests(spec, BaseURL),
		middleware.ForOperations(web.MustOperations(spec, BaseURL,
			"POST /api/v1/auth/login", "POST /api/v1/auth/password/set", "GET /api/v1/me"), middleware.NoStore),
	}
	for _, l := range authLimits {
		mws = append(mws, middleware.ForOperations(web.MustOperations(spec, BaseURL, l.op),
			middleware.RateLimit(l.every, l.burst)))
	}
	strict := api.NewStrictHandlerWithOptions(h, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  web.RequestError,
		ResponseErrorHandlerFunc: web.WriteProblem,
	})
	api.HandlerWithOptions(strict, api.ChiServerOptions{
		BaseRouter:       r,
		BaseURL:          BaseURL,
		Middlewares:      mws,
		ErrorHandlerFunc: web.RequestError,
	})
	return nil
}

// Spec cho trang tài liệu API
func Spec() (*openapi3.T, error) { return api.GetSwagger() }
