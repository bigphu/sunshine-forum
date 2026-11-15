package domain

import "sunshine/internal/screening/contract"

// Cảnh báo trước khi gửi (COMMUNITY_RULES.md, phần 2). Không tính là vi phạm. Quy tắc 6
// và 7 không có cảnh báo.
var ruleWarnings = map[int]string{
	1: "Đoạn này có thể xúc phạm người khác (Quy tắc 1). Bạn có muốn sửa lại, tập trung vào vấn đề không?",
	2: "Đoạn này có thể mang tính phân biệt đối xử (Quy tắc 2). Bạn có muốn sửa lại trước khi đăng không?",
	3: "Đoạn này có thể bị xem là đe dọa hoặc quấy rối (Quy tắc 3). Bạn có muốn sửa lại trước khi đăng không?",
	4: "Đoạn này có từ ngữ có thể bị xem là thô tục (Quy tắc 4). Bạn có muốn thay bằng cách nói khác không?",
	5: "Có vẻ bạn đang đăng thông tin cá nhân (Quy tắc 5). Hãy chắc chắn rằng bạn được phép chia sẻ thông tin này.",
}

// Nhãn hate không chỉ ra được một quy tắc nên dùng cảnh báo chung
const generalWarning = "Đoạn này có thể công kích hoặc xúc phạm người khác (Quy tắc 1, 2 hoặc 3). Bạn có muốn sửa lại trước khi đăng không?"

type Warning struct {
	Rule      *int
	Message   string
	Sentences []contract.Sentence
}

func RuleWarning(rule int) (Warning, bool) {
	msg, ok := ruleWarnings[rule]
	return Warning{Rule: ptr(rule), Message: msg}, ok
}

func HateWarning(s []contract.Sentence) Warning {
	return Warning{Message: generalWarning, Sentences: s}
}

func OffensiveWarning(s []contract.Sentence) Warning {
	return Warning{Rule: ptr(4), Message: ruleWarnings[4], Sentences: s}
}
