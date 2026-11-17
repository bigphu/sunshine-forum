// cmd/mockmodel là dịch vụ mô hình giả, nói đúng hợp đồng HTTP của dịch vụ thật
// (internal/screening/scorer/model-service.yaml). Điểm lấy từ scorer.StandIn nên là điểm
// mẫu. Dùng để chạy và thử đường HTTP (SCREENING_MODEL_URL) khi chưa có mô hình đã huấn
// luyện; không dùng ở production.
//
//	go run ./cmd/mockmodel            // :8000, hoặc MOCK_MODEL_ADDR
//
// Giả lập sự cố khi đang chạy (không cần dừng container):
//
//	PUT /mock/state {"offline": true}       /score và /health trả 503
//	PUT /mock/state {"delay_ms": 300}       chậm hơn hạn 200 ms của backend
//	GET /mock/state
package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"sunshine/internal/screening/contract"
	"sunshine/internal/screening/scorer"
)

// Version ghi vào kết quả sàng lọc và nhật ký, để không lẫn với mô hình thật
const Version = "mock-0.1 (điểm mẫu)"

type state struct {
	Offline bool `json:"offline"`
	DelayMS int  `json:"delay_ms"`
}

type server struct {
	mu sync.Mutex
	st state
}

func (s *server) current() state {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.st
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// wait áp độ trễ giả lập; trả false nếu bên gọi đã bỏ đi
func wait(r *http.Request, st state) bool {
	if st.DelayMS <= 0 {
		return true
	}
	select {
	case <-time.After(time.Duration(st.DelayMS) * time.Millisecond):
		return true
	case <-r.Context().Done():
		return false
	}
}

func (s *server) score(w http.ResponseWriter, r *http.Request) {
	st := s.current()
	if !wait(r, st) {
		return
	}
	if st.Offline {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"detail": "model offline (mock)"})
		return
	}
	var in struct {
		Text *string `json:"text"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil || in.Text == nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "body must be {\"text\": string}"})
		return
	}
	if len([]rune(*in.Text)) > contract.MaxChars {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "text is longer than 5000 characters"})
		return
	}
	res, err := scorer.StandIn{}.Score(r.Context(), *in.Text)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": err.Error()})
		return
	}
	sentences := res.Sentences
	if sentences == nil {
		sentences = []contract.Sentence{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"model_version": Version, "labels": res.Labels, "sentences": sentences})
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	st := s.current()
	if !wait(r, st) {
		return
	}
	if st.Offline {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "offline"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "model_version": Version})
}

func (s *server) mockState(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPut {
		var next state
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&next); err != nil || next.DelayMS < 0 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"detail": "body must be {\"offline\": bool, \"delay_ms\": int >= 0}"})
			return
		}
		s.mu.Lock()
		s.st = next
		s.mu.Unlock()
		slog.Info("mock state changed", slog.Bool("offline", next.Offline), slog.Int("delay_ms", next.DelayMS))
	}
	writeJSON(w, http.StatusOK, s.current())
}

func main() {
	addr := os.Getenv("MOCK_MODEL_ADDR")
	if addr == "" {
		addr = ":8000"
	}
	s := &server{}
	if v := os.Getenv("MOCK_MODEL_DELAY_MS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			slog.Error("MOCK_MODEL_DELAY_MS must be a non-negative integer")
			os.Exit(1)
		}
		s.st.DelayMS = n
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /score", s.score)
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /mock/state", s.mockState)
	mux.HandleFunc("PUT /mock/state", s.mockState)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	slog.Info("mock model service listening", slog.String("addr", addr), slog.String("model_version", Version))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("mock model service stopped", slog.String("err", err.Error()))
		os.Exit(1)
	}
}
