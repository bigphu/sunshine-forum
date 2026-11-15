<script setup lang="ts">
import { useQueryClient } from '@tanstack/vue-query'
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppDialog from '@/components/AppDialog.vue'
import Icon from '@/components/Icon.vue'
import LoadError from '@/components/LoadError.vue'
import { api, ApiError, errorText, unwrap } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'
import { dm, fold } from '@/lib/format'
import { toast } from '@/lib/toast'
import type { Account, Role } from '@/lib/types'
import AdminPage from './AdminPage.vue'
import { adminKeys, useAccounts, useReadOnly } from './queries'

// Người dùng (FR-1, UC-02/03/04): mời cư dân từ danh sách đang thuê, nhập danh sách, ngừng
// truy cập, gửi lại lời mời, đổi vai trò nhân viên.
const route = useRoute()
const auth = useAuth()
const qc = useQueryClient()
const accounts = useAccounts()
const readOnly = useReadOnly()
const roles = computed(() => route.params.tab === 'vai-tro')
const ROLE: Record<Role, string> = { resident: 'Cư dân', manager: 'Quản lý', admin: 'Quản trị viên' }

type St = 'all' | 'on' | 'inv' | 'ro' | 'off'
const q = ref('')
const st = ref<St>('all')
const roUntil = (a: Account) => readOnly.data.value?.find((r) => r.account_id === a.id)?.ends_at
const stateOf = (a: Account): St =>
  a.status === 'invited' ? 'inv' : a.status === 'active' ? (roUntil(a) ? 'ro' : 'on') : 'off'
const residents = computed(() => (accounts.data.value ?? []).filter((a) => a.role === 'resident' && a.status !== 'purged'))
const staff = computed(() => (accounts.data.value ?? []).filter((a) => a.role !== 'resident' && a.status !== 'purged'))
const shown = computed(() => {
  const f = fold(q.value.trim())
  return residents.value.filter((a) => (st.value === 'all' || stateOf(a) === st.value) && (!f || fold(`${a.name} ${a.unit} ${a.email}`).includes(f)))
})
const expired = (a: Account) => !!a.invite_expires_at && new Date(a.invite_expires_at) < new Date()
const purgeOn = (a: Account) => dm(new Date(new Date(a.deactivated_at ?? Date.now()).getTime() + 90 * 864e5))
const refresh = () => qc.invalidateQueries({ queryKey: adminKeys.accounts })

// Mời
const inviting = ref<'resident' | 'manager' | null>(null)
const form = ref({ name: '', unit: '', email: '' })
const fe = ref({ name: '', unit: '', email: '', form: '' })
const busy = ref(false)
function openInvite(kind: 'resident' | 'manager') {
  form.value = { name: '', unit: '', email: '' }
  fe.value = { name: '', unit: '', email: '', form: '' }
  inviting.value = kind
}
async function invite(close: () => void) {
  const f = form.value
  fe.value = {
    name: f.name.trim() ? '' : 'Hãy nhập họ và tên.',
    unit: inviting.value === 'resident' && !f.unit.trim() ? 'Hãy nhập căn hộ.' : '',
    email: /^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(f.email.trim()) ? '' : 'Email chưa đúng định dạng.',
    form: '',
  }
  if (fe.value.name || fe.value.unit || fe.value.email) return
  busy.value = true
  try {
    await unwrap(
      api.identity.POST('/accounts', {
        body: { name: f.name.trim(), email: f.email.trim(), role: inviting.value!, ...(f.unit.trim() ? { unit: f.unit.trim() } : {}) },
      }),
    )
    close()
    toast(`Đã gửi lời mời đến ${f.email.trim()}.`)
    refresh()
  } catch (e) {
    const field = e instanceof ApiError ? e.fields?.find((x) => ['email', 'name', 'unit'].includes(x.field)) : undefined
    if (field) fe.value[field.field as 'email'] = field.detail
    else fe.value.form = errorText(e)
  } finally {
    busy.value = false
  }
}

async function reinvite(a: Account) {
  try {
    await unwrap(api.identity.POST('/accounts/{account_id}/reinvite', { params: { path: { account_id: a.id } } }))
    toast(`Đã gửi lời mời mới cho ${a.name}. Liên kết cũ đã hết hiệu lực.`)
    refresh()
  } catch (e) {
    toast(errorText(e))
  }
}

const deact = ref<Account | null>(null)
const deErr = ref('')
async function deactivate(close: () => void) {
  if (!deact.value) return
  deErr.value = ''
  try {
    await unwrap(api.identity.POST('/accounts/{account_id}/deactivate', { params: { path: { account_id: deact.value.id } } }))
    toast(`Đã ngừng truy cập của ${deact.value.name}.`)
    close()
    refresh()
  } catch (e) {
    deErr.value = errorText(e)
  }
}

const roleFor = ref<Account | null>(null)
const newRole = ref<Role>('manager')
const roleErr = ref('')
function openRole(a: Account) {
  roleFor.value = a
  newRole.value = a.role
  roleErr.value = ''
}
async function saveRole(close: () => void) {
  if (!roleFor.value) return
  try {
    await unwrap(api.identity.PUT('/accounts/{account_id}/role', { params: { path: { account_id: roleFor.value.id } }, body: { role: newRole.value } }))
    toast(`Đã đổi vai trò của ${roleFor.value.name} thành ${ROLE[newRole.value]}.`)
    close()
    refresh()
  } catch (e) {
    roleErr.value = errorText(e)
  }
}

// Nhập danh sách cư dân từ tệp CSV: họ tên, căn hộ, email (dòng tiêu đề được bỏ qua)
const importing = ref(false)
const people = ref<{ name: string; unit: string; email: string }[]>([])
const fileErr = ref('')
const result = ref<{ invited: number; skipped: { row: number; email: string; reason: string }[] } | null>(null)
function openImport() {
  people.value = []
  fileErr.value = ''
  result.value = null
  importing.value = true
}
async function readFile(e: Event) {
  fileErr.value = ''
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const rows = (await file.text())
    .split(/\r?\n/)
    .map((l) => l.split(/[,;\t]/).map((x) => x.trim().replace(/^"|"$/g, '')))
    .filter((r) => r.some(Boolean))
  const body = rows.filter((r) => r.some((x) => x.includes('@')))
  people.value = body.map((r) => {
    const email = r.find((x) => x.includes('@')) ?? ''
    const rest = r.filter((x) => x !== email)
    return { name: rest[0] ?? '', unit: rest[1] ?? '', email }
  })
  if (!people.value.length) fileErr.value = 'Không tìm thấy dòng nào có email. Tệp cần các cột: họ tên, căn hộ, email.'
}
async function runImport() {
  if (!people.value.length) return
  busy.value = true
  try {
    result.value = await unwrap(api.identity.POST('/accounts/import', { body: { people: people.value } }))
    refresh()
  } catch (e) {
    fileErr.value = errorText(e)
  } finally {
    busy.value = false
  }
}
const CHIPS: [St, string][] = [
  ['all', 'Tất cả'],
  ['on', 'Đang hoạt động'],
  ['inv', 'Đã mời'],
  ['ro', 'Chỉ đọc'],
  ['off', 'Đã ngừng'],
]
</script>

<template>
  <AdminPage title="Người dùng" lead="Mời cư dân từ danh sách đang thuê, ngừng truy cập khi hết hợp đồng, và giao vai trò cho nhân viên.">
    <nav class="rtab" aria-label="Người dùng">
      <RouterLink to="/quan-tri/nguoi-dung" :aria-current="roles ? undefined : 'page'"><Icon name="users" />Cư dân</RouterLink>
      <RouterLink to="/quan-tri/nguoi-dung/vai-tro" :aria-current="roles ? 'page' : undefined"><Icon name="shield" />Vai trò</RouterLink>
    </nav>
    <LoadError v-if="accounts.isError.value" title="Không tải được danh sách tài khoản" @retry="accounts.refetch()" />
    <template v-else-if="roles">
      <div class="atb">
        <p class="tcount">{{ staff.length }} tài khoản nhân viên</p>
        <button class="btn pri" @click="openInvite('manager')"><Icon name="plus" />Mời nhân viên</button>
      </div>
      <section class="card tc">
        <table class="tbl">
          <caption class="sr">Nhân viên và vai trò</caption>
          <thead><tr><th scope="col">Người dùng</th><th scope="col">Vai trò</th><th scope="col" class="tac"><span class="sr">Thao tác</span></th></tr></thead>
          <tbody>
            <tr v-for="a in staff" :key="a.id">
              <td><b>{{ a.name }}</b><small>{{ a.email }}</small></td>
              <td data-l="Vai trò"><span class="pill" :class="{ mu: a.role !== 'admin' }">{{ ROLE[a.role] }}</span></td>
              <td class="tac">
                <span v-if="a.id === auth.me?.id" class="ac0">Tài khoản của bạn</span>
                <span v-else-if="a.status === 'deactivated'" class="ac0">Không còn truy cập</span>
                <button v-else class="btn" :aria-label="`Đổi vai trò của ${a.name}`" @click="openRole(a)">Đổi vai trò</button>
              </td>
            </tr>
          </tbody>
        </table>
      </section>
      <p class="hint">
        Quản lý duyệt nội dung và đăng thông báo. Quản trị viên làm mọi việc của quản lý, cộng với các trang quản trị. Vai trò mới có hiệu
        lực ngay từ thao tác tiếp theo của người đó.
      </p>
    </template>
    <template v-else>
      <div class="atb">
        <label class="af" for="uq-i"><span class="sr">Tìm tên, căn hộ hoặc email</span><Icon name="search" /><input id="uq-i" v-model="q" class="field" type="search" placeholder="Tìm tên, căn hộ hoặc email" /></label>
        <button class="btn" @click="openImport">Nhập danh sách cư dân</button>
        <button class="btn pri" @click="openInvite('resident')"><Icon name="plus" />Mời cư dân</button>
      </div>
      <div class="pick" role="group" aria-label="Trạng thái">
        <span>Trạng thái</span>
        <button v-for="[v, t] in CHIPS" :key="v" type="button" :aria-pressed="st === v" @click="st = v">{{ t }}</button>
      </div>
      <section class="card tc">
        <table v-if="shown.length" class="tbl">
          <caption class="sr">Tài khoản cư dân</caption>
          <thead>
            <tr><th scope="col">Cư dân</th><th scope="col">Căn hộ</th><th scope="col">Trạng thái</th><th scope="col" class="tac"><span class="sr">Thao tác</span></th></tr>
          </thead>
          <tbody>
            <tr v-for="a in shown" :key="a.id">
              <td><b>{{ a.name }}</b><small>{{ a.email }}</small></td>
              <td data-l="Căn hộ">{{ a.unit }}</td>
              <td data-l="Trạng thái">
                <span v-if="stateOf(a) === 'on'" class="pill ok"><Icon name="check" />Đang hoạt động</span>
                <span v-else-if="stateOf(a) === 'inv' && expired(a)" class="pill mu"><Icon name="x" />Lời mời hết hạn ngày {{ dm(a.invite_expires_at!) }}</span>
                <span v-else-if="stateOf(a) === 'inv'" class="pill"><Icon name="clock" />Đã mời ngày {{ dm(a.created_at) }}, hết hạn {{ a.invite_expires_at ? dm(a.invite_expires_at) : '' }}</span>
                <span v-else-if="stateOf(a) === 'ro'" class="pill"><Icon name="alert" />Chỉ đọc đến {{ dm(roUntil(a)!) }}</span>
                <span v-else class="pill mu"><Icon name="x" />Đã ngừng, xóa ngày {{ purgeOn(a) }}</span>
              </td>
              <td class="tac">
                <button v-if="a.status === 'invited'" class="btn" :aria-label="`Gửi lời mời mới cho ${a.name}`" @click="reinvite(a)">Gửi lời mời mới</button>
                <span v-else-if="a.status === 'deactivated'" class="ac0">Không còn truy cập</span>
                <button v-else class="btn" :aria-label="`Ngừng truy cập của ${a.name}`" @click="(deact = a), (deErr = '')">Ngừng truy cập</button>
              </td>
            </tr>
          </tbody>
        </table>
        <p v-else-if="!accounts.isPending.value" class="tbl-e">Không có tài khoản nào khớp.</p>
      </section>
      <p class="hint">
        Hiển thị {{ shown.length }} trong {{ residents.length }} tài khoản. Mỗi người lớn có một tài khoản và một email. Lời mời gửi liên kết
        đặt mật khẩu, có hiệu lực trong 3 ngày. Gửi lời mời mới sẽ làm liên kết cũ hết hiệu lực.
      </p>
    </template>

    <AppDialog v-if="inviting" :title="inviting === 'manager' ? 'Mời nhân viên' : 'Mời cư dân'" @close="inviting = null">
      <template #default="{ close }">
        <form class="dfm" novalidate @submit.prevent="invite(close)">
          <p class="rp-n">
            {{
              inviting === 'manager'
                ? 'Người này nhận email có liên kết để đặt mật khẩu, với vai trò Quản lý. Liên kết có hiệu lực trong 3 ngày.'
                : 'Cư dân nhận email có liên kết để đặt mật khẩu, có hiệu lực trong 3 ngày. Mỗi người lớn một tài khoản.'
            }}
          </p>
          <div>
            <label class="lb" for="iv-n">Họ và tên</label>
            <input id="iv-n" v-model="form.name" class="field" aria-required="true" :aria-invalid="fe.name ? 'true' : undefined" aria-describedby="iv-n-e" />
            <div id="iv-n-e" aria-live="polite"><p v-if="fe.name" class="err">{{ fe.name }}</p></div>
          </div>
          <div v-if="inviting === 'resident'">
            <label class="lb" for="iv-a">Căn hộ</label>
            <input id="iv-a" v-model="form.unit" class="field" aria-required="true" :aria-invalid="fe.unit ? 'true' : undefined" aria-describedby="iv-a-e iv-a-h" />
            <p id="iv-a-h" class="hint">Ví dụ: B-1204. Căn hộ phải có trong danh sách đang thuê.</p>
            <div id="iv-a-e" aria-live="polite"><p v-if="fe.unit" class="err">{{ fe.unit }}</p></div>
          </div>
          <div>
            <label class="lb" for="iv-e">Email</label>
            <input id="iv-e" v-model="form.email" class="field" type="email" aria-required="true" :aria-invalid="fe.email ? 'true' : undefined" aria-describedby="iv-e-e" />
            <div id="iv-e-e" aria-live="polite"><p v-if="fe.email" class="err">{{ fe.email }}</p></div>
          </div>
          <p v-if="fe.form" class="err" role="alert">{{ fe.form }}</p>
          <div class="de">
            <button type="button" class="btn ghost" @click="close">Hủy</button>
            <button class="btn pri" type="submit" :aria-busy="busy">Gửi lời mời</button>
          </div>
        </form>
      </template>
    </AppDialog>

    <AppDialog v-if="deact" :title="`Ngừng truy cập của ${deact.name}?`" @close="deact = null">
      <template #default="{ close }">
        <form novalidate @submit.prevent="deactivate(close)">
          <p class="rp-n">
            {{ deact.unit ? `Căn hộ ${deact.unit}. ` : '' }}Người này bị đăng xuất ngay và không đăng nhập lại được. Tài khoản và thông tin cá
            nhân bị xóa vĩnh viễn sau 90 ngày, vào ngày {{ purgeOn({ ...deact, deactivated_at: new Date().toISOString() }) }}.
          </p>
          <p v-if="deErr" class="err" role="alert">{{ deErr }}</p>
          <div class="de">
            <button type="button" class="btn ghost" @click="close">Hủy</button>
            <button class="btn dz" type="submit">Ngừng truy cập</button>
          </div>
        </form>
      </template>
    </AppDialog>

    <AppDialog v-if="roleFor" :title="`Vai trò của ${roleFor.name}`" @close="roleFor = null">
      <template #default="{ close }">
        <form novalidate @submit.prevent="saveRole(close)">
          <fieldset class="rul">
            <legend class="sr">Vai trò</legend>
            <label class="rd"><input v-model="newRole" type="radio" name="sro" value="manager" /><span>Quản lý<small>Duyệt nội dung, trả lời chính thức và đăng thông báo.</small></span></label>
            <label class="rd"><input v-model="newRole" type="radio" name="sro" value="resident" /><span>Cư dân<small>Gỡ quyền quản lý. Người này dùng diễn đàn như cư dân.</small></span></label>
          </fieldset>
          <p class="hint">Có hiệu lực ngay từ thao tác tiếp theo của người này.</p>
          <p v-if="roleErr" class="err" role="alert">{{ roleErr }}</p>
          <div class="de">
            <button type="button" class="btn ghost" @click="close">Hủy</button>
            <button class="btn pri" type="submit">Lưu vai trò</button>
          </div>
        </form>
      </template>
    </AppDialog>

    <AppDialog v-if="importing" title="Nhập danh sách cư dân" @close="importing = false">
      <template #default="{ close }">
        <div v-if="result" class="dfm">
          <p class="rp-n">Đã gửi lời mời cho {{ result.invited }} cư dân.</p>
          <template v-if="result.skipped.length">
            <p class="lb">Bỏ qua {{ result.skipped.length }} dòng</p>
            <ul class="rights exl2">
              <li v-for="s in result.skipped" :key="s.row" class="no"><Icon name="x" /><span>Dòng {{ s.row }} ({{ s.email }}): {{ s.reason }}</span></li>
            </ul>
          </template>
          <div class="de"><button type="button" class="btn pri" @click="close">Xong</button></div>
        </div>
        <form v-else class="dfm" novalidate @submit.prevent="runImport">
          <p class="rp-n">
            Chọn tệp CSV xuất từ danh sách cư dân đang thuê, mỗi dòng một người lớn: họ tên, căn hộ, email. Mỗi người nhận một lời mời. Email
            đã có tài khoản được bỏ qua.
          </p>
          <div>
            <label class="lb" for="imp-f">Tệp CSV</label>
            <input id="imp-f" class="field" type="file" accept=".csv,text/csv" aria-describedby="imp-e" @change="readFile" />
            <div id="imp-e" aria-live="polite"><p v-if="fileErr" class="err">{{ fileErr }}</p></div>
          </div>
          <p v-if="people.length" class="hint">Đọc được {{ people.length }} người, ví dụ: {{ people[0].name }} · {{ people[0].unit }} · {{ people[0].email }}.</p>
          <div class="de">
            <button type="button" class="btn ghost" @click="close">Hủy</button>
            <button class="btn pri" type="submit" :disabled="!people.length" :aria-busy="busy">Gửi {{ people.length || '' }} lời mời</button>
          </div>
        </form>
      </template>
    </AppDialog>
  </AdminPage>
</template>
