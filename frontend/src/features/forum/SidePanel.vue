<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/Icon.vue'
import { useAuth } from '@/lib/auth'
import { CHANNEL_NOTE, RULE_SHORT } from '@/lib/forum'
import { useChannels } from '@/lib/queries'

// Cột phải (prototype panel + act): nút đăng bài, lưu ý của kênh, bảy quy tắc, số khẩn cấp
const props = defineProps<{ slug?: string; composeLabel?: string }>()
const emit = defineEmits<{ compose: [] }>()
const auth = useAuth()
const { data: channels } = useChannels()
const channel = computed(() => channels.value?.find((c) => c.slug === props.slug))
const note = computed(() => (props.slug ? CHANNEL_NOTE[props.slug] : undefined))
</script>

<template>
  <div class="col-r">
    <div v-if="!auth.readOnlyUntil" class="act">
      <button class="btn pri" @click="emit('compose')"><Icon name="plus" />{{ composeLabel ?? 'Đăng bài' }}</button>
    </div>
    <aside class="pn" aria-label="Thông tin thêm">
      <section v-if="channel && note" class="pn-c">
        <h2>Lưu ý ở kênh {{ channel.name }}</h2>
        <p>{{ note[0] }}</p>
        <RouterLink v-if="note[1]" class="u" :to="`/quy-tac/${note[1]}`">Xem Quy tắc {{ note[1] }}</RouterLink>
      </section>
      <section>
        <h2>Bảy quy tắc</h2>
        <ol class="prl">
          <li v-for="(t, n) in RULE_SHORT" :key="n">
            <RouterLink :to="`/quy-tac/${n}`"><span class="rn">{{ n }}</span>{{ t }}</RouterLink>
          </li>
        </ol>
        <RouterLink class="u" to="/quy-tac">Đọc đầy đủ và quyền của bạn</RouterLink>
      </section>
      <section>
        <h2>Trường hợp khẩn cấp</h2>
        <p>Diễn đàn không phải là kênh khẩn cấp. Nếu có nguy hiểm, hãy gọi ngay:</p>
        <div class="tels">
          <a href="tel:113" aria-label="Gọi 113, công an">113</a>
          <a href="tel:114" aria-label="Gọi 114, cứu hỏa">114</a>
          <a href="tel:115" aria-label="Gọi 115, cấp cứu">115</a>
        </div>
      </section>
    </aside>
  </div>
</template>
