<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/Icon.vue'
import Mark from '@/components/Mark.vue'
import { useAuth } from '@/lib/auth'
import { fold } from '@/lib/format'
import type { Post } from '@/lib/types'
import { excerpt } from './channels'
import ImitationNote from './ImitationNote.vue'
import PostHead from './PostHead.vue'

// Một bài trong bảng tin (prototype item). q: tô từ khoá khi ở trang tìm kiếm.
const props = defineProps<{ post: Post; q?: string }>()
const emit = defineEmits<{ remove: [post: Post] }>()
const auth = useAuth()

// Tô phần khớp, so khớp không dấu như backend
function marks(text: string): { t: string; hit: boolean }[] {
  const q = fold((props.q ?? '').trim())
  if (!q) return [{ t: text, hit: false }]
  const chars = [...text]
  const folded = chars.map((c) => (fold(c) || c)[0]).join('')
  const out: { t: string; hit: boolean }[] = []
  let i = 0
  let j: number
  while ((j = folded.indexOf(q, i)) > -1) {
    out.push({ t: chars.slice(i, j).join(''), hit: false }, { t: chars.slice(j, j + q.length).join(''), hit: true })
    i = j + q.length
  }
  out.push({ t: chars.slice(i).join(''), hit: false })
  return out
}
const title = computed(() => marks(props.post.title))
const body = computed(() => marks(excerpt(props.post.body)))
const canRemove = computed(() => auth.can('moderation.remove') && !props.post.mine && !props.post.official)
</script>

<template>
  <li>
    <article class="it">
      <p v-if="post.pinned" class="ctx"><Icon name="pin" />Đã ghim</p>
      <PostHead :post="post" />
      <ImitationNote v-if="post.imitates_official" />
      <h3 class="tt">
        <RouterLink :to="`/t/${post.id}`"
          ><template v-for="(m, i) in title" :key="i"><mark v-if="m.hit" class="hit">{{ m.t }}</mark
            ><template v-else>{{ m.t }}</template></template
          ></RouterLink
        >
      </h3>
      <p class="ex">
        <template v-for="(m, i) in body" :key="i"><mark v-if="m.hit" class="hit">{{ m.t }}</mark><template v-else>{{ m.t }}</template></template>
      </p>
      <div class="ac">
        <RouterLink class="st" :to="`/t/${post.id}`"
          ><Icon name="msg" />{{ post.comment_count }} bình luận<span class="sr">: {{ post.title }}</span></RouterLink
        >
        <button v-if="canRemove" class="st st-r" :aria-label="`Gỡ bài: ${post.title}`" @click="emit('remove', post)">
          <Icon name="x" />Gỡ nội dung
        </button>
      </div>
      <p v-if="post.official_replied && !post.official" class="rp"><Mark :size="16" />Ban quản lý đã trả lời</p>
    </article>
  </li>
</template>
