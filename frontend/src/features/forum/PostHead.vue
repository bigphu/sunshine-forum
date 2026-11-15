<script setup lang="ts">
import Avatar from '@/components/Avatar.vue'
import Mark from '@/components/Mark.vue'
import TimeAgo from '@/components/TimeAgo.vue'
import type { Post } from '@/lib/types'

// Đầu bài: avatar, tên, dấu "Chính thức" của ban quản lý, kênh, thời gian
defineProps<{ post: Post }>()
</script>

<template>
  <div class="hd">
    <Avatar :name="post.author.name" :official="post.official" />
    <div>
      <b
        >{{ post.author.name }}<span v-if="post.official" class="offp"><Mark :size="16" />Chính thức</span></b
      >
      <small
        ><RouterLink :to="`/kenh/${post.channel.slug}`">{{ post.channel.name }}</RouterLink> • <TimeAgo :at="post.created_at" />{{
          post.edited ? ' • đã sửa' : ''
        }}</small
      >
    </div>
  </div>
</template>
