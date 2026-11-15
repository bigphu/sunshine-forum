<script setup lang="ts">
import { computed, ref } from 'vue'
import CharLimit from '@/components/CharLimit.vue'
import DraftWarning from '@/components/DraftWarning.vue'
import SaveFailure from '@/components/SaveFailure.vue'
import { useDraftCheck } from '@/composables/useDraftCheck'
import { api, ApiError, errorText, unwrap } from '@/lib/api/client'
import { charLen, MAX_CHARS, OUTAGE_HINT } from '@/lib/forum'
import { useScreeningStatus } from '@/lib/queries'
import type { Submitted } from '@/lib/types'

// Ô viết bình luận, trả lời hoặc sửa bình luận. Kiểm tra bản nháp khi gõ; gửi thất bại thì
// giữ nguyên chữ.
const props = defineProps<{
  id: string
  postId: string
  channelId: string
  parentId?: string
  commentId?: string
  initial?: string
  label: string
  labelHidden?: boolean
  submitLabel: string
  cancelLabel?: string
  hint?: string
}>()
const emit = defineEmits<{ done: [res: Submitted]; cancel: [] }>()

const body = ref(props.initial ?? '')
const area = ref<HTMLTextAreaElement>()
const busy = ref(false)
const failure = ref<ApiError | null>(null)
const empty = ref(false)
const { data: status } = useScreeningStatus()
const { warning, reset } = useDraftCheck([body], computed(() => props.channelId))
const over = computed(() => charLen(body.value) > MAX_CHARS)

async function submit() {
  empty.value = !body.value.trim()
  if (empty.value || over.value) {
    area.value?.focus()
    return
  }
  busy.value = true
  failure.value = null
  try {
    const res = props.commentId
      ? await unwrap(api.content.PATCH('/comments/{comment_id}', { params: { path: { comment_id: props.commentId } }, body: { body: body.value } }))
      : await unwrap(
          api.content.POST('/posts/{post_id}/comments', {
            params: { path: { post_id: props.postId } },
            body: { body: body.value, ...(props.parentId ? { parent_id: props.parentId } : {}) },
          }),
        )
    body.value = ''
    reset()
    emit('done', res)
  } catch (e) {
    failure.value = e instanceof ApiError ? e : new ApiError(0, errorText(e))
  } finally {
    busy.value = false
  }
}
defineExpose({ focus: () => area.value?.focus() })
</script>

<template>
  <form novalidate @submit.prevent="submit">
    <label class="lb" :class="{ sr: labelHidden }" :for="id">{{ label }}</label>
    <textarea
      :id="id"
      ref="area"
      v-model="body"
      class="field"
      :aria-invalid="empty || over ? 'true' : undefined"
      :aria-describedby="`${id}-h ${id}-w ${id}-lim`"
      @input="empty = false"
    />
    <p v-if="empty" class="err" role="alert">Hãy viết nội dung trước khi gửi.</p>
    <CharLimit :id="`${id}-lim`" :value="body" />
    <p :id="`${id}-h`" class="hint">
      {{ status && !status.model_online ? OUTAGE_HINT : '' }}{{ hint ?? 'Đừng chia sẻ thông tin riêng tư.' }}
    </p>
    <div :id="`${id}-w`"><DraftWarning :warning="warning" @fix="area?.focus()" /></div>
    <SaveFailure
      v-if="failure"
      :message="failure.message"
      :save="!!commentId"
      :offline="failure.status === 0"
      @retry="submit"
    />
    <div class="fm-e">
      <button v-if="cancelLabel" type="button" class="btn ghost" @click="emit('cancel')">{{ cancelLabel }}</button>
      <span v-else />
      <button class="btn" :class="{ pri: !warning }" type="submit" :aria-busy="busy">
        <span v-if="busy" class="spin" aria-hidden="true" />{{ busy ? (commentId ? 'Đang lưu' : 'Đang gửi') : warning ? 'Vẫn gửi' : submitLabel }}
      </button>
    </div>
  </form>
</template>
