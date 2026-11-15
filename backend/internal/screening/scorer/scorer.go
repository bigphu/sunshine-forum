// Package scorer là chỗ cắm mô hình chấm độc hại. Hợp đồng (Result) theo định dạng đã
// thống nhất với nhóm mô hình (SYSTEM_DESIGN §5): model_version, điểm clean/offensive/hate
// đã hiệu chỉnh 0–1 và điểm theo câu.
//
// Hiện chỉ có StandIn (điểm mẫu). Khi nhóm mô hình giao dịch vụ FastAPI, thêm một
// Scorer gọi HTTP (POST văn bản, nhận Result, /health cho ModelOnline) và chọn nó trong
// screening.New; phần còn lại của backend không đổi. Timeout 200 ms do service đặt.
package scorer

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"

	"sunshine/internal/screening/contract"
)

type Result struct {
	ModelVersion string
	Labels       map[string]float64 // clean, offensive, hate
	Sentences    []contract.Sentence
}

type Scorer interface {
	Score(ctx context.Context, text string) (Result, error)
}

var ErrOffline = errors.New("scorer: model service offline")

// StandIn là dịch vụ chấm điểm tạm: dò vài từ khoá để tạo điểm mẫu, giúp làm và trình
// diễn cảnh báo, hàng chờ, kháng nghị khi chưa có mô hình. Offline() cho phép Admin giả
// lập sự cố để thử chính sách NFR-4.2.
type StandIn struct {
	Offline func(ctx context.Context) bool
}

const StandInVersion = "stand-in-0.1 (điểm mẫu)"

var (
	offensiveTerms = []string{"đm", "dm", "dmm", "vcl", "vkl", "vl", "clgt", "đéo", "deo", "địt", "cứt", "đếch"}
	hateTerms      = []string{"ngu", "óc chó", "đồ khốn", "thằng khốn", "con khốn", "súc vật", "cút đi", "giết", "đánh chết", "mọi rợ", "đồ rác rưởi"}
	sentenceSplit  = regexp.MustCompile(`[^.!?\n]+[.!?\n]*`)
)

func wordsOnly(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}), " ")
}

func countHits(lower string, terms []string) int {
	padded := " " + wordsOnly(lower) + " "
	n := 0
	for _, t := range terms {
		n += strings.Count(padded, " "+t+" ")
	}
	return n
}

func hitScore(hits int) float64 {
	if hits == 0 {
		return 0.04
	}
	return min(0.97, 0.62+0.17*float64(hits-1))
}

func (s StandIn) Score(ctx context.Context, text string) (Result, error) {
	if s.Offline != nil && s.Offline(ctx) {
		return Result{}, ErrOffline
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	res := Result{ModelVersion: StandInVersion, Labels: map[string]float64{"clean": 0.96, "offensive": 0.04, "hate": 0.04}}
	// Chấm từng câu và lấy điểm cao nhất (FR-3.5)
	for _, loc := range sentenceSplit.FindAllStringIndex(text, -1) {
		lower := strings.ToLower(text[loc[0]:loc[1]])
		off := hitScore(countHits(lower, offensiveTerms))
		hate := hitScore(countHits(lower, hateTerms))
		if off > 0.5 || hate > 0.5 {
			top, score := "hate", hate
			if off > hate {
				top, score = "offensive", off
			}
			res.Sentences = append(res.Sentences, contract.Sentence{
				Start: len([]rune(text[:loc[0]])), End: len([]rune(text[:loc[1]])), TopLabel: top, Score: score,
			})
		}
		res.Labels["offensive"] = max(res.Labels["offensive"], off)
		res.Labels["hate"] = max(res.Labels["hate"], hate)
	}
	res.Labels["clean"] = 1 - max(res.Labels["offensive"], res.Labels["hate"])
	return res, nil
}
