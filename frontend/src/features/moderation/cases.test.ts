import { describe, expect, it } from 'vitest'
import type { QueueItem } from '@/lib/types'
import { type Case, groupOf, segments, shortTitle, suggestedRules, toCases } from './cases'

const item = (over: Partial<QueueItem['screening']> = {}, body = 'Nội dung bình thường'): QueueItem => ({
  ref: { type: 'comment', id: 'c1' },
  post_id: 'p1',
  post_title: 'Bài',
  title: '',
  body,
  channel_id: 'ch',
  channel_name: 'Thảo luận chung',
  author: { id: 'a', name: 'Lan', unit: 'A-1' },
  status: 'pending',
  screening: { reason: 'model', label: 'offensive', model_version: 'x', sentences: [], ...over },
  created_at: '2026-11-15T00:00:00Z',
})
const asCase = (it: QueueItem): Case => toCases([it], [])[0]

describe('groupOf', () => {
  it('đặt từ khóa chặn, thù ghét và phần còn lại vào đúng nhóm', () => {
    expect(groupOf(item({ reason: 'blocklist', rule: 6 }))).toBe('kw')
    expect(groupOf(item({ label: 'hate' }))).toBe('hi')
    expect(groupOf(item({ label: 'offensive' }))).toBe('cx')
    expect(groupOf(item({ reason: 'language', label: '' }))).toBe('cx')
  })
})

describe('suggestedRules', () => {
  it('nội dung thù ghét chỉ được chọn Quy tắc 1, 2 hoặc 3', () => {
    expect(suggestedRules(asCase(item({ label: 'hate' })))).toEqual({ suggested: [1, 2, 3], rest: [], onlySuggested: true })
  })
  it('gợi ý quy tắc của từ khóa chặn, còn lại vẫn chọn được', () => {
    const r = suggestedRules(asCase(item({ reason: 'blocklist', rule: 6 })))
    expect(r.suggested).toEqual([6])
    expect(r.rest).toEqual([1, 2, 3, 4, 5, 7])
  })
})

describe('segments', () => {
  it('ẩn câu bị mô hình gắn cờ theo vị trí ký tự (không theo byte)', () => {
    const body = 'Đẹp quá. Đồ ngu.'
    const c = asCase(item({ sentences: [{ start: 9, end: 16, top_label: 'offensive', score: 0.7 }] }, body))
    const segs = segments(body, c, [], true)
    expect(segs).toEqual([
      { t: 'Đẹp quá. ', kind: 'plain' },
      { t: 'Đồ ngu.', kind: 'hide' },
    ])
    expect(shortTitle(c)).toBe('Đẹp quá. •••')
  })
  it('tô từ khóa chặn, không phân biệt hoa thường', () => {
    const c = asCase(item({ reason: 'blocklist', rule: 6 }, 'Ai cần VAY NHANH thì nhắn'))
    expect(segments('Ai cần VAY NHANH thì nhắn', c, ['vay nhanh'], false)).toEqual([
      { t: 'Ai cần ', kind: 'plain' },
      { t: 'VAY NHANH', kind: 'flag' },
      { t: ' thì nhắn', kind: 'plain' },
    ])
  })
  it('bỏ qua vị trí nằm ngoài văn bản (điểm tính trên tiêu đề của bài)', () => {
    const c = asCase(item({ sentences: [{ start: 0, end: 99, top_label: 'hate', score: 0.9 }] }, 'Ngắn'))
    expect(segments('Ngắn', c, [], true)).toEqual([{ t: 'Ngắn', kind: 'plain' }])
  })
})
