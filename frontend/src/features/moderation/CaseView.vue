<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Avatar from '@/components/Avatar.vue'
import Icon from '@/components/Icon.vue'
import TimeAgo from '@/components/TimeAgo.vue'
import { api, errorText, unwrap } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { dm, score2, trunc } from '@/lib/format'
import { useBlocklist, useRules } from '@/lib/queries'
import {
  aiText,
  appealMessage,
  type Case,
  decisionHint,
  groupName,
  hasHidden,
  kindName,
  rejectMessage,
  segments,
  suggestedRules,
  whyHere,
} from './cases'

// Một trường hợp trong hàng chờ (prototype caseV, dpanel, ctxD): nội dung với đoạn bị ẩn,
// lý do ở đây, bảng quyết định, chi tiết. Phím tắt H, A, R gọi qua expose.
const props = defineProps<{ c: Case; standalone: boolean; autoDelete: boolean }>()
const emit = defineEmits<{ decided: [c: Case, label: string] }>()
const auth = useAuth()
const { data: rules } = useRules()
const { data: blocklist } = useBlocklist()

const revealed = ref(false)
const open = ref<'ok' | 'no' | null>(null)
const rule = ref(0)
const ruleErr = ref('')
const warnOk = ref(false)
const warnGo = ref(false)
const busy = ref(false)
const err = ref('')
watch(
  () => props.c.id,
  () => {
    revealed.value = false
    open.value = null
    rule.value = 0
    ruleErr.value = err.value = ''
    warnOk.value = warnGo.value = false
  },
)

const it = computed(() => props.c.item)
const ap = computed(() => props.c.appeal)
const terms = computed(() => (it.value.screening.reason === 'blocklist' ? (blocklist.value ?? []).map((b) => b.term) : []))
// Điểm câu tính trên bình luận hoặc nội dung bài
const bodySegs = computed(() => segments(it.value.body, props.c, terms.value, true))
const titleSegs = computed(() => (it.value.title ? segments(it.value.title, props.c, terms.value, false) : []))
const hidden = computed(() => hasHidden(bodySegs.value))
const lang = computed(() => (it.value.screening.reason === 'language' ? 'en' : undefined))
const sug = computed(() => suggestedRules(props.c))
const ruleOf = (n: number) => rules.value?.find((r) => r.number === n)
const choices = computed(() => (ap.value ? ['Chấp nhận yêu cầu', 'Giữ quyết định'] : ['Cho phép đăng', 'Từ chối']))
const blocked = computed(() => !!ap.value && !ap.value.can_review)
const accept = computed(() => !!ap.value && open.value === 'ok')
const goLabel = computed(() => (accept.value ? 'Gửi và khôi phục' : ap.value ? 'Gửi và giữ quyết định' : 'Gửi và từ chối'))
const still = (l: string) => 'Vẫn ' + l[0].toLowerCase() + l.slice(1)
const message = computed(() =>
  ap.value ? appealMessage(accept.value, ruleOf(ap.value.decision.rule)) : rejectMessage(it.value.channel_name, ruleOf(rule.value)),
)
const history = computed(() => {
  const n = it.value.author_strikes ?? 0
  const w = auth.standing?.strike_window_days ?? 90
  return n ? `${n} vi phạm đã xác nhận trong ${w} ngày` : `Chưa có vi phạm nào trong ${w} ngày`
})

function choose(d: 'ok' | 'no') {
  if (blocked.value) return
  err.value = ''
  // Cho phép đăng khi chưa xem đoạn bị ẩn: cảnh báo một lần, bấm lại để xác nhận
  if (d === 'ok' && !ap.value) {
    if (hidden.value && !revealed.value && !warnOk.value) {
      warnOk.value = true
      return
    }
    return send('approve')
  }
  open.value = open.value === d ? null : d
  warnGo.value = false
}
async function go() {
  if (!ap.value && !rule.value) {
    ruleErr.value = 'Hãy chọn quy tắc bị vi phạm.'
    document.querySelector<HTMLInputElement>('#mr-g input')?.focus()
    return
  }
  if (!accept.value && hidden.value && !revealed.value && !warnGo.value) {
    warnGo.value = true
    return
  }
  if (ap.value) return review(open.value === 'ok')
  return send('reject')
}
async function send(action: 'approve' | 'reject') {
  busy.value = true
  try {
    await unwrap(
      api.moderation.POST('/moderation/decisions', {
        body: { items: [it.value.ref], action, ...(action === 'reject' ? { rule: rule.value } : {}) },
      }),
    )
    emit('decided', props.c, action === 'approve' ? 'cho phép đăng' : `từ chối, Quy tắc ${rule.value}`)
  } catch (e) {
    err.value = errorText(e)
  } finally {
    busy.value = false
  }
}
async function review(accept: boolean) {
  busy.value = true
  try {
    await unwrap(api.moderation.POST('/moderation/appeals/{appeal_id}/review', { params: { path: { appeal_id: props.c.id } }, body: { accept } }))
    emit('decided', props.c, accept ? 'chấp nhận' : 'giữ quyết định')
  } catch (e) {
    err.value = errorText(e)
  } finally {
    busy.value = false
  }
}
defineExpose({
  reveal: () => hidden.value && (revealed.value = !revealed.value),
  first: () => choose('ok'),
  second: () => choose('no'),
})
</script>

<template>
  <div>
    <RouterLink class="bk bk-q" to="/kiem-duyet"><Icon name="back" />Hàng chờ</RouterLink>
    <article class="card case" :aria-labelledby="standalone ? 'focus-t' : 'case-h'">
      <p class="ccx">
        <span class="chip">{{ it.channel_name }}</span><span>{{ groupName(c.group) }}</span>
        <span v-if="it.ref.type === 'comment'">Trong bài <RouterLink class="u" :to="`/t/${it.post_id}`">{{ trunc(it.post_title, 60) }}</RouterLink></span>
        <RouterLink v-else-if="it.status === 'published' || ap" class="u" :to="`/t/${it.post_id}`">Xem trong diễn đàn</RouterLink>
      </p>
      <h2 :id="standalone ? 'focus-t' : 'case-h'" tabindex="-1" class="case-h">{{ kindName(c) }} của {{ it.author.name }}</h2>
      <div class="fc">
        <Avatar :name="it.author.name" />
        <div>
          <div class="c-w">
            <b>{{ it.author.name }}</b><span v-if="it.author.unit">{{ it.author.unit }}</span>
            <span :class="{ 'strk': (it.author_strikes ?? 0) > 0 }">{{ history }}</span><span><TimeAgo :at="it.created_at" /></span>
          </div>
          <p v-if="titleSegs.length" class="fc-tt" :lang="lang">
            <template v-for="(s, i) in titleSegs" :key="i"
              ><mark v-if="s.kind === 'flag'"><span class="sr">Đoạn bị gắn cờ: </span>{{ s.t }}</mark
              ><template v-else>{{ s.t }}</template></template
            >
          </p>
          <p class="fc-t" :lang="lang">
            <template v-for="(s, i) in bodySegs" :key="i"
              ><mark v-if="s.kind === 'hide' && revealed"><span class="sr">Đoạn bị ẩn: </span>{{ s.t }}</mark
              ><mark v-else-if="s.kind === 'hide'" class="hid"><span class="sr">Đoạn bị ẩn</span><span aria-hidden="true">•••••</span></mark
              ><mark v-else-if="s.kind === 'flag'"><span class="sr">Đoạn bị gắn cờ: </span>{{ s.t }}</mark
              ><template v-else>{{ s.t }}</template></template
            >
          </p>
          <button v-if="hidden" class="btn rvb" @click="revealed = !revealed">
            {{ revealed ? 'Ẩn lại đoạn này' : 'Hiện đoạn bị ẩn' }}
          </button>
        </div>
      </div>
      <div v-if="ap" class="pv apw">
        <b>Lý do {{ it.author.name }} yêu cầu xem xét lại</b>
        <p>{{ ap.appeal.reason }}</p>
      </div>
      <p class="why"><b>Vì sao ở đây:</b> {{ whyHere(c, autoDelete) }}</p>

      <section class="dp" aria-labelledby="dp-h">
        <h3 id="dp-h" class="sr">Quyết định</h3>
        <div class="dp-b">
          <!-- data-dc: CSS của prototype tô đĩa biểu tượng xanh/đỏ và màu khi rê chuột hay khi mở -->
          <button class="btn" data-dc="ok" :disabled="blocked || busy" :aria-expanded="ap ? open === 'ok' : undefined" aria-controls="rj" @click="choose('ok')">
            <Icon name="check" />{{ warnOk && !open ? still(choices[0]) : choices[0] }}
          </button>
          <button class="btn" data-dc="no" :disabled="blocked || busy" :aria-expanded="open === 'no'" aria-controls="rj" @click="choose('no')">
            <Icon name="x" />{{ choices[1] }}
          </button>
        </div>
        <p v-if="decisionHint(c)" class="hint">{{ decisionHint(c) }}</p>
        <div v-if="blocked" class="pend">
          <Icon name="alert" /><span>Bạn là người đã ra quyết định trước. Một quản lý khác hoặc quản trị viên cần xem xét lại yêu cầu này.</span>
        </div>
        <div v-if="warnOk && !open" class="warn" role="alert">
          <Icon name="alert" />
          <div><span>Bạn chưa xem đoạn bị ẩn</span><span class="wc">Nên xem trước khi quyết định. Nếu bạn chắc chắn, bấm “{{ still(choices[0]) }}”.</span></div>
        </div>
        <div v-if="open" id="rj" class="rj">
          <fieldset v-if="!ap" id="mr-g" class="rul" aria-describedby="mr-e">
            <legend class="lb">{{ sug.onlySuggested ? 'Chọn Quy tắc 1, 2 hoặc 3' : 'Nội dung vi phạm quy tắc nào?' }}</legend>
            <label v-for="n in sug.suggested" :key="n" class="rd">
              <input v-model="rule" type="radio" name="mr" :value="n" @change="ruleErr = ''" />
              <span>Quy tắc {{ n }}. {{ ruleOf(n)?.title }}<span v-if="!sug.onlySuggested" class="sg">Gợi ý</span></span>
            </label>
            <details v-if="sug.rest.length && sug.suggested.length" class="rn-d" :open="sug.rest.includes(rule)">
              <summary>Quy tắc khác</summary>
              <label v-for="n in sug.rest" :key="n" class="rd">
                <input v-model="rule" type="radio" name="mr" :value="n" @change="ruleErr = ''" /><span>Quy tắc {{ n }}. {{ ruleOf(n)?.title }}</span>
              </label>
            </details>
            <template v-else>
              <label v-for="n in sug.rest" :key="n" class="rd">
                <input v-model="rule" type="radio" name="mr" :value="n" @change="ruleErr = ''" /><span>Quy tắc {{ n }}. {{ ruleOf(n)?.title }}</span>
              </label>
            </template>
          </fieldset>
          <div id="mr-e" aria-live="polite"><p v-if="ruleErr" class="err">{{ ruleErr }}</p></div>
          <div class="msgp">
            <p id="msg-l" class="lb">Tin nhắn {{ it.author.name }} sẽ nhận</p>
            <p class="q" aria-labelledby="msg-l">{{ message || 'Chọn một quy tắc để xem tin nhắn.' }}</p>
          </div>
          <div v-if="warnGo" class="warn" role="alert">
            <Icon name="alert" />
            <div><span>Bạn chưa xem đoạn bị ẩn</span><span class="wc">Nên xem trước khi quyết định. Nếu bạn chắc chắn, bấm “{{ still(goLabel) }}”.</span></div>
          </div>
          <p v-if="err" class="err" role="alert">{{ err }}</p>
          <div class="fm-e">
            <button class="btn ghost" @click="open = null">Hủy</button>
            <button class="btn" :class="accept ? 'pri' : 'dz'" :aria-busy="busy" @click="go">
              <span v-if="busy" class="spin" aria-hidden="true" />{{ warnGo ? still(goLabel) : goLabel }}
            </button>
          </div>
        </div>
        <p v-else-if="err" class="err" role="alert">{{ err }}</p>
      </section>

      <!-- Chi tiết đánh giá: gập lại mặc định, mở sẵn với yêu cầu xem xét lại vì quyết định trước nằm ở đây -->
      <details class="cxd" :open="!!ap">
        <summary><Icon name="chev" /><span>Chi tiết đánh giá</span></summary>
        <dl class="fx">
          <dt>Đánh giá tự động</dt>
          <dd class="fx-n">
            {{ aiText(c) }}
            <span v-if="it.screening.score != null" class="hint" style="display: block; margin: 4px 0 0">
              {{ it.screening.label }} {{ score2(it.screening.score) }} · mô hình {{ it.screening.model_version }}{{
                /điểm mẫu|toy/.test(it.screening.model_version) ? ' (điểm mẫu, không phải mô hình thật)' : ''
              }}
            </span>
          </dd>
          <template v-if="ap">
            <dt>Quyết định trước</dt>
            <dd class="fx-n">
              {{ ap.decision.moderator?.name ?? 'Bộ lọc AI' }} · {{ ap.decision.action === 'remove' ? 'Gỡ' : ap.decision.action === 'auto_delete' ? 'Tự động xóa' : 'Từ chối' }} ·
              Quy tắc {{ ap.decision.rule }} · {{ dm(ap.decision.created_at) }}
            </dd>
          </template>
        </dl>
      </details>
    </article>
  </div>
</template>
