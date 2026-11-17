<script setup lang="ts">
import { useQueryClient } from '@tanstack/vue-query'
import { ref } from 'vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import { api, errorText, unwrap } from '@/lib/api/client'
import { dm } from '@/lib/format'
import { toast } from '@/lib/toast'
import type { ExportItem } from '@/lib/types'
import AdminPage from './AdminPage.vue'
import { adminKeys, useExports } from './queries'

// Xuất dữ liệu huấn luyện (FR-7.3): sửa đổi đã hoàn tất từ 30 đến 90 ngày trước, đã thay
// thông tin cá nhân; mỗi bản xuất tự xóa trước khi bản gốc bị xóa.
const qc = useQueryClient()
const list = useExports()
const purpose = ref<'training' | 'evaluation'>('training')
const busy = ref(false)
const PURPOSE = { training: 'Huấn luyện', evaluation: 'Đánh giá' } as const
const RULES: [boolean, string][] = [
  [true, 'Gồm văn bản đã sửa và nhãn cuối cùng của quản lý.'],
  [true, 'Không có mã người dùng hay mã bài viết.'],
  [true, 'Số điện thoại, email và thông tin tương tự được thay tự động.'],
  [false, 'Không gồm các trường hợp về thông tin riêng tư (Quy tắc 5).'],
  [false, 'Không gồm nội dung người viết đã tự xóa.'],
]

async function create() {
  busy.value = true
  try {
    const x = await unwrap(api.administration.POST('/admin/exports', { body: { purpose: purpose.value } }))
    toast(`Đã tạo bản xuất ${x.item_count} mục.`)
    qc.invalidateQueries({ queryKey: adminKeys.exports })
  } catch (e) {
    toast(errorText(e))
  } finally {
    busy.value = false
  }
}
// Tải về bằng liên kết thường để trình duyệt tự lưu tệp ndjson (phiên đi theo cookie)
function download(x: ExportItem) {
  const a = document.createElement('a')
  a.href = `/api/v1/admin/exports/${x.id}`
  a.download = `sunshine-${x.purpose}-${x.created_at.slice(0, 10)}.ndjson`
  a.click()
}
async function remove(x: ExportItem) {
  try {
    await api.administration.DELETE('/admin/exports/{export_id}', { params: { path: { export_id: x.id } } })
    toast('Đã xóa bản xuất.')
    qc.invalidateQueries({ queryKey: adminKeys.exports })
  } catch (e) {
    toast(errorText(e))
  }
}
</script>

<template>
  <AdminPage title="Xuất dữ liệu huấn luyện" lead="Xuất các sửa đổi đã hoàn tất để huấn luyện hoặc đánh giá lại mô hình. Không có kho lưu trữ dài hạn riêng.">
    <LoadError v-if="list.isError.value" title="Không tải được bản xuất" @retry="list.refetch()" />
    <template v-else-if="list.data.value">
      <section class="card" aria-labelledby="xe-h">
        <h2 id="xe-h">{{ list.data.value.eligible }} sửa đổi đủ điều kiện</h2>
        <p>Quyết định đã hoàn tất từ 30 đến 90 ngày trước, từ {{ dm(list.data.value.period_from) }} đến {{ dm(list.data.value.period_to) }}.</p>
        <ul class="rights exl2">
          <li v-for="[ok, t] in RULES" :key="t" :class="{ no: !ok }"><Icon :name="ok ? 'check' : 'x'" /><span>{{ t }}</span></li>
        </ul>
        <form novalidate @submit.prevent="create">
          <fieldset class="rul">
            <legend class="lb">Dùng để</legend>
            <label class="rd"><input v-model="purpose" type="radio" name="xpu" value="training" /><span>Huấn luyện<small>Huấn luyện lại mô hình.</small></span></label>
            <label class="rd"><input v-model="purpose" type="radio" name="xpu" value="evaluation" /><span>Đánh giá<small>Kiểm tra độ chính xác trước khi dùng mô hình mới.</small></span></label>
          </fieldset>
          <div class="row-a">
            <button class="btn pri" type="submit" :disabled="!list.data.value.eligible" :aria-busy="busy">Tạo bản xuất</button>
          </div>
        </form>
      </section>
      <section class="psec" aria-labelledby="xl-h">
        <h2 id="xl-h">Bản xuất đang lưu</h2>
        <ul v-if="list.data.value.exports.length" class="rqs">
          <li v-for="x in list.data.value.exports" :key="x.id" class="exr">
            <span><b>Bản xuất {{ dm(x.created_at) }} · {{ PURPOSE[x.purpose] }}</b><small>{{ x.item_count }} mục · Tự xóa ngày {{ dm(x.delete_after) }}</small></span>
            <span class="exa">
              <button class="btn ghost" @click="download(x)">Tải xuống</button>
              <button class="btn ghost" @click="remove(x)">Xóa ngay</button>
            </span>
          </li>
        </ul>
        <p v-else class="hint">Chưa có bản xuất nào.</p>
        <p class="hint">Mỗi bản xuất tự xóa trước khi bản gốc cũ nhất trong đó bị xóa vĩnh viễn sau 90 ngày.</p>
      </section>
    </template>
    <p v-else role="status">Đang tải…</p>
  </AdminPage>
</template>
