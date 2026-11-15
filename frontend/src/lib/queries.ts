// Truy vấn dùng chung giữa nhiều trang (Vue Query). Truy vấn riêng của một trang nằm cạnh trang đó.
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { api, unwrap } from './api/client'
import { useAuth } from './auth'

export const keys = {
  channels: ['channels'] as const,
  rules: ['rules'] as const,
  inbox: ['notifications'] as const,
  queue: ['moderation', 'queue'] as const,
  status: ['screening', 'status'] as const,
  blocklist: ['screening', 'blocklist'] as const,
  feed: (slug: string) => ['feed', slug] as const,
  thread: (id: string) => ['thread', id] as const,
}

export function useChannels() {
  return useQuery({
    queryKey: keys.channels,
    queryFn: () => unwrap(api.content.GET('/channels')),
    staleTime: Infinity,
  })
}

export function useRules() {
  return useQuery({ queryKey: keys.rules, queryFn: () => unwrap(api.screening.GET('/rules')), staleTime: Infinity })
}

export function useInbox() {
  const auth = useAuth()
  return useQuery({
    queryKey: keys.inbox,
    queryFn: () => unwrap(api.notifications.GET('/notifications')),
    enabled: computed(() => !!auth.me),
  })
}

export function useQueue() {
  const auth = useAuth()
  return useQuery({
    queryKey: keys.queue,
    queryFn: () => unwrap(api.moderation.GET('/moderation/queue')),
    enabled: computed(() => auth.isStaff),
    refetchInterval: 30_000,
  })
}

// Trạng thái mô hình: composer báo "kiểm tra đang chậm" khi mô hình sự cố
export function useScreeningStatus() {
  return useQuery({
    queryKey: keys.status,
    queryFn: () => unwrap(api.screening.GET('/screening/status')),
    staleTime: 30_000,
  })
}

// Danh sách từ khoá chặn: trình duyệt cảnh báo ngay khi gõ (FR-3.1)
export function useBlocklist() {
  return useQuery({
    queryKey: keys.blocklist,
    queryFn: () => unwrap(api.screening.GET('/screening/blocklist')),
    staleTime: 5 * 60_000,
  })
}

// Luồng Server-Sent Events: có thông báo mới thì tải lại hộp thư, và (với ban quản lý) hàng
// chờ. Trình duyệt tự nối lại khi mất kết nối.
export function useNotificationStream() {
  const qc = useQueryClient()
  let es: EventSource | null = null
  onMounted(() => {
    es = new EventSource('/api/v1/notifications/stream')
    const refresh = () => {
      qc.invalidateQueries({ queryKey: keys.inbox })
      qc.invalidateQueries({ queryKey: keys.queue })
    }
    es.addEventListener('notification', refresh)
    es.addEventListener('ready', refresh)
  })
  onBeforeUnmount(() => es?.close())
}
