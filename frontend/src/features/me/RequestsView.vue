<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { computed } from 'vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import PageShell from '@/components/PageShell.vue'
import ReadOnlyCard from '@/components/ReadOnlyCard.vue'
import { api, unwrap } from '@/lib/api/client'
import { dm } from '@/lib/format'
import type { MyDecision } from '@/lib/types'
import { appealStatus, decisionTitle } from './decisions'

// Yêu cầu của tôi: nội dung đang chờ và nội dung bị xử lý, kèm trạng thái xem xét lại (FR-5.5)
const pending = useQuery({ queryKey: ['me', 'pending'], queryFn: () => unwrap(api.content.GET('/me/pending')) })
const decisions = useQuery({ queryKey: ['me', 'decisions'], queryFn: () => unwrap(api.moderation.GET('/me/decisions')) })

const waiting = computed(() => [
  ...(pending.data.value?.posts ?? []).map((p) => ({ id: p.id, t: p.title, m: `Bài viết trong kênh ${p.channel.name}`, to: `/t/${p.id}` })),
  ...(pending.data.value?.comments ?? []).map(({ comment: c, post: p }) => ({
    id: c.id,
    t: c.body,
    m: `${c.parent_id ? 'Trả lời' : 'Bình luận'} trong “${p.title}”`,
    to: `/t/${p.id}#c-${c.id}`,
  })),
])
const handled = computed(() => (decisions.data.value ?? []).filter((d) => d.decision.action !== 'approve'))

function meta(d: MyDecision) {
  let tail = 'Đã hết hạn yêu cầu xem xét lại'
  if (d.appeal) tail = d.appeal.status === 'pending' ? 'Đã gửi yêu cầu xem xét lại' : 'Quyết định cuối cùng'
  else if (d.can_appeal && d.decision.appeal_deadline) tail = `Hạn yêu cầu xem xét lại: ${dm(d.decision.appeal_deadline)}`
  return `Quy tắc ${d.decision.rule} • ${tail}`
}
</script>

<template>
  <PageShell title="Yêu cầu của tôi">
    <ReadOnlyCard />
    <p style="margin: 0">Theo dõi nội dung đang chờ và yêu cầu xem xét lại của bạn. Bạn cũng sẽ nhận thông báo khi có kết quả.</p>
    <section class="card">
      <h2>Đang chờ xem xét <span class="hc">({{ waiting.length }})</span></h2>
      <LoadError v-if="pending.isError.value" title="Không tải được nội dung đang chờ" @retry="pending.refetch()" />
      <ul v-else-if="waiting.length" class="rqs">
        <li v-for="x in waiting" :key="x.id">
          <RouterLink class="rq" :to="x.to">
            <span class="rq-t">{{ x.t }}</span><small>{{ x.m }}</small><span class="pill"><Icon name="clock" />Đang chờ</span>
          </RouterLink>
        </li>
      </ul>
      <p v-else-if="!pending.isPending.value" class="hint">Không có nội dung nào đang chờ xem xét.</p>
    </section>
    <section class="card">
      <h2>Nội dung bị xử lý <span class="hc">({{ handled.length }})</span></h2>
      <LoadError v-if="decisions.isError.value" title="Không tải được quyết định" @retry="decisions.refetch()" />
      <ul v-else-if="handled.length" class="rqs">
        <li v-for="d in handled" :key="d.decision.id">
          <RouterLink class="rq" :to="`/xem-xet-lai/${d.decision.id}`">
            <span class="rq-t">{{ decisionTitle(d) }}</span><small>{{ meta(d) }}</small>
            <span class="pill"><Icon :name="appealStatus(d)[1]" />{{ appealStatus(d)[0] }}</span>
          </RouterLink>
        </li>
      </ul>
      <p v-else-if="!decisions.isPending.value" class="hint">Không có nội dung nào bị xử lý.</p>
    </section>
  </PageShell>
</template>
