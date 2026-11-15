package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/bigphu/goat/auth"
	"github.com/bigphu/goat/logger"
	"github.com/bigphu/goat/web"

	"sunshine/internal/identity/domain"
	"sunshine/internal/identity/service"
)

const CookieName = "sunshine_session"

// CSRFHeader: client gửi lại csrf_token của phiên trong header này ở mọi request ghi
const CSRFHeader = "X-CSRF-Token"

type sessionKey struct{}

func sessionFrom(ctx context.Context) (domain.Session, bool) {
	s, ok := ctx.Value(sessionKey{}).(domain.Session)
	return s, ok
}

// SessionMiddleware gắn ở router gốc, trước mọi module. Có cookie phiên hợp lệ thì đặt
// Actor (quyền của vai trò hiện tại, đọc lại ở mỗi request) vào ctx; không có thì để
// trống và service trả 401 khi cần. Request ghi có phiên phải mang X-CSRF-Token đúng.
// Cookie hỏng hay hết hạn bị xoá.
func SessionMiddleware(svc *service.Service, cookie CookieSettings) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(CookieName)
			if err != nil || c.Value == "" {
				next.ServeHTTP(w, r)
				return
			}
			sess, actor, err := svc.Authenticate(r.Context(), c.Value)
			if errors.Is(err, domain.ErrSessionNotFound) {
				http.SetCookie(w, cookie.clear())
				next.ServeHTTP(w, r)
				return
			}
			if err != nil {
				web.WriteProblem(w, r, err)
				return
			}
			if !safeMethod(r.Method) {
				if err := service.CheckCSRF(sess, r.Header.Get(CSRFHeader)); err != nil {
					web.WriteProblem(w, r, err)
					return
				}
			}
			ctx := auth.WithActor(r.Context(), actor)
			ctx = context.WithValue(ctx, sessionKey{}, sess)
			logger.AddToScope(ctx, slog.String("actor_id", actor.AccountID.String()))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func safeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}
