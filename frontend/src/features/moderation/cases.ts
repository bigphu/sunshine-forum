// Hàng chờ kiểm duyệt ở dạng "trường hợp" như prototype: yêu cầu xem xét lại và nội dung bị
// giữ, chia bốn nhóm theo mức ưu tiên.
import { dm } from '@/lib/format'
import type { QueueAppeal, QueueItem, Rule } from '@/lib/types'

export type Group = 'ap' | 'hi' | 'cx' | 'kw'
export const GROUPS: [Group, string][] = [
  ['ap', 'Xem xét lại'],
  ['hi', 'Mức cao'],
  ['cx', 'Cần xem'],
  ['kw', 'Từ khóa chặn'],
]
export const GROUP_ICON = { ap: 'clock', hi: 'alert', cx: 'eye', kw: 'filter' } as const
export const groupName = (g: Group) => GROUPS.find((x) => x[0] === g)![1]

export interface Case {
  id: string
  group: Group
  item: QueueItem
  appeal?: QueueAppeal
}

export function groupOf(it: QueueItem): Group {
  if (it.screening.reason === 'blocklist') return 'kw'
  if (it.screening.reason === 'model' && it.screening.label === 'hate') return 'hi'
  return 'cx'
}

export function toCases(items: QueueItem[], appeals: QueueAppeal[]): Case[] {
  return [
    ...appeals.map((a) => ({ id: a.appeal.id, group: 'ap' as const, item: a.item, appeal: a })),
    ...items.map((it) => ({ id: it.ref.id, group: groupOf(it), item: it })),
  ]
}

export const kindName = (c: Case) => (c.appeal ? 'Yêu cầu xem xét lại' : c.item.ref.type === 'post' ? 'Bài viết' : 'Bình luận')
export const caseName = (c: Case) =>
  `${c.appeal ? 'yêu cầu xem xét lại' : c.item.ref.type === 'post' ? 'bài viết' : 'bình luận'} của ${c.item.author.name}`

// Vì sao nội dung nằm trong hàng chờ (prototype why)
export function whyHere(c: Case, autoDelete: boolean): string {
  const s = c.item.screening
  if (c.appeal) {
    const d = c.appeal.decision
    return `${c.item.author.name} yêu cầu xem xét lại quyết định ${d.action === 'auto_delete' ? 'tự động xóa' : d.action === 'remove' ? 'gỡ nội dung' : 'từ chối'} theo Quy tắc ${d.rule}.`
  }
  if (s.reason === 'outage') return 'Bộ lọc AI đang gián đoạn nên nội dung được giữ lại; hệ thống sẽ tự kiểm tra lại khi bộ lọc hoạt động trở lại.'
  if (s.reason === 'language') return 'Nội dung không phải tiếng Việt nên bộ lọc AI không đánh giá được.'
  if (s.reason === 'blocklist') return `Nội dung khớp một từ khóa chặn của Quy tắc ${s.rule}.`
  if (s.label === 'hate') return 'Bộ lọc AI giữ lại nội dung này vì có thể công kích hoặc thù ghét.'
  return autoDelete
    ? 'Bộ lọc AI giữ lại vì có thể có từ ngữ thô tục, nhưng chưa đủ chắc để tự động xóa.'
    : 'Bộ lọc AI giữ lại vì có thể có từ ngữ thô tục.'
}

// Đánh giá tự động trong phần Chi tiết (prototype ctxD)
export function aiText(c: Case): string {
  const s = c.item.screening
  if (c.appeal) {
    const d = c.appeal.decision
    return d.action === 'auto_delete'
      ? 'Bộ lọc AI đã tự động xóa nội dung này.'
      : `${d.moderator?.name ?? 'Ban quản lý'} đã ${d.action === 'remove' ? 'gỡ' : 'từ chối'}, không phải bộ lọc AI.`
  }
  if (s.reason === 'blocklist') return 'Không dùng AI: hệ thống chặn theo từ khóa.'
  if (s.reason === 'language') return 'Không có điểm: bộ lọc chỉ đánh giá tiếng Việt.'
  if (s.reason === 'outage') return 'Chưa có điểm: bộ lọc AI đang gián đoạn.'
  if (s.label === 'hate') return 'Bộ lọc AI khá chắc đây là nội dung công kích hoặc thù ghét, nhưng không xác định được quy tắc nào.'
  return 'Bộ lọc AI cho rằng có thể có từ ngữ thô tục, nhưng chưa chắc chắn.'
}

export function decisionHint(c: Case): string {
  const s = c.item.screening
  if (c.appeal) return 'Đây là quyết định cuối cùng. Nếu chấp nhận, nội dung được khôi phục và lần vi phạm được gỡ bỏ.'
  if (s.label === 'hate') return 'Nếu từ chối, bạn cần chọn Quy tắc 1, 2 hoặc 3.'
  if (s.reason === 'language' || s.reason === 'outage')
    return 'Nội dung đang chờ này chưa phải là vi phạm. Nếu không có vấn đề, hãy cho phép đăng.'
  return 'Nếu từ chối, người viết nhận tin nhắn nêu quy tắc và có thể yêu cầu xem xét lại.'
}

// Quy tắc gợi ý; với nội dung thù ghét chỉ được chọn Quy tắc 1, 2 hoặc 3 (FR-5.2)
export function suggestedRules(c: Case): { suggested: number[]; rest: number[]; onlySuggested: boolean } {
  const s = c.item.screening
  if (s.label === 'hate') return { suggested: [1, 2, 3], rest: [], onlySuggested: true }
  const suggested = s.reason === 'blocklist' && s.rule ? [s.rule] : s.label === 'offensive' ? [4] : []
  return { suggested, rest: [1, 2, 3, 4, 5, 6, 7].filter((n) => !suggested.includes(n)), onlySuggested: false }
}

const in30 = () => dm(new Date(Date.now() + 30 * 864e5))
export function rejectMessage(channel: string, r: Rule | undefined): string {
  return r
    ? `Nội dung của bạn trong kênh ${channel} đã bị từ chối vì vi phạm Quy tắc ${r.number}: ${r.title}. Bạn có thể yêu cầu xem xét lại một lần trước ngày ${in30()}.`
    : ''
}
export function appealMessage(accept: boolean, r: Rule | undefined): string {
  return accept
    ? 'Yêu cầu xem xét lại của bạn đã được chấp nhận. Nội dung đã được khôi phục và lần vi phạm này đã được gỡ bỏ.'
    : `Yêu cầu xem xét lại của bạn không được chấp nhận. Nội dung vẫn bị từ chối vì vi phạm Quy tắc ${r?.number}: ${r?.title}. Đây là quyết định cuối cùng.`
}

// Chia văn bản thành đoạn thường, đoạn bị gắn cờ và đoạn bị ẩn (prototype segT). Điểm câu từ
// mô hình tính theo ký tự; từ khoá chặn được tìm lại trong văn bản.
export interface Seg {
  t: string
  kind: 'plain' | 'flag' | 'hide'
}
export function segments(text: string, c: Case, terms: string[], useOffsets: boolean): Seg[] {
  const chars = [...text]
  const mark: Seg['kind'][] = chars.map(() => 'plain')
  if (useOffsets)
    for (const s of c.item.screening.sentences) {
      if (s.top_label === 'clean' || s.end > chars.length) continue
      for (let i = s.start; i < s.end; i++) mark[i] = 'hide'
    }
  const low = text.toLowerCase()
  for (const term of terms) {
    const t = term.toLowerCase()
    let j = low.indexOf(t)
    while (t && j > -1) {
      const a = [...text.slice(0, j)].length
      for (let i = a; i < a + [...term].length; i++) if (mark[i] === 'plain') mark[i] = 'flag'
      j = low.indexOf(t, j + t.length)
    }
  }
  const out: Seg[] = []
  chars.forEach((ch, i) => {
    const last = out[out.length - 1]
    if (last && last.kind === mark[i]) last.t += ch
    else out.push({ t: ch, kind: mark[i] })
  })
  return out
}
export const hasHidden = (segs: Seg[]) => segs.some((s) => s.kind === 'hide')
// Tiêu đề trong danh sách: đoạn bị ẩn thay bằng ••• (prototype plainH)
export const shortTitle = (c: Case) => {
  const t = c.item.title || segments(c.item.body, c, [], true).map((s) => (s.kind === 'hide' ? '•••' : s.t)).join('')
  return t.length > 140 ? t.slice(0, 139).trimEnd() + '…' : t
}
