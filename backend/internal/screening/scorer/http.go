package scorer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"sunshine/internal/screening/contract"
)

// HTTP gọi dịch vụ mô hình theo hợp đồng đã thống nhất (SYSTEM_DESIGN §5, mô tả đầy đủ ở
// model-service.yaml cạnh file này):
//
//	POST {base}/score  {"text": "..."}
//	  -> {"model_version": "...", "labels": {"clean": .., "offensive": .., "hate": ..},
//	      "sentences": [{"start": 0, "end": 42, "top_label": "hate", "score": 0.91}]}
//	GET  {base}/health -> 200 {"model_version": "..."} khi mô hình đã nạp xong
//
// Mọi lỗi (mạng, quá hạn, mã khác 2xx, JSON sai hợp đồng) trả về error, và screening áp
// chính sách khi mô hình sự cố (NFR-4.2). Thời hạn 200 ms do service đặt qua ctx.
type HTTP struct {
	base   string
	client *http.Client
}

// maxResponse: phản hồi hợp lệ của một văn bản 5.000 ký tự nhỏ hơn nhiều
const maxResponse = 1 << 20

// NewHTTP: base là địa chỉ gốc của dịch vụ, ví dụ http://model:8000
func NewHTTP(base string) (*HTTP, error) {
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("scorer: invalid model service URL %q", base)
	}
	// Giữ kết nối sống: mỗi bài gọi hai lần (tiêu đề và nội dung) trong 200 ms
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.MaxIdleConnsPerHost = 32
	tr.IdleConnTimeout = 90 * time.Second
	return &HTTP{base: strings.TrimRight(base, "/"), client: &http.Client{Transport: tr}}, nil
}

type scoreRequest struct {
	Text string `json:"text"`
}

type scoreResponse struct {
	ModelVersion string              `json:"model_version"`
	Labels       map[string]float64  `json:"labels"`
	Sentences    []contract.Sentence `json:"sentences"`
}

type healthResponse struct {
	ModelVersion string `json:"model_version"`
}

func (h *HTTP) Score(ctx context.Context, text string) (Result, error) {
	body, err := json.Marshal(scoreRequest{Text: text})
	if err != nil {
		return Result{}, err
	}
	var out scoreResponse
	if err := h.do(ctx, http.MethodPost, "/score", body, &out); err != nil {
		return Result{}, err
	}
	if err := out.validate([]rune(text)); err != nil {
		return Result{}, fmt.Errorf("scorer: bad response from model service: %w", err)
	}
	return Result(out), nil
}

// Health trả về phiên bản mô hình khi dịch vụ sẵn sàng
func (h *HTTP) Health(ctx context.Context) (string, error) {
	var out healthResponse
	if err := h.do(ctx, http.MethodGet, "/health", nil, &out); err != nil {
		return "", err
	}
	if out.ModelVersion == "" {
		return "", errors.New("scorer: model service health has no model_version")
	}
	return out.ModelVersion, nil
}

func (h *HTTP) do(ctx context.Context, method, path string, body []byte, out any) error {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, h.base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrOffline, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode/100 != 2 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponse))
		return fmt.Errorf("%w: %s %s returned %d", ErrOffline, method, path, resp.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponse)).Decode(out); err != nil {
		return fmt.Errorf("scorer: decode %s: %w", path, err)
	}
	return nil
}

var labelNames = []string{"clean", "offensive", "hate"}

// validate: không tin dịch vụ ngoài. Điểm lạ đi vào ngưỡng và hàng chờ, vị trí lạ làm
// lệch phần bị ẩn trong màn kiểm duyệt.
func (r scoreResponse) validate(text []rune) error {
	if r.ModelVersion == "" {
		return errors.New("model_version is empty")
	}
	inUnit := func(v float64) bool { return !math.IsNaN(v) && v >= 0 && v <= 1 }
	for _, l := range labelNames {
		v, ok := r.Labels[l]
		if !ok {
			return fmt.Errorf("labels.%s is missing", l)
		}
		if !inUnit(v) {
			return fmt.Errorf("labels.%s = %v is outside 0..1", l, v)
		}
	}
	for i, s := range r.Sentences {
		if s.Start < 0 || s.End <= s.Start || s.End > len(text) {
			return fmt.Errorf("sentences[%d] range %d..%d does not fit a %d-character text", i, s.Start, s.End, len(text))
		}
		if s.TopLabel != "offensive" && s.TopLabel != "hate" && s.TopLabel != "clean" {
			return fmt.Errorf("sentences[%d].top_label %q is unknown", i, s.TopLabel)
		}
		if !inUnit(s.Score) {
			return fmt.Errorf("sentences[%d].score = %v is outside 0..1", i, s.Score)
		}
	}
	return nil
}
