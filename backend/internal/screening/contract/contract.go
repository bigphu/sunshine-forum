// Package contract là phần công khai của screening: content gọi Screener trước khi lưu
// bài hay bình luận; mọi module dùng danh sách quy tắc cộng đồng ở đây.
package contract

import (
	"context"

	"github.com/google/uuid"
)

// Kết quả sàng lọc
const (
	OutcomePublished   = "published"
	OutcomePending     = "pending"
	OutcomeAutoDeleted = "auto_deleted"
)

// Lý do nội dung bị giữ lại (hoặc được chấm bởi mô hình)
const (
	ReasonModel     = "model"
	ReasonBlocklist = "blocklist"
	ReasonLanguage  = "language"
	ReasonOutage    = "outage"
)

// MaxChars là giới hạn đầu vào của mô hình; bài và bình luận không vượt quá (FR-2.7)
const MaxChars = 5000

// Sentence là đoạn văn bản mô hình gắn cờ, vị trí tính theo ký tự (rune)
type Sentence struct {
	Start    int     `json:"start"`
	End      int     `json:"end"`
	TopLabel string  `json:"top_label"`
	Score    float64 `json:"score"`
}

// Result là quyết định sàng lọc cho một đoạn văn bản
type Result struct {
	Outcome      string
	Reason       string // rỗng khi không chấm (không bao giờ xảy ra với Screen)
	Label        string // clean | offensive | hate; rỗng khi không chấm điểm
	Score        *float64
	ModelVersion string
	// Rule: quy tắc của mục blocklist khớp, hoặc 4 với nhãn offensive
	Rule      *int
	Sentences []Sentence
	// NeedsRescan: mô hình gặp sự cố, cần chấm lại khi mô hình hoạt động lại (NFR-4.2)
	NeedsRescan bool
}

// Target là nơi nội dung được đăng; ảnh hưởng tới ngưỡng và chính sách sự cố
type Target struct {
	ChannelID uuid.UUID
	// Bình luận trong Thông báo chung vẫn bị giữ khi chính sách là fail open
	AnnouncementComment bool
}

type Screener interface {
	// Screen chạy blocklist, kiểm tra ngôn ngữ, gọi mô hình (tối đa 200 ms) rồi áp ngưỡng.
	// Mô hình lỗi không làm Screen lỗi: chính sách sự cố quyết định (NFR-4.2).
	Screen(ctx context.Context, text string, t Target) (Result, error)
	// Worse chọn kết quả nặng hơn (tiêu đề và nội dung được chấm riêng)
	Worse(a, b Result) Result
	// ModelOnline: false khi mô hình đang sự cố, để job quét lại chờ tiếp
	ModelOnline(ctx context.Context) bool
}

// Rule là một quy tắc cộng đồng (COMMUNITY_RULES.md, phần 1)
type Rule struct {
	Number int
	Title  string
	Text   string
}

var Rules = []Rule{
	{1, "Không lăng mạ, xúc phạm người khác", "Không chửi bới, miệt thị, chế giễu hoặc gọi người khác bằng những từ xúc phạm."},
	{2, "Không phát ngôn thù ghét hoặc phân biệt đối xử", "Không hạ thấp hoặc kích động chống lại người khác vì dân tộc, vùng miền, tôn giáo, giới tính, xu hướng tính dục, khuyết tật, tuổi tác hoặc hoàn cảnh kinh tế."},
	{3, "Không đe dọa hoặc quấy rối", "Không đe dọa gây hại, kêu gọi bạo lực, liên tục làm phiền một người hoặc quấy rối tình dục."},
	{4, "Không dùng ngôn từ tục tĩu", "Không dùng từ ngữ thô tục hoặc đăng nội dung khiêu dâm, kể cả khi không nhắm vào ai."},
	{5, "Không tiết lộ thông tin riêng tư của người khác", "Không đăng số điện thoại, số căn hộ, biển số xe, giấy tờ tùy thân, hình ảnh riêng tư hoặc lịch sinh hoạt của người khác khi chưa được họ đồng ý. Bạn cũng nên cân nhắc trước khi chia sẻ thông tin của chính mình, chẳng hạn như thời gian nhà không có người."},
	{6, "Không spam, lừa đảo hoặc mua bán hàng cấm", "Không đăng lặp lại cùng một nội dung, không lừa đảo và không mua bán hàng hóa bị pháp luật cấm. Tin mua bán chỉ được đăng trong kênh Marketplace."},
	{7, "Không mạo danh", "Không giả danh ban quản lý hoặc cư dân khác, và không đăng nội dung giả làm thông báo chính thức."},
}

func RuleTitle(n int) string {
	if ValidRule(n) {
		return Rules[n-1].Title
	}
	return ""
}

func ValidRule(n int) bool { return n >= 1 && n <= len(Rules) }
