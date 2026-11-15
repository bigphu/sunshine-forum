package handler

import (
	"net/http"
	"time"
)

// CookieSettings: Secure phải tắt được khi dev chạy http (trình duyệt bỏ cookie Secure
// trên http, kể cả localhost)
type CookieSettings struct {
	Secure bool
	Domain string
	MaxAge time.Duration
}

// set là cookie phiên: HttpOnly để script không đọc được (chống XSS lấy phiên),
// SameSite=Lax cùng với X-CSRF-Token chống CSRF (NFR-3.2)
func (c CookieSettings) set(token string) *http.Cookie {
	return &http.Cookie{
		Name: CookieName, Value: token, Path: "/", Domain: c.Domain,
		MaxAge: int(c.MaxAge.Seconds()), HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteLaxMode,
	}
}

func (c CookieSettings) clear() *http.Cookie {
	return &http.Cookie{
		Name: CookieName, Value: "", Path: "/", Domain: c.Domain,
		MaxAge: -1, HttpOnly: true, Secure: c.Secure, SameSite: http.SameSiteLaxMode,
	}
}

func cookieHeader(c *http.Cookie) *string {
	s := c.String()
	return &s
}
