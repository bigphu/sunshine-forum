<script setup lang="ts">
import { computed } from 'vue'
import { useRules } from '@/lib/queries'
import type { DraftWarning } from '@/lib/types'
import Icon from './Icon.vue'

// Thẻ cảnh báo trước khi gửi (DESIGN.md "Pre-send warning"): "Sửa lại" là nút chính, gửi
// tiếp là nút phụ "Vẫn gửi" của form.
const props = defineProps<{ warning: DraftWarning | null }>()
const emit = defineEmits<{ fix: [] }>()
const { data: rules } = useRules()
const rule = computed(() => rules.value?.find((r) => r.number === props.warning?.rule))
</script>

<template>
  <div aria-live="polite">
    <div v-if="warning" class="warn">
      <Icon name="alert" />
      <div>
        <span>{{ warning.message }}</span>
        <div class="w-a"><button type="button" class="btn pri" @click="emit('fix')">Sửa lại</button></div>
        <span class="wc">Sửa lại sẽ không bị tính là vi phạm.</span>
        <details v-if="warning.rule !== 5" class="wd">
          <summary>Nếu vẫn gửi thì sao?</summary>
          <p>Nội dung có thể được đăng ngay hoặc được giữ lại để ban quản lý xem xét.</p>
        </details>
        <details v-if="rule" class="wd">
          <summary>Xem Quy tắc {{ rule.number }}: {{ rule.title }}</summary>
          <p>{{ rule.text }}</p>
        </details>
        <details v-else-if="warning.rule == null" class="wd">
          <summary>Xem Quy tắc 1, 2 và 3</summary>
          <p v-for="r in rules?.slice(0, 3)" :key="r.number"><b>Quy tắc {{ r.number }}.</b> {{ r.text }}</p>
        </details>
      </div>
    </div>
  </div>
</template>
