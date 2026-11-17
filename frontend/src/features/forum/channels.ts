// Đoạn văn của bài: tách theo dòng trống
export const paragraphs = (body: string) =>
  body
    .split(/\n\s*\n/)
    .map((p) => p.trim())
    .filter(Boolean)

export const excerpt = (body: string) => body.replace(/\s+/g, ' ').trim()
