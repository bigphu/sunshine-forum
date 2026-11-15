import { ref } from 'vue'

// Thông báo nhanh dạng viên thuốc màu mực (.toast của prototype). Có thể kèm nút hoàn tác.
export const current = ref<{ id: number; text: string } | null>(null)
let next = 1

export function toast(text: string) {
  const id = next++
  current.value = { id, text }
  // Đọc cho trình đọc màn hình qua vùng aria-live của ToastHost
  setTimeout(() => {
    if (current.value?.id === id) current.value = null
  }, Math.max(6000, text.length * 90))
}
