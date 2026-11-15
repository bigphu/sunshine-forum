package domain

import "github.com/bigphu/goat/errs"

// ErrSessionNotFound không bao giờ tới client: middleware coi như chưa đăng nhập
var ErrSessionNotFound = errs.Unauthorized("Phiên đăng nhập đã hết hạn. Hãy đăng nhập lại.",
	errs.WithType("/errors/session-expired"), errs.WithTitle("Session expired"))

// ErrCSRF: request ghi có cookie phiên nhưng thiếu hoặc sai header X-CSRF-Token
var ErrCSRF = errs.Forbidden("Yêu cầu không hợp lệ. Hãy tải lại trang rồi thử lại.",
	errs.WithType("/errors/csrf"), errs.WithTitle("CSRF token missing or invalid"))
