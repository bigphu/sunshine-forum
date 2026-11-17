package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sunshine/internal/screening/scorer"
)

func newMock(t *testing.T) (*server, *scorer.HTTP, string) {
	t.Helper()
	s := &server{}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /score", s.score)
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("PUT /mock/state", s.mockState)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	sc, err := scorer.NewHTTP(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return s, sc, srv.URL
}

// Mock phải qua được kiểm tra hợp đồng của chính client backend
func TestMockSpeaksTheContract(t *testing.T) {
	_, sc, _ := newMock(t)
	ctx := context.Background()
	if v, err := sc.Health(ctx); err != nil || v != Version {
		t.Fatalf("health: %q, %v", v, err)
	}
	clean, err := sc.Score(ctx, "Hồ bơi cuối tuần mở mấy giờ vậy mọi người?")
	if err != nil || clean.Labels["clean"] < 0.5 || len(clean.Sentences) != 0 {
		t.Fatalf("clean text: %+v, %v", clean, err)
	}
	text := "Mọi người ơi. Thằng đó đúng là súc vật!"
	bad, err := sc.Score(ctx, text)
	if err != nil || bad.Labels["hate"] < 0.5 || len(bad.Sentences) != 1 {
		t.Fatalf("hateful text: %+v, %v", bad, err)
	}
	if got := string([]rune(text)[bad.Sentences[0].Start:bad.Sentences[0].End]); !strings.Contains(got, "súc vật") {
		t.Fatalf("sentence offsets point at %q", got)
	}
}

func TestMockOffline(t *testing.T) {
	_, sc, url := newMock(t)
	req, _ := http.NewRequest(http.MethodPut, url+"/mock/state", strings.NewReader(`{"offline":true}`))
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("set state: %v %v", resp, err)
	}
	_ = resp.Body.Close()
	if _, err := sc.Score(context.Background(), "abc"); err == nil {
		t.Fatal("want an error while offline")
	}
	if _, err := sc.Health(context.Background()); err == nil {
		t.Fatal("want health to fail while offline")
	}
}
