<script setup lang="ts">
import PageShell from '@/components/PageShell.vue'
</script>

<template>
  <PageShell title="Không tìm thấy">
    <section class="card">
      <p>Trang có thể đã bị xóa hoặc đường dẫn không đúng.</p>
      <div class="row-a"><RouterLink class="btn" to="/">Về bảng tin</RouterLink></div>
    </section>
  </PageShell>
</template>
