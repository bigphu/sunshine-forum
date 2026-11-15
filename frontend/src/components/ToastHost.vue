<script setup lang="ts">
import { current } from '@/lib/toast'
</script>

<template>
  <div class="sr" aria-live="polite">{{ current?.text }}</div>
  <div v-if="current" :key="current.id" class="toast" aria-hidden="true">
    <span>{{ current.text }}</span>
  </div>
</template>
