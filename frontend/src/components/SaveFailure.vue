<script setup lang="ts">
import Icon from './Icon.vue'

// Gửi hoặc lưu thất bại: ở lại trong form, nội dung không bị xoá (DESIGN.md)
const props = defineProps<{ message: string; save?: boolean; offline?: boolean }>()
const emit = defineEmits<{ retry: [] }>()
</script>

<template>
  <div class="sfail" role="alert">
    <Icon name="alert" />
    <div>
      <b>{{ props.offline ? (save ? 'Chưa lưu được vì mất kết nối' : 'Chưa gửi được vì mất kết nối') : save ? 'Chưa lưu được' : 'Chưa gửi được' }}</b>
      <p>{{ message }} Nội dung bạn viết vẫn còn nguyên.</p>
    </div>
    <button type="button" class="btn" @click="emit('retry')">Thử lại</button>
  </div>
</template>
