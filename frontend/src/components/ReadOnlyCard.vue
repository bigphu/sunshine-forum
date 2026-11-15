<script setup lang="ts">
import { useAuth } from '@/lib/auth'
import { weekdayDm } from '@/lib/format'
import Icon from './Icon.vue'

// Thay mọi ô soạn thảo khi tài khoản tạm chỉ đọc (FR-1.3)
const auth = useAuth()
</script>

<template>
  <section v-if="auth.readOnlyUntil" class="card roc" aria-labelledby="roc-h">
    <Icon name="clock" />
    <div>
      <h2 id="roc-h">Tài khoản tạm chỉ được đọc đến {{ weekdayDm(auth.readOnlyUntil) }}</h2>
      <p>
        Vì có {{ auth.standing?.strike_limit }} vi phạm đã được xác nhận trong {{ auth.standing?.strike_window_days }} ngày.
        Trong thời gian này bạn chưa đăng bài, bình luận hay sửa nội dung được, nhưng vẫn đọc và yêu cầu xem xét lại như bình
        thường. Nếu một yêu cầu xem xét lại được chấp nhận, vi phạm đó được gỡ bỏ.
      </p>
      <RouterLink class="u" to="/yeu-cau">Xem các vi phạm</RouterLink>
    </div>
  </section>
</template>
