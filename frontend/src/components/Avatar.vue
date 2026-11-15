<script setup lang="ts">
import { hue, initial } from '@/lib/format'
import Mark from './Mark.vue'

// Avatar chữ cái đầu, bốn màu theo tên. Ban quản lý dùng dấu Sunshine thay chữ.
const props = defineProps<{ name: string; official?: boolean; small?: boolean; empty?: boolean }>()
</script>

<template>
  <span v-if="empty" class="av s nil" aria-hidden="true" />
  <span v-else-if="official" class="av off" :class="{ s: small }" aria-hidden="true"><Mark :size="small ? 19 : 24" /></span>
  <span v-else class="av" :class="[hue(props.name), { s: small }]" aria-hidden="true">{{ initial(props.name) }}</span>
</template>
