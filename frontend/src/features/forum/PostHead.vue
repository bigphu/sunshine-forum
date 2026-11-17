<script setup lang="ts">
import Avatar from '@/components/Avatar.vue'
import Mark from '@/components/Mark.vue'
import TimeAgo from '@/components/TimeAgo.vue'
import { longDate, noticeDate } from '@/lib/format'
import type { Post } from '@/lib/types'

// Đầu bài. Bài của cư dân: avatar, tên, kênh, thời gian tương đối. Thông báo của ban quản lý
// đọc như tờ thông báo dán ở sảnh: ngày đầy đủ đứng đầu, rồi người đăng kèm "Chính thức".
defineProps<{ post: Post }>()
</script>

<template>
  <div v-if="post.official" class="hd hd-n">
    <Avatar :name="post.author.name" official />
    <div>
      <b><time :datetime="post.created_at" :title="longDate(post.created_at)">{{ noticeDate(post.created_at) }}</time></b>
      <small
        >{{ post.author.name }}<span class="offp"><Mark :size="16" />Chính thức</span> •
        <RouterLink :to="`/kenh/${post.channel.slug}`">{{ post.channel.name }}</RouterLink>{{ post.edited ? ' • đã sửa' : '' }}</small
      >
    </div>
  </div>
  <div v-else class="hd">
    <Avatar :name="post.author.name" />
    <div>
      <b>{{ post.author.name }}</b>
      <small
        ><RouterLink :to="`/kenh/${post.channel.slug}`">{{ post.channel.name }}</RouterLink> • <TimeAgo :at="post.created_at" />{{
          post.edited ? ' • đã sửa' : ''
        }}</small
      >
    </div>
  </div>
</template>
