<script setup lang="ts">
import { ref } from 'vue'

// Ô mật khẩu có nút "Hiện" (aria-pressed), nhãn luôn ở trên ô
defineProps<{ id: string; label: string; autocomplete: string; error?: string; describedby?: string }>()
const model = defineModel<string>({ required: true })
const shown = ref(false)
</script>

<template>
  <div>
    <label class="lb" :for="id">{{ label }}</label>
    <div class="pwf">
      <input
        :id="id"
        v-model="model"
        class="field"
        :type="shown ? 'text' : 'password'"
        :autocomplete="autocomplete"
        aria-required="true"
        :aria-invalid="error ? 'true' : undefined"
        :aria-describedby="`${id}-e${describedby ? ' ' + describedby : ''}`"
      />
      <button type="button" class="btn ghost pwt" :aria-pressed="shown" :aria-label="shown ? 'Ẩn mật khẩu' : 'Hiện mật khẩu'" @click="shown = !shown">
        {{ shown ? 'Ẩn' : 'Hiện' }}
      </button>
    </div>
    <div :id="`${id}-e`" aria-live="polite"><p v-if="error" class="err">{{ error }}</p></div>
  </div>
</template>
