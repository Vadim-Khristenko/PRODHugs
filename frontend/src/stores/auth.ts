import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { authApi, setForceLogoutHandler } from '@/api/client'
import { accessToken, clearAccessToken, setAccessToken } from '@/lib/token'
import router from '@/router'

export type Gender = 'male' | 'female'

export interface User {
  id: string
  username: string
  role: string
  gender?: Gender | null
  display_name?: string | null
  tag?: string | null
  special_tag?: string | null
  telegram_id?: number | null
  matrix_id?: string | null
  captcha_type: 'none' | 'sudoku' | 'casino'
  captcha_cooldown_until?: string | null
  promoted_until?: string | null
  promotion_message?: string | null
  promotion_bid?: number | null
  vip_remaining_seconds?: number
  vip_cooldown_until?: string | null
  balance?: number
}

export const useAuthStore = defineStore('auth', () => {
  const token = accessToken // shared ref from token.ts — stays in sync after refresh
  const user = ref<User | null>(
    localStorage.getItem('user') ? JSON.parse(localStorage.getItem('user')!) : null,
  )
  const loading = ref(false)
  const error = ref<string | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  async function register(username: string, password: string, gender?: string) {
    loading.value = true
    error.value = null
    try {
      const res = await authApi.register(username, password, gender)
      token.value = res.data.token
      user.value = res.data.user
      setAccessToken(res.data.token)
      localStorage.setItem('user', JSON.stringify(res.data.user))
      await router.push('/dashboard')
    } catch (e: unknown) {
      const err = e as { response?: { data?: { message?: string } } }
      error.value = err.response?.data?.message || 'Ошибка регистрации'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function login(username: string, password: string) {
    loading.value = true
    error.value = null
    try {
      const res = await authApi.login(username, password)
      token.value = res.data.token
      user.value = res.data.user
      setAccessToken(res.data.token)
      localStorage.setItem('user', JSON.stringify(res.data.user))
      await router.push('/dashboard')
    } catch (e: unknown) {
      const err = e as { response?: { data?: { message?: string } } }
      error.value = err.response?.data?.message || 'Неверные данные'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchMe() {
    try {
      const res = await authApi.me()
      user.value = res.data
      localStorage.setItem('user', JSON.stringify(res.data))
    } catch {
      logout()
    }
  }

  async function logout() {
    // Clear the refresh token cookie on the server
    try {
      await authApi.logout()
    } catch {
      // Ignore — we're logging out anyway
    }
    token.value = null
    user.value = null
    clearAccessToken()
    localStorage.removeItem('user')
    router.push('/login')
  }

  // Register handler so the API client's forceLogout() can clear reactive state
  // (avoids circular import: client.ts cannot import this store).
  setForceLogoutHandler(() => {
    token.value = null
    user.value = null
    clearAccessToken()
  })

  return { token, user, loading, error, isAuthenticated, register, login, fetchMe, logout }
})
