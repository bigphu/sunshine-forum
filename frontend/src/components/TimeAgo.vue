<script setup lang="ts">
import { age, longDate } from '@/lib/format'

defineProps<{ at: string }>()
</script>

<template>
  <time :datetime="at" :title="longDate(at)">{{ age(at) }}</time>
</template>
