package scorer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func serve(t *testing.T, h http.HandlerFunc) *HTTP {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	sc, err := NewHTTP(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func TestHTTPScore(t *testing.T) {
	text := "Đẹp quá. Đồ ngu."
	sc := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/score" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		var in scoreRequest
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Text != text {
			t.Errorf("request body: %v %q", err, in.Text)
		}
		_, _ = w.Write([]byte(`{"model_version":"svm-test","labels":{"clean":0.1,"offensive":0.2,"hate":0.7},
			"sentences":[{"start":9,"end":16,"top_label":"hate","score":0.7}],"extra":"ignored"}`))
	})
	res, err := sc.Score(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	if res.ModelVersion != "svm-test" || res.Labels["hate"] != 0.7 || len(res.Sentences) != 1 || res.Sentences[0].End != 16 {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestHTTPScoreErrors(t *testing.T) {
	cases := []struct {
		name    string
		handler http.HandlerFunc
		offline bool // lỗi mạng hoặc mã 5xx: ErrOffline
	}{
		{"server error", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "boom", 500) }, true},
		{"not json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("hello")) }, false},
		{"missing label", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"model_version":"v","labels":{"clean":0.9,"offensive":0.1}}`))
		}, false},
		{"score above 1", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"model_version":"v","labels":{"clean":0.1,"offensive":1.4,"hate":0}}`))
		}, false},
		{"no version", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"labels":{"clean":1,"offensive":0,"hate":0}}`))
		}, false},
		{"sentence past the end", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"model_version":"v","labels":{"clean":0,"offensive":1,"hate":0},
				"sentences":[{"start":0,"end":99,"top_label":"offensive","score":1}]}`))
		}, false},
		{"unknown label", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"model_version":"v","labels":{"clean":0,"offensive":1,"hate":0},
				"sentences":[{"start":0,"end":3,"top_label":"spam","score":1}]}`))
		}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := serve(t, c.handler).Score(context.Background(), "abc def")
			if err == nil {
				t.Fatal("want an error")
			}
			if errors.Is(err, ErrOffline) != c.offline {
				t.Fatalf("ErrOffline = %v, want %v (%v)", errors.Is(err, ErrOffline), c.offline, err)
			}
		})
	}
}

func TestHTTPScoreTimeout(t *testing.T) {
	sc := serve(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
		case <-r.Context().Done():
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := sc.Score(ctx, "abc"); !errors.Is(err, ErrOffline) {
		t.Fatalf("want ErrOffline, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatal("the deadline was not honoured")
	}
}

func TestHTTPScoreUnreachable(t *testing.T) {
	sc, err := NewHTTP("http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sc.Score(context.Background(), "abc"); !errors.Is(err, ErrOffline) {
		t.Fatalf("want ErrOffline, got %v", err)
	}
}

func TestHTTPHealth(t *testing.T) {
	sc := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"status":"ok","model_version":"svm-test"}`))
	})
	v, err := sc.Health(context.Background())
	if err != nil || v != "svm-test" {
		t.Fatalf("got %q, %v", v, err)
	}
	down := serve(t, func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "loading", http.StatusServiceUnavailable) })
	if _, err := down.Health(context.Background()); !errors.Is(err, ErrOffline) {
		t.Fatalf("want ErrOffline while loading, got %v", err)
	}
}

func TestNewHTTPRejectsBadURL(t *testing.T) {
	for _, u := range []string{"", "model:8000", "ftp://model", "http://"} {
		if _, err := NewHTTP(u); err == nil || !strings.Contains(err.Error(), "invalid") {
			t.Errorf("%q: want invalid URL error, got %v", u, err)
		}
	}
}
