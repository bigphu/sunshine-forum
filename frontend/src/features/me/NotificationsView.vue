<script setup lang="ts">
import { useQueryClient } from '@tanstack/vue-query'
import { computed, onMounted } from 'vue'
import Avatar from '@/components/Avatar.vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import PageShell from '@/components/PageShell.vue'
import TimeAgo from '@/components/TimeAgo.vue'
import { api } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { keys, useInbox } from '@/lib/queries'
import type { Notification } from '@/lib/types'

// Hộp thông báo trong ứng dụng (FR-6). Mở trang là đánh dấu đã đọc; mục mới vẫn được tô
// cho tới lần mở sau.
const auth = useAuth()
const qc = useQueryClient()
const inbox = useInbox()
const items = computed(() => inbox.data.value?.items ?? [])
const fresh = computed(() => items.value.filter((n) => !n.read_at))
const older = computed(() => items.value.filter((n) => n.read_at))
const FROM_STAFF = new Set(['moderation', 'appeal', 'read_only', 'announcement'])

onMounted(async () => {
  await inbox.suspense()
  if (inbox.data.value?.unread) {
    await api.notifications.POST('/notifications/read')
    qc.invalidateQueries({ queryKey: keys.inbox, refetchType: 'none' })
  }
})
const who = (n: Notification) => (FROM_STAFF.has(n.kind) ? '' : 'Cư dân')
</script>

<template>
  <PageShell title="Thông báo">
    <p v-if="auth.isStaff" style="margin: 0">Thông báo dành cho ban quản lý. Yêu cầu xem xét lại mới cũng nằm trong hàng chờ.</p>
    <LoadError v-if="inbox.isError.value" title="Không tải được thông báo" @retry="inbox.refetch()" />
    <p v-else-if="inbox.isPending.value" role="status">Đang tải…</p>
    <template v-else>
      <template v-for="[label, list] in ([['Mới', fresh], ['Trước đó', older]] as const)" :key="label">
        <template v-if="list.length">
          <h2 class="nh">{{ label }}</h2>
          <ul class="stack nl">
            <li v-for="n in list" :key="n.id">
              <RouterLink class="nt" :to="n.link">
                <Avatar :name="who(n)" :official="!who(n)" />
                <div><b>{{ n.body }}</b><small><TimeAgo :at="n.created_at" /></small></div>
                <span v-if="!n.read_at" class="new">Mới</span>
              </RouterLink>
            </li>
          </ul>
        </template>
      </template>
      <div v-if="!items.length" class="ep"><p>Chưa có thông báo nào.</p></div>
    </template>
    <div class="row-a">
      <RouterLink v-if="auth.isStaff" class="btn" to="/kiem-duyet"><Icon name="inbox" />Mở hàng chờ</RouterLink>
      <RouterLink v-else class="btn" to="/yeu-cau"><Icon name="inbox" />Yêu cầu của tôi</RouterLink>
    </div>
  </PageShell>
</template>
