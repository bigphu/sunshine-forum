<script setup lang="ts">
import { useQueryClient } from '@tanstack/vue-query'
import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppDialog from '@/components/AppDialog.vue'
import Avatar from '@/components/Avatar.vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import { api, errorText, unwrap } from '@/lib/api/client'
import { age } from '@/lib/format'
import { useChannels, useQueue, useScreeningStatus } from '@/lib/queries'
import { toast } from '@/lib/toast'
import BulkRejectDialog from './BulkRejectDialog.vue'
import { type Case, caseName, GROUP_ICON, groupName, GROUPS, shortTitle, toCases } from './cases'
import CaseView from './CaseView.vue'

// Hàng chờ kiểm duyệt (prototype modV, UC-17/18): danh sách bên trái, trường hợp đang xem bên
// phải; trên điện thoại chỉ một trong hai.
const route = useRoute()
const router = useRouter()
const qc = useQueryClient()
const queue = useQueue()
const { data: channels } = useChannels()
const { data: status } = useScreeningStatus()

// Tùy chọn hiển thị, nhớ trên thiết bị này
type Sort = 'pri' | 'new' | 'old'
type Density = 'cozy' | 'compact'
const SORTS: Record<Sort, string> = { pri: 'Ưu tiên, theo nhóm', new: 'Mới nhất trước', old: 'Cũ nhất trước' }
const DENS: Record<Density, string> = { cozy: 'Thoáng', compact: 'Gọn' }
const view = ref<{ qs: Sort; qd: Density; mf: string }>({ qs: 'pri', qd: 'cozy', mf: 'all' })
try {
  Object.assign(view.value, JSON.parse(localStorage.getItem('sunshine-qv') ?? '{}'))
} catch {
  /* bỏ qua */
}
watch(view, (v) => {
  try {
    localStorage.setItem('sunshine-qv', JSON.stringify(v))
  } catch {
    /* bỏ qua */
  }
}, { deep: true })
const changed = computed(() => +(view.value.qs !== 'pri') + +(view.value.qd !== 'cozy') + +(view.value.mf !== 'all'))
const menu = ref(false)

const all = computed(() => toCases(queue.data.value?.items ?? [], queue.data.value?.appeals ?? []))
const order = (g: string) => GROUPS.findIndex((x) => x[0] === g)
const list = computed(() => {
  const l = all.value.filter((c) => view.value.mf === 'all' || c.item.channel_id === view.value.mf)
  const t = (c: Case) => new Date(c.appeal?.appeal.created_at ?? c.item.created_at).getTime()
  if (view.value.qs === 'new') return [...l].sort((a, b) => t(b) - t(a))
  if (view.value.qs === 'old') return [...l].sort((a, b) => t(a) - t(b))
  return [...l].sort((a, b) => order(a.group) - order(b.group) || t(a) - t(b))
})
const appeals = computed(() => all.value.filter((c) => c.group === 'ap').length)
const caseId = computed(() => (route.params.case as string | undefined) ?? '')
const current = computed(() => (caseId.value ? all.value.find((c) => c.id === caseId.value) : list.value[0]))
const channelName = (id: string) => channels.value?.find((c) => c.id === id)?.name ?? ''

const mark = (c: Case): [string, string] | null =>
  c.appeal
    ? ['clock', 'Ưu tiên']
    : c.item.screening.sentences.some((s) => s.top_label !== 'clean')
      ? ['eye', 'Có đoạn ẩn']
      : c.item.screening.reason === 'language'
        ? ['msg', 'Ngôn ngữ khác']
        : null

// Xác nhận quyết định ngay trong trang, không dùng toast (DESIGN.md "Moderation Queue")
const confirmation = ref('')
const cap = (t: string) => t[0].toUpperCase() + t.slice(1)

// Đã xử lý trong phiên này
const done = ref<{ c: Case; label: string }[]>([])
const reader = ref<InstanceType<typeof CaseView>>()
async function decided(c: Case, label: string) {
  done.value.unshift({ c, label })
  const i = list.value.findIndex((x) => x.id === c.id)
  const next = list.value[i + 1] ?? list.value[i - 1]
  confirmation.value = `${cap(caseName(c))}: đã ${label}. Đã gửi thông báo cho người viết.`
  await queue.refetch()
  qc.invalidateQueries({ queryKey: ['feed'] })
  if (next && next.id !== c.id) router.replace(`/kiem-duyet/${next.id}`)
  else router.replace('/kiem-duyet')
}

// Chọn nhiều để cho phép hoặc từ chối cùng lúc; yêu cầu xem xét lại quyết định từng mục
const selecting = ref(false)
const selected = ref(new Set<string>())
const picked = computed(() => list.value.filter((c) => selected.value.has(c.id) && !c.appeal))
const bulk = ref(false)
function toggle(id: string) {
  const s = new Set(selected.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  selected.value = s
}
function selectGroup(g: string) {
  selected.value = new Set([...selected.value, ...list.value.filter((c) => c.group === g).map((c) => c.id)])
}
watch(selecting, (v) => !v && (selected.value = new Set()))
// Cho phép hàng loạt: nếu có mục chứa đoạn bị ẩn, cảnh báo một lần rồi mới gửi
const bulkWarn = ref(false)
watch(selected, () => (bulkWarn.value = false))
async function bulkApprove() {
  const hidden = picked.value.some((c) => c.item.screening.sentences.some((s) => s.top_label !== 'clean'))
  if (hidden && !bulkWarn.value) {
    bulkWarn.value = true
    return
  }
  try {
    const n = picked.value.length
    await unwrap(api.moderation.POST('/moderation/decisions', { body: { items: picked.value.map((c) => c.item.ref), action: 'approve' } }))
    await bulkDone(n, 'cho phép đăng')
  } catch (e) {
    toast(errorText(e))
  }
}
async function bulkDone(n: number, label: string) {
  selected.value = new Set()
  confirmation.value = `Đã ${label} ${n} mục. Đã gửi thông báo cho từng người viết.`
  await queue.refetch()
  if (caseId.value && !all.value.some((c) => c.id === caseId.value)) router.replace('/kiem-duyet')
}

// Phím tắt: J/K đổi trường hợp, H hiện đoạn ẩn, A và R hai lựa chọn, ? mở bảng phím tắt
const keysOpen = ref(false)
function onKey(e: KeyboardEvent) {
  if (e.ctrlKey || e.metaKey || e.altKey) return
  const t = e.target as HTMLElement
  if (t.closest?.('input,textarea,select,[contenteditable]') || document.querySelector('dialog[open]')) return
  const k = e.key.toLowerCase()
  if (e.key === '?') {
    e.preventDefault()
    keysOpen.value = true
  } else if ((k === 'j' || k === 'k') && list.value.length) {
    e.preventDefault()
    const i = current.value ? list.value.findIndex((x) => x.id === current.value!.id) : -1
    const n = list.value[(i + (k === 'j' ? 1 : -1) + list.value.length) % list.value.length]
    router.replace(`/kiem-duyet/${n.id}`)
  } else if (k === 'h') reader.value?.reveal()
  else if (k === 'a') reader.value?.first()
  else if (k === 'r') reader.value?.second()
}
onMounted(() => document.addEventListener('keydown', onKey))
onBeforeUnmount(() => document.removeEventListener('keydown', onKey))
watch(caseId, async (id) => {
  if (!id) return
  await nextTick()
  document.getElementById('case-h')?.focus({ preventScroll: true })
})
</script>

<template>
  <div class="ctn wide">
    <div class="modw" :class="{ 'has-case': !!caseId }">
      <header class="mh">
        <div>
          <h1 :id="caseId ? 'mq-h' : 'focus-t'" tabindex="-1">Hàng chờ kiểm duyệt</h1>
          <p>
            {{
              all.length
                ? `${all.length} trường hợp đang chờ${appeals ? ` · ${appeals} yêu cầu xem xét lại được ưu tiên` : ''}`
                : 'Không có trường hợp nào đang chờ'
            }}
          </p>
        </div>
      </header>
      <div v-if="status && !status.model_online" class="warn ob">
        <Icon name="alert" />
        <div>
          <span>Bộ lọc AI đang ngừng hoạt động</span>
          <span class="wc">Nội dung mới được xử lý theo chính sách khi bộ lọc lỗi và sẽ tự động kiểm tra lại khi bộ lọc hoạt động trở lại. Bạn không cần duyệt từng mục.</span>
        </div>
      </div>
      <div v-if="confirmation" class="und dn" role="status">
        <Icon name="check" /><span>{{ confirmation }}</span>
      </div>
      <div class="mod">
        <div class="mqw">
          <div class="qtb">
            <div class="fw">
              <PopoverRoot v-model:open="menu">
                <PopoverTrigger class="btn" :class="{ on: changed }" :aria-label="`Tùy chọn hiển thị${changed ? `, đã đổi ${changed} tùy chọn` : ''}`">
                  <Icon name="filter" />Hiển thị<span v-if="changed" class="cnt" aria-hidden="true">{{ changed }}</span>
                </PopoverTrigger>
                <PopoverPortal>
                  <PopoverContent class="fmenu" role="group" aria-label="Tùy chọn hiển thị" align="start" :side-offset="8">
                    <p>Sắp xếp</p>
                    <button v-for="(t, v) in SORTS" :key="v" :aria-pressed="view.qs === v" @click="view.qs = v">
                      <Icon v-if="view.qs === v" name="check" /><span v-else class="ck" />{{ t }}
                    </button>
                    <p>Mật độ</p>
                    <button v-for="(t, v) in DENS" :key="v" :aria-pressed="view.qd === v" @click="view.qd = v">
                      <Icon v-if="view.qd === v" name="check" /><span v-else class="ck" />{{ t }}
                    </button>
                    <p>Kênh</p>
                    <button :aria-pressed="view.mf === 'all'" @click="view.mf = 'all'">
                      <Icon v-if="view.mf === 'all'" name="check" /><span v-else class="ck" />Tất cả kênh
                    </button>
                    <button v-for="ch in channels" :key="ch.id" :aria-pressed="view.mf === ch.id" @click="view.mf = ch.id">
                      <Icon v-if="view.mf === ch.id" name="check" /><span v-else class="ck" />{{ ch.name }}
                    </button>
                  </PopoverContent>
                </PopoverPortal>
              </PopoverRoot>
            </div>
            <button class="btn" :aria-pressed="selecting" @click="selecting = !selecting">Chọn nhiều</button>
            <button class="ib kb-b" aria-label="Phím tắt" @click="keysOpen = true"><Icon name="kbd" /></button>
            <p v-if="changed" class="qsum">
              <span
                >Đang xem:
                <template v-if="view.qs !== 'pri'"><b>{{ SORTS[view.qs] }}</b></template>
                <template v-if="view.mf !== 'all'">{{ view.qs !== 'pri' ? ' · ' : '' }}kênh <b>{{ channelName(view.mf) }}</b></template>
                <template v-if="view.qd !== 'cozy'">{{ view.qs !== 'pri' || view.mf !== 'all' ? ' · ' : '' }}mật độ <b>{{ DENS[view.qd] }}</b></template>
              </span>
              <button class="btn ghost" @click="view = { qs: 'pri', qd: 'cozy', mf: 'all' }">Đặt lại</button>
            </p>
          </div>
          <section class="mq" :class="{ compact: view.qd === 'compact' }" aria-label="Danh sách trường hợp">
            <LoadError v-if="queue.isError.value" title="Không tải được hàng chờ" @retry="queue.refetch()" />
            <p v-else-if="queue.isPending.value" role="status">Đang tải…</p>
            <template v-else-if="list.length">
              <template v-if="view.qs === 'pri'">
                <template v-for="[g, n] in GROUPS" :key="g">
                  <section v-if="list.some((c) => c.group === g)" class="qg" :class="`g-${g}`" :aria-labelledby="`qg-${g}`">
                    <div class="qg-h">
                      <h2 :id="`qg-${g}`">
                        <Icon :name="GROUP_ICON[g]" /><span>{{ n }}</span><span class="qc">{{ list.filter((c) => c.group === g).length }}</span>
                      </h2>
                      <button v-if="selecting && g !== 'ap' && list.filter((c) => c.group === g).length > 1" class="btn ghost" @click="selectGroup(g)">
                        Chọn cả nhóm
                      </button>
                    </div>
                    <ul>
                      <li v-for="c in list.filter((x) => x.group === g)" :key="c.id" class="qr">
                        <label v-if="selecting && !c.appeal" class="qck">
                          <input type="checkbox" :checked="selected.has(c.id)" :aria-label="`Chọn ${caseName(c)}`" @change="toggle(c.id)" />
                        </label>
                        <RouterLink class="qa" :to="`/kiem-duyet/${c.id}`" :aria-current="current?.id === c.id ? 'true' : undefined">
                          <Avatar :name="c.item.author.name" small /><b>{{ shortTitle(c) }}</b>
                          <small>
                            <span>{{ c.item.ref.type === 'post' ? 'Bài viết' : 'Bình luận' }} · {{ c.item.channel_name }} · {{ age(c.item.created_at) }}</span>
                            <span v-if="mark(c)" class="qm"><Icon :name="mark(c)![0] as 'eye'" />{{ mark(c)![1] }}</span>
                          </small>
                        </RouterLink>
                      </li>
                    </ul>
                  </section>
                </template>
              </template>
              <ul v-else class="qfl">
                <li v-for="c in list" :key="c.id" class="qr">
                  <label v-if="selecting && !c.appeal" class="qck">
                    <input type="checkbox" :checked="selected.has(c.id)" :aria-label="`Chọn ${caseName(c)}`" @change="toggle(c.id)" />
                  </label>
                  <RouterLink class="qa" :to="`/kiem-duyet/${c.id}`" :aria-current="current?.id === c.id ? 'true' : undefined">
                    <Avatar :name="c.item.author.name" small /><b>{{ shortTitle(c) }}</b>
                    <small>
                      <span>{{ groupName(c.group) }} · {{ c.item.ref.type === 'post' ? 'Bài viết' : 'Bình luận' }} · {{ c.item.channel_name }} · {{ age(c.item.created_at) }}</span>
                      <span v-if="mark(c)" class="qm"><Icon :name="mark(c)![0] as 'eye'" />{{ mark(c)![1] }}</span>
                    </small>
                  </RouterLink>
                </li>
              </ul>
            </template>
            <div v-else-if="all.length" class="ep">
              <p>Không có trường hợp nào trong kênh {{ channelName(view.mf) }}.</p>
              <div class="row-a" style="justify-content: center"><button class="btn" @click="view.mf = 'all'">Xóa bộ lọc</button></div>
            </div>
            <div v-else class="ep">
              <p><b>Hàng chờ đã trống.</b></p>
              <p>Nội dung bị giữ lại sẽ xuất hiện ở đây.</p>
            </div>
            <p v-if="selecting && list.some((c) => c.appeal)" class="hint qn">Yêu cầu xem xét lại cần được quyết định từng mục.</p>
            <details v-if="done.length" class="qd">
              <summary><Icon name="chev" />Đã xử lý trong phiên này ({{ done.length }})</summary>
              <ul>
                <li v-for="(x, i) in done" :key="i">{{ cap(caseName(x.c)) }}: {{ x.label }}</li>
              </ul>
            </details>
            <div v-if="selecting" class="bb" role="region" aria-label="Thao tác hàng loạt">
              <b>{{ picked.length ? `Đã chọn ${picked.length} mục` : 'Đánh dấu các mục để xử lý cùng lúc' }}</b>
              <p v-if="bulkWarn" class="hint">Có mục có đoạn bị ẩn bạn chưa xem. Bấm “Vẫn cho phép” nếu bạn chắc chắn.</p>
              <div v-if="picked.length" class="bb-a">
                <button class="btn" @click="bulkApprove">{{ bulkWarn ? 'Vẫn cho phép' : 'Cho phép' }}</button>
                <button class="btn" @click="bulk = true">Từ chối…</button>
                <button class="btn ghost" @click="selected = new Set()">Bỏ chọn</button>
              </div>
            </div>
          </section>
        </div>
        <section class="rdr" aria-label="Trường hợp đang xem">
          <CaseView v-if="current" ref="reader" :key="current.id" :c="current" :standalone="!!caseId" :auto-delete="false" @decided="decided" />
          <template v-else-if="caseId && !queue.isPending.value">
            <RouterLink class="bk bk-q" to="/kiem-duyet"><Icon name="back" />Hàng chờ</RouterLink>
            <div class="card">
              <h2 id="focus-t" tabindex="-1" class="case-h" style="margin: 0">Trường hợp này đã được xử lý</h2>
              <p>Hãy chọn một trường hợp khác trong hàng chờ.</p>
            </div>
          </template>
        </section>
      </div>
    </div>
    <BulkRejectDialog v-if="bulk" :cases="picked" @close="bulk = false" @done="bulkDone" />
    <AppDialog v-if="keysOpen" title="Phím tắt" @close="keysOpen = false">
      <template #default="{ close }">
        <p class="rp-n">Chỉ dùng được khi bạn không gõ chữ. Mọi thao tác đều có nút bấm tương ứng.</p>
        <dl class="kbl">
          <div v-for="[k, t] in [['J', 'Trường hợp tiếp theo'], ['K', 'Trường hợp trước'], ['H', 'Hiện hoặc ẩn đoạn bị ẩn'], ['A', 'Lựa chọn thứ nhất, ví dụ Cho phép đăng'], ['R', 'Từ chối hoặc Giữ quyết định'], ['?', 'Mở bảng này']]" :key="k">
            <dt><kbd>{{ k }}</kbd></dt>
            <dd>{{ t }}</dd>
          </div>
        </dl>
        <div class="de"><button type="button" class="btn pri" @click="close">Đóng</button></div>
      </template>
    </AppDialog>
  </div>
</template>
