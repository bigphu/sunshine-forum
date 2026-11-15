package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/bigphu/goat/web"
)

// Stream là GET /api/v1/notifications/stream (text/event-stream). Phiên đăng nhập do
// SessionMiddleware ở router gốc kiểm tra. nginx phải tắt buffering cho route này
// (deploy/nginx), nếu không thông báo tới trễ.
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	ch, cancel, err := h.svc.Subscribe(r.Context())
	if err != nil {
		web.WriteProblem(w, r, err)
		return
	}
	defer cancel()
	flusher, ok := w.(http.Flusher)
	if !ok {
		web.WriteProblem(w, r, fmt.Errorf("notifications: response writer cannot flush"))
		return
	}
	hd := w.Header()
	hd.Set("Content-Type", "text/event-stream")
	hd.Set("Cache-Control", "no-store")
	hd.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	// Báo client đã nối được, để nó tải hộp thư một lần
	_, _ = fmt.Fprint(w, "retry: 5000\nevent: ready\ndata: {}\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ch:
			if _, err := fmt.Fprint(w, "event: notification\ndata: {}\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
