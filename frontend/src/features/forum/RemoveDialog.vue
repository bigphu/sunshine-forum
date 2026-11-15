<script setup lang="ts">
import { useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import AppDialog from '@/components/AppDialog.vue'
import { api, errorText } from '@/lib/api/client'
import { dm } from '@/lib/format'
import { useRules } from '@/lib/queries'
import { toast } from '@/lib/toast'

// Ban quản lý gỡ nội dung bộ lọc đã cho đăng (FR-4.1, UC-19): chọn quy tắc, xem đúng tin
// nhắn người viết sẽ nhận, rồi "Gửi và gỡ"
const props = defineProps<{ type: 'post' | 'comment'; id: string; author: string; channel: string }>()
const emit = defineEmits<{ close: []; removed: [] }>()
const qc = useQueryClient()
const { data: rules } = useRules()
const rule = ref<number | null>(null)
const err = ref('')
const busy = ref(false)
const dlg = ref<InstanceType<typeof AppDialog>>()

const deadline = dm(new Date(Date.now() + 30 * 86400_000))
const message = computed(() => {
  const r = rules.value?.find((x) => x.number === rule.value)
  return r
    ? `Nội dung của bạn trong kênh ${props.channel} đã bị gỡ vì vi phạm Quy tắc ${r.number}: ${r.title}. Bạn có thể yêu cầu xem xét lại một lần trước ngày ${deadline}.`
    : ''
})

async function submit() {
  if (!rule.value) {
    err.value = 'Hãy chọn quy tắc bị vi phạm.'
    return
  }
  busy.value = true
  try {
    await api.moderation.POST('/moderation/removals', { body: { item: { type: props.type, id: props.id }, rule: rule.value } })
    toast(`Đã gỡ ${props.type === 'post' ? 'bài viết' : 'bình luận'} của ${props.author}, Quy tắc ${rule.value}. Đã gửi thông báo cho người viết.`)
    qc.invalidateQueries({ queryKey: ['feed'] })
    qc.invalidateQueries({ queryKey: ['thread'] })
    emit('removed')
    dlg.value?.close()
  } catch (e) {
    err.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AppDialog ref="dlg" :title="`Gỡ ${type === 'post' ? 'bài viết' : 'bình luận'} của ${author}`" @close="emit('close')">
    <template #default="{ close }">
      <form novalidate @submit.prevent="submit">
        <p class="rp-n">
          Nội dung sẽ bị ẩn khỏi diễn đàn. Người viết nhận tin nhắn nêu quy tắc và có thể yêu cầu xem xét lại. Hệ thống ghi nhận
          đây là lần AI bỏ sót.
        </p>
        <fieldset aria-describedby="re">
          <legend class="lb">Nội dung này vi phạm quy tắc nào?</legend>
          <label v-for="r in rules" :key="r.number" class="rd">
            <input v-model="rule" type="radio" name="r" :value="r.number" @change="err = ''" /><span
              >Quy tắc {{ r.number }}. {{ r.title }}</span
            >
          </label>
        </fieldset>
        <div id="re" aria-live="polite"><p v-if="err" class="err">{{ err }}</p></div>
        <div class="msgp">
          <p id="rmv-l" class="lb">Tin nhắn {{ author }} sẽ nhận</p>
          <p class="q" aria-labelledby="rmv-l">{{ message || 'Chọn một quy tắc để xem tin nhắn.' }}</p>
        </div>
        <div class="de">
          <button type="button" class="btn ghost" @click="close">Hủy</button>
          <button class="btn dz" type="submit" :aria-busy="busy"><span v-if="busy" class="spin" aria-hidden="true" />Gửi và gỡ</button>
        </div>
      </form>
    </template>
  </AppDialog>
</template>
