// Ngày giờ và văn bản hiển thị, theo giờ Việt Nam

const TZ = 'Asia/Ho_Chi_Minh'
const DAYS = ['Chủ nhật', 'Thứ Hai', 'Thứ Ba', 'Thứ Tư', 'Thứ Năm', 'Thứ Sáu', 'Thứ Bảy']
const DAYS_SHORT = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7']

const parts = (d: Date) => {
  const p = new Intl.DateTimeFormat('en-GB', {
    timeZone: TZ,
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    weekday: 'short',
    hour12: false,
  }).formatToParts(d)
  const get = (t: string) => p.find((x) => x.type === t)?.value ?? ''
  const wd = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].indexOf(get('weekday'))
  return { day: get('day'), month: get('month'), year: get('year'), hour: get('hour'), minute: get('minute'), wd }
}

const toDate = (v: string | Date) => (typeof v === 'string' ? new Date(v) : v)

// "14/11"
export const dm = (v: string | Date) => {
  const p = parts(toDate(v))
  return `${p.day}/${p.month}`
}

// "14/11/2026"
export const dmy = (v: string | Date) => {
  const p = parts(toDate(v))
  return `${p.day}/${p.month}/${p.year}`
}

// "16:20 14/11"
export const timeDm = (v: string | Date) => {
  const p = parts(toDate(v))
  return `${p.hour}:${p.minute} ${p.day}/${p.month}`
}

// "Thứ Bảy 21/11"
export const weekdayDm = (v: string | Date) => {
  const p = parts(toDate(v))
  return `${DAYS[p.wd]} ${p.day}/${p.month}`
}

export const weekdayShort = (v: string | Date) => DAYS_SHORT[parts(toDate(v)).wd]

// Mô tả đầy đủ cho title của <time>
export const longDate = (v: string | Date) =>
  new Intl.DateTimeFormat('vi-VN', {
    timeZone: TZ,
    weekday: 'long',
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(toDate(v))

// "25 phút", "3 giờ", "2 ngày"; quá 7 ngày thì ghi ngày
export function age(v: string | Date): string {
  const s = Math.max(0, (Date.now() - toDate(v).getTime()) / 1000)
  if (s < 60) return 'Vừa xong'
  if (s < 3600) return `${Math.floor(s / 60)} phút trước`
  if (s < 86400) return `${Math.floor(s / 3600)} giờ trước`
  if (s < 7 * 86400) return `${Math.floor(s / 86400)} ngày trước`
  return dmy(v)
}

export const num = (n: number) => n.toLocaleString('vi-VN')
export const pct = (v: number) => `${Math.round(v * 100)}%`
export const score100 = (v: number | null | undefined) => (v == null ? '–' : String(Math.round(v * 100)))
export const score2 = (v: number | null | undefined) => (v == null ? '–' : v.toFixed(2).replace('.', ','))

export const trunc = (t: string, n: number) => (t.length > n ? t.slice(0, n - 1).trimEnd() + '…' : t)

// Chữ cái đầu của tên cuối, cho avatar
export const initial = (name: string) => {
  const words = name.replace(/\(.*?\)/g, '').trim().split(/\s+/)
  return (words[words.length - 1]?.[0] ?? '?').toUpperCase()
}

// Màu avatar chọn theo tên, bốn màu như prototype
export const hue = (name: string) => 'a' + ([...name].reduce((a, c) => a + c.charCodeAt(0), 0) % 4)

// Bỏ dấu, viết thường: so khớp tìm kiếm không dấu (giống vntext.Fold ở backend)
export const fold = (s: string) =>
  s.normalize('NFD').replace(/\p{M}/gu, '').replace(/đ/g, 'd').replace(/Đ/g, 'D').toLowerCase()
