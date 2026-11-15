// Package vntext chuẩn hoá văn bản tiếng Việt để so khớp: viết thường, bỏ dấu (đ → d),
// bỏ dấu câu. Dùng chung cho blocklist (screening) và tìm kiếm (content), để gõ không
// dấu vẫn khớp. Không phụ thuộc module nào.
package vntext

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Fold: "Vay NÓNG, không thế chấp!" → "vay nong khong the chap"
func Fold(s string) string {
	var b strings.Builder
	for _, r := range norm.NFD.String(strings.ToLower(s)) {
		switch {
		case unicode.Is(unicode.Mn, r):
			continue
		case r == 'đ':
			b.WriteRune('d')
		default:
			b.WriteRune(r)
		}
	}
	return WordsOnly(b.String())
}

// WordsOnly thay dấu câu bằng khoảng trắng và gộp khoảng trắng
func WordsOnly(s string) string {
	return strings.Join(strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}), " ")
}

// ContainsPhrase so khớp nguyên cụm từ trên văn bản đã Fold
func ContainsPhrase(haystack, phrase string) bool {
	return strings.Contains(" "+haystack+" ", " "+phrase+" ")
}
