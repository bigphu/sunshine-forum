// Đường dẫn tiếng Việt như prototype (prototype dùng #/…, app dùng history API)
import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { setUnauthorizedHandler } from '@/lib/api/client'
import { useAuth } from '@/lib/auth'

declare module 'vue-router' {
  interface RouteMeta {
    public?: boolean
    staff?: boolean
    admin?: boolean
    title?: string
  }
}

const forum = () => import('@/features/forum/FeedView.vue')
const thread = () => import('@/features/forum/ThreadView.vue')
const compose = () => import('@/features/forum/ComposeView.vue')

const routes: RouteRecordRaw[] = [
  { path: '/dang-nhap', component: () => import('@/features/auth/LoginView.vue'), meta: { public: true, title: 'Đăng nhập' } },
  { path: '/quen-mat-khau', component: () => import('@/features/auth/ForgotView.vue'), meta: { public: true, title: 'Quên mật khẩu' } },
  { path: '/dat-mat-khau', component: () => import('@/features/auth/SetPasswordView.vue'), meta: { public: true, title: 'Đặt mật khẩu' } },
  {
    path: '/',
    component: () => import('./AppLayout.vue'),
    children: [
      { path: '', component: forum, meta: { title: 'Bảng tin' } },
      { path: 'kenh/:slug', component: forum },
      { path: 't/:id', component: thread },
      { path: 'dang-bai', component: compose, meta: { title: 'Bài viết mới' } },
      { path: 'dang-bai/thong-bao', component: compose, meta: { title: 'Soạn tin ban quản lý', staff: true } },
      { path: 'sua/:id', component: compose, meta: { title: 'Sửa bài viết' } },
      { path: 'tim/:q?', component: () => import('@/features/forum/SearchView.vue'), meta: { title: 'Tìm kiếm' } },
      { path: 'thong-bao', component: () => import('@/features/me/NotificationsView.vue'), meta: { title: 'Thông báo' } },
      { path: 'yeu-cau', component: () => import('@/features/me/RequestsView.vue'), meta: { title: 'Yêu cầu của tôi' } },
      { path: 'xem-xet-lai/:id', component: () => import('@/features/me/AppealView.vue'), meta: { title: 'Yêu cầu xem xét lại' } },
      { path: 'quy-tac/:n?', component: () => import('@/features/me/RulesView.vue'), meta: { title: 'Quy tắc cộng đồng' } },
      { path: 'cai-dat', component: () => import('@/features/me/SettingsView.vue'), meta: { title: 'Cài đặt' } },
      {
        path: 'kiem-duyet/:case?',
        component: () => import('@/features/moderation/ModerationView.vue'),
        meta: { title: 'Hàng chờ kiểm duyệt', staff: true },
      },
      { path: 'quan-tri', component: () => import('@/features/admin/AdminHome.vue'), meta: { title: 'Quản trị', admin: true } },
      {
        path: 'quan-tri/nguoi-dung/:tab?',
        component: () => import('@/features/admin/UsersView.vue'),
        meta: { title: 'Người dùng', admin: true },
      },
      { path: 'quan-tri/bo-loc', component: () => import('@/features/admin/FilterView.vue'), meta: { title: 'Bộ lọc AI', admin: true } },
      { path: 'quan-tri/tu-khoa', component: () => import('@/features/admin/KeywordsView.vue'), meta: { title: 'Từ khóa chặn', admin: true } },
      { path: 'quan-tri/phan-tich', component: () => import('@/features/admin/AnalyticsView.vue'), meta: { title: 'Phân tích', admin: true } },
      { path: 'quan-tri/nhat-ky', component: () => import('@/features/admin/AuditView.vue'), meta: { title: 'Nhật ký kiểm tra', admin: true } },
      {
        path: 'quan-tri/xuat-du-lieu',
        component: () => import('@/features/admin/ExportView.vue'),
        meta: { title: 'Xuất dữ liệu huấn luyện', admin: true },
      },
      { path: ':pathMatch(.*)*', component: () => import('@/features/me/NotFoundView.vue'), meta: { title: 'Không tìm thấy' } },
    ],
  },
]

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: (to, _from, saved) => saved ?? (to.hash ? { el: to.hash, top: 96 } : { top: 0 }),
})

router.beforeEach(async (to) => {
  const auth = useAuth()
  if (!auth.loaded) await auth.load()
  if (to.meta.public) return to.path === '/dang-nhap' && auth.me ? '/' : true
  if (!auth.me) return { path: '/dang-nhap', query: to.fullPath === '/' ? {} : { next: to.fullPath } }
  if (to.meta.admin && !auth.isAdmin) return '/'
  if (to.meta.staff && !auth.isStaff) return '/'
  return true
})

router.afterEach((to) => {
  document.title = `${to.meta.title ?? 'Diễn đàn'} · Sunshine Apartment Forum`
  // Đưa focus về tiêu đề trang để trình đọc màn hình đọc trang mới (prototype: #focus-t)
  setTimeout(() => document.getElementById('focus-t')?.focus({ preventScroll: true }), 0)
})

// Phiên hết hạn giữa chừng: về trang đăng nhập, quay lại trang cũ sau khi đăng nhập
setUnauthorizedHandler(() => {
  const auth = useAuth()
  auth.set(null)
  const cur = router.currentRoute.value
  if (!cur.meta.public) router.replace({ path: '/dang-nhap', query: { next: cur.fullPath } })
})
