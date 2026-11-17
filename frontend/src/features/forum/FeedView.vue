<script setup lang="ts">
import { useInfiniteQuery } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import ReadOnlyCard from '@/components/ReadOnlyCard.vue'
import TimeAgo from '@/components/TimeAgo.vue'
import { api, unwrap } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { ALL_FEED } from '@/lib/forum'
import { keys, useChannels } from '@/lib/queries'
import type { Post } from '@/lib/types'
import ChannelBanner from './ChannelBanner.vue'
import PostItem from './PostItem.vue'
import RemoveDialog from './RemoveDialog.vue'
import SidePanel from './SidePanel.vue'

// Bảng tin và trang kênh (prototype home)
const route = useRoute()
const router = useRouter()
const auth = useAuth()
const slug = computed(() => (route.params.slug as string | undefined) ?? '')
const { data: channels } = useChannels()
const channel = computed(() => channels.value?.find((c) => c.slug === slug.value))
const official = computed(() => !!channel.value?.official_only)

const feed = useInfiniteQuery({
  queryKey: keys.feed(slug.value),
  queryFn: ({ pageParam }) =>
    unwrap(api.content.GET('/posts', { params: { query: { channel: slug.value || undefined, cursor: pageParam || undefined } } })),
  initialPageParam: '',
  getNextPageParam: (last) => last.next_cursor,
})
const pinned = computed(() => feed.data.value?.pages[0]?.pinned ?? [])
// Mới nhất trước, như server trả về. Bản đầu không có sắp xếp hay bộ lọc: chúng không có
// trong spec, và chỉ áp được lên những bài đã tải nên dễ gây hiểu nhầm.
const list = computed(() => feed.data.value?.pages.flatMap((p) => p.items) ?? [])

// Thẻ chào mừng cho cư dân, ẩn sau khi bấm "Đã hiểu"
const welcome = ref(false)
try {
  welcome.value = localStorage.getItem('sunshine-welcome') !== 'seen'
} catch {
  /* bỏ qua */
}
function dismissWelcome() {
  welcome.value = false
  try {
    localStorage.setItem('sunshine-welcome', 'seen')
  } catch {
    /* bỏ qua */
  }
}

const removing = ref<Post | null>(null)
const composeLabel = computed(() => (auth.isStaff && official.value ? 'Soạn tin ban quản lý' : 'Đăng bài'))
function compose() {
  if (auth.isStaff && official.value) router.push('/dang-bai/thong-bao')
  else router.push(slug.value && !official.value ? `/dang-bai?kenh=${slug.value}` : '/dang-bai')
}
</script>

<template>
  <div class="ctn">
    <ChannelBanner
      :slug="slug"
      :name="channel?.name ?? ALL_FEED.name"
      :description="channel?.description ?? ALL_FEED.description"
    />
    <div class="cols">
      <div class="col-m">
        <ReadOnlyCard v-if="!slug" />
        <section v-if="!slug && welcome && !auth.isStaff" class="card wel" aria-labelledby="wel-h">
          <h2 id="wel-h">Chào mừng bạn đến với diễn đàn cư dân</h2>
          <p>Đây là nơi cư dân và ban quản lý Sunshine Apartment trao đổi, hỏi han và cập nhật tin tức tòa nhà.</p>
          <ul class="rights">
            <li><Icon name="check" /><span><b>Góp ý và phê bình.</b> Kể cả với ban quản lý, miễn là không công kích cá nhân.</span></li>
            <li><Icon name="check" /><span><b>Biết lý do.</b> Khi nội dung bị xử lý, bạn luôn được cho biết vì sao.</span></li>
            <li><Icon name="check" /><span><b>Yêu cầu xem xét lại.</b> Một người thật sẽ đọc lại và quyết định.</span></li>
          </ul>
          <div class="row-a">
            <RouterLink class="btn ghost" to="/quy-tac">Xem bảy quy tắc</RouterLink>
            <button class="btn pri" @click="dismissWelcome">Đã hiểu</button>
          </div>
        </section>

        <template v-if="!official && pinned.length">
          <h2 class="sr">Thông báo được ghim</h2>
          <ul class="prs">
            <li v-for="p in pinned" :key="p.id">
              <RouterLink class="pr" :to="`/t/${p.id}`">
                <Icon name="pin" /><span class="sr">Đã ghim: </span><span class="pr-t">{{ p.title }}</span
                ><span class="pr-m"><TimeAgo :at="p.created_at" /></span>
              </RouterLink>
            </li>
          </ul>
        </template>

        <p v-if="official" class="chn">
          {{
            auth.isStaff
              ? 'Đăng thông báo bằng nút Soạn tin ban quản lý. Cư dân bình luận trong từng thông báo.'
              : 'Chỉ ban quản lý đăng bài tại đây. Bạn có thể bình luận trong từng thông báo.'
          }}
        </p>
        <div v-else class="hdg"><h2>Bài viết mới nhất</h2></div>

        <template v-if="official && pinned.length">
          <h2 class="sr">Thông báo được ghim</h2>
          <ul class="fd">
            <PostItem v-for="p in pinned" :key="p.id" :post="p" @remove="removing = $event" />
          </ul>
        </template>

        <div v-if="feed.isPending.value" class="skl" aria-busy="true">
          <p class="sr" role="status">Đang tải…</p>
          <div v-for="i in 4" :key="i" class="skr">
            <span class="sk rd" style="width: 40px; height: 40px" />
            <div class="skc">
              <span class="sk" style="width: 34%; height: 12px" /><span class="sk" style="width: 86%; height: 18px" /><span
                class="sk"
                style="width: 62%; height: 12px"
              />
            </div>
          </div>
        </div>
        <LoadError v-else-if="feed.isError.value" title="Không tải được danh sách bài viết" @retry="feed.refetch()" />
        <template v-else-if="list.length">
          <ul class="fd">
            <PostItem v-for="p in list" :key="p.id" :post="p" @remove="removing = $event" />
          </ul>
          <div v-if="feed.hasNextPage.value" class="row-a">
            <button class="btn" :aria-busy="feed.isFetchingNextPage.value" @click="feed.fetchNextPage()">
              <span v-if="feed.isFetchingNextPage.value" class="spin" aria-hidden="true" />Xem thêm bài viết
            </button>
          </div>
        </template>
        <div v-else-if="!pinned.length" class="ep">
          <p>Chưa có bài viết nào. Hãy là người mở lời đầu tiên.</p>
          <div v-if="!auth.readOnlyUntil && !(official && !auth.isStaff)" class="row-a" style="justify-content: center">
            <button class="btn pri" @click="compose">{{ composeLabel }}</button>
          </div>
        </div>
      </div>
      <SidePanel :slug="slug" :compose-label="composeLabel" @compose="compose" />
    </div>
    <RemoveDialog
      v-if="removing"
      type="post"
      :id="removing.id"
      :author="removing.author.name"
      :channel="removing.channel.name"
      @close="removing = null"
      @removed="feed.refetch()"
    />
  </div>
</template>
