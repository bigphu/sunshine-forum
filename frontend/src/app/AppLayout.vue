<script setup lang="ts">
import { PopoverContent, PopoverPortal, PopoverRoot, PopoverTrigger } from 'reka-ui'
import { useQueryClient } from '@tanstack/vue-query'
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppDialog from '@/components/AppDialog.vue'
import Icon from '@/components/Icon.vue'
import { useAuth } from '@/lib/auth'
import { CHANNEL_ICON } from '@/lib/forum'
import type { IconName } from '@/lib/icons'
import { useChannels, useInbox, useNotificationStream, useQueue } from '@/lib/queries'

// Khung của mọi trang sau khi đăng nhập (prototype: top_, sideNav, mob)
const auth = useAuth()
const route = useRoute()
const router = useRouter()
const qc = useQueryClient()
const { data: channels } = useChannels()
const { data: inbox } = useInbox()
const { data: queue } = useQueue()
useNotificationStream()

const unread = computed(() => inbox.value?.unread ?? 0)
const queueCount = computed(() => (queue.value ? queue.value.items.length + queue.value.appeals.length : 0))

const ADMIN_NAV: [string, string, IconName][] = [
  ['nguoi-dung', 'Người dùng', 'users'],
  ['bo-loc', 'Bộ lọc AI', 'sliders'],
  ['tu-khoa', 'Từ khóa chặn', 'filter'],
  ['phan-tich', 'Phân tích', 'chart'],
  ['nhat-ky', 'Nhật ký kiểm tra', 'list'],
  ['xuat-du-lieu', 'Xuất dữ liệu', 'download'],
]

// Thanh bên thu gọn được, nhớ theo thiết bị
const nav = ref(true)
try {
  nav.value = localStorage.getItem('sunshine-nav') !== 'off'
} catch {
  /* bộ nhớ trình duyệt bị chặn: giữ mặc định */
}
watch(nav, (v) => {
  try {
    localStorage.setItem('sunshine-nav', v ? 'on' : 'off')
  } catch {
    /* bỏ qua */
  }
})

const q = ref('')
function search() {
  if (q.value.trim()) router.push(`/tim/${encodeURIComponent(q.value.trim())}`)
}

const current = (path: string) => (route.path === path ? 'page' : undefined)
const inChannel = computed(() => route.path.startsWith('/kenh/') || route.path.startsWith('/t/'))
const sheet = ref(false)
const menu = ref(false)
watch(
  () => route.fullPath,
  () => {
    menu.value = false
  },
)

function compose() {
  const slug = route.path.startsWith('/kenh/') ? String(route.params.slug) : ''
  router.push(slug ? `/dang-bai?kenh=${slug}` : '/dang-bai')
}

async function logout() {
  await auth.logout()
  qc.clear()
  router.push('/dang-nhap')
}

const roleLabel = computed(() => (auth.me?.role === 'admin' ? 'Quản trị viên' : 'Ban quản lý'))
</script>

<template>
  <a class="skip" href="#mc">Đến nội dung chính</a>
  <header class="tb">
    <RouterLink class="logo" to="/" aria-label="Sunshine Apartment Forum, trang chủ">
      <img src="/brand/sunshine-lockup-horizontal-on-light.svg" alt="" />
    </RouterLink>
    <form class="srch" role="search" @submit.prevent="search">
      <label class="sr" for="q-t">Tìm trong diễn đàn</label>
      <Icon name="search" />
      <input id="q-t" v-model="q" type="search" placeholder="Tìm trong diễn đàn" />
      <button type="submit">Tìm</button>
    </form>
    <div class="tb-r">
      <button v-if="!auth.readOnlyUntil" class="btn pri tb-p" @click="compose">Đăng bài</button>
      <RouterLink class="ib tb-s" to="/tim" aria-label="Tìm kiếm"><Icon name="search" /></RouterLink>
      <RouterLink class="ib tb-b" to="/thong-bao" :aria-label="`Thông báo${unread ? `, ${unread} mới` : ''}`">
        <Icon name="bell" /><span v-if="unread" class="dot" aria-hidden="true">{{ unread }}</span>
      </RouterLink>
      <div class="amw">
        <PopoverRoot v-model:open="menu">
          <PopoverTrigger
            class="ib me"
            :class="{ mg: auth.isStaff }"
            :aria-label="`Tài khoản của ${auth.me?.name}${auth.isStaff ? `, ${roleLabel.toLowerCase()}` : ''}`"
          >
            {{ (auth.me?.name ?? '?').replace(/\(.*?\)/g, '').trim().split(/\s+/).pop()?.[0] }}
          </PopoverTrigger>
          <PopoverPortal>
            <PopoverContent class="fmenu am" role="group" aria-label="Tài khoản" align="end" :side-offset="6">
              <p>
                <b>{{ auth.me?.name }}</b
                ><small>{{ auth.me?.unit ? `Căn hộ ${auth.me.unit}` : roleLabel }}</small>
              </p>
              <RouterLink to="/cai-dat"><Icon name="sliders" />Cài đặt</RouterLink>
              <RouterLink v-if="!auth.isStaff" to="/yeu-cau"><Icon name="inbox" />Yêu cầu của tôi</RouterLink>
              <button @click="logout"><Icon name="back" />Đăng xuất</button>
            </PopoverContent>
          </PopoverPortal>
        </PopoverRoot>
      </div>
    </div>
  </header>

  <div class="shell" :class="{ 'nav-off': !nav }">
    <div class="sbw">
      <nav class="sb" aria-label="Điều hướng chính">
        <template v-if="auth.isStaff">
          <RouterLink class="sb-h" to="/kiem-duyet" :aria-current="route.path.startsWith('/kiem-duyet') ? 'page' : undefined">
            <Icon name="inbox" /><span class="sb-x">Kiểm duyệt</span>
            <template v-if="queueCount"
              ><span class="cnt" aria-hidden="true">{{ queueCount }}</span
              ><span class="sr">, {{ queueCount }} trường hợp</span></template
            >
          </RouterLink>
          <RouterLink class="sb-h" to="/dang-bai/thong-bao" :aria-current="current('/dang-bai/thong-bao')">
            <Icon name="mega" /><span class="sb-x">Soạn tin<span class="sr"> ban quản lý</span></span>
          </RouterLink>
        </template>
        <RouterLink class="sb-h" to="/" :aria-current="current('/')"><Icon name="home" />Bảng tin</RouterLink>
        <div class="sb-g">
          <RouterLink
            v-for="c in channels"
            :key="c.id"
            class="sb-i"
            :to="`/kenh/${c.slug}`"
            :aria-current="current(`/kenh/${c.slug}`)"
          >
            <Icon :name="CHANNEL_ICON[c.slug] ?? 'msg'" /><span class="sb-x" :title="c.name">{{ c.name }}</span>
          </RouterLink>
        </div>
        <div class="sb-h"><Icon name="shield" />Minh bạch</div>
        <div class="sb-g">
          <RouterLink class="sb-i" to="/quy-tac" :aria-current="route.path.startsWith('/quy-tac') ? 'page' : undefined">
            <Icon name="book" /><span class="sb-x">Bảy quy tắc</span>
          </RouterLink>
          <RouterLink
            v-if="!auth.isStaff"
            class="sb-i"
            to="/yeu-cau"
            :aria-current="route.path.startsWith('/yeu-cau') || route.path.startsWith('/xem-xet-lai') ? 'page' : undefined"
          >
            <Icon name="inbox" /><span class="sb-x">Yêu cầu của tôi</span>
          </RouterLink>
        </div>
        <template v-if="auth.isAdmin">
          <div class="sb-h"><Icon name="sliders" />Quản trị</div>
          <div class="sb-g">
            <RouterLink
              v-for="[id, t, i] in ADMIN_NAV"
              :key="id"
              class="sb-i"
              :to="`/quan-tri/${id}`"
              :aria-current="route.path.startsWith(`/quan-tri/${id}`) ? 'page' : undefined"
            >
              <Icon :name="i" /><span class="sb-x" :title="t">{{ t }}</span>
            </RouterLink>
          </div>
        </template>
      </nav>
      <button class="sb-t" :aria-expanded="nav" :aria-label="nav ? 'Thu gọn thanh bên' : 'Mở thanh bên'" @click="nav = !nav">
        <Icon :name="nav ? 'chevl' : 'chev'" />
      </button>
    </div>
    <main id="mc" tabindex="-1">
      <!-- Hàng chờ giữ nguyên khi đổi trường hợp (danh sách đã xử lý, cuộn); trang khác dựng lại theo đường dẫn -->
      <RouterView :key="route.path.startsWith('/kiem-duyet') ? '/kiem-duyet' : route.path" />
    </main>
  </div>

  <nav class="mb" aria-label="Điều hướng trên điện thoại">
    <RouterLink to="/" :aria-current="current('/')"><Icon name="home" />Bảng tin</RouterLink>
    <button :aria-current="inChannel ? 'true' : undefined" @click="sheet = true"><Icon name="list" />Kênh</button>
    <button v-if="!auth.isAdmin && !auth.readOnlyUntil" :aria-current="route.path.startsWith('/dang-bai') ? 'page' : undefined" @click="compose">
      <Icon name="plus" />Đăng bài
    </button>
    <RouterLink v-if="auth.isStaff" to="/kiem-duyet" :aria-current="route.path.startsWith('/kiem-duyet') ? 'page' : undefined">
      <Icon name="inbox" />Kiểm duyệt
      <template v-if="queueCount"
        ><span class="dot" aria-hidden="true">{{ queueCount }}</span><span class="sr">, {{ queueCount }} trường hợp</span></template
      >
    </RouterLink>
    <RouterLink v-if="auth.isAdmin" to="/quan-tri" :aria-current="route.path.startsWith('/quan-tri') ? 'page' : undefined">
      <Icon name="sliders" />Quản trị
    </RouterLink>
    <RouterLink to="/thong-bao" :aria-current="current('/thong-bao')">
      <Icon name="bell" />Thông báo<span v-if="unread" class="dot" aria-hidden="true">{{ unread }}</span>
    </RouterLink>
    <RouterLink v-if="!auth.isStaff" to="/quy-tac" :aria-current="route.path.startsWith('/quy-tac') ? 'page' : undefined">
      <Icon name="book" />Quy tắc
    </RouterLink>
  </nav>

  <AppDialog v-if="sheet" title="Chọn kênh" @close="sheet = false">
    <template #default="{ close }">
      <div class="dc" @click="close">
        <RouterLink to="/" :aria-current="current('/')"><Icon name="home" />Bảng tin</RouterLink>
        <RouterLink v-for="c in channels" :key="c.id" :to="`/kenh/${c.slug}`" :aria-current="current(`/kenh/${c.slug}`)">
          <Icon :name="CHANNEL_ICON[c.slug] ?? 'msg'" />{{ c.name }}
        </RouterLink>
        <RouterLink v-if="auth.isStaff" to="/quy-tac"><Icon name="book" />Quy tắc</RouterLink>
        <RouterLink to="/cai-dat" :aria-current="current('/cai-dat')"><Icon name="sliders" />Cài đặt</RouterLink>
      </div>
      <div class="de"><button type="button" class="btn ghost" @click="close">Đóng</button></div>
    </template>
  </AppDialog>
</template>
