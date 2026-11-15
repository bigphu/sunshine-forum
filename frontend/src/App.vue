<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Icon from '@/components/Icon.vue'
import ToastHost from '@/components/ToastHost.vue'

// Mất mạng: dải báo dưới thanh trên cùng; bản nháp vẫn nằm trong ô soạn thảo
const offline = ref(!navigator.onLine)
const set = () => (offline.value = !navigator.onLine)
onMounted(() => {
  addEventListener('online', set)
  addEventListener('offline', set)
})
onBeforeUnmount(() => {
  removeEventListener('online', set)
  removeEventListener('offline', set)
})
</script>

<template>
  <div v-if="offline" class="offb" role="status">
    <Icon name="alert" /><span
      ><b>Mất kết nối mạng.</b> Bạn vẫn đọc được những gì đã tải. Bản nháp được giữ trong ô soạn thảo và gửi được khi có mạng
      lại.</span
    >
  </div>
  <RouterView />
  <ToastHost />
</template>
