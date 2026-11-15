<script setup lang="ts">
import Icon from '@/components/Icon.vue'
import { useScreeningStatus } from '@/lib/queries'

// Khung trang quản trị (prototype admP): quay lại Quản trị, tiêu đề, câu dẫn, báo bộ lọc lỗi
defineProps<{ title: string; lead?: string; home?: boolean }>()
const { data: status } = useScreeningStatus()
</script>

<template>
  <div class="ctn wide">
    <div class="adm">
      <RouterLink v-if="!home" class="bk" to="/quan-tri"><Icon name="back" />Quản trị</RouterLink>
      <header class="pg-h">
        <h1 id="focus-t" tabindex="-1">{{ title }}</h1>
        <p v-if="lead" class="lead">{{ lead }}</p>
      </header>
      <div v-if="status && !status.model_online" class="stack">
        <div class="warn ob">
          <Icon name="alert" />
          <div>
            <span>Bộ lọc AI đang ngừng hoạt động</span>
            <span class="wc">Nội dung mới được xử lý theo chính sách khi bộ lọc lỗi và sẽ tự động kiểm tra lại khi bộ lọc hoạt động trở lại.</span>
          </div>
        </div>
      </div>
      <div class="stack"><slot /></div>
    </div>
  </div>
</template>
