<script setup lang="ts">
import { computed, ref } from 'vue'
import AppDialog from '@/components/AppDialog.vue'
import { api, errorText, unwrap } from '@/lib/api/client'
import { useRules } from '@/lib/queries'
import { type Case, rejectMessage, shortTitle } from './cases'

// Từ chối nhiều mục cùng một quy tắc (prototype openBulk). Bỏ chọn mục nào không muốn từ chối.
const props = defineProps<{ cases: Case[] }>()
const emit = defineEmits<{ close: []; done: [n: number, label: string] }>()
const { data: rules } = useRules()
const keep = ref(new Set(props.cases.map((c) => c.id)))
const rule = ref(0)
const err = ref('')
const busy = ref(false)
const chosen = computed(() => props.cases.filter((c) => keep.value.has(c.id)))
const hiddenCount = computed(() => props.cases.filter((c) => c.item.screening.sentences.some((s) => s.top_label !== 'clean')).length)
// Nội dung thù ghét chỉ được từ chối theo Quy tắc 1, 2 hoặc 3
const hate = computed(() => chosen.value.some((c) => c.item.screening.label === 'hate'))
const allowed = computed(() => (rules.value ?? []).filter((r) => !hate.value || r.number <= 3))
const sample = computed(() => rejectMessage('§', rules.value?.find((r) => r.number === rule.value)))

function toggle(id: string) {
  const s = new Set(keep.value)
  if (s.has(id)) s.delete(id)
  else s.add(id)
  keep.value = s
}
async function submit(close: () => void) {
  err.value = !chosen.value.length ? 'Hãy chọn ít nhất một mục.' : !allowed.value.some((r) => r.number === rule.value) ? 'Hãy chọn quy tắc bị vi phạm.' : ''
  if (err.value) return
  busy.value = true
  try {
    const n = chosen.value.length
    await unwrap(
      api.moderation.POST('/moderation/decisions', { body: { items: chosen.value.map((c) => c.item.ref), action: 'reject', rule: rule.value } }),
    )
    close()
    emit('done', n, `từ chối, Quy tắc ${rule.value},`)
  } catch (e) {
    err.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AppDialog :title="`Từ chối ${chosen.length} mục`" @close="emit('close')">
    <template #default="{ close }">
      <form novalidate @submit.prevent="submit(close)">
        <p class="rp-n">
          Mỗi người viết sẽ nhận tin nhắn nêu quy tắc đã vi phạm. Bỏ chọn mục nào bạn không muốn từ chối.{{
            hiddenCount ? ` ${hiddenCount} mục có đoạn bị ẩn bạn chưa xem.` : ''
          }}
        </p>
        <fieldset class="bkl">
          <legend class="lb">Các mục đã chọn</legend>
          <label v-for="c in cases" :key="c.id" class="rd">
            <input type="checkbox" :checked="keep.has(c.id)" @change="toggle(c.id)" />
            <span>{{ shortTitle(c) }}<small>{{ c.item.author.name }} · {{ c.item.channel_name }}</small></span>
          </label>
        </fieldset>
        <fieldset aria-describedby="bk-e">
          <legend class="lb">{{ hate ? 'Có nội dung thù ghét: chọn Quy tắc 1, 2 hoặc 3' : 'Các mục này vi phạm quy tắc nào?' }}</legend>
          <label v-for="r in allowed" :key="r.number" class="rd">
            <input v-model="rule" type="radio" name="br" :value="r.number" @change="err = ''" /><span>Quy tắc {{ r.number }}. {{ r.title }}</span>
          </label>
        </fieldset>
        <div id="bk-e" aria-live="polite"><p v-if="err" class="err">{{ err }}</p></div>
        <p class="lb" style="margin-top: 12px">Tin nhắn mẫu</p>
        <p class="q">
          <template v-if="sample">{{ sample.split('§')[0] }}<i>[kênh của từng mục]</i>{{ sample.split('§')[1] }}</template>
          <template v-else>Chọn một quy tắc để xem tin nhắn.</template>
        </p>
        <div class="de">
          <button type="button" class="btn ghost" @click="close">Hủy</button>
          <button class="btn dz" type="submit" :aria-busy="busy">Gửi và từ chối {{ chosen.length }} mục</button>
        </div>
      </form>
    </template>
  </AppDialog>
</template>
