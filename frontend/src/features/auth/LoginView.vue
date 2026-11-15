<script setup lang="ts">
import { nextTick, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import PasswordField from '@/components/PasswordField.vue'
import { errorText } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { PHONE, PHONE_TEXT } from '@/lib/forum'
import AuthPage from './AuthPage.vue'

const auth = useAuth()
const router = useRouter()
const route = useRoute()
const email = ref('')
const password = ref('')
const errEmail = ref('')
const errPw = ref('')
const errForm = ref('')
const busy = ref(false)

async function submit() {
  errEmail.value = /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(email.value.trim()) ? '' : 'Hãy nhập email hợp lệ.'
  errPw.value = password.value ? '' : 'Hãy nhập mật khẩu.'
  errForm.value = ''
  if (errEmail.value || errPw.value) {
    await nextTick()
    document.getElementById(errEmail.value ? 'li-e' : 'li-p')?.focus()
    return
  }
  busy.value = true
  try {
    await auth.login(email.value.trim(), password.value)
    const next = typeof route.query.next === 'string' && route.query.next.startsWith('/') ? route.query.next : '/'
    router.replace(next)
  } catch (e) {
    errForm.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthPage title="Đăng nhập">
    <p class="au-s">Dành cho cư dân và ban quản lý Sunshine Apartment.</p>
    <form class="dfm" novalidate @submit.prevent="submit">
      <div>
        <label class="lb" for="li-e">Email</label>
        <input
          id="li-e"
          v-model="email"
          class="field"
          type="email"
          autocomplete="username"
          aria-required="true"
          :aria-invalid="errEmail ? 'true' : undefined"
          aria-describedby="li-e-e"
        />
        <div id="li-e-e" aria-live="polite"><p v-if="errEmail" class="err">{{ errEmail }}</p></div>
      </div>
      <PasswordField id="li-p" v-model="password" label="Mật khẩu" autocomplete="current-password" :error="errPw" />
      <div aria-live="polite"><p v-if="errForm" class="err">{{ errForm }}</p></div>
      <button class="btn pri au-b" type="submit" :aria-busy="busy">
        <span v-if="busy" class="spin" aria-hidden="true" />{{ busy ? 'Đang đăng nhập' : 'Đăng nhập' }}
      </button>
    </form>
    <RouterLink class="u au-q" to="/quen-mat-khau">Quên mật khẩu?</RouterLink>
    <div class="au-n">
      <p>
        <b>Chưa có tài khoản?</b> Diễn đàn không có đăng ký công khai. Ban quản lý mời từng cư dân theo danh sách đang thuê.
        Hãy liên hệ văn phòng ban quản lý qua số <a :href="PHONE">{{ PHONE_TEXT }}</a>.
      </p>
    </div>
  </AuthPage>
</template>
