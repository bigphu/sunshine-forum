<script setup lang="ts">
import { useQueryClient } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import AppDialog from '@/components/AppDialog.vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import { api, errorText, unwrap } from '@/lib/api/client'
import { keys, useChannels, useScreeningStatus } from '@/lib/queries'
import { toast } from '@/lib/toast'
import type { PenaltySettings, ScreeningSettings } from '@/lib/types'
import AdminPage from './AdminPage.vue'
import { adminKeys, usePenaltySettings, useScreeningSettings } from './queries'

// Bộ lọc AI (FR-4, UC-20): ngưỡng chung và theo kênh, tự động xóa (mặc định tắt), chính sách
// khi bộ lọc lỗi, ngưỡng chỉ đọc. Có hiệu lực ngay sau khi lưu, và được ghi vào nhật ký.
const qc = useQueryClient()
const settings = useScreeningSettings()
const penalties = usePenaltySettings()
const { data: channels } = useChannels()
const { data: status } = useScreeningStatus()
const pubChannels = computed(() => channels.value ?? [])

// Bản nháp dạng điểm 0–100 như trên giao diện
// Ô số trả về số khi đã sửa, chuỗi khi chưa
type Num = string | number
interface Draft {
  hold: Num
  delOn: boolean
  del: Num
  ch: Record<string, [Num, Num]>
  ro: [Num, Num, Num]
}
const to100 = (v: number | null | undefined) => (v == null ? '' : String(Math.round(v * 100)))
function fromServer(s: ScreeningSettings, p: PenaltySettings): Draft {
  const ch: Draft['ch'] = {}
  for (const c of pubChannels.value) {
    const t = s.channel_thresholds.find((x) => x.channel_id === c.id)
    ch[c.id] = [to100(t?.hold_threshold), to100(t?.auto_delete_threshold)]
  }
  return { hold: to100(s.hold_threshold), delOn: s.auto_delete_enabled, del: to100(s.auto_delete_threshold), ch, ro: [String(p.strike_limit), String(p.strike_window_days), String(p.timeout_days)] }
}
const base = ref<Draft | null>(null)
const draft = ref<Draft | null>(null)
watch(
  [() => settings.data.value, () => penalties.data.value, pubChannels],
  ([s, p]) => {
    if (!s || !p || !pubChannels.value.length) return
    base.value = fromServer(s, p)
    draft.value = JSON.parse(JSON.stringify(base.value))
  },
  { immediate: true },
)

// v-model trên input type=number trả về số; so sánh theo chuỗi
const str = (v: unknown) => String(v ?? '').trim()
const diffs = computed(() => {
  const a = base.value
  const b = draft.value
  if (!a || !b) return [] as string[]
  const out: string[] = []
  if (str(a.hold) !== str(b.hold)) out.push('hold')
  if (a.delOn !== b.delOn) out.push('delOn')
  if (b.delOn && str(a.del) !== str(b.del)) out.push('del')
  for (const id of Object.keys(b.ch)) {
    if (str(a.ch[id][0]) !== str(b.ch[id][0]) || (b.delOn && str(a.ch[id][1]) !== str(b.ch[id][1]))) out.push(`ch.${id}`)
  }
  b.ro.forEach((v, i) => str(a.ro[i]) !== str(v) && out.push(`ro.${i}`))
  return out
})

// Dải ngưỡng: đăng ngay, giữ lại, tự động xóa
const strip = computed(() => {
  const d = draft.value
  const a = Number(d?.hold)
  const b = d?.delOn ? Number(d.del) : 100
  const ok = a >= 0 && b <= 100 && a < b
  const x = ok ? a : Number(base.value?.hold ?? 50)
  const y = ok ? b : d?.delOn ? Number(base.value?.del ?? 100) : 100
  return { a: x, b: y }
})
const stripText = computed(() =>
  draft.value?.delOn
    ? `Đăng ngay: dưới ${strip.value.a} · Giữ lại: ${strip.value.a} đến dưới ${strip.value.b} · Tự động xóa (chỉ nhãn offensive): từ ${strip.value.b}.`
    : `Đăng ngay: dưới ${strip.value.a} · Giữ lại: từ ${strip.value.a}. Tự động xóa đang tắt.`,
)

const err = ref('')
const bad = ref(new Set<string>())
const saving = ref(false)
const num = (v: unknown) => (/^\d+$/.test(str(v)) ? Number(v) : NaN)
function validate(): string[] {
  const d = draft.value!
  const errs: string[] = []
  const b = new Set<string>()
  const pair = (h: Num, del: Num, name: string, hk: string, dk: string, optional = false) => {
    if (optional && !str(h)) return
    const x = num(h)
    const y = num(del)
    if (isNaN(x) || x < 0 || x > 100) return errs.push(`${name}: Giữ lại phải từ 0 đến 100`), b.add(hk)
    if (!d.delOn || (optional && !str(del))) return
    if (isNaN(y) || y < 0 || y > 100) return errs.push(`${name}: Tự động xóa phải từ 0 đến 100`), b.add(dk)
    if (x >= y) errs.push(`${name}: Giữ lại phải nhỏ hơn Tự động xóa`), b.add(hk)
  }
  pair(d.hold, d.del, 'Ngưỡng chung', 'hold', 'del')
  for (const c of pubChannels.value) pair(d.ch[c.id][0], d.ch[c.id][1], c.name, `ch.${c.id}.0`, `ch.${c.id}.1`, true)
  const lim: [number, number, string][] = [
    [1, 20, 'Số vi phạm'],
    [7, 365, 'Số ngày tính vi phạm'],
    [1, 90, 'Số ngày chỉ đọc'],
  ]
  d.ro.forEach((v, i) => {
    const n = num(v)
    if (isNaN(n) || n < lim[i][0] || n > lim[i][1]) errs.push(`${lim[i][2]} phải từ ${lim[i][0]} đến ${lim[i][1]}`), b.add(`ro.${i}`)
  })
  bad.value = b
  return errs
}
async function save() {
  const errs = validate()
  if (errs.length) {
    err.value = `Chưa lưu được: ${errs.join('; ')}.`
    document.getElementById(`cf-${[...bad.value][0]}`)?.focus()
    return
  }
  const d = draft.value!
  const s = settings.data.value!
  const n = diffs.value.length
  saving.value = true
  err.value = ''
  try {
    if (diffs.value.some((x) => x === 'hold' || x === 'del' || x === 'delOn'))
      await unwrap(
        api.screening.PUT('/screening/settings', {
          body: { hold_threshold: num(d.hold) / 100, auto_delete_enabled: d.delOn, auto_delete_threshold: (d.delOn ? num(d.del) : num(base.value!.del)) / 100, outage_policy: s.outage_policy },
        }),
      )
    for (const c of pubChannels.value) {
      if (!diffs.value.includes(`ch.${c.id}`)) continue
      const [h, del] = d.ch[c.id]
      await unwrap(
        api.screening.PUT('/screening/channels/{channel_id}/threshold', {
          params: { path: { channel_id: c.id } },
          body: { hold_threshold: str(h) ? num(h) / 100 : null, auto_delete_threshold: str(del) ? num(del) / 100 : null },
        }),
      )
    }
    if (diffs.value.some((x) => x.startsWith('ro.')))
      await unwrap(
        api.penalties.PUT('/penalties/settings', { body: { strike_limit: num(d.ro[0]), strike_window_days: num(d.ro[1]), timeout_days: num(d.ro[2]) } }),
      )
    await Promise.all([qc.invalidateQueries({ queryKey: adminKeys.settings }), qc.invalidateQueries({ queryKey: adminKeys.penalties })])
    toast(`Đã lưu ${n} thay đổi. Bộ lọc dùng giá trị mới ngay.`)
  } catch (e) {
    err.value = errorText(e)
  } finally {
    saving.value = false
  }
}
function discard() {
  draft.value = JSON.parse(JSON.stringify(base.value))
  err.value = ''
  bad.value = new Set()
}

// Chính sách khi bộ lọc lỗi: đổi cần xác nhận
const POLICY = {
  fail_closed: ['Giữ lại để kiểm tra sau', 'Mặc định. Nội dung mới được giữ lại và tự động kiểm tra lại khi bộ lọc hoạt động trở lại. Quản lý không phải duyệt từng mục.'],
  fail_open: ['Vẫn đăng, chỉ chạy từ khóa chặn', 'Nội dung mới được đăng nếu không khớp từ khóa chặn, rồi được kiểm tra lại khi bộ lọc hoạt động trở lại và bị ẩn nếu vi phạm.'],
} as const
type Policy = keyof typeof POLICY
const policyTo = ref<Policy | null>(null)
const policyKey = ref(0)
async function changePolicy(close: () => void) {
  const s = settings.data.value!
  try {
    await unwrap(
      api.screening.PUT('/screening/settings', {
        body: { hold_threshold: s.hold_threshold, auto_delete_enabled: s.auto_delete_enabled, auto_delete_threshold: s.auto_delete_threshold, outage_policy: policyTo.value! },
      }),
    )
    close()
    toast('Đã đổi chính sách khi bộ lọc lỗi.')
    qc.invalidateQueries({ queryKey: adminKeys.settings })
  } catch (e) {
    toast(errorText(e))
  }
}

// Chỉ có với bộ chấm mẫu: giả lập bộ lọc AI ngừng hoạt động để thử luồng sự cố
async function setOffline(offline: boolean) {
  try {
    await unwrap(api.screening.PUT('/screening/stand-in', { body: { offline } }))
    toast(offline ? 'Đã giả lập bộ lọc AI ngừng hoạt động.' : 'Bộ lọc AI đã hoạt động trở lại. Nội dung bị giữ sẽ được kiểm tra lại.')
    qc.invalidateQueries({ queryKey: adminKeys.settings })
    qc.invalidateQueries({ queryKey: keys.status })
  } catch (e) {
    toast(errorText(e))
  }
}
const inv = (k: string) => (bad.value.has(k) ? 'true' : undefined)
</script>

<template>
  <AdminPage title="Bộ lọc AI" lead="Thay đổi có hiệu lực ngay sau khi lưu, không cần khởi động lại.">
    <LoadError v-if="settings.isError.value || penalties.isError.value" title="Không tải được cấu hình bộ lọc" @retry="(settings.refetch(), penalties.refetch())" />
    <p v-else-if="!draft" role="status">Đang tải…</p>
    <template v-else>
      <section class="card">
        <h2>Trạng thái</h2>
        <p class="fst">
          <span v-if="status && !status.model_online" class="pill"><Icon name="alert" />Đang ngừng hoạt động</span>
          <span v-else class="pill ok"><Icon name="check" />Đang hoạt động</span>
          <span
            >{{ status?.stand_in ? 'Bộ chấm mẫu cho điểm mẫu, mô hình thật chưa được kết nối' : `Mô hình ${status?.model_version ?? ''}` }}. Điểm từ 0 đến
            100.</span
          >
        </p>
        <label v-if="settings.data.value?.stand_in" class="tg">
          <span><b>Giả lập bộ lọc ngừng hoạt động</b><small>Chỉ có khi dùng bộ chấm mẫu, để thử luồng sự cố trước khi nối mô hình thật.</small></span>
          <input type="checkbox" role="switch" :checked="settings.data.value.stand_in_offline" @change="setOffline(($event.target as HTMLInputElement).checked)" />
        </label>
      </section>

      <section class="card" aria-labelledby="pol-h">
        <h2 id="pol-h">Khi bộ lọc AI lỗi</h2>
        <fieldset :key="policyKey" class="rul">
          <legend class="sr">Chính sách khi bộ lọc lỗi</legend>
          <label v-for="(v, k) in POLICY" :key="k" class="rd">
            <input
              type="radio"
              name="pol"
              :value="k"
              :checked="settings.data.value?.outage_policy === k"
              @change="policyTo = k"
            /><span>{{ v[0] }}<small>{{ v[1] }}</small></span>
          </label>
        </fieldset>
        <p class="hint">Luôn áp dụng: bình luận trong Tin ban quản lý vẫn bị giữ lại. Đổi chính sách cần xác nhận và được ghi vào nhật ký kiểm tra.</p>
      </section>

      <section class="card" aria-labelledby="thg-h">
        <h2 id="thg-h">Ngưỡng chung</h2>
        <p>Áp dụng cho mọi kênh chưa có ngưỡng riêng.</p>
        <div class="thr" aria-hidden="true" :style="{ '--a': strip.a + '%', '--b': strip.b - strip.a + '%' }">
          <span class="t0">Đăng ngay</span><span class="t1">Giữ lại</span><span v-if="draft.delOn" class="t2">Tự động xóa</span>
        </div>
        <p id="thr-l" class="hint thr-l">{{ stripText }}</p>
        <div class="thg">
          <div>
            <label class="lb" for="cf-hold">Giữ lại từ điểm</label>
            <div class="nu"><input id="cf-hold" v-model="draft.hold" class="field nin" type="number" inputmode="numeric" min="0" max="100" :aria-invalid="inv('hold')" /><span>/100</span></div>
          </div>
          <div v-if="draft.delOn">
            <label class="lb" for="cf-del">Tự động xóa từ điểm</label>
            <div class="nu"><input id="cf-del" v-model="draft.del" class="field nin" type="number" inputmode="numeric" min="0" max="100" :aria-invalid="inv('del')" /><span>/100</span></div>
          </div>
        </div>
        <label class="tg">
          <span
            ><b>Tự động xóa</b
            ><small>Mặc định tắt: nội dung có điểm cao được giữ lại để quản lý xem xét. Chỉ nên bật sau khi đã kiểm tra độ chính xác của mô hình trên nội dung của tòa nhà.</small></span
          >
          <input v-model="draft.delOn" type="checkbox" role="switch" />
        </label>
        <p class="hint">{{ draft.delOn ? 'Tự động xóa chỉ áp dụng cho nhãn offensive (Quy tắc 4). ' : '' }}Nhãn hate luôn được giữ lại để quản lý chọn quy tắc.</p>
      </section>

      <section class="card" aria-labelledby="tch-h">
        <h2 id="tch-h">Theo kênh</h2>
        <table class="tbl">
          <thead>
            <tr><th scope="col">Kênh</th><th scope="col">Giữ lại từ</th><th v-if="draft.delOn" scope="col">Tự động xóa từ</th></tr>
          </thead>
          <tbody>
            <tr v-for="c in pubChannels" :key="c.id">
              <th scope="row" data-l="Kênh">{{ c.name }}</th>
              <td data-l="Giữ lại từ">
                <div class="nu">
                  <input :id="`cf-ch.${c.id}.0`" v-model="draft.ch[c.id][0]" class="field nin" type="number" inputmode="numeric" min="0" max="100" :placeholder="String(draft.hold)" :aria-label="`Giữ lại từ, ${c.name}`" :aria-invalid="inv(`ch.${c.id}.0`)" /><span>/100</span>
                </div>
              </td>
              <td v-if="draft.delOn" data-l="Tự động xóa từ">
                <div class="nu">
                  <input :id="`cf-ch.${c.id}.1`" v-model="draft.ch[c.id][1]" class="field nin" type="number" inputmode="numeric" min="0" max="100" :placeholder="String(draft.del)" :aria-label="`Tự động xóa từ, ${c.name}`" :aria-invalid="inv(`ch.${c.id}.1`)" /><span>/100</span>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
        <p class="hint">Để trống để dùng ngưỡng chung. Ở Tin ban quản lý, ngưỡng áp dụng cho cả bình luận của cư dân và bài chính thức.</p>
      </section>

      <section class="card" aria-labelledby="ro-h">
        <h2 id="ro-h">Tạm chỉ được đọc</h2>
        <p class="sent">
          Sau <input id="cf-ro.0" v-model="draft.ro[0]" class="field nin" type="number" min="1" max="20" aria-label="Số vi phạm" :aria-invalid="inv('ro.0')" /> vi phạm đã xác nhận
          trong <input id="cf-ro.1" v-model="draft.ro[1]" class="field nin" type="number" min="7" max="365" aria-label="Số ngày tính vi phạm" :aria-invalid="inv('ro.1')" /> ngày, tài
          khoản chỉ được đọc trong <input id="cf-ro.2" v-model="draft.ro[2]" class="field nin" type="number" min="1" max="90" aria-label="Số ngày chỉ đọc" :aria-invalid="inv('ro.2')" /> ngày.
        </p>
        <p class="hint">Cảnh báo trước khi gửi và nội dung đang chờ không bị tính. Yêu cầu xem xét lại được chấp nhận sẽ gỡ vi phạm.</p>
      </section>

      <div v-if="diffs.length" id="sbar" class="sbar" role="region" aria-label="Thay đổi chưa lưu">
        <span><b>{{ diffs.length }} thay đổi chưa lưu.</b> <span id="cfg-e" role="alert">{{ err }}</span></span>
        <button class="btn ghost" @click="discard">Hủy thay đổi</button>
        <button class="btn pri" :aria-busy="saving" @click="save">Lưu thay đổi</button>
      </div>
    </template>

    <AppDialog v-if="policyTo" title="Đổi chính sách khi bộ lọc lỗi?" @close="(policyTo = null), policyKey++">
      <template #default="{ close }">
        <form novalidate @submit.prevent="changePolicy(close)">
          <p class="rp-n"><b>{{ POLICY[policyTo][0] }}.</b> {{ POLICY[policyTo][1] }}</p>
          <p class="hint">Thay đổi có hiệu lực ngay{{ status && !status.model_online ? ', kể cả với sự cố đang diễn ra,' : '' }} và được ghi vào nhật ký kiểm tra.</p>
          <div class="de">
            <button type="button" class="btn ghost" @click="close">Hủy</button>
            <button class="btn pri" type="submit">Đổi chính sách</button>
          </div>
        </form>
      </template>
    </AppDialog>
  </AdminPage>
</template>
