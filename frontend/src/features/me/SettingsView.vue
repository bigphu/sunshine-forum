<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import Icon from '@/components/Icon.vue'
import PageShell from '@/components/PageShell.vue'
import PasswordField from '@/components/PasswordField.vue'
import { api, ApiError, errorText } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { toast } from '@/lib/toast'

// Cài đặt: thông tin tài khoản (chỉ đọc), đổi mật khẩu, cách nhận tin. Bản đầu chưa có thông
// báo đẩy, nên chỉ nói rõ thông báo trong ứng dụng và email.
const auth = useAuth()
const router = useRouter()
const me = computed(() => auth.me!)
const resident = computed(() => me.value.role === 'resident')
const ROLE: Record<string, string> = { resident: 'Cư dân', manager: 'Quản lý', admin: 'Quản trị viên' }

const open = ref(false)
const cur = ref('')
const next = ref('')
const again = ref('')
const e = ref({ cur: '', next: '', again: '', form: '' })
const busy = ref(false)
const rules = computed(() => [
  { ok: next.value.length >= 8, text: 'Ít nhất 8 ký tự' },
  { ok: next.value.length > 0 && next.value.toLowerCase() !== me.value.email.toLowerCase(), text: 'Khác với email đăng nhập' },
])

function cancel() {
  open.value = false
  cur.value = next.value = again.value = ''
  e.value = { cur: '', next: '', again: '', form: '' }
}
async function save() {
  e.value = {
    cur: cur.value ? '' : 'Hãy nhập mật khẩu hiện tại.',
    next: !rules.value.every((r) => r.ok) ? 'Mật khẩu mới chưa đạt yêu cầu bên dưới.' : '',
    again: again.value !== next.value ? 'Hai mật khẩu chưa khớp.' : '',
    form: '',
  }
  if (e.value.cur || e.value.next || e.value.again) return
  busy.value = true
  try {
    await api.identity.PUT('/me/password', { body: { current_password: cur.value, new_password: next.value } })
    cancel()
    toast('Đã đổi mật khẩu. Các thiết bị khác cần đăng nhập lại.')
  } catch (err) {
    const msg = errorText(err)
    if (err instanceof ApiError && msg.includes('hiện tại')) e.value.cur = msg
    else e.value.form = msg
  } finally {
    busy.value = false
  }
}
async function logout() {
  await auth.logout()
  router.replace('/dang-nhap')
}
</script>

<template>
  <PageShell title="Cài đặt">
    <section class="psec" aria-labelledby="sa-h">
      <h2 id="sa-h">Tài khoản</h2>
      <dl class="fx">
        <dt>Họ và tên</dt>
        <dd>{{ me.name }}</dd>
        <template v-if="me.unit">
          <dt>Căn hộ</dt>
          <dd>{{ me.unit }}</dd>
        </template>
        <dt>Email đăng nhập</dt>
        <dd>{{ me.email }}</dd>
        <dt>Vai trò</dt>
        <dd>{{ ROLE[me.role] }}</dd>
      </dl>
      <p class="hint">
        {{
          resident
            ? 'Muốn sửa, hãy liên hệ văn phòng ban quản lý.'
            : 'Muốn sửa tên hoặc email, hãy nhờ quản trị viên.'
        }}
      </p>
    </section>

    <section class="card" aria-labelledby="sp-h">
      <h2 id="sp-h">Mật khẩu</h2>
      <form v-if="open" class="dfm" novalidate @submit.prevent="save">
        <PasswordField id="cp-o" v-model="cur" label="Mật khẩu hiện tại" autocomplete="current-password" :error="e.cur" />
        <PasswordField id="cp-n" v-model="next" label="Mật khẩu mới" autocomplete="new-password" :error="e.next" describedby="cp-r" />
        <ul id="cp-r" class="pwr">
          <li v-for="r in rules" :key="r.text" :class="{ ok: r.ok }"><Icon name="check" /><span>{{ r.text }}</span></li>
        </ul>
        <PasswordField id="cp-c" v-model="again" label="Nhập lại mật khẩu mới" autocomplete="new-password" :error="e.again" />
        <p v-if="e.form" class="err" role="alert">{{ e.form }}</p>
        <div class="fm-e" style="margin: 0">
          <button type="button" class="btn ghost" @click="cancel">Hủy</button>
          <button class="btn pri" type="submit" :aria-busy="busy"><span v-if="busy" class="spin" aria-hidden="true" />Lưu mật khẩu</button>
        </div>
      </form>
      <template v-else>
        <p>Sau khi đổi, các thiết bị khác phải đăng nhập lại.</p>
        <div class="row-a"><button class="btn" @click="open = true">Đổi mật khẩu</button></div>
      </template>
    </section>

    <section class="psec" aria-labelledby="sn-h">
      <h2 id="sn-h">Cách nhận tin</h2>
      <ul class="rights exl2">
        <li><Icon name="check" /><span>Mọi thông báo có trong mục Thông báo.</span></li>
        <li><Icon name="check" /><span>Lời mời và liên kết đặt lại mật khẩu được gửi đến {{ me.email }}.</span></li>
        <li class="no"><Icon name="x" /><span>Các thông báo khác không gửi qua email.</span></li>
      </ul>
    </section>
    <div class="row-a" style="margin: 0"><button class="btn ghost" @click="logout">Đăng xuất</button></div>
  </PageShell>
</template>
