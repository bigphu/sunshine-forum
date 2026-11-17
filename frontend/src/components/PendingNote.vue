<script setup lang="ts">
import { PENDING_TEXT } from '@/lib/forum'
import Icon from './Icon.vue'

// Nội dung đang chờ duyệt: chưa hiển thị và chưa phải vi phạm
defineProps<{ link?: boolean; text?: string }>()
</script>

<template>
  <div class="pend">
    <Icon name="clock" />
    <span
      >{{ text ?? PENDING_TEXT }}
      <RouterLink v-if="link" class="u" to="/yeu-cau">Xem trong Yêu cầu của tôi</RouterLink></span
    >
  </div>
</template>
