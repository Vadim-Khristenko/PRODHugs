<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { validateLoginForm, parseBackendError, type FieldError } from '@/lib/validation'
import { useTelegramLogin } from '@/composables/useTelegramLogin'
import { useMatrixLogin } from '@/composables/useMatrixLogin'
import { authApi } from '@/api/client'
import { setAccessToken } from '@/lib/token'
import type { TelegramWidgetUser } from '@/lib/telegram'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Send, Loader2 } from 'lucide-vue-next'
import TelegramLoginWidget from '@/components/TelegramLoginWidget.vue'
import router from '@/router'

const auth = useAuthStore()
const {
  telegramPolling,
  telegramError,
  telegramLoading,
  startTelegramLogin,
  cancelTelegramLogin,
} = useTelegramLogin()
const {
  matrixPolling,
  matrixError,
  matrixLoading,
  matrixCommand,
  matrixBotId,
  startMatrixLogin,
  cancelMatrixLogin,
  copyCommand,
} = useMatrixLogin()
const username = ref('')
const password = ref('')
const serverError = ref('')
const fieldErrors = ref<FieldError[]>([])
const submitted = ref(false)
const widgetError = ref<string | null>(null)

function errorFor(field: string): string | undefined {
  return fieldErrors.value.find((e) => e.field === field)?.message
}

const hasErrors = computed(() => fieldErrors.value.length > 0)

function validate() {
  fieldErrors.value = validateLoginForm(username.value, password.value)
}

async function handleLogin() {
  submitted.value = true
  serverError.value = ''
  validate()
  if (hasErrors.value) return

  try {
    await auth.login(username.value, password.value)
  } catch (e: unknown) {
    const parsed = parseBackendError(e)
    if (parsed.fieldErrors.length > 0) {
      fieldErrors.value = [...fieldErrors.value, ...parsed.fieldErrors]
    }
    if (parsed.generalError) {
      serverError.value = parsed.generalError
    }
    if (!parsed.generalError && parsed.fieldErrors.length === 0) {
      serverError.value = 'Неверное имя пользователя или пароль'
    }
  }
}

async function handleWidgetAuth(user: TelegramWidgetUser) {
  widgetError.value = null
  try {
    const res = await authApi.telegramWidgetLogin(user)
    const { token, user: loggedInUser } = res.data
    setAccessToken(token)
    auth.user = loggedInUser
    localStorage.setItem('user', JSON.stringify(loggedInUser))
    await router.push('/dashboard')
  } catch {
    widgetError.value = 'Не удалось войти через Telegram Widget. Попробуйте снова'
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background p-4">
    <Card class="w-full max-w-sm relative">
      <!-- Telegram polling overlay -->
      <div
        v-if="telegramPolling"
        class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-4 rounded-lg bg-background/95 backdrop-blur-sm"
      >
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
        <p class="text-sm text-center text-muted-foreground px-6">
          Нажмите <strong>Start</strong> в открывшемся боте, чтобы войти
        </p>
        <Button
          variant="ghost"
          size="sm"
          @click="cancelTelegramLogin"
        >
          Отмена
        </Button>
      </div>

      <!-- Matrix polling overlay -->
      <div
        v-if="matrixPolling"
        class="absolute inset-0 z-10 flex flex-col items-center justify-center gap-4 rounded-lg bg-background/95 backdrop-blur-sm px-6"
      >
        <Loader2 class="h-8 w-8 animate-spin text-primary" />
        <div class="w-full space-y-2 text-center">
          <p class="text-sm font-medium">Войти через Matrix</p>
          <p class="text-xs text-muted-foreground">
            Бот: <span class="font-mono text-foreground">{{ matrixBotId }}</span>
          </p>
          <p class="text-xs text-muted-foreground">
            Отправьте эту команду боту в личном чате в Matrix:
          </p>
          <div class="flex items-center gap-2">
            <code class="flex-1 truncate rounded-md border bg-muted px-3 py-2 font-mono text-sm text-left">{{ matrixCommand }}</code>
            <Button
              variant="outline"
              size="sm"
              class="shrink-0 rounded-[21px]"
              @click="copyCommand"
            >
              Копировать
            </Button>
          </div>
        </div>
        <Button
          variant="ghost"
          size="sm"
          @click="cancelMatrixLogin"
        >
          Отмена
        </Button>
      </div>

      <CardHeader class="text-center">
        <img src="/logo.webp" alt="PROD" class="mx-auto mb-2 size-12 rounded-lg object-contain" />
        <CardTitle class="text-xl">Вход</CardTitle>
        <CardDescription>Войди в свой аккаунт PRODнимашек</CardDescription>
      </CardHeader>
      <CardContent>
        <form @submit.prevent="handleLogin" class="grid gap-4">
          <div class="grid gap-2">
            <Label for="username">Имя пользователя</Label>
            <Input
              id="username"
              v-model="username"
              type="text"
              placeholder="username"
              :class="{ 'border-destructive': submitted && errorFor('username') }"
              @input="submitted && validate()"
            />
            <p v-if="submitted && errorFor('username')" class="text-xs text-destructive">
              {{ errorFor('username') }}
            </p>
          </div>
          <div class="grid gap-2">
            <Label for="password">Пароль</Label>
            <Input
              id="password"
              v-model="password"
              type="password"
              placeholder="********"
              :class="{ 'border-destructive': submitted && errorFor('password') }"
              @input="submitted && validate()"
            />
            <p v-if="submitted && errorFor('password')" class="text-xs text-destructive">
              {{ errorFor('password') }}
            </p>
          </div>
          <p v-if="serverError" class="text-sm text-destructive text-center">
            {{ serverError }}
          </p>
          <Button
            type="submit"
            variant="yellow"
            class="w-full rounded-[21px]"
            :disabled="auth.loading"
          >
            {{ auth.loading ? 'Вход...' : 'Войти' }}
          </Button>
        </form>
        <!-- Telegram / Matrix login section -->
        <div class="mt-6 flex flex-col items-center">
          <Separator class="my-2 w-full" />
          <p class="mb-2 text-xs text-muted-foreground">Войти через мессенджер</p>
          <div class="w-full flex flex-col gap-2">
            <Button
              type="button"
              variant="outline"
              class="w-full flex items-center justify-center gap-2"
              :disabled="telegramLoading"
              @click="startTelegramLogin"
            >
              <Send class="w-4 h-4" />
              {{ telegramLoading ? 'Открывается бот...' : 'Войти через Telegram' }}
            </Button>
            <p v-if="telegramError" class="text-sm text-destructive text-center">
              {{ telegramError }}
            </p>
            <Button
              type="button"
              variant="outline"
              class="w-full flex items-center justify-center gap-2"
              :disabled="matrixLoading"
              @click="startMatrixLogin"
            >
              <Send class="w-4 h-4" />
              {{ matrixLoading ? 'Подготовка...' : 'Войти через Matrix' }}
            </Button>
            <p v-if="matrixError" class="text-sm text-destructive text-center">
              {{ matrixError }}
            </p>
          </div>
          <!-- Telegram Login Widget (only rendered when VITE_TELEGRAM_BOT_USERNAME is set) -->
          <div class="mt-3 flex flex-col items-center gap-1">
            <TelegramLoginWidget @auth="handleWidgetAuth" />
            <p v-if="widgetError" class="mt-1 text-sm text-destructive text-center">
              {{ widgetError }}
            </p>
          </div>
        </div>
      </CardContent>
      <CardFooter class="justify-center">
        <p class="text-sm text-muted-foreground">
          Нет аккаунта?
          <RouterLink
            to="/register"
            class="text-foreground underline underline-offset-4 hover:text-primary"
          >
            Зарегистрироваться
          </RouterLink>
        </p>
      </CardFooter>
    </Card>
  </div>
</template>
