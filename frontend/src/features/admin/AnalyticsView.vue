<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import { api, unwrap } from '@/lib/api/client'
import { RULE_SHORT } from '@/lib/forum'
import { dm, num } from '@/lib/format'
import { useScreeningStatus } from '@/lib/queries'
import AdminPage from './AdminPage.vue'
import { useAccounts, usePenaltySettings, useReadOnly, useScreeningSettings } from './queries'

// Phân tích (FR-7.1): bộ lọc AI có đang làm tốt không. 90 ngày được gộp theo tuần.
type Days = 7 | 30 | 90
const days = ref<Days>(7)
const data = useQuery({
  queryKey: computed(() => ['admin', 'analytics', days.value]),
  queryFn: () => unwrap(api.administration.GET('/admin/analytics', { params: { query: { days: days.value } } })),
})
const a = computed(() => data.data.value)
const { data: status } = useScreeningStatus()
const settings = useScreeningSettings()
const penalties = usePenaltySettings()
const readOnly = useReadOnly()
const accounts = useAccounts()
const DW = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7']

const series = computed(() => {
  const d = a.value?.flagged_by_day ?? []
  if (days.value === 90) {
    const weeks: { v: number; label: string; full: string }[] = []
    for (let i = 0; i < d.length; i += 7) {
      const chunk = d.slice(i, i + 7)
      weeks.push({ v: chunk.reduce((x, y) => x + y.count, 0), label: weeks.length % 3 === 0 ? dm(chunk[0].day) : '', full: `Tuần từ ${dm(chunk[0].day)}` })
    }
    return weeks
  }
  return d.map((x, i) => {
    const wd = DW[new Date(x.day + 'T00:00:00').getDay()]
    return { v: x.count, label: days.value === 7 ? wd : i % 5 === 0 ? dm(x.day) : '', full: `${wd} ${dm(x.day)}` }
  })
})
const total = computed(() => series.value.reduce((x, y) => x + y.v, 0))
const max = computed(() => Math.max(0, ...series.value.map((s) => s.v)))
const top = computed(() => Math.max(5, Math.ceil(max.value / 5) * 5))
const peak = computed(() => series.value.find((s) => s.v === max.value))
const overturned = computed(() => (a.value ? a.value.overturned.held_wrong + a.value.overturned.missed + a.value.overturned.deleted_wrong : 0))
const pctText = (v: number) => `${(Math.round(v * 1000) / 10).toString().replace('.', ',')}%`
const labels = computed(() => {
  const x = a.value
  if (!x) return []
  const rate = (n: number) => (x.checked ? n / x.checked : 0)
  return [
    ['hate', 'Công kích hoặc thù ghét, Quy tắc 1 đến 3', rate(x.by_label.hate)],
    ['offensive', 'Thô tục, Quy tắc 4', rate(x.by_label.offensive)],
    ['Từ khóa chặn', 'Theo quy tắc của từ khóa', rate(x.by_label.blocklist)],
    ['Ngôn ngữ khác', 'Không chấm điểm, chờ quản lý', rate(x.by_label.language)],
  ] as [string, string, number][]
})
const labelMax = computed(() => Math.max(0.0001, ...labels.value.map((l) => l[2])))
const nameOf = (id: string) => accounts.data.value?.find((x) => x.id === id)
</script>

<template>
  <AdminPage
    title="Phân tích"
    :lead="`Bộ lọc AI có đang làm tốt không.${status?.stand_in ? ' Điểm là điểm mẫu, vì mô hình chưa được kết nối.' : ''}`"
  >
    <div class="pick" role="group" aria-label="Khoảng thời gian">
      <span>Khoảng thời gian</span>
      <button v-for="d in [7, 30, 90] as const" :key="d" type="button" :aria-pressed="days === d" @click="days = d">{{ d }} ngày</button>
    </div>
    <LoadError v-if="data.isError.value" title="Không tải được số liệu" @retry="data.refetch()" />
    <p v-else-if="!a" role="status">Đang tải…</p>
    <template v-else>
      <section class="card" aria-labelledby="fg-h">
        <h2 id="fg-h">Tổng quan {{ days }} ngày</h2>
        <dl class="tl t4">
          <div><dt>{{ num(a.flagged) }}</dt><dd>nội dung bị gắn cờ, trên {{ num(a.checked) }} nội dung được kiểm tra</dd></div>
          <div><dt>{{ a.agreement == null ? '–' : pctText(a.agreement) }}</dt><dd>quyết định của quản lý đồng ý với AI</dd></div>
          <div>
            <dt>{{ overturned }}</dt>
            <dd>
              lần AI bị lật: {{ a.overturned.held_wrong }} giữ nhầm, {{ a.overturned.missed }} bỏ sót{{
                settings.data.value?.auto_delete_enabled || a.overturned.deleted_wrong ? `, ${a.overturned.deleted_wrong} xóa nhầm` : ''
              }}
            </dd>
          </div>
          <div><dt>{{ a.read_only_accounts }}</dt><dd>tài khoản đang chỉ đọc</dd></div>
        </dl>
      </section>

      <section class="card" aria-labelledby="ch1-h">
        <h2 id="ch1-h">Nội dung bị gắn cờ {{ days === 90 ? 'mỗi tuần' : 'mỗi ngày' }}</h2>
        <p class="hint" style="margin-top: 0">Gồm nội dung bị bộ lọc giữ lại hoặc khớp từ khóa chặn.</p>
        <div class="bc">
          <div class="bc-y" aria-hidden="true"><span>{{ top }}</span><span>{{ top / 2 }}</span><span>0</span></div>
          <div
            class="bc-p"
            role="img"
            :aria-label="`Biểu đồ cột: ${total} nội dung bị gắn cờ trong ${days} ngày${max ? `, cao nhất ${max} vào ${peak?.full}` : ''}. Xem dạng bảng bên dưới.`"
          >
            <i class="gl" style="bottom: 50%" /><i class="gl" style="bottom: 100%" />
            <div v-for="(s, i) in series" :key="i" class="bc-b" :style="{ '--v': (s.v / top).toFixed(3) }" :title="`${s.full}: ${s.v}`"><i /><b>{{ s.v }}</b></div>
          </div>
          <div class="bc-x" aria-hidden="true"><span v-for="(s, i) in series" :key="i">{{ s.label }}</span></div>
        </div>
        <details class="wd tbd">
          <summary>Xem dạng bảng</summary>
          <table class="tbl mini">
            <thead><tr><th scope="col">{{ days === 90 ? 'Tuần' : 'Ngày' }}</th><th scope="col" class="num">Bị gắn cờ</th></tr></thead>
            <tbody><tr v-for="(s, i) in series" :key="i"><td>{{ s.full }}</td><td class="num">{{ s.v }}</td></tr></tbody>
          </table>
        </details>
      </section>

      <section class="card" aria-labelledby="ch2-h">
        <h2 id="ch2-h">Tỉ lệ gắn cờ theo nhãn</h2>
        <p class="hint" style="margin-top: 0">Phần trăm trên tổng nội dung được kiểm tra.</p>
        <ul class="hb">
          <li v-for="[t, x, v] in labels" :key="t">
            <span class="hb-l"><b>{{ t }}</b><small>{{ x }}</small></span>
            <span class="hb-t" aria-hidden="true"><i :style="{ '--v': (v / labelMax).toFixed(3) }" /></span>
            <b class="hb-v">{{ pctText(v) }}</b>
          </li>
        </ul>
      </section>

      <section class="card" aria-labelledby="dq-h">
        <h2 id="dq-h">Quyết định và yêu cầu xem xét lại</h2>
        <dl class="tl t4">
          <div><dt>{{ a.approved }}</dt><dd>cho phép đăng</dd></div>
          <div><dt>{{ a.rejected + a.removed }}</dt><dd>từ chối hoặc gỡ</dd></div>
          <div><dt>{{ a.appeals_pending }}</dt><dd>yêu cầu xem xét lại đang chờ</dd></div>
          <div><dt>{{ a.appeals_accepted }} / {{ a.appeals_accepted + a.appeals_declined }}</dt><dd>yêu cầu được chấp nhận</dd></div>
        </dl>
      </section>

      <section class="card" aria-labelledby="aw-h">
        <h2 id="aw-h">Hình phạt đang có hiệu lực</h2>
        <ul v-if="readOnly.data.value?.length" class="rqs">
          <li v-for="r in readOnly.data.value" :key="r.account_id">
            <RouterLink class="rq" to="/quan-tri/nguoi-dung">
              <span class="rq-t">{{ nameOf(r.account_id)?.name ?? 'Tài khoản' }}</span>
              <small
                >{{ nameOf(r.account_id)?.unit ? `Căn hộ ${nameOf(r.account_id)?.unit} · ` : '' }}đủ {{ penalties.data.value?.strike_limit }} vi phạm trong
                {{ penalties.data.value?.strike_window_days }} ngày · gần nhất Quy tắc {{ r.last_rule }}: {{ RULE_SHORT[r.last_rule] }}</small
              >
              <span class="pill"><Icon name="alert" />Chỉ đọc đến {{ dm(r.ends_at) }}</span>
            </RouterLink>
          </li>
        </ul>
        <p v-else class="hint">Không có tài khoản nào đang chỉ đọc.</p>
      </section>
    </template>
  </AdminPage>
</template>
