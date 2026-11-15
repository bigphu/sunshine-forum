<script setup lang="ts">
import { MARK } from '@/lib/icons'

// Dấu hai nửa đĩa của Sunshine: giọng chính thức của ban quản lý
const props = withDefaults(defineProps<{ size?: number }>(), { size: 16 })
const width = Math.round((props.size * 66) / 90)
</script>

<template>
  <svg class="mk" viewBox="17 5 66 90" :width="width" :height="size" aria-hidden="true"><path :d="MARK" /></svg>
</template>
