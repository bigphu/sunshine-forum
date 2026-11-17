<script setup lang="ts">
import { computed } from 'vue'

// Thanh trượt ngưỡng 0–100 (Bộ lọc AI). Dùng <input type="range"> gốc nên phím mũi tên,
// Home/End, PageUp/PageDown và trình đọc màn hình đều chạy sẵn; giá trị hiện ngay cạnh nhãn.
const props = defineProps<{ id: string; label: string; invalid?: boolean; hint?: string }>()
const model = defineModel<string | number>({ required: true })
const value = computed(() => Number(model.value) || 0)
</script>

<template>
  <div class="ths" :class="{ bad: invalid }">
    <div class="ths-h">
      <label class="lb" :for="id">{{ label }}</label>
      <output :for="id" class="ths-v" aria-hidden="true">{{ value }}<span>/100</span></output>
    </div>
    <input
      :id="id"
      v-model="model"
      type="range"
      min="0"
      max="100"
      step="1"
      :style="{ '--p': value + '%' }"
      :aria-valuetext="`${value} trên 100`"
      :aria-invalid="invalid ? 'true' : undefined"
      :aria-describedby="hint ? `${id}-h` : undefined"
    />
    <p v-if="hint" :id="`${id}-h`" class="hint">{{ props.hint }}</p>
  </div>
</template>
