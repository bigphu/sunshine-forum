<script setup lang="ts">
import { MARK } from '@/lib/icons'
import { BANNER_ART } from './channels'

// Banner nền xanh đêm với hai nửa đĩa vàng, xếp khác nhau theo kênh (DESIGN.md)
defineProps<{ slug: string; name: string; description: string }>()
</script>

<template>
  <section class="bn" :data-ch="BANNER_ART[slug] ?? 'all'" aria-labelledby="focus-t">
    <div class="bn-art" aria-hidden="true">
      <svg class="m1" viewBox="17 5 66 90"><path :d="MARK" /></svg>
      <svg class="m2" viewBox="17 5 66 90"><path :d="MARK" /></svg>
    </div>
    <div class="bn-t">
      <h1 id="focus-t" tabindex="-1">{{ name }}</h1>
      <p>{{ description }}</p>
    </div>
  </section>
</template>
