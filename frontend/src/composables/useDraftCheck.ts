// Kiểm tra bản nháp trước khi gửi (FR-3.1). Từ khoá chặn và số điện thoại được dò ngay trong
// trình duyệt; mô hình chạy trên server sau khi người viết ngừng gõ một chút. Bản nháp không
// được lưu, cảnh báo không bao giờ tính là vi phạm.
import { computed, onBeforeUnmount, ref, watch, type Ref } from 'vue'
import { api } from '@/lib/api/client'
import { fold } from '@/lib/format'
import { useBlocklist } from '@/lib/queries'
import type { DraftWarning } from '@/lib/types'

// Câu cảnh báo theo quy tắc (COMMUNITY_RULES.md phần 2); Quy tắc 6, 7 không cảnh báo
const RULE_WARNINGS: Record<number, string> = {
  1: 'Đoạn này có thể xúc phạm người khác (Quy tắc 1). Bạn có muốn sửa lại, tập trung vào vấn đề không?',
  2: 'Đoạn này có thể mang tính phân biệt đối xử (Quy tắc 2). Bạn có muốn sửa lại trước khi đăng không?',
  3: 'Đoạn này có thể bị xem là đe dọa hoặc quấy rối (Quy tắc 3). Bạn có muốn sửa lại trước khi đăng không?',
  4: 'Đoạn này có từ ngữ có thể bị xem là thô tục (Quy tắc 4). Bạn có muốn thay bằng cách nói khác không?',
  5: 'Có vẻ bạn đang đăng thông tin cá nhân (Quy tắc 5). Hãy chắc chắn rằng bạn được phép chia sẻ thông tin này.',
}
const PHONE = /(?:\+84|0)[\s.-]?\d{2,3}[\s.-]?\d{3}[\s.-]?\d{3,4}/
const words = (s: string) => ' ' + fold(s).replace(/[^\p{L}\p{N}]+/gu, ' ').trim() + ' '

export function useDraftCheck(texts: Ref<string>[], channelId?: Ref<string | undefined>, delay = 600) {
  const { data: blocklist } = useBlocklist()
  const server = ref<DraftWarning[] | null>(null)
  const modelOffline = ref(false)
  let timer: ReturnType<typeof setTimeout> | undefined
  let seq = 0

  const text = computed(() => texts.map((t) => t.value).join('\n'))

  // Cảnh báo dò ngay trong trình duyệt
  const local = computed<DraftWarning[]>(() => {
    const t = text.value
    if (!t.trim()) return []
    const out: DraftWarning[] = []
    const w = words(t)
    const hit = blocklist.value?.find((b) => w.includes(' ' + fold(b.term).trim() + ' '))
    if (hit && RULE_WARNINGS[hit.rule]) out.push({ rule: hit.rule, message: RULE_WARNINGS[hit.rule], sentences: [] })
    if (PHONE.test(t)) out.push({ rule: 5, message: RULE_WARNINGS[5], sentences: [] })
    return out
  })

  async function run() {
    const t = text.value
    const mine = ++seq
    if (t.trim().length < 3) {
      server.value = []
      return
    }
    try {
      const { data } = await api.screening.POST('/screening/check', {
        body: { text: t, ...(channelId?.value ? { channel_id: channelId.value } : {}) },
      })
      if (mine !== seq || !data) return
      server.value = data.warnings
      modelOffline.value = data.model_offline
    } catch {
      // Kiểm tra lỗi không chặn việc viết: backend vẫn sàng lọc khi gửi
    }
  }

  watch(text, () => {
    server.value = null
    clearTimeout(timer)
    timer = setTimeout(run, delay)
  })
  onBeforeUnmount(() => clearTimeout(timer))

  // Một cảnh báo mỗi lúc, như prototype: kết quả server nếu đã có, không thì kết quả cục bộ
  const warning = computed<DraftWarning | null>(() => (server.value ?? local.value)[0] ?? null)

  function reset() {
    server.value = []
    seq++
  }

  return { warning, modelOffline, reset }
}
