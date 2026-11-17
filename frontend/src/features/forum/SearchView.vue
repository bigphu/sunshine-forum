<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import PageShell from '@/components/PageShell.vue'
import { api, unwrap } from '@/lib/api/client'
import PostItem from './PostItem.vue'

// Tìm bài viết và thông báo theo tiêu đề và nội dung, gõ không dấu cũng được (FR-2.6)
const route = useRoute()
const router = useRouter()
const q = computed(() => String(route.params.q ?? '').trim())
const input = ref(q.value)
watch(q, (v) => (input.value = v))

const res = useQuery({
  queryKey: computed(() => ['search', q.value]),
  queryFn: () => unwrap(api.content.GET('/search', { params: { query: { q: q.value } } })),
  enabled: computed(() => !!q.value),
})
function go() {
  const v = input.value.trim()
  router.push(v ? `/tim/${encodeURIComponent(v)}` : '/tim')
}
</script>

<template>
  <PageShell :title="q ? `Kết quả cho “${q}”` : 'Tìm trong diễn đàn'">
    <form class="srch srch-p" role="search" @submit.prevent="go">
      <label class="sr" for="q-p">Tìm trong diễn đàn</label><Icon name="search" />
      <input id="q-p" v-model="input" type="search" placeholder="Tìm trong diễn đàn" />
      <button type="submit">Tìm</button>
    </form>
    <p v-if="q && res.isPending.value" role="status">Đang tìm…</p>
    <LoadError v-else-if="res.isError.value" title="Không tìm được lúc này" @retry="res.refetch()" />
    <template v-else-if="res.data.value">
      <p role="status">
        {{ res.data.value.length ? `Tìm thấy ${res.data.value.length} bài viết.` : 'Không tìm thấy bài viết nào. Hãy thử từ khác.' }}
      </p>
      <ul v-if="res.data.value.length" class="fd">
        <PostItem v-for="p in res.data.value" :key="p.id" :post="p" :q="q" />
      </ul>
    </template>
  </PageShell>
</template>
