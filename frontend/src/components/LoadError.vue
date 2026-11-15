<script setup lang="ts">
import Icon from './Icon.vue'

// Lỗi tải một vùng dữ liệu: chỉ thay vùng đó, phần còn lại của trang vẫn dùng được
defineProps<{ title: string }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="lerr" role="alert">
    <Icon name="alert" />
    <div>
      <b>{{ title }}</b>
      <p>Có thể do mạng chập chờn hoặc máy chủ đang bận. Phần còn lại của trang vẫn dùng được.</p>
      <button class="btn" @click="emit('retry')">Thử lại</button>
    </div>
  </div>
</template>
