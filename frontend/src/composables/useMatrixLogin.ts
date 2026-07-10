import { ref, onUnmounted } from 'vue'
import { type AxiosError } from 'axios'
import { toast } from 'vue-sonner'
import { authApi } from '@/api/client'
import { setAccessToken } from '@/lib/token'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'

export function useMatrixLogin() {
  const auth = useAuthStore()

  const matrixPolling = ref(false)
  const matrixError = ref<string | null>(null)
  const matrixLoading = ref(false)
  const matrixCommand = ref('')
  const matrixBotId = ref('')

  let pollInterval: ReturnType<typeof setInterval> | null = null
  let pollToken: string | null = null
  let pollAttempts = 0
  const MAX_POLL_ATTEMPTS = 150 // 5 minutes at 2-second intervals

  function stopPolling() {
    if (pollInterval) {
      clearInterval(pollInterval)
      pollInterval = null
    }
    matrixPolling.value = false
    pollToken = null
    pollAttempts = 0
  }

  async function startMatrixLogin() {
    matrixError.value = null
    matrixLoading.value = true

    try {
      const res = await authApi.initMatrixLogin()
      const { command, bot_user_id, poll_token } = res.data

      matrixCommand.value = command
      matrixBotId.value = bot_user_id
      pollToken = poll_token
      pollAttempts = 0
      matrixPolling.value = true

      // Start polling
      pollInterval = setInterval(async () => {
        if (!pollToken) {
          stopPolling()
          return
        }

        pollAttempts++
        if (pollAttempts >= MAX_POLL_ATTEMPTS) {
          stopPolling()
          matrixError.value = 'Время ожидания истекло. Попробуйте снова'
          return
        }

        try {
          const pollRes = await authApi.pollMatrixLogin(pollToken)

          if (pollRes.status === 200) {
            // Login successful
            stopPolling()
            const data = pollRes.data
            auth.token = data.token
            auth.user = data.user
            setAccessToken(data.token)
            localStorage.setItem('user', JSON.stringify(data.user))
            await router.push('/dashboard')
          }
          // 202 = still pending, keep polling
        } catch (err: unknown) {
          const axiosErr = err as AxiosError<{ message?: string }>
          if (axiosErr.response?.status === 403) {
            stopPolling()
            matrixError.value = axiosErr.response?.data?.message || 'Аккаунт заблокирован'
          } else if (axiosErr.response?.status === 404) {
            stopPolling()
            matrixError.value = 'Сессия истекла. Попробуйте снова'
          }
          // Other errors: keep polling (transient network issues)
        }
      }, 2000)
    } catch (err: unknown) {
      const axiosErr = err as AxiosError
      if (axiosErr.response?.status === 503) {
        matrixError.value = 'Вход через Matrix недоступен'
      } else {
        matrixError.value = 'Не удалось начать вход через Matrix'
      }
    } finally {
      matrixLoading.value = false
    }
  }

  function cancelMatrixLogin() {
    stopPolling()
    matrixError.value = null
    matrixCommand.value = ''
    matrixBotId.value = ''
  }

  async function copyCommand() {
    try {
      await navigator.clipboard.writeText(matrixCommand.value)
      toast.success('Команда скопирована')
    } catch {
      toast.error('Не удалось скопировать')
    }
  }

  onUnmounted(() => {
    stopPolling()
  })

  return {
    matrixPolling,
    matrixError,
    matrixLoading,
    matrixCommand,
    matrixBotId,
    startMatrixLogin,
    cancelMatrixLogin,
    copyCommand,
  }
}
