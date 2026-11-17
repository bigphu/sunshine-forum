<script setup lang="ts">
import Icon from '@/components/Icon.vue'
import { CHANNEL_ICON } from '@/lib/forum'
import { MARK } from '@/lib/icons'

// Chỉ Bảng tin có banner xanh đêm với nửa đĩa vàng. Trang kênh dùng tiêu đề trơn: bốn banner
// cùng một hình xếp khác nhau trông như khuôn mẫu (DESIGN.md "Simplification Pass").
defineProps<{ slug: string; name: string; description: string }>()
</script>

<template>
  <section v-if="!slug" class="bn" data-ch="all" aria-labelledby="focus-t">
    <div class="bn-art" aria-hidden="true">
      <svg class="m1" viewBox="17 5 66 90"><path :d="MARK" /></svg>
      <svg class="m2" viewBox="17 5 66 90"><path :d="MARK" /></svg>
    </div>
    <div class="bn-t">
      <h1 id="focus-t" tabindex="-1">{{ name }}</h1>
      <p>{{ description }}</p>
    </div>
  </section>
  <header v-else class="chh" aria-labelledby="focus-t">
    <span class="chh-i" aria-hidden="true"><Icon :name="CHANNEL_ICON[slug] ?? 'msg'" /></span>
    <div>
      <h1 id="focus-t" tabindex="-1">{{ name }}</h1>
      <p>{{ description }}</p>
    </div>
  </header>
</template>
