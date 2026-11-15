<script setup lang="ts">
import { computed } from 'vue'
import { charLen, MAX_CHARS, SHOW_COUNT_FROM } from '@/lib/forum'
import { num } from '@/lib/format'

// Đếm ký tự còn lại khi gần giới hạn 5.000 (FR-2.7); vượt thì báo lỗi, giữ nguyên bản nháp
const props = defineProps<{ value: string; id: string }>()
const n = computed(() => charLen(props.value))
</script>

<template>
  <p v-if="n > MAX_CHARS" :id="id" class="err" role="alert">
    Thừa {{ num(n - MAX_CHARS) }} ký tự so với giới hạn {{ num(MAX_CHARS) }} ký tự.
  </p>
  <p v-else-if="n >= SHOW_COUNT_FROM" :id="id" class="hint">Còn {{ num(MAX_CHARS - n) }} ký tự.</p>
</template>
