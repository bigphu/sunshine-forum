<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/Icon.vue'
import type { IconName } from '@/lib/icons'
import { useBlocklist, useScreeningStatus } from '@/lib/queries'
import AdminPage from './AdminPage.vue'
import { useAccounts, useExports, useReadOnly, useScreeningSettings } from './queries'

// Trang chủ quản trị: các mục kèm một dòng trạng thái (prototype admHome)
const accounts = useAccounts()
const readOnly = useReadOnly()
const settings = useScreeningSettings()
const { data: status } = useScreeningStatus()
const { data: blocklist } = useBlocklist()
const exports = useExports()

const lines = computed<Record<string, string>>(() => {
  const a = accounts.data.value ?? []
  const residents = a.filter((x) => x.role === 'resident' && x.status !== 'purged' && x.status !== 'deactivated').length
  const invited = a.filter((x) => x.status === 'invited').length
  const s = settings.data.value
  const ex = exports.data.value
  return {
    'nguoi-dung': `${residents} tài khoản cư dân · ${invited} lời mời chưa dùng · ${readOnly.data.value?.length ?? 0} tài khoản chỉ đọc`,
    'bo-loc': `${status.value && !status.value.model_online ? 'Đang ngừng hoạt động' : 'Đang hoạt động'} · ${s?.outage_policy === 'fail_open' ? 'Vẫn đăng khi lỗi' : 'Giữ lại khi lỗi'}`,
    'tu-khoa': `${blocklist.value?.length ?? 0} từ khóa`,
    'phan-tich': 'Số liệu 7, 30 hoặc 90 ngày',
    'nhat-ky': 'Chỉ ghi thêm',
    'xuat-du-lieu': `${ex?.exports.length ? `${ex.exports.length} bản xuất đang lưu` : 'Chưa có bản xuất'} · ${ex?.eligible ?? 0} sửa đổi đủ điều kiện`,
  }
})
const ITEMS: [string, IconName, string][] = [
  ['nguoi-dung', 'users', 'Người dùng'],
  ['bo-loc', 'sliders', 'Bộ lọc AI'],
  ['tu-khoa', 'filter', 'Từ khóa chặn'],
  ['phan-tich', 'chart', 'Phân tích'],
  ['nhat-ky', 'list', 'Nhật ký kiểm tra'],
  ['xuat-du-lieu', 'download', 'Xuất dữ liệu huấn luyện'],
]
</script>

<template>
  <AdminPage title="Quản trị" lead="Quyền truy cập, bộ lọc AI, số liệu và nhật ký của diễn đàn." home>
    <nav class="card ail" aria-label="Các mục quản trị">
      <ul>
        <li v-for="[id, icon, t] in ITEMS" :key="id">
          <RouterLink class="ai" :to="`/quan-tri/${id}`">
            <Icon :name="icon" /><span><b>{{ t }}</b><small>{{ lines[id] }}</small></span><Icon name="chev" />
          </RouterLink>
        </li>
      </ul>
    </nav>
  </AdminPage>
</template>
