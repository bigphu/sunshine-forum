<script setup lang="ts">
import { ICONS, type IconName } from '@/lib/icons'

defineProps<{ name: IconName }>()
</script>

<template>
  <!-- eslint-disable-next-line vue/no-v-html -- đường vẽ tĩnh trong lib/icons -->
  <svg class="i" viewBox="0 0 24 24" aria-hidden="true" v-html="ICONS[name]" />
</template>
