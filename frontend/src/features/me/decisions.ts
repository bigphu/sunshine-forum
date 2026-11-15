// Câu chữ dùng chung cho quyết định kiểm duyệt ở trang Yêu cầu và Xem xét lại
import type { IconName } from '@/lib/icons'
import type { MyDecision } from '@/lib/types'

export const VERB: Record<string, string> = { reject: 'từ chối', remove: 'gỡ', auto_delete: 'tự động xóa', approve: 'cho phép đăng' }
export const kindText = (d: MyDecision) => (d.content_type === 'post' ? 'Bài viết' : 'Bình luận')
export const decisionTitle = (d: MyDecision) => `${kindText(d)} của bạn trong kênh ${d.channel_name} đã bị ${VERB[d.decision.action]}`

export function appealStatus(d: MyDecision): [string, IconName] {
  if (d.appeal?.status === 'accepted') return ['Đã chấp nhận', 'check']
  if (d.appeal?.status === 'declined') return ['Không được chấp nhận', 'x']
  if (d.appeal) return ['Đang xem xét lại', 'clock']
  return d.can_appeal ? ['Có thể yêu cầu xem xét lại', 'arrow'] : ['Đã hết hạn yêu cầu', 'alert']
}
