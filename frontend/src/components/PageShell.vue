<script setup lang="ts">
import Icon from './Icon.vue'

// Khung trang một cột (prototype page): nút quay lại, tiêu đề nhận focus, các thẻ xếp chồng
defineProps<{ title: string; back?: [string, string] }>()
</script>

<template>
  <div class="ctn">
    <div class="pg">
      <header class="pg-h">
        <RouterLink v-if="back" class="bk" :to="back[0]"><Icon name="back" />{{ back[1] }}</RouterLink>
        <h1 id="focus-t" tabindex="-1">{{ title }}</h1>
      </header>
      <div class="stack"><slot /></div>
    </div>
  </div>
</template>
