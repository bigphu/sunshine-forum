<script setup lang="ts">
import { nextTick, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import PageShell from '@/components/PageShell.vue'
import { PHONE } from '@/lib/forum'
import { RULE_EXAMPLES } from '@/lib/forum'
import { useRules, useScreeningStatus } from '@/lib/queries'

// Quy tắc cộng đồng (COMMUNITY_RULES.md): quyền của cư dân, bảy quy tắc kèm ví dụ, cách diễn
// đàn hoạt động. /quy-tac/3 mở sẵn và tô quy tắc 3.
const route = useRoute()
const rules = useRules()
useScreeningStatus()
const open = Number(route.params.n ?? 0)

onMounted(async () => {
  if (!open) return
  await rules.suspense()
  await nextTick()
  const el = document.getElementById(`r-${open}`)
  el?.scrollIntoView({ block: 'center' })
  el?.focus({ preventScroll: true })
})

const RIGHTS: [string, string][] = [
  ['Góp ý và phê bình', 'Kể cả với ban quản lý, miễn là bạn tập trung vào vấn đề và không công kích cá nhân.'],
  ['Biết lý do', 'Khi nội dung bị xử lý, bạn luôn được cho biết quy tắc nào đã bị vi phạm.'],
  ['Yêu cầu xem xét lại', 'Một lần, trong vòng 30 ngày. Một người thật sẽ đọc lại và quyết định.'],
  ['Quy tắc công khai', 'Chỉ chủ tòa nhà được thay đổi quy tắc, và quy tắc luôn được công khai tại đây.'],
]
const HOW: [string, string][] = [
  [
    'Khi đăng bài',
    'Hệ thống có thể cảnh báo nếu bản nháp có khả năng vi phạm. Nếu bạn sửa lại, cảnh báo này không bị tính là vi phạm. Mọi bài viết và bình luận, kể cả nội dung đã chỉnh sửa, đều được kiểm tra tự động. Nội dung có thể được đăng ngay hoặc được giữ lại để ban quản lý xem xét.',
  ],
  [
    'Khi nội dung bị xử lý',
    'Ban quản lý có thể gỡ nội dung đã đăng nếu nội dung đó vi phạm quy tắc. Bạn sẽ nhận được thông báo cho biết quy tắc đã vi phạm. Khi vi phạm nhiều lần, tài khoản của bạn có thể tạm thời chỉ được đọc, và bạn sẽ được thông báo về lý do và thời hạn.',
  ],
  [
    'Yêu cầu xem xét lại',
    'Bạn có thể yêu cầu xem xét lại nội dung bị từ chối, bị gỡ hoặc bị tự động xóa một lần, trong vòng 30 ngày, từ thông báo bạn nhận được. Quyết định sau khi xem xét lại là quyết định cuối cùng. Nếu yêu cầu được chấp nhận, lần vi phạm đó sẽ được gỡ bỏ.',
  ],
  [
    'Cải thiện bộ lọc',
    'Nội dung đã được ban quản lý xem xét có thể được dùng tạm thời để cải thiện bộ lọc tự động, sau khi loại bỏ số điện thoại, email và các thông tin tương tự. Bản sao này sẽ bị xóa trong vòng 90 ngày kể từ khi có quyết định.',
  ],
]
</script>

<template>
  <PageShell title="Quy tắc cộng đồng">
    <section class="psec">
      <p>Diễn đàn này dành cho cư dân và ban quản lý Sunshine Apartment. Hãy trao đổi với nhau như khi gặp hàng xóm ở sảnh chung cư.</p>
      <h2 class="rt-h">Quyền của bạn</h2>
      <ul class="rights">
        <li v-for="[b, t] in RIGHTS" :key="b"><Icon name="check" /><span><b>{{ b }}.</b> {{ t }}</span></li>
      </ul>
    </section>
    <LoadError v-if="rules.isError.value" title="Không tải được quy tắc" @retry="rules.refetch()" />
    <section v-else class="psec">
      <h2>Bảy quy tắc</h2>
      <ol class="rl">
        <li v-for="r in rules.data.value" :id="`r-${r.number}`" :key="r.number" tabindex="-1" :class="{ hl: open === r.number }">
          <span class="rn">{{ r.number }}</span>
          <div>
            <b>{{ r.title }}</b>
            <p>{{ r.text }}</p>
            <details v-if="RULE_EXAMPLES[r.number]" class="exd" :open="open === r.number">
              <summary>Xem ví dụ</summary>
              <ul class="exl">
                <li class="ok"><Icon name="check" /><span><b>Được:</b> “{{ RULE_EXAMPLES[r.number][0] }}”</span></li>
                <li class="no"><Icon name="x" /><span><b>Không được:</b> “{{ RULE_EXAMPLES[r.number][1] }}”</span></li>
              </ul>
            </details>
          </div>
        </li>
      </ol>
    </section>
    <section class="psec">
      <h2>Cách diễn đàn hoạt động</h2>
      <details v-for="[t, b] in HOW" :key="t" class="acd">
        <summary><Icon name="chev" /><span>{{ t }}</span></summary>
        <div>{{ b }}</div>
      </details>
    </section>
    <section class="psec">
      <h2>Trường hợp khẩn cấp</h2>
      <p>Diễn đàn không phải là kênh khẩn cấp. Nếu có nguy hiểm, hãy gọi ngay:</p>
      <div class="tels">
        <a class="btn" href="tel:113">113 (công an)</a><a class="btn" href="tel:114">114 (cứu hỏa)</a><a class="btn" href="tel:115">115 (cấp cứu)</a>
      </div>
      <p class="hint">Việc không khẩn cấp, hãy gọi văn phòng ban quản lý: <a class="u" :href="PHONE">028 3812 3456</a>.</p>
    </section>
  </PageShell>
</template>
