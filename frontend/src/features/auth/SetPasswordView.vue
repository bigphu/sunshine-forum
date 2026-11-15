<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '@/components/Icon.vue'
import PasswordField from '@/components/PasswordField.vue'
import { api, errorText, unwrap } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { PHONE, PHONE_TEXT } from '@/lib/forum'
import AuthPage from './AuthPage.vue'

// Mở từ link trong thư mời hoặc thư đặt lại: /dat-mat-khau#token=… (token ở fragment nên
// không tới log của server)
const router = useRouter()
const auth = useAuth()
const token = new URLSearchParams(location.hash.slice(1)).get('token') ?? ''
const info = ref<{ name: string; email: string; purpose: 'invite' | 'reset' } | null>(null)
const dead = ref(false)
const pw = ref('')
const pw2 = ref('')
const errPw = ref('')
const errPw2 = ref('')
const errForm = ref('')
const busy = ref(false)

const reset = computed(() => info.value?.purpose === 'reset')
const rules = computed(() => [
  { ok: pw.value.length >= 8, text: 'Ít nhất 8 ký tự' },
  { ok: pw.value.length > 0 && pw.value.toLowerCase() !== (info.value?.email ?? '').toLowerCase(), text: 'Khác với email đăng nhập' },
])

onMounted(async () => {
  if (!token) {
    dead.value = true
    return
  }
  try {
    info.value = await unwrap(api.identity.POST('/auth/password/inspect', { body: { token } }))
  } catch {
    dead.value = true
  }
})

async function submit() {
  errPw.value =
    pw.value.length < 8
      ? 'Mật khẩu cần ít nhất 8 ký tự.'
      : pw.value.toLowerCase() === info.value?.email.toLowerCase()
        ? 'Mật khẩu không được trùng với email.'
        : ''
  errPw2.value = !errPw.value && pw2.value !== pw.value ? 'Hai mật khẩu chưa khớp.' : ''
  errForm.value = ''
  if (errPw.value || errPw2.value) return
  busy.value = true
  try {
    auth.set(await unwrap(api.identity.POST('/auth/password/set', { body: { token, password: pw.value } })))
    await auth.loadStanding()
    history.replaceState(null, '', location.pathname)
    router.replace('/')
  } catch (e) {
    errForm.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <AuthPage v-if="dead" title="Liên kết đã hết hạn">
    <p class="au-s">
      Liên kết đặt mật khẩu đã được dùng, đã bị thay bằng liên kết mới, hoặc đã hết hạn (lời mời 3 ngày, đặt lại 1 ngày). Nếu
      bạn đang đặt lại mật khẩu, hãy yêu cầu liên kết mới. Nếu đây là lời mời, hãy liên hệ văn phòng ban quản lý qua số
      <a :href="PHONE">{{ PHONE_TEXT }}</a> để được gửi lời mời mới.
    </p>
    <RouterLink class="btn pri au-b" to="/quen-mat-khau">Gửi liên kết đặt lại mới</RouterLink>
    <RouterLink class="u au-q" to="/dang-nhap">Về trang đăng nhập</RouterLink>
  </AuthPage>
  <AuthPage v-else-if="info" :title="reset ? 'Đặt mật khẩu mới' : 'Đặt mật khẩu'">
    <p class="au-s">
      <template v-if="reset">Chọn mật khẩu mới cho tài khoản của bạn.</template>
      <template v-else
        >Chào <b>{{ info.name }}</b
        >. Ban quản lý đã mời bạn tham gia diễn đàn. Hãy đặt mật khẩu để bắt đầu.</template
      >
    </p>
    <form class="dfm" novalidate @submit.prevent="submit">
      <dl class="fx au-x">
        <dt>Email đăng nhập</dt>
        <dd>{{ info.email }}</dd>
      </dl>
      <PasswordField id="sp-p" v-model="pw" label="Mật khẩu mới" autocomplete="new-password" :error="errPw" describedby="sp-r" />
      <ul id="sp-r" class="pwr">
        <li v-for="r in rules" :key="r.text" :class="{ ok: r.ok }"><Icon name="check" /><span>{{ r.text }}</span></li>
      </ul>
      <PasswordField id="sp-c" v-model="pw2" label="Nhập lại mật khẩu" autocomplete="new-password" :error="errPw2" />
      <div aria-live="polite"><p v-if="errForm" class="err">{{ errForm }}</p></div>
      <button class="btn pri au-b" type="submit" :aria-busy="busy">
        <span v-if="busy" class="spin" aria-hidden="true" />{{ reset ? 'Lưu mật khẩu và đăng nhập' : 'Đặt mật khẩu và vào diễn đàn' }}
      </button>
    </form>
    <p v-if="reset" class="hint">Sau khi đổi, các thiết bị khác phải đăng nhập lại.</p>
    <p v-else class="hint">Email không đúng hoặc bạn không ở Sunshine Apartment? Hãy bỏ qua email mời này và báo văn phòng ban quản lý.</p>
  </AuthPage>
</template>
