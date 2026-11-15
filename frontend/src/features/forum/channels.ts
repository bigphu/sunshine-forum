// Ánh xạ slug của kênh sang mã ảnh banner của prototype (CSS dùng [data-ch])
export const BANNER_ART: Record<string, string> = {
  '': 'all',
  'thong-bao': 'ann',
  'thao-luan': 'talk',
  'bao-tri': 'fix',
  marketplace: 'market',
}

// Đoạn văn của bài: tách theo dòng trống
export const paragraphs = (body: string) =>
  body
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter(Boolean)

export const excerpt = (body: string) => body.replace(/\s+/g, ' ').trim()
