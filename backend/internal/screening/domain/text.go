// Package domain là quy tắc sàng lọc của screening: chuẩn hoá văn bản, kiểm tra ngôn ngữ,
// áp ngưỡng và cảnh báo trước khi gửi. Không biết HTTP hay SQL.
package domain

import (
	"regexp"
	"strings"

	"sunshine/internal/vntext"
)

// Fold, ContainsPhrase: xem package vntext
func Fold(s string) string { return vntext.Fold(s) }

func ContainsPhrase(haystack, phrase string) bool { return vntext.ContainsPhrase(haystack, phrase) }

var (
	vietnameseMarks = regexp.MustCompile(`[ăâđêôơưàáảãạằắẳẵặầấẩẫậèéẻẽẹềếểễệìíỉĩịòóỏõọồốổỗộờớởỡợùúủũụừứửữựỳýỷỹỵ]`)
	englishWords    = set("the", "and", "is", "are", "you", "this", "that", "with", "for", "have", "not", "what", "of", "to", "it", "your", "my", "be", "will", "can", "was", "they", "we", "he", "she")
	vietWords       = set("la", "cua", "va", "khong", "co", "duoc", "nha", "toi", "ban", "minh", "cho", "nay", "mot", "nguoi", "cac", "nhung", "de", "thi", "voi", "da", "se", "rat", "lam", "anh", "chi", "em", "o", "tang", "can", "ho", "quan", "ly", "nuoc", "dien", "may", "gi", "sao", "roi", "nhe", "a", "oi")
	phonePattern    = regexp.MustCompile(`(?:\+84|0)[\s.\-]?\d{2,3}[\s.\-]?\d{3}[\s.\-]?\d{3,4}`)
)

func set(words ...string) map[string]bool {
	m := make(map[string]bool, len(words))
	for _, w := range words {
		m[w] = true
	}
	return m
}

// IsVietnamese là bước kiểm tra ngôn ngữ đơn giản (FR-3.4). Văn bản có dấu
// tiếng Việt luôn tính là tiếng Việt; văn bản không dấu chỉ bị coi là ngôn ngữ
// khác khi có rõ nhiều từ tiếng Anh hơn từ tiếng Việt không dấu.
func IsVietnamese(text string) bool {
	if vietnameseMarks.MatchString(strings.ToLower(text)) {
		return true
	}
	words := strings.Fields(Fold(text))
	if len(words) < 4 {
		return true
	}
	en, vi := 0, 0
	for _, w := range words {
		if englishWords[w] {
			en++
		}
		if vietWords[w] {
			vi++
		}
	}
	return en < 2 || en <= vi
}

func HasPhoneNumber(text string) bool { return phonePattern.MatchString(text) }
