<script setup lang="ts">
import { computed, ref } from 'vue'
import Avatar from '@/components/Avatar.vue'
import Icon from '@/components/Icon.vue'
import Mark from '@/components/Mark.vue'
import PendingNote from '@/components/PendingNote.vue'
import TimeAgo from '@/components/TimeAgo.vue'
import { useAuth } from '@/lib/auth'
import type { Comment, Submitted } from '@/lib/types'
import CommentForm from './CommentForm.vue'

// Một bình luận và các trả lời của nó (hai cấp, prototype node)
const props = defineProps<{
  comment: Comment
  replies?: Comment[]
  postId: string
  channelId: string
  reply?: boolean
  highlight?: string
  replyOpen?: boolean
}>()
const emit = defineEmits<{
  reply: [to: Comment]
  closeReply: []
  remove: [c: Comment]
  delete: [c: Comment]
  changed: [res?: Submitted]
}>()
const auth = useAuth()
const editing = ref(false)
const c = computed(() => props.comment)
const hasKids = computed(() => !!props.replies?.length)
const canRemove = computed(() => auth.can('moderation.remove') && !c.value.mine && !c.value.official && c.value.status === 'published')
const name = computed(() => c.value.author?.name ?? '')
</script>

<template>
  <li :class="{ kid: reply }">
    <article :id="`c-${c.id}`" class="c" :class="{ par: hasKids || replyOpen, hl: highlight === c.id }" tabindex="-1">
      <template v-if="c.placeholder">
        <Avatar name="" empty />
        <div>
          <div class="c-w"><span><TimeAgo :at="c.created_at" /></span></div>
          <p class="gone">
            {{ c.status === 'deleted' ? 'Bình luận này đã được người viết xóa.' : 'Bình luận này đã bị ẩn vì vi phạm quy tắc cộng đồng.' }}
          </p>
        </div>
      </template>
      <template v-else>
        <Avatar :name="name" :official="c.official" small />
        <div>
          <div class="c-w">
            <b>{{ name }}</b>
            <span v-if="c.official" class="offp"><Mark :size="16" />Phản hồi chính thức</span>
            <span><TimeAgo :at="c.created_at" /></span>
            <span v-if="c.edited">đã sửa</span>
          </div>
          <CommentForm
            v-if="editing"
            :id="`ce-${c.id}`"
            class="cef"
            :post-id="postId"
            :channel-id="channelId"
            :comment-id="c.id"
            :initial="c.body"
            label="Sửa bình luận"
            label-hidden
            submit-label="Lưu"
            cancel-label="Hủy"
            hint="Bình luận đã sửa được kiểm tra tự động lại."
            @cancel="editing = false"
            @done="(res) => ((editing = false), emit('changed', res))"
          />
          <template v-else>
            <p>{{ c.body }}</p>
            <PendingNote v-if="c.status === 'pending'" />
            <div v-else class="ca">
              <button
                v-if="!c.mine && !auth.readOnlyUntil && c.status === 'published'"
                class="st"
                :aria-label="`Trả lời ${name}`"
                @click="emit('reply', c)"
              >
                <Icon name="reply" />Trả lời
              </button>
              <button v-if="canRemove" class="st st-r" :aria-label="`Gỡ bình luận của ${name}`" @click="emit('remove', c)">
                <Icon name="x" />Gỡ nội dung
              </button>
              <template v-if="c.mine && c.status === 'published'">
                <button v-if="!auth.readOnlyUntil" class="st" aria-label="Sửa bình luận của bạn" @click="editing = true">
                  <Icon name="pen" />Sửa
                </button>
                <button class="st" aria-label="Xóa bình luận của bạn" @click="emit('delete', c)"><Icon name="trash" />Xóa</button>
              </template>
            </div>
          </template>
        </div>
      </template>
    </article>
    <ul v-if="hasKids || replyOpen" aria-label="Trả lời">
      <CommentNode
        v-for="k in replies"
        :key="k.id"
        :comment="k"
        :post-id="postId"
        :channel-id="channelId"
        :highlight="highlight"
        reply
        @reply="emit('reply', $event)"
        @remove="emit('remove', $event)"
        @delete="emit('delete', $event)"
        @changed="emit('changed', $event)"
      />
      <li v-if="replyOpen" class="kid">
        <div class="c ri">
          <Avatar :name="auth.me?.name ?? ''" :official="auth.isStaff" small />
          <CommentForm
            :id="`ri-${c.id}`"
            :post-id="postId"
            :channel-id="channelId"
            :parent-id="c.id"
            :label="`Trả lời ${name}`"
            submit-label="Gửi trả lời"
            cancel-label="Thu gọn"
            @cancel="emit('closeReply')"
            @done="(res) => (emit('closeReply'), emit('changed', res))"
          />
        </div>
      </li>
    </ul>
  </li>
</template>
