<script setup lang="ts">
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppDialog from '@/components/AppDialog.vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import PendingNote from '@/components/PendingNote.vue'
import ReadOnlyCard from '@/components/ReadOnlyCard.vue'
import { api, ApiError, errorText, unwrap } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { PENDING_TEXT } from '@/lib/forum'
import { keys } from '@/lib/queries'
import { toast } from '@/lib/toast'
import type { Comment, Submitted } from '@/lib/types'
import { paragraphs } from './channels'
import CommentForm from './CommentForm.vue'
import CommentNode from './CommentNode.vue'
import ImitationNote from './ImitationNote.vue'
import PostHead from './PostHead.vue'
import RemoveDialog from './RemoveDialog.vue'
import SidePanel from './SidePanel.vue'

// Một chủ đề: bài, ô bình luận, cây bình luận hai cấp (prototype thread)
const route = useRoute()
const router = useRouter()
const auth = useAuth()
const qc = useQueryClient()
const id = String(route.params.id)
const fresh = route.query.moi === '1'

const q = useQuery({ queryKey: keys.thread(id), queryFn: () => unwrap(api.content.GET('/posts/{post_id}', { params: { path: { post_id: id } } })) })
const post = computed(() => q.data.value?.post)
const notFound = computed(() => q.error.value instanceof ApiError && q.error.value.status === 404)
// Tiêu đề tab theo tên bài (router đặt "Diễn đàn" trước khi bài tải xong)
watch(post, (p) => p && (document.title = `${p.title} · Sunshine Apartment Forum`), { immediate: true })

const threads = computed(() => {
  const all = q.data.value?.comments ?? []
  return all.filter((c) => !c.parent_id).map((c) => ({ c, kids: all.filter((k) => k.parent_id === c.id) }))
})

// Bình luận được liên kết tới (thông báo, tìm kiếm): tô và đưa vào giữa màn hình
const highlight = ref(route.hash.startsWith('#c-') ? route.hash.slice(3) : '')
onMounted(async () => {
  await q.suspense()
  await nextTick()
  if (highlight.value) {
    const el = document.getElementById(`c-${highlight.value}`)
    el?.scrollIntoView({ block: 'center' })
    el?.focus({ preventScroll: true })
  }
})

const replyTo = ref<string | null>(null)
const pendingNote = ref<string | null>(null)
function onReply(to: Comment) {
  replyTo.value = to.parent_id ?? to.id
}
function changed(res?: Submitted) {
  if (res?.status === 'pending' && res.comment_id) pendingNote.value = res.comment_id
  else if (res?.status === 'published' && res.comment_id) toast('Đã gửi bình luận.')
  q.refetch()
  qc.invalidateQueries({ queryKey: ['feed'] })
}

const removing = ref<{ type: 'post' | 'comment'; id: string; author: string } | null>(null)
const deleting = ref<'post' | Comment | null>(null)
const delErr = ref('')
async function confirmDelete(close: () => void) {
  delErr.value = ''
  try {
    if (deleting.value === 'post') {
      await api.content.DELETE('/posts/{post_id}', { params: { path: { post_id: id } } })
      close()
      toast('Đã xóa bài viết.')
      qc.invalidateQueries({ queryKey: ['feed'] })
      router.replace(post.value ? `/kenh/${post.value.channel.slug}` : '/')
      return
    }
    if (deleting.value) {
      await api.content.DELETE('/comments/{comment_id}', { params: { path: { comment_id: deleting.value.id } } })
      close()
      toast('Đã xóa bình luận.')
      q.refetch()
    }
  } catch (e) {
    delErr.value = errorText(e)
  }
}

async function togglePin() {
  if (!post.value) return
  try {
    await api.content.PUT('/posts/{post_id}/pin', { params: { path: { post_id: id } }, body: { pinned: !post.value.pinned } })
    toast(post.value.pinned ? 'Đã bỏ ghim bài viết.' : 'Đã ghim bài viết lên đầu kênh và Bảng tin.')
    q.refetch()
    qc.invalidateQueries({ queryKey: ['feed'] })
  } catch (e) {
    toast(errorText(e))
  }
}

function onRemoved() {
  if (removing.value?.type === 'post') router.replace(post.value ? `/kenh/${post.value.channel.slug}` : '/')
  else q.refetch()
}
const statusText: Record<string, string> = {
  rejected: 'Bài viết này đã bị từ chối và không hiển thị công khai. Xem lý do trong Yêu cầu của tôi.',
  removed: 'Bài viết này đã bị gỡ và không hiển thị công khai. Xem lý do trong Yêu cầu của tôi.',
  auto_deleted: 'Bài viết này đã bị tự động xóa. Xem lý do trong Yêu cầu của tôi.',
}
</script>

<template>
  <div class="ctn">
    <RouterLink class="bk" :to="post ? `/kenh/${post.channel.slug}` : '/'"><Icon name="back" />{{ post?.channel.name ?? 'Bảng tin' }}</RouterLink>
    <div class="cols">
      <div class="col-m">
        <div v-if="q.isPending.value" aria-busy="true">
          <p class="sr" role="status">Đang tải…</p>
          <div class="card skx">
            <div class="skr"><span class="sk rd" style="width: 40px; height: 40px" /><div class="skc"><span class="sk" style="width: 34%; height: 12px" /></div></div>
            <span class="sk" style="width: 72%; height: 28px" /><span class="sk" style="width: 100%; height: 14px" /><span class="sk" style="width: 90%; height: 14px" />
          </div>
        </div>
        <section v-else-if="notFound" class="card">
          <h1 id="focus-t" tabindex="-1" class="pt">Không tìm thấy bài viết</h1>
          <p>Bài viết có thể đã bị xóa hoặc đường dẫn không đúng.</p>
          <div class="row-a"><RouterLink class="btn" to="/">Về bảng tin</RouterLink></div>
        </section>
        <LoadError v-else-if="q.isError.value" title="Không tải được bài viết" @retry="q.refetch()" />
        <template v-else-if="post">
          <div v-if="fresh && post.status === 'published'" class="okn" role="status">
            <Icon name="check" /><span>Đã đăng trong kênh {{ post.channel.name }}. Bạn sẽ nhận thông báo khi có người bình luận.</span>
          </div>
          <article class="card post">
            <PostHead :post="post" />
            <ImitationNote v-if="post.imitates_official" />
            <h1 id="focus-t" class="pt" tabindex="-1">{{ post.title }}</h1>
            <div class="pb"><p v-for="(p, i) in paragraphs(post.body)" :key="i">{{ p }}</p></div>
            <PendingNote v-if="post.status === 'pending'" link />
            <div v-else-if="post.status !== 'published'" class="pend"><Icon name="alert" /><span>{{ statusText[post.status] }} <RouterLink class="u" to="/yeu-cau">Xem trong Yêu cầu của tôi</RouterLink></span></div>
            <div v-else class="ac">
              <span class="st st-d"><Icon name="msg" />{{ post.comment_count }} bình luận</span>
              <button v-if="auth.can('content.pin')" class="st" :aria-pressed="post.pinned" @click="togglePin">
                <Icon name="pin" />{{ post.pinned ? 'Bỏ ghim' : 'Ghim' }}
              </button>
              <span v-if="post.mine" class="st-o">
                <RouterLink v-if="!auth.readOnlyUntil" class="st" :to="`/sua/${post.id}`"><Icon name="pen" />Sửa</RouterLink>
                <button class="st" :aria-label="`Xóa bài viết: ${post.title}`" @click="deleting = 'post'"><Icon name="trash" />Xóa</button>
              </span>
              <button
                v-else-if="auth.can('moderation.remove') && !post.official"
                class="st st-r"
                :aria-label="`Gỡ bài: ${post.title}`"
                @click="removing = { type: 'post', id: post.id, author: post.author.name }"
              >
                <Icon name="x" />Gỡ nội dung
              </button>
            </div>
            <div v-if="post.mine && post.status === 'pending'" class="row-a">
              <RouterLink class="btn" :to="`/sua/${post.id}`"><Icon name="pen" />Sửa bài</RouterLink>
              <button class="btn ghost" @click="deleting = 'post'"><Icon name="trash" />Xóa</button>
            </div>
          </article>

          <template v-if="post.status === 'published'">
            <ReadOnlyCard />
            <CommentForm
              v-if="!auth.readOnlyUntil"
              id="reply"
              class="card rep"
              :post-id="post.id"
              :channel-id="post.channel.id"
              label="Viết bình luận"
              submit-label="Gửi bình luận"
              @done="changed"
            />
            <div v-if="pendingNote" class="pend rcp" role="status">
              <Icon name="clock" /><span>{{ PENDING_TEXT }} Bạn sẽ nhận được thông báo khi có quyết định.</span>
              <div class="rcp-a"><button type="button" class="btn ghost" @click="pendingNote = null">Đóng</button></div>
            </div>
            <section class="cs" aria-labelledby="ch">
              <div class="cs-h"><h2 id="ch">{{ post.comment_count }} bình luận</h2></div>
              <ul v-if="threads.length">
                <CommentNode
                  v-for="t in threads"
                  :key="t.c.id"
                  :comment="t.c"
                  :replies="t.kids"
                  :post-id="post.id"
                  :channel-id="post.channel.id"
                  :highlight="highlight"
                  :reply-open="replyTo === t.c.id"
                  @reply="onReply"
                  @close-reply="replyTo = null"
                  @remove="(c) => (removing = { type: 'comment', id: c.id, author: c.author?.name ?? '' })"
                  @delete="(c) => (deleting = c)"
                  @changed="changed"
                />
              </ul>
              <p v-else>Chưa có bình luận nào. Hãy là người đầu tiên chia sẻ.</p>
            </section>
          </template>
        </template>
      </div>
      <SidePanel
        :slug="post?.channel.slug"
        @compose="router.push(post && !post.channel.official_only ? `/dang-bai?kenh=${post.channel.slug}` : '/dang-bai')"
      />
    </div>
    <RemoveDialog
      v-if="removing && post"
      :type="removing.type"
      :id="removing.id"
      :author="removing.author"
      :channel="post.channel.name"
      @close="removing = null"
      @removed="onRemoved"
    />
    <AppDialog
      v-if="deleting"
      :title="deleting === 'post' ? 'Xóa bài viết này?' : 'Xóa bình luận này?'"
      @close="(deleting = null), (delErr = '')"
    >
      <template #default="{ close }">
        <form novalidate @submit.prevent="confirmDelete(close)">
          <p class="rp-n">
            <template v-if="deleting === 'post'"
              >“{{ post?.title }}” sẽ bị gỡ khỏi diễn đàn{{ post?.comment_count ? `, cùng ${post.comment_count} bình luận trong bài` : '' }}.
              Bạn không thể hoàn tác.</template
            >
            <template v-else>Bình luận sẽ bị gỡ khỏi chủ đề. Nếu đã có trả lời, một dòng thông báo sẽ giữ chỗ để các trả lời vẫn có ngữ cảnh.</template>
          </p>
          <p v-if="delErr" class="err" role="alert">{{ delErr }}</p>
          <div class="de">
            <button type="button" class="btn ghost" @click="close">Hủy</button>
            <button class="btn dz" type="submit">{{ deleting === 'post' ? 'Xóa bài viết' : 'Xóa bình luận' }}</button>
          </div>
        </form>
      </template>
    </AppDialog>
  </div>
</template>
