// Client API: một client openapi-fetch cho mỗi module, kiểu sinh từ openapi.yaml của backend
// (npm run gen:api). Mọi request ghi gửi kèm X-CSRF-Token của phiên; lỗi problem+json được
// đổi thành ApiError với câu tiếng Việt trong `detail`.
import createClient, { type Middleware } from 'openapi-fetch'
import type { paths as administration } from './administration'
import type { paths as content } from './content'
import type { paths as identity } from './identity'
import type { paths as moderation } from './moderation'
import type { paths as notifications } from './notifications'
import type { paths as penalties } from './penalties'
import type { paths as screening } from './screening'

export class ApiError extends Error {
  status: number
  type: string
  fields: { field: string; detail: string }[]
  constructor(status: number, message: string, type = '', fields: { field: string; detail: string }[] = []) {
    super(message)
    this.status = status
    this.type = type
    this.fields = fields
  }
}

let csrfToken = ''
export function setCsrfToken(t: string) {
  csrfToken = t
}

// Gọi khi phiên hết hạn ở giữa chừng (401): store auth đưa người dùng về trang đăng nhập
let onUnauthorized: () => void = () => {}
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

const SAFE = new Set(['GET', 'HEAD', 'OPTIONS'])

const middleware: Middleware = {
  onRequest({ request }) {
    if (!SAFE.has(request.method) && csrfToken) request.headers.set('X-CSRF-Token', csrfToken)
    return request
  },
  async onResponse({ response, request }) {
    if (response.ok) return response
    const body = await response
      .clone()
      .json()
      .catch(() => null)
    if (response.status === 401 && !request.url.includes('/auth/') && !request.url.endsWith('/me')) onUnauthorized()
    throw new ApiError(
      response.status,
      body?.detail || 'Đã có lỗi xảy ra. Vui lòng thử lại.',
      body?.type ?? '',
      body?.errors ?? [],
    )
  },
  onError() {
    throw new ApiError(0, 'Không kết nối được máy chủ. Nội dung bạn nhập vẫn còn, hãy thử lại.', '/errors/offline')
  },
}

function client<P extends {}>() {
  const c = createClient<P>({ baseUrl: '/api/v1', credentials: 'same-origin' })
  c.use(middleware)
  return c
}

export const api = {
  identity: client<identity>(),
  screening: client<screening>(),
  content: client<content>(),
  moderation: client<moderation>(),
  penalties: client<penalties>(),
  notifications: client<notifications>(),
  administration: client<administration>(),
}

// unwrap: lỗi đã được middleware ném ra, nên ở đây chỉ còn data
export async function unwrap<T>(p: Promise<{ data?: T }>): Promise<T> {
  const { data } = await p
  return data as T
}

export function errorText(e: unknown): string {
  return e instanceof Error ? e.message : 'Đã có lỗi xảy ra.'
}
