<script setup lang="ts">
import { ref } from 'vue'
import Icon from '@/components/Icon.vue'
import { api, errorText } from '@/lib/api/client'
import AuthPage from './AuthPage.vue'

const email = ref('')
const sent = ref('')
const err = ref('')
const busy = ref(false)

async function submit() {
  err.value = /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email.value.trim()) ? '' : 'Hãy nhập email hợp lệ.'
  if (err.value) return
  busy.value = true
  try {
    await api.identity.POST('/auth/password/forgot', { body: { email: email.value.trim() } })
    sent.value = email.value.trim()
  } catch (e) {
    err.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthPage title="Quên mật khẩu">
    <template v-if="sent">
      <div class="okn" role="status">
        <Icon name="check" />
        <span
          >Nếu {{ sent }} có trong danh sách tài khoản, bạn sẽ nhận được email có liên kết đặt mật khẩu mới trong vài phút.
          Liên kết có hiệu lực trong 1 ngày. Hãy xem cả thư mục thư rác.</span
        >
      </div>
      <RouterLink class="btn au-b" to="/dang-nhap">Về trang đăng nhập</RouterLink>
    </template>
    <template v-else>
      <p class="au-s">Nhập email bạn dùng để đăng nhập. Chúng tôi sẽ gửi liên kết để đặt mật khẩu mới.</p>
      <form class="dfm" novalidate @submit.prevent="submit">
        <div>
          <label class="lb" for="fp-e">Email</label>
          <input
            id="fp-e"
            v-model="email"
            class="field"
            type="email"
            autocomplete="username"
            aria-required="true"
            :aria-invalid="err ? 'true' : undefined"
            aria-describedby="fp-e-e"
          />
          <div id="fp-e-e" aria-live="polite"><p v-if="err" class="err">{{ err }}</p></div>
        </div>
        <button class="btn pri au-b" type="submit" :aria-busy="busy">
          <span v-if="busy" class="spin" aria-hidden="true" />{{ busy ? 'Đang gửi' : 'Gửi liên kết' }}
        </button>
      </form>
      <RouterLink class="u au-q" to="/dang-nhap">Quay lại đăng nhập</RouterLink>
    </template>
  </AuthPage>
</template>
