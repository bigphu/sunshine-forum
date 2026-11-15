// Dữ liệu tĩnh của giao diện, lấy từ prototype và COMMUNITY_RULES.md: biểu tượng và lưu ý của
// từng kênh, tên ngắn và ví dụ của bảy quy tắc, câu chữ dùng lại nhiều nơi.
import type { IconName } from './icons'

// Kênh nhận diện bằng slug (seed ở migration 00003)
export const CHANNEL_ICON: Record<string, IconName> = {
  'thong-bao': 'mega',
  'thao-luan': 'msg',
  'bao-tri': 'wrench',
  marketplace: 'tag',
}

// Lưu ý ở cột bên phải của kênh, và quy tắc liên quan nhất
export const CHANNEL_NOTE: Record<string, [string, number]> = {
  'thong-bao': ['Chỉ ban quản lý đăng bài ở đây. Bình luận trong kênh này được kiểm tra chặt hơn các kênh khác.', 0],
  'thao-luan': ['Phê bình thì được, công kích cá nhân thì không.', 1],
  'bao-tri': ['Ghi rõ tòa, tầng và vị trí sự cố. Đừng ghi số căn hộ hay biển số xe của người khác.', 5],
  marketplace: ['Chỉ đăng tin mua bán và cho tặng ở kênh này. Nên giao nhận ở sảnh và đừng chuyển tiền trước cho người lạ.', 6],
}

export const ALL_FEED = {
  name: 'Bảng tin',
  description: 'Mọi bài viết và thông báo trong tòa nhà.',
}

// Tên ngắn của quy tắc cho danh sách ở cột phải và bảng từ khóa
export const RULE_SHORT: Record<number, string> = {
  1: 'Không lăng mạ',
  2: 'Không phân biệt đối xử',
  3: 'Không đe dọa, quấy rối',
  4: 'Không nói tục',
  5: 'Không lộ thông tin riêng',
  6: 'Không spam, lừa đảo',
  7: 'Không mạo danh',
}

// Ví dụ được / không được của từng quy tắc (trang Quy tắc)
export const RULE_EXAMPLES: Record<number, [string, string]> = {
  1: ['Thang máy hỏng ba ngày rồi, mong ban quản lý xử lý sớm hơn.', 'Ban quản lý toàn người vô dụng.'],
  2: ['Nhà mình mới chuyển đến, rất vui được làm quen với mọi người.', 'Dân tỉnh lẻ thuê nhà ở đây toàn gây chuyện.'],
  3: ['Tối qua tầng 12 khá ồn sau 22 giờ, mong mọi người giữ yên lặng giúp.', 'Còn ồn nữa thì đừng trách tôi ra tay.'],
  4: ['Mệt thật sự, thang máy lại hỏng nữa rồi.', 'Cái thang máy chết tiệt này lại hỏng.'],
  5: ['Ai làm rơi chùm chìa khóa ở sảnh tòa A thì liên hệ bảo vệ nhé.', 'Xe biển số 51A-123.45 đỗ sai chỗ, chủ xe là anh ở căn 8C.'],
  6: ['Tặng lại xe đạp trẻ em, ai cần thì để lại bình luận (trong kênh Marketplace).', 'Vay nhanh lãi cực thấp, nhắn tin ngay.'],
  7: ['Mình chụp lại thông báo cắt điện mà ban quản lý dán ở thang máy.', '[THÔNG BÁO BQL] Từ mai mỗi hộ nộp thêm 200.000đ phí vệ sinh.'],
}

// Câu chữ trạng thái (COMMUNITY_RULES.md phần 2 và prototype)
export const PENDING_TEXT =
  'Nội dung của bạn đang chờ ban quản lý xem xét và chưa hiển thị công khai. Đây chưa phải là một lần vi phạm.'
export const PENDING_OUTAGE_TEXT =
  'Hệ thống kiểm tra tự động đang tạm gián đoạn. Nội dung của bạn sẽ hiển thị sau khi được kiểm tra. Đây chưa phải là một lần vi phạm.'
export const OUTAGE_HINT = 'Kiểm tra tự động đang chậm: nội dung sẽ hiển thị sau khi được kiểm tra. '
export const APPEAL_ACCEPTED =
  'Yêu cầu xem xét lại của bạn đã được chấp nhận. Nội dung đã được khôi phục và lần vi phạm này đã được gỡ bỏ.'

export const MAX_CHARS = 5000
export const SHOW_COUNT_FROM = 4500

// Đếm ký tự như backend (theo ký tự, không theo byte)
export const charLen = (v: string) => [...v.trim().normalize('NFC')].length

export const PHONE = 'tel:02838123456'
export const PHONE_TEXT = '028 3812 3456 (số mẫu)'
