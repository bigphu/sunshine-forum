<script setup lang="ts">
import { useInfiniteQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import { api, unwrap } from '@/lib/api/client'
import { score100, timeDm } from '@/lib/format'
import AdminPage from './AdminPage.vue'

// Nhật ký kiểm tra (FR-7.2): chỉ ghi thêm, không lưu nội dung, người dùng ghi bằng mã ẩn danh
type F = 'all' | 'overturned' | 'config'
const filter = ref<F>('all')
const log = useInfiniteQuery({
  queryKey: computed(() => ['audit', filter.value]),
  queryFn: ({ pageParam }) =>
    unwrap(
      api.moderation.GET('/audit', {
        params: { query: { ...(pageParam ? { before: pageParam } : {}), ...(filter.value !== 'all' ? { filter: filter.value } : {}) } },
      }),
    ),
  initialPageParam: 0,
  getNextPageParam: (last) => last.next_before,
})
const rows = computed(() => log.data.value?.pages.flatMap((p) => p.items) ?? [])
// Mã trong nhật ký là ổn định cho máy đọc; giao diện hiển thị tên tiếng Việt
const ACTION: Record<string, string> = {
  approve: 'Cho phép đăng',
  reject: 'Từ chối',
  remove: 'Gỡ nội dung',
  auto_delete: 'Tự động xóa',
  appeal_accepted: 'Chấp nhận xem xét lại',
  appeal_declined: 'Giữ quyết định',
  config_change: 'Đổi cấu hình',
}
const OUTCOME: Record<string, string> = {
  published: 'Đã đăng',
  rejected: 'Đã từ chối',
  removed: 'Đã gỡ',
  auto_deleted: 'Đã tự động xóa',
  restored: 'Đã khôi phục',
  permanently_rejected: 'Từ chối vĩnh viễn',
}
// Người kiểm duyệt sửa lại đánh giá của AI: gọi đúng tên lỗi của AI theo hành động
function aiMistake(action: string): string {
  if (action === 'approve') return 'AI giữ nhầm'
  if (action === 'remove') return 'AI bỏ sót'
  if (action === 'appeal_accepted') return 'AI xóa nhầm'
  return 'AI đánh giá sai'
}
const CHIPS: [F, string][] = [
  ['all', 'Tất cả'],
  ['overturned', 'AI đánh giá sai'],
  ['config', 'Thay đổi cấu hình'],
]
</script>

<template>
  <AdminPage title="Nhật ký kiểm tra" lead="Mọi quyết định của người kiểm duyệt và mọi thay đổi cấu hình, kể cả khi quản lý sửa lại đánh giá của AI.">
    <div class="pend">
      <Icon name="shield" /><span
        >Nhật ký chỉ ghi thêm. Không ai sửa hoặc xóa được, kể cả quản trị viên. Nhật ký không lưu nội dung bài viết hay thông tin cá nhân;
        người dùng được ghi bằng mã ẩn danh.</span
      >
    </div>
    <div class="pick" role="group" aria-label="Hiển thị">
      <span>Hiển thị</span>
      <button v-for="[v, t] in CHIPS" :key="v" type="button" :aria-pressed="filter === v" @click="filter = v">{{ t }}</button>
    </div>
    <LoadError v-if="log.isError.value" title="Không tải được nhật ký" @retry="log.refetch()" />
    <section v-else class="card tc wide-t">
      <table v-if="rows.length" class="tbl">
        <caption class="sr">Nhật ký kiểm tra</caption>
        <thead>
          <tr>
            <th scope="col">Thời gian</th><th scope="col">Người thực hiện</th><th scope="col">Hành động</th><th scope="col">Đánh giá AI</th>
            <th scope="col">Quy tắc</th><th scope="col">Kết quả</th><th scope="col">Người dùng</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="e in rows" :key="e.id">
            <td data-l="Thời gian" class="nw">{{ timeDm(e.created_at) }}</td>
            <td data-l="Người thực hiện">{{ e.moderator || 'Bộ lọc tự động' }}</td>
            <td data-l="Hành động"><b>{{ ACTION[e.action] ?? e.action }}</b></td>
            <td data-l="Đánh giá AI">
              {{ e.ai_label ? `${e.ai_label}${e.ai_score != null ? ` ${score100(e.ai_score)}` : ''}` : '—' }}
              <small v-if="e.model_version">Mô hình {{ e.model_version }}</small>
            </td>
            <td data-l="Quy tắc">{{ e.rule ?? '—' }}</td>
            <td data-l="Kết quả">
              {{ OUTCOME[e.outcome] ?? e.outcome }}
              <span v-if="e.overturned_ai" class="pill aip"><Icon name="alert" />{{ aiMistake(e.action) }}</span>
            </td>
            <td data-l="Người dùng" class="nw">{{ e.subject }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else-if="!log.isPending.value" class="tbl-e">Không có mục nào.</p>
    </section>
    <div v-if="log.hasNextPage.value" class="row-a">
      <button class="btn" :aria-busy="log.isFetchingNextPage.value" @click="log.fetchNextPage()">Xem mục cũ hơn</button>
    </div>
  </AdminPage>
</template>
