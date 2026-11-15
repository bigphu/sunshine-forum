// Store phiên đăng nhập: tài khoản hiện tại, quyền, và CSRF token gửi kèm mọi request ghi.
// Quyền được đọc lại từ server ở mỗi request (phiên lưu trong DB), nên đổi vai trò hay khoá
// tài khoản có hiệu lực ngay; store chỉ dùng để hiển thị.
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { api, ApiError, setCsrfToken, unwrap } from './api/client'
import type { Me, Standing } from './types'

export const useAuth = defineStore('auth', () => {
  const me = ref<Me | null>(null)
  const standing = ref<Standing | null>(null)
  const loaded = ref(false)

  const can = (perm: string) => !!me.value?.permissions.includes(perm)
  const isStaff = computed(() => can('moderation.review'))
  const isAdmin = computed(() => me.value?.role === 'admin')
  const readOnlyUntil = computed(() => {
    const t = standing.value?.read_only_until
    return t && new Date(t) > new Date() ? t : null
  })

  function set(m: Me | null) {
    me.value = m
    setCsrfToken(m?.csrf_token ?? '')
  }

  async function load() {
    try {
      set(await unwrap(api.identity.GET('/me')))
      await loadStanding()
    } catch (e) {
      if (!(e instanceof ApiError) || e.status !== 401) throw e
      set(null)
    } finally {
      loaded.value = true
    }
  }

  async function loadStanding() {
    standing.value = me.value ? await unwrap(api.penalties.GET('/me/standing')) : null
  }

  async function login(email: string, password: string) {
    set(await unwrap(api.identity.POST('/auth/login', { body: { email, password } })))
    await loadStanding()
  }

  async function logout() {
    await api.identity.POST('/auth/logout', { params: {} })
    set(null)
    standing.value = null
  }

  return { me, standing, loaded, can, isStaff, isAdmin, readOnlyUntil, set, load, loadStanding, login, logout }
})
