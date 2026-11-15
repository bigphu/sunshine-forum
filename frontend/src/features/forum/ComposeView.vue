<script setup lang="ts">
import { useQueryClient } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CharLimit from '@/components/CharLimit.vue'
import DraftWarning from '@/components/DraftWarning.vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import PageShell from '@/components/PageShell.vue'
import ReadOnlyCard from '@/components/ReadOnlyCard.vue'
import SaveFailure from '@/components/SaveFailure.vue'
import { useDraftCheck } from '@/composables/useDraftCheck'
import { api, ApiError, errorText, unwrap } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { CHANNEL_ICON, charLen, MAX_CHARS, OUTAGE_HINT, PENDING_OUTAGE_TEXT } from '@/lib/forum'
import { useChannels, useScreeningStatus } from '@/lib/queries'
import { toast } from '@/lib/toast'

// Bài viết mới, tin ban quản lý và sửa bài (prototype composer, editV). Bản nháp được giữ
// trong sessionStorage nếu người viết rời trang.
const route = useRoute()
const router = useRouter()
const auth = useAuth()
const qc = useQueryClient()
const { data: channels } = useChannels()
const { data: status } = useScreeningStatus()

const editId = route.params.id ? String(route.params.id) : ''
const official = route.path === '/dang-bai/thong-bao'
const DRAFT = 'sunshine-draft'

const title = ref('')
const body = ref('')
const slug = ref(official ? 'thong-bao' : String(route.query.kenh ?? 'thao-luan'))
const loadErr = ref(false)
const editChannel = ref('')

if (!editId) {
  try {
    const d = JSON.parse(sessionStorage.getItem(DRAFT) ?? 'null')
    if (d && !official) {
      title.value = d.title ?? ''
      body.value = d.body ?? ''
      if (!route.query.kenh && d.slug) slug.value = d.slug
    }
  } catch {
    /* bỏ qua */
  }
  watch([title, body, slug], () => {
    try {
      if (official) return
      if (title.value || body.value) sessionStorage.setItem(DRAFT, JSON.stringify({ title: title.value, body: body.value, slug: slug.value }))
      else sessionStorage.removeItem(DRAFT)
    } catch {
      /* bỏ qua */
    }
  })
} else {
  loadPost()
}
async function loadPost() {
  loadErr.value = false
  try {
    const t = await unwrap(api.content.GET('/posts/{post_id}', { params: { path: { post_id: editId } } }))
    if (!t.post.mine) return router.replace(`/t/${editId}`)
    title.value = t.post.title
    body.value = t.post.body
    slug.value = t.post.channel.slug
    editChannel.value = t.post.channel.name
  } catch {
    loadErr.value = true
  }
}

const choices = computed(() => (channels.value ?? []).filter((c) => (official ? c.official_only : !c.official_only)))
const channel = computed(() => channels.value?.find((c) => c.slug === slug.value))
const channelId = computed(() => channel.value?.id)
const { warning, reset } = useDraftCheck([title, body], channelId)

const titleErr = ref('')
const bodyErr = ref('')
const busy = ref(false)
const failure = ref<ApiError | null>(null)
const tEl = ref<HTMLInputElement>()
const bEl = ref<HTMLTextAreaElement>()
const over = computed(() => charLen(body.value) > MAX_CHARS)
const heading = computed(() => (editId ? 'Sửa bài viết' : official ? 'Soạn tin ban quản lý' : 'Bài viết mới'))

async function submit() {
  titleErr.value = title.value.trim() ? '' : 'Hãy nhập tiêu đề.'
  bodyErr.value = body.value.trim() ? '' : 'Hãy viết nội dung trước khi đăng.'
  if (titleErr.value) return tEl.value?.focus()
  if (bodyErr.value || over.value) return bEl.value?.focus()
  if (!channelId.value) return
  busy.value = true
  failure.value = null
  try {
    const res = editId
      ? await unwrap(api.content.PATCH('/posts/{post_id}', { params: { path: { post_id: editId } }, body: { title: title.value, body: body.value } }))
      : await unwrap(api.content.POST('/posts', { body: { channel_id: channelId.value, title: title.value, body: body.value } }))
    reset()
    try {
      if (!editId) sessionStorage.removeItem(DRAFT)
    } catch {
      /* bỏ qua */
    }
    qc.invalidateQueries({ queryKey: ['feed'] })
    qc.invalidateQueries({ queryKey: ['thread', res.post_id] })
    if (editId) toast(res.status === 'pending' ? 'Đã lưu. Bài viết đang chờ xem xét lại.' : 'Đã lưu thay đổi.')
    router.push(`/t/${res.post_id}${!editId && res.status === 'published' ? '?moi=1' : ''}`)
  } catch (e) {
    failure.value = e instanceof ApiError ? e : new ApiError(0, errorText(e))
  } finally {
    busy.value = false
  }
}

function cancel() {
  if (editId) router.push(`/t/${editId}`)
  else if (window.history.state?.back) router.back()
  else router.push('/')
}
</script>

<template>
  <PageShell :title="heading" :back="editId ? [`/t/${editId}`, 'Bài viết'] : ['/', 'Bảng tin']">
      <ReadOnlyCard />
      <LoadError v-if="loadErr" title="Không tải được bài viết" @retry="loadPost" />
      <template v-else-if="!auth.readOnlyUntil">
        <div v-if="status && !status.model_online" class="pend" role="status" style="margin: 0">
          <Icon name="clock" /><span>{{ PENDING_OUTAGE_TEXT }}</span>
        </div>
        <p v-if="!editId && !official" class="hint" style="margin: 0">Bản nháp được giữ lại nếu bạn rời trang này.</p>
        <form class="card comp" novalidate @submit.prevent="submit">
          <div v-if="!editId && choices.length > 1" class="pick" role="group" aria-label="Chọn kênh đăng">
            <span>Đăng vào</span>
            <button v-for="c in choices" :key="c.id" type="button" :aria-pressed="slug === c.slug" @click="slug = c.slug">
              <Icon :name="CHANNEL_ICON[c.slug] ?? 'msg'" />{{ c.name }}
            </button>
          </div>
          <p v-else class="hint" style="margin-top: 0">Kênh: <b>{{ editChannel || channel?.name }}</b></p>
          <label class="lb" for="n-t">Tiêu đề</label>
          <input
            id="n-t"
            ref="tEl"
            v-model="title"
            class="field"
            maxlength="150"
            aria-required="true"
            :aria-invalid="titleErr ? 'true' : undefined"
            aria-describedby="n-e w-n"
            @input="titleErr = ''"
          />
          <div id="n-e" aria-live="polite"><p v-if="titleErr" class="err">{{ titleErr }}</p></div>
          <label class="lb" for="n-b">Nội dung</label>
          <textarea
            id="n-b"
            ref="bEl"
            v-model="body"
            class="field"
            :aria-invalid="bodyErr || over ? 'true' : undefined"
            aria-describedby="n-h w-n n-lim"
            @input="bodyErr = ''"
          />
          <p v-if="bodyErr" class="err" role="alert">{{ bodyErr }}</p>
          <CharLimit id="n-lim" :value="body" />
          <p id="n-h" class="hint">
            {{ status && !status.model_online ? OUTAGE_HINT : editId ? 'Bài đã sửa được kiểm tra tự động lại. ' : 'Mọi bài viết đều được kiểm tra tự động. ' }}Đừng
            chia sẻ thông tin riêng tư.
          </p>
          <div id="w-n"><DraftWarning :warning="warning" @fix="bEl?.focus()" /></div>
          <SaveFailure v-if="failure" :message="failure.message" :save="!!editId" :offline="failure.status === 0" @retry="submit" />
          <div class="fm-e">
            <button type="button" class="btn ghost" @click="cancel">Hủy</button>
            <button class="btn" :class="{ pri: !warning }" type="submit" :aria-busy="busy">
              <span v-if="busy" class="spin" aria-hidden="true" />{{
                busy ? (editId ? 'Đang lưu' : 'Đang đăng') : warning ? 'Vẫn gửi' : editId ? 'Lưu thay đổi' : official ? 'Đăng thông báo' : 'Đăng bài'
              }}
            </button>
          </div>
        </form>
      </template>
  </PageShell>
</template>
