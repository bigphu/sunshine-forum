<script setup lang="ts">
import { onMounted, ref } from 'vue'

// Hộp thoại modal bằng <dialog> gốc (prototype dùng cùng phần tử và CSS): giữ focus bên
// trong, Esc để đóng, focus trở lại nút đã mở khi đóng.
defineProps<{ title: string }>()
const emit = defineEmits<{ close: [] }>()
const el = ref<HTMLDialogElement>()
const opener = document.activeElement as HTMLElement | null

onMounted(() => el.value?.showModal())

function close() {
  el.value?.close()
}

function onClose() {
  emit('close')
  opener?.focus?.({ preventScroll: true })
}
defineExpose({ close })
</script>

<template>
  <dialog ref="el" aria-labelledby="dt" @close="onClose">
    <h2 id="dt">{{ title }}</h2>
    <slot :close="close" />
  </dialog>
</template>
