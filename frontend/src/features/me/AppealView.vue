<script setup lang="ts">
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, nextTick, ref } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import PageShell from '@/components/PageShell.vue'
import { api, errorText, unwrap } from '@/lib/api/client'
import { APPEAL_ACCEPTED } from '@/lib/forum'
import { dm } from '@/lib/format'
import { useRules } from '@/lib/queries'
import { decisionTitle } from './decisions'

// Yêu cầu xem xét lại (FR-5.5, UC-15): một lần, trong 30 ngày, lý do 20 đến 2.000 ký tự.
// Có bước xem lại trước khi gửi vì không sửa được sau khi gửi.
const route = useRoute()
const qc = useQueryClient()
const id = String(route.params.id)
const list = useQuery({ queryKey: ['me', 'decisions'], queryFn: () => unwrap(api.moderation.GET('/me/decisions')) })
const d = computed(() => list.data.value?.find((x) => x.decision.id === id))
const { data: rules } = useRules()
const rule = computed(() => rules.value?.find((r) => r.number === d.value?.decision.rule))

const why = ref('')
const err = ref('')
const confirm = ref(false)
const busy = ref(false)
const sendErr = ref('')
const len = computed(() => [...why.value.trim()].length)
const status = computed(() => d.value?.appeal?.status)

async function review() {
  err.value = len.value < 20 || len.value > 2000 ? 'Hãy viết lý do từ 20 đến 2.000 ký tự.' : ''
  if (err.value) return document.getElementById('why')?.focus()
  confirm.value = true
  await nextTick()
  document.getElementById('cfh')?.focus()
}
async function send() {
  busy.value = true
  sendErr.value = ''
  try {
    await unwrap(api.moderation.POST('/appeals', { body: { decision_id: id, reason: why.value.trim() } }))
    await qc.invalidateQueries({ queryKey: ['me', 'decisions'] })
    confirm.value = false
    await nextTick()
    document.getElementById('focus-t')?.focus()
  } catch (e) {
    sendErr.value = errorText(e)
  } finally {
    busy.value = false
  }
}
const title = computed(() => (!status.value ? 'Yêu cầu xem xét lại' : status.value === 'pending' ? 'Đã gửi yêu cầu' : 'Kết quả xem xét lại'))
</script>

<template>
  <PageShell :title="title" :back="['/yeu-cau', 'Yêu cầu của tôi']">
    <p v-if="list.isPending.value" role="status">Đang tải…</p>
    <LoadError v-else-if="list.isError.value" title="Không tải được quyết định" @retry="list.refetch()" />
    <section v-else-if="!d" class="card">
      <p>Không tìm thấy quyết định này.</p>
      <div class="row-a"><RouterLink class="btn" to="/yeu-cau">Xem yêu cầu của tôi</RouterLink></div>
    </section>
    <template v-else>
      <template v-if="d.appeal">
        <section v-if="status !== 'pending'" class="card orc" :class="status === 'accepted' ? 'ok' : 'no'" aria-labelledby="or-h">
          <Icon :name="status === 'accepted' ? 'check' : 'x'" />
          <div>
            <h2 id="or-h">{{ status === 'accepted' ? 'Yêu cầu đã được chấp nhận' : 'Yêu cầu không được chấp nhận' }}</h2>
            <p v-if="status === 'accepted'">{{ APPEAL_ACCEPTED }}</p>
            <p v-else>
              Yêu cầu xem xét lại của bạn không được chấp nhận. Nội dung vẫn bị {{ d.decision.action === 'remove' ? 'gỡ' : 'từ chối' }} vì vi
              phạm Quy tắc {{ d.decision.rule }}: {{ rule?.title }}. Đây là quyết định cuối cùng.
            </p>
          </div>
        </section>
        <section class="card">
          <ol aria-label="Tiến trình">
            <li class="sp done">
              <span class="bl"><Icon name="check" /></span>
              <div><b><span class="sr">Đã xong: </span>Đã nhận yêu cầu</b><p>{{ dm(d.appeal.created_at) }}</p></div>
            </li>
            <li v-if="status === 'pending'" class="sp now" aria-current="step">
              <span class="bl" />
              <div>
                <b><span class="sr">Đang diễn ra: </span>Đang xem xét</b>
                <p>Một người khác sẽ đọc lại nội dung của bạn. Bạn sẽ nhận được thông báo khi có quyết định.</p>
              </div>
            </li>
            <li v-else class="sp done">
              <span class="bl"><Icon name="check" /></span>
              <div><b><span class="sr">Đã xong: </span>Đã xem xét</b><p>Một người khác đã đọc lại nội dung của bạn.</p></div>
            </li>
            <li v-if="status === 'pending'" class="sp">
              <span class="bl" />
              <div>
                <b><span class="sr">Chưa tới: </span>Quyết định cuối cùng</b>
                <p>Nếu được chấp nhận, nội dung sẽ được khôi phục và lần vi phạm này sẽ được gỡ bỏ.</p>
              </div>
            </li>
            <li v-else class="sp done" :class="{ no: status === 'declined' }">
              <span class="bl"><Icon :name="status === 'accepted' ? 'check' : 'x'" /></span>
              <div>
                <b><span class="sr">Đã xong: </span>Quyết định cuối cùng</b>
                <p>{{ status === 'accepted' ? 'Nội dung đã được khôi phục và lần vi phạm đã được gỡ bỏ.' : 'Nội dung vẫn bị từ chối.' }}</p>
              </div>
            </li>
          </ol>
        </section>
      </template>

      <section class="card" aria-labelledby="dh">
        <h2 id="dh">{{ decisionTitle(d) }}</h2>
        <q v-if="d.body" class="q">{{ d.body }}</q>
        <p v-else class="hint">Nội dung đã được xóa khỏi hệ thống sau thời hạn lưu.</p>
        <dl class="fx">
          <dt>Trong</dt>
          <dd>{{ d.title }}</dd>
          <dt>Quy tắc</dt>
          <dd>
            <RouterLink class="u" :to="`/quy-tac/${d.decision.rule}`">Quy tắc {{ d.decision.rule }}: {{ rule?.title }}</RouterLink>
          </dd>
          <dt>Nội dung</dt>
          <dd class="fx-n">{{ rule?.text }}</dd>
          <template v-if="!d.appeal && d.decision.appeal_deadline">
            <dt>Hạn yêu cầu</dt>
            <dd>Trước ngày {{ dm(d.decision.appeal_deadline) }}</dd>
          </template>
        </dl>
      </section>
      <section v-if="d.appeal" class="card">
        <h2>Lý do của bạn</h2>
        <p>{{ d.appeal.reason }}</p>
      </section>

      <template v-if="!d.appeal">
        <section v-if="!d.can_appeal" class="card"><p>Đã quá hạn yêu cầu xem xét lại cho nội dung này.</p></section>
        <section v-else-if="confirm" class="card cf" aria-labelledby="cfh">
          <h2 id="cfh" tabindex="-1">Kiểm tra lại trước khi gửi</h2>
          <p>Bạn chỉ có một lần yêu cầu cho nội dung này. Sau khi gửi, bạn không thể sửa hoặc gửi thêm.</p>
          <dl class="fx">
            <dt>Lý do</dt>
            <dd class="fx-n">{{ why.trim() }}</dd>
          </dl>
          <p v-if="sendErr" class="err" role="alert">{{ sendErr }}</p>
          <div class="fm-e" style="margin: 16px 0 0">
            <button class="btn ghost" @click="confirm = false">Sửa lại</button>
            <button class="btn pri" :aria-busy="busy" @click="send">
              <span v-if="busy" class="spin" aria-hidden="true" />Gửi yêu cầu (chỉ một lần)
            </button>
          </div>
        </section>
        <form v-else class="card fm" novalidate @submit.prevent="review">
          <p>
            Bạn có một lần yêu cầu. Một người thật sẽ đọc lại và quyết định cuối cùng. Nếu yêu cầu được chấp nhận, nội dung sẽ được khôi
            phục và lần vi phạm này sẽ được gỡ bỏ.
          </p>
          <div>
            <label class="lb" for="why">Vì sao bạn cho rằng quyết định này chưa đúng?</label>
            <textarea
              id="why"
              v-model="why"
              class="field"
              maxlength="2000"
              aria-required="true"
              :aria-invalid="err ? 'true' : undefined"
              aria-describedby="wcnt wh we"
              @input="err = ''"
            />
            <p id="wcnt" class="hint">{{ len }}/2.000 ký tự{{ len && len < 20 ? `, cần thêm ${20 - len}` : '' }}</p>
            <p id="wh" class="hint">
              Bắt buộc, từ 20 đến 2.000 ký tự. Hãy nói rõ vì sao nội dung này không vi phạm quy tắc, ví dụ hệ thống đã hiểu sai từ ngữ hoặc
              ngữ cảnh.
            </p>
            <div id="we" aria-live="polite"><p v-if="err" class="err">{{ err }}</p></div>
          </div>
          <div class="fm-e" style="margin: 0">
            <RouterLink class="btn ghost" to="/yeu-cau">Để sau</RouterLink>
            <button class="btn pri" type="submit">Xem lại trước khi gửi</button>
          </div>
        </form>
      </template>
      <div v-if="d.appeal" class="row-a">
        <RouterLink class="btn" to="/yeu-cau">Xem yêu cầu của tôi</RouterLink>
        <RouterLink class="btn" to="/">Về bảng tin</RouterLink>
      </div>
    </template>
  </PageShell>
</template>
