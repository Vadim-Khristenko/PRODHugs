import axios, { type AxiosError, type InternalAxiosRequestConfig } from 'axios'
import router from '@/router'
import { clearAccessToken, getAccessToken, setAccessToken } from '@/lib/token'
import type { User } from '@/stores/auth'

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true, // send HttpOnly cookies with every request
})

// v2 axios instance — shares cookies, JWT, and the silent-refresh interceptor
// installed below by piggybacking on the same response interceptor.
const apiV2 = axios.create({
  baseURL: '/api/v2',
  headers: {
    'Content-Type': 'application/json',
  },
  withCredentials: true,
})

apiV2.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// ── Request interceptor: attach access token ──
api.interceptors.request.use((config) => {
  const token = getAccessToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// ── Response interceptor: silent refresh on 401 ──
let isRefreshing = false
let pendingQueue: Array<{
  resolve: (token: string) => void
  reject: (err: unknown) => void
}> = []

function processPendingQueue(token: string | null, error: unknown) {
  for (const p of pendingQueue) {
    if (token) {
      p.resolve(token)
    } else {
      p.reject(error)
    }
  }
  pendingQueue = []
}

const AUTH_PATHS = [
  '/auth/login',
  '/auth/register',
  '/auth/refresh',
  '/auth/logout',
  '/auth/telegram/init',
  '/auth/telegram/poll',
  '/auth/telegram/widget',
]

function isAuthRequest(config: InternalAxiosRequestConfig | undefined): boolean {
  if (!config?.url) return false
  return AUTH_PATHS.some((p) => config.url!.endsWith(p))
}

// Optional callback set by the auth store to clear its reactive state on force-logout.
let onForceLogout: (() => void) | null = null

export function setForceLogoutHandler(handler: () => void) {
  onForceLogout = handler
}

function forceLogout() {
  clearAccessToken()
  localStorage.removeItem('user')
  // Clear Pinia auth store state to keep it in sync with localStorage.
  onForceLogout?.()
  router.push('/login')
}

function makeRefreshInterceptor(client: typeof api) {
  return async (error: AxiosError) => {
    const originalRequest = error.config

    if (error.response?.status !== 401 || !originalRequest || isAuthRequest(originalRequest)) {
      return Promise.reject(error)
    }

    if ((originalRequest as InternalAxiosRequestConfig & { _retried?: boolean })._retried) {
      forceLogout()
      return Promise.reject(error)
    }

    if (isRefreshing) {
      return new Promise<string>((resolve, reject) => {
        pendingQueue.push({ resolve, reject })
      }).then((newToken) => {
        originalRequest.headers.Authorization = `Bearer ${newToken}`
        return client(originalRequest)
      })
    }

    isRefreshing = true
    ;(originalRequest as InternalAxiosRequestConfig & { _retried?: boolean })._retried = true

    try {
      // The refresh endpoint lives under /api/v1 — always call it through `api`.
      const res = await api.post('/auth/refresh')
      const newToken: string = res.data.token

      setAccessToken(newToken)
      originalRequest.headers.Authorization = `Bearer ${newToken}`

      processPendingQueue(newToken, null)
      return client(originalRequest)
    } catch (refreshError) {
      processPendingQueue(null, refreshError)
      forceLogout()
      return Promise.reject(refreshError)
    } finally {
      isRefreshing = false
    }
  }
}

api.interceptors.response.use((response) => response, makeRefreshInterceptor(api))
apiV2.interceptors.response.use((response) => response, makeRefreshInterceptor(apiV2))

export default api

// Auth
export const authApi = {
  register: (username: string, password: string, gender?: string) =>
    api.post('/auth/register', { username, password, ...(gender ? { gender } : {}) }),
  login: (username: string, password: string) => api.post('/auth/login', { username, password }),
  logout: () => api.post('/auth/logout'),
  me: () => api.get('/users/me'),
  checkUsername: (username: string) =>
    api.get<{ available: boolean }>('/auth/check-username', { params: { username } }),
  initTelegramLogin: () =>
    api.post<{ bot_url: string; poll_token: string }>('/auth/telegram/init'),
  pollTelegramLogin: (pollToken: string) =>
    api.post<{ token: string; user: User }>(
      '/auth/telegram/poll',
      { poll_token: pollToken },
    ),
  telegramWidgetLogin: (data: {
    id: number
    first_name: string
    last_name?: string
    username?: string
    photo_url?: string
    auth_date: number
    hash: string
  }) => api.post<{ token: string; user: User }>('/auth/telegram/widget', data),
}

// Hugs
export const hugsApi = {
  suggest: (userId: string, hugType?: string, comment?: string, captchaToken?: string) =>
    api.post(
      `/hugs/${userId}`,
      hugType || comment || captchaToken ? { 
        ...(hugType ? { hug_type: hugType } : {}), 
        ...(comment ? { comment } : {}),
        ...(captchaToken ? { captcha_token: captchaToken } : {}) 
      } : undefined,
    ),
  accept: (hugId: string) => api.post(`/hugs/${hugId}/accept`),
  decline: (hugId: string) => api.post(`/hugs/${hugId}/decline`),
  cancel: (hugId: string) => api.post(`/hugs/${hugId}/cancel`),
  getDetail: (hugId: string) => api.get(`/hugs/detail/${hugId}`),
  getInbox: () => api.get('/hugs/inbox'),
  getOutgoing: () => api.get('/hugs/outgoing'),
  getInboxCount: () => api.get('/hugs/inbox/count'),
  getCooldown: (userId: string) => api.get(`/hugs/cooldown/${userId}`),
  upgradeCooldown: (userId: string) => api.post(`/hugs/cooldown/${userId}/upgrade`),
  getHistory: () => api.get('/hugs/history'),
  getFeed: (limit = 50, offset = 0) => api.get('/hugs/feed', { params: { limit, offset } }),
  getActivity: () => api.get('/hugs/activity'),
  buySlot: () => api.post('/hugs/slots/buy'),
}

// Intimacy
export const intimacyApi = {
  getPairIntimacy: (userId: string) => api.get(`/intimacy/${userId}`),
  getConnections: (limit = 20, offset = 0) =>
    api.get('/intimacy/connections', { params: { limit, offset } }),
  getLeaderboard: (limit = 20, offset = 0) =>
    api.get('/intimacy/leaderboard', { params: { limit, offset } }),
}

// Streaks
export const streaksApi = {
  getPairStreak: (userId: string) => api.get(`/streaks/${userId}`),
  getTopStreaks: () => api.get('/streaks/top'),
}

// Balance
export const balanceApi = {
  get: () => api.get('/balance'),
  claimDaily: () => api.post('/daily-reward'),
}

// Users
export const usersApi = {
  search: (q = '', limit = 20, offset = 0) =>
    api.get('/users/search', { params: { q, limit, offset } }),
  getVIPs: () => api.get('/users/vips'),
  getProfile: (userId: string) => api.get(`/users/${userId}/profile`),
  updateSettings: (data: { gender?: string; display_name?: string | null }) =>
    api.put('/users/me/settings', data),
  createTelegramLinkToken: () =>
    api.post<{ token: string; bot_url: string }>('/users/me/telegram/link-token'),
  unlinkTelegram: () => api.delete('/users/me/telegram'),
  createMatrixLinkToken: () =>
    api.post<{ token: string; command: string; bot_user_id: string }>('/users/me/matrix/link-token'),
  unlinkMatrix: () => api.delete<User>('/users/me/matrix'),
  changePassword: (oldPassword: string, newPassword: string) =>
    api.put('/users/me/password', { old_password: oldPassword, new_password: newPassword }),
  blockUser: (userId: string) => api.post(`/users/${userId}/block`),
  unblockUser: (userId: string) => api.delete(`/users/${userId}/block`),
  getBlockedUsers: () => api.get('/users/me/blocked'),
  promote: (bid: number, message?: string) => api.post('/users/promote', { bid, message }),
}

// Leaderboard
export const leaderboardApi = {
  get: (limit = 20, offset = 0) => api.get('/leaderboard', { params: { limit, offset } }),
}

// Announcements
export const announcementsApi = {
  getActive: () => api.get('/announcements/active'),
  dismiss: (id: string) => api.post(`/announcements/${id}/dismiss`),
}

// Admin
export const adminApi = {
  getStats: () => api.get('/admin/stats'),
  getUsers: (limit = 20, offset = 0, q?: string) =>
    api.get('/admin/users', { params: { limit, offset, ...(q ? { q } : {}) } }),
  banUser: (userId: string) => api.put(`/admin/users/${userId}/ban`),
  unbanUser: (userId: string) => api.delete(`/admin/users/${userId}/ban`),
  updateUsername: (userId: string, username: string) =>
    api.put(`/admin/users/${userId}/username`, { username }),
  updateGender: (userId: string, gender: string | null) =>
    api.put(`/admin/users/${userId}/gender`, { gender }),
  updateDisplayName: (userId: string, displayName: string | null) =>
    api.put(`/admin/users/${userId}/display-name`, { display_name: displayName }),
  updateTag: (userId: string, tag: string | null) =>
    api.put(`/admin/users/${userId}/tag`, { tag }),
  updateSpecialTag: (userId: string, specialTag: string | null) =>
    api.put(`/admin/users/${userId}/special-tag`, { special_tag: specialTag }),
  updatePassword: (userId: string, password: string) =>
    api.put(`/admin/users/${userId}/password`, { password }),
  updateBalance: (userId: string, amount: number) =>
    api.put(`/admin/users/${userId}/balance`, { amount }),
  updateCaptchaType: (userId: string, captchaType: string) =>
    api.put(`/admin/users/${userId}/captcha-type`, { captcha_type: captchaType }),
  clearPromotion: (userId: string) => api.delete(`/admin/users/${userId}/promotion`),
  deleteUser: (userId: string) => api.delete(`/admin/users/${userId}`),
  createAnnouncement: (message: string) => api.post('/admin/announcements', { message }),
  deleteAnnouncement: (id: string) => api.delete(`/admin/announcements/${id}`),
}

// ── v2 API (spec at backend/api/openapi-v2.yaml) ──
// v2 endpoints currently power: profile lookup by username, "@username"
// search, and read-only daily-reward status. v1 endpoints stay available as
// legacy.

export const usersApiV2 = {
  // q can include a leading "@" to filter by username substring only.
  search: (q = '', limit = 20, offset = 0) =>
    apiV2.get('/users/search', { params: { q, limit, offset } }),
  // usernameOrId may be a UUID, a bare username, or "@username".
  getProfile: (usernameOrId: string) =>
    apiV2.get(`/users/${encodeURIComponent(usernameOrId)}/profile`),
}

export const balanceApiV2 = {
  getDailyRewardStatus: () =>
    apiV2.get<{
      can_claim: boolean
      next_claim_at: string
      streak_days: number
      last_claimed_at?: string | null
    }>('/daily-reward/status'),
}

export interface UserNoteDTO {
  target_id: string
  target_username?: string
  target_display_name?: string | null
  content: string
  updated_at: string
}

export const notesApiV2 = {
  // usernameOrId may be a UUID, a bare username, or "@username".
  get: (usernameOrId: string) =>
    apiV2.get<UserNoteDTO>(`/users/${encodeURIComponent(usernameOrId)}/note`),
  upsert: (usernameOrId: string, content: string) =>
    apiV2.put<UserNoteDTO>(`/users/${encodeURIComponent(usernameOrId)}/note`, {
      content,
    }),
  remove: (usernameOrId: string) =>
    apiV2.delete<void>(`/users/${encodeURIComponent(usernameOrId)}/note`),
  list: (limit = 50, offset = 0) =>
    apiV2.get<UserNoteDTO[]>('/notes', { params: { limit, offset } }),
}

// Captcha
export const captchaApi = {
  getSudoku: (targetId: string) => api.get<{ id: string; puzzle: number[][] }>('/captcha/sudoku', { params: { target_id: targetId } }),
  verifyCell: (id: string, row: number, col: number, value: number) =>
    api.post<{ correct: boolean; errors: number; failed: boolean }>(`/captcha/sudoku/${id}/verify-cell`, {
      row,
      col,
      value,
    }),
  completeSudoku: (id: string) =>
    api.post<{ captcha_token: string }>(`/captcha/sudoku/${id}/complete`),
  getCasino: () => api.get<{ id: string; expires_at: string }>('/captcha/casino'),
  spinCasino: (id: string) =>
    api.post<{ win: boolean; captcha_token?: string; cooldown_until?: string }>(
      `/captcha/casino/${id}/spin`,
    ),
}

