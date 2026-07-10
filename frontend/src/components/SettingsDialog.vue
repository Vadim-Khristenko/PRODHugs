<script setup lang="ts">
import { ref, watch, onUnmounted } from 'vue'
import { toast } from 'vue-sonner'
import { ShieldX, Send } from 'lucide-vue-next'
import { useAuthStore, type Gender } from '@/stores/auth'
import { useHugsStore, type BlockedUser } from '@/stores/hugs'
import { authApi, usersApi } from '@/api/client'
import { plural } from '@/lib/utils'
import { profileLink } from '@/lib/profileLink'
import { validateChangePasswordForm, parseBackendError, type FieldError } from '@/lib/validation'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import PasswordRequirements from '@/components/PasswordRequirements.vue'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'

const open = defineModel<boolean>('open', { required: true })

const auth = useAuthStore()
const hugsStore = useHugsStore()

// ── Display name + Gender + Tag ──
const displayName = ref(auth.user?.display_name ?? '')
const gender = ref<Gender | ''>((auth.user?.gender as Gender) ?? '')
const tag = ref(auth.user?.tag ?? '')
const savingProfile = ref(false)

// ── Telegram ──
const telegramLinked = ref(auth.user?.telegram_id != null)
const telegramLoading = ref(false)
const telegramPolling = ref(false)
const telegramError = ref('')
let pollTimer: ReturnType<typeof setInterval> | null = null

// ── Matrix ──
const matrixLinked = ref(auth.user?.matrix_id != null)
const matrixLoading = ref(false)
const matrixPolling = ref(false)
const matrixError = ref('')
const matrixCommand = ref('')
const matrixBotId = ref('')
let matrixPollTimer: ReturnType<typeof setInterval> | null = null

// ── Blocked users ──
const blockedUsers = ref<BlockedUser[]>([])
const loadingBlocked = ref(false)
const unblockingId = ref<string | null>(null)

async function fetchBlocked() {
  loadingBlocked.value = true
  try {
    blockedUsers.value = await hugsStore.getBlockedUsers()
  } catch {
    // Ignore
  } finally {
    loadingBlocked.value = false
  }
}

async function unblock(userId: string) {
  if (unblockingId.value) return
  unblockingId.value = userId
  try {
    await hugsStore.unblockUser(userId)
    blockedUsers.value = blockedUsers.value.filter((u) => u.id !== userId)
    toast.success('Пользователь разблокирован')
  } catch {
    toast.error('Не удалось разблокировать')
  } finally {
    unblockingId.value = null
  }
}

watch(open, (isOpen) => {
  if (isOpen) {
    displayName.value = auth.user?.display_name ?? ''
    gender.value = (auth.user?.gender as Gender) ?? ''
    tag.value = auth.user?.tag ?? ''
    telegramLinked.value = auth.user?.telegram_id != null
    telegramError.value = ''
    stopPolling()
    matrixLinked.value = auth.user?.matrix_id != null
    matrixError.value = ''
    stopMatrixPolling()
    resetPasswordForm()
    fetchBlocked()
  }
})

async function saveProfile() {
  savingProfile.value = true
  try {
    const trimmed = displayName.value.trim()
    const trimmedTag = tag.value.trim()
    const payload: { gender?: string; display_name?: string | null; tag?: string | null } = {}
    if (gender.value) payload.gender = gender.value
    payload.display_name = trimmed || null
    payload.tag = trimmedTag || null
    const res = await usersApi.updateSettings(payload)
    auth.user = res.data
    localStorage.setItem('user', JSON.stringify(res.data))
    toast.success('Настройки сохранены')
  } catch (e) {
    const parsed = parseBackendError(e)
    toast.error(parsed.generalError ?? 'Ошибка сохранения')
  } finally {
    savingProfile.value = false
  }
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  telegramPolling.value = false
}

async function linkTelegram() {
  telegramError.value = ''
  telegramLoading.value = true
  try {
    const res = await usersApi.createTelegramLinkToken()
    window.open(res.data.bot_url, '_blank')
    // Start polling for link confirmation
    telegramPolling.value = true
    telegramLoading.value = false
    let attempts = 0
    pollTimer = setInterval(async () => {
      attempts++
      if (attempts > 60) {
        stopPolling()
        telegramError.value = 'Время ожидания истекло. Попробуйте снова.'
        return
      }
      try {
        const me = await authApi.me()
        if (me.data.telegram_id != null) {
          auth.user = me.data
          localStorage.setItem('user', JSON.stringify(me.data))
          telegramLinked.value = true
          stopPolling()
          toast.success('Telegram привязан')
        }
      } catch {
        // ignore polling errors
      }
    }, 2000)
  } catch (e) {
    const parsed = parseBackendError(e)
    telegramError.value = parsed.generalError ?? 'Ошибка'
    telegramLoading.value = false
  }
}

async function unlinkTelegram() {
  telegramError.value = ''
  telegramLoading.value = true
  try {
    const res = await usersApi.unlinkTelegram()
    auth.user = res.data
    localStorage.setItem('user', JSON.stringify(res.data))
    telegramLinked.value = false
    toast.success('Telegram отвязан')
  } catch (e) {
    const parsed = parseBackendError(e)
    telegramError.value = parsed.generalError ?? 'Ошибка'
  } finally {
    telegramLoading.value = false
  }
}

function stopMatrixPolling() {
  if (matrixPollTimer) {
    clearInterval(matrixPollTimer)
    matrixPollTimer = null
  }
  matrixPolling.value = false
}

async function linkMatrix() {
  matrixError.value = ''
  matrixLoading.value = true
  try {
    const res = await usersApi.createMatrixLinkToken()
    matrixCommand.value = res.data.command
    matrixBotId.value = res.data.bot_user_id
    matrixLoading.value = false
    // Start polling for link confirmation
    matrixPolling.value = true
    let attempts = 0
    matrixPollTimer = setInterval(async () => {
      attempts++
      if (attempts > 60) {
        stopMatrixPolling()
        matrixError.value = 'Время ожидания истекло. Попробуйте снова.'
        return
      }
      try {
        const me = await authApi.me()
        if (me.data.matrix_id != null) {
          auth.user = me.data
          localStorage.setItem('user', JSON.stringify(me.data))
          matrixLinked.value = true
          stopMatrixPolling()
          matrixCommand.value = ''
          matrixBotId.value = ''
          toast.success('Matrix привязан')
        }
      } catch {
        // ignore polling errors
      }
    }, 2000)
  } catch (e: unknown) {
    const err = e as { response?: { status?: number; data?: { message?: string } } }
    if (err.response?.status === 503) {
      matrixError.value = 'Matrix-привязка недоступна на сервере'
    } else {
      const parsed = parseBackendError(e)
      matrixError.value = parsed.generalError ?? 'Ошибка'
    }
    matrixLoading.value = false
  }
}

async function unlinkMatrix() {
  matrixError.value = ''
  matrixLoading.value = true
  try {
    const res = await usersApi.unlinkMatrix()
    auth.user = res.data
    localStorage.setItem('user', JSON.stringify(res.data))
    matrixLinked.value = false
    toast.success('Matrix отвязан')
  } catch (e) {
    const parsed = parseBackendError(e)
    matrixError.value = parsed.generalError ?? 'Ошибка'
  } finally {
    matrixLoading.value = false
  }
}

async function copyMatrixCommand() {
  try {
    await navigator.clipboard.writeText(matrixCommand.value)
    toast.success('Команда скопирована')
  } catch {
    toast.error('Не удалось скопировать')
  }
}

onUnmounted(() => {
  stopPolling()
  stopMatrixPolling()
})

// ── Password ──
const oldPassword = ref('')
const newPassword = ref('')
const newPasswordConfirm = ref('')
const passwordErrors = ref<FieldError[]>([])
const passwordServerError = ref('')
const savingPassword = ref(false)
const passwordSubmitted = ref(false)

function resetPasswordForm() {
  oldPassword.value = ''
  newPassword.value = ''
  newPasswordConfirm.value = ''
  passwordErrors.value = []
  passwordServerError.value = ''
  passwordSubmitted.value = false
}

function passwordErrorFor(field: string): string | undefined {
  return passwordErrors.value.find((e) => e.field === field)?.message
}

function validatePasswordForm() {
  passwordErrors.value = validateChangePasswordForm(
    oldPassword.value,
    newPassword.value,
    newPasswordConfirm.value,
  )
}

async function savePassword() {
  passwordSubmitted.value = true
  passwordServerError.value = ''
  validatePasswordForm()
  if (passwordErrors.value.length > 0) return

  savingPassword.value = true
  try {
    await usersApi.changePassword(oldPassword.value, newPassword.value)
    toast.success('Пароль изменён')
    resetPasswordForm()
  } catch (e) {
    const parsed = parseBackendError(e)
    if (parsed.fieldErrors.length > 0) {
      passwordErrors.value = [...passwordErrors.value, ...parsed.fieldErrors]
    }
    if (parsed.generalError) {
      passwordServerError.value = parsed.generalError
    }
  } finally {
    savingPassword.value = false
  }
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="sm:max-w-md max-h-[calc(100dvh-2rem)] flex flex-col overflow-hidden">
      <DialogHeader>
        <DialogTitle>Настройки</DialogTitle>
        <DialogDescription>Управление профилем и безопасностью</DialogDescription>
      </DialogHeader>

      <div class="-mx-4 flex-1 space-y-6 overflow-y-auto overscroll-contain px-4 pb-1">
        <!-- Profile section -->
        <div class="space-y-3">
          <Label class="text-sm font-medium">Профиль</Label>
          <div class="grid gap-1.5">
            <Label for="settings-display-name" class="text-xs text-muted-foreground"
              >Отображаемое имя</Label
            >
            <Input
              id="settings-display-name"
              v-model="displayName"
              maxlength="32"
              placeholder="Как тебя называть"
            />
            <p class="text-[11px] text-muted-foreground">
              Оставь пустым, чтобы использовать имя пользователя.
            </p>
          </div>
          <div class="grid gap-1.5">
            <Label class="text-xs text-muted-foreground">Пол</Label>
            <RadioGroup v-model="gender" class="flex gap-4">
              <div class="flex items-center gap-2">
                <RadioGroupItem id="gender-male" value="male" />
                <Label for="gender-male" class="font-normal cursor-pointer">Мужской</Label>
              </div>
              <div class="flex items-center gap-2">
                <RadioGroupItem id="gender-female" value="female" />
                <Label for="gender-female" class="font-normal cursor-pointer">Женский</Label>
              </div>
            </RadioGroup>
          </div>
          <div class="grid gap-1.5">
            <Label for="settings-tag" class="text-xs text-muted-foreground">Тег</Label>
            <Input
              id="settings-tag"
              v-model="tag"
              maxlength="20"
              placeholder="Мой тег"
            />
            <p class="text-[11px] text-muted-foreground">
              Будет виден в рейтинге и на странице пользователей. Смена тега стоит {{ plural(5, 'обниманя', 'обнимани', 'обнимань') }}.
            </p>
          </div>
          <Button
            variant="yellow"
            size="sm"
            class="rounded-[21px]"
            :disabled="savingProfile"
            @click="saveProfile"
          >
            {{ savingProfile ? 'Сохранение...' : 'Сохранить' }}
          </Button>
        </div>

        <Separator />

        <!-- Telegram section -->
        <div class="space-y-3">
          <Label class="text-sm font-medium">Telegram уведомления</Label>

          <!-- Linked state -->
          <div v-if="telegramLinked" class="space-y-2">
            <div
              class="flex items-center gap-2 rounded-md border border-green-800/40 bg-green-950/30 px-3 py-2 text-sm"
            >
              <Send class="size-4 text-green-400" />
              <span>Telegram привязан</span>
            </div>
            <Button
              variant="outline"
              size="sm"
              class="rounded-[21px]"
              :disabled="telegramLoading"
              @click="unlinkTelegram"
            >
              {{ telegramLoading ? 'Отвязка...' : 'Отвязать Telegram' }}
            </Button>
          </div>

          <!-- Not linked — polling state -->
          <div v-else-if="telegramPolling" class="space-y-2">
            <p class="text-sm text-muted-foreground">
              Нажмите <strong>Start</strong> в открывшемся боте...
            </p>
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <span class="inline-block size-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
              Ожидание привязки
            </div>
            <button
              type="button"
              class="text-xs text-muted-foreground underline underline-offset-2 hover:text-foreground"
              @click="stopPolling(); telegramError = ''"
            >
              Отмена
            </button>
          </div>

          <!-- Not linked — idle state -->
          <div v-else class="space-y-2">
            <p class="text-[11px] text-muted-foreground">
              Привяжите Telegram, чтобы получать уведомления об объятиях.
            </p>
            <Button
              variant="yellow"
              size="sm"
              class="rounded-[21px]"
              :disabled="telegramLoading"
              @click="linkTelegram"
            >
              {{ telegramLoading ? 'Загрузка...' : 'Привязать Telegram' }}
            </Button>
          </div>

          <p v-if="telegramError" class="text-xs text-destructive">
            {{ telegramError }}
          </p>
        </div>

        <Separator />

        <!-- Matrix section -->
        <div class="space-y-3">
          <Label class="text-sm font-medium">Matrix уведомления</Label>

          <!-- Linked state -->
          <div v-if="matrixLinked" class="space-y-2">
            <div
              class="flex items-center gap-2 rounded-md border border-green-800/40 bg-green-950/30 px-3 py-2 text-sm"
            >
              <Send class="size-4 text-green-400" />
              <span>Matrix привязан</span>
            </div>
            <Button
              variant="outline"
              size="sm"
              class="rounded-[21px]"
              :disabled="matrixLoading"
              @click="unlinkMatrix"
            >
              {{ matrixLoading ? 'Отвязка...' : 'Отвязать Matrix' }}
            </Button>
          </div>

          <!-- Polling / command shown state -->
          <div v-else-if="matrixPolling" class="space-y-2">
            <p class="text-[11px] text-muted-foreground">
              Откройте личный чат с ботом в Matrix и отправьте эту команду.
            </p>
            <p class="text-xs text-muted-foreground">
              Бот:
              <span class="font-mono text-foreground">{{ matrixBotId }}</span>
            </p>
            <div class="flex items-center gap-2">
              <code
                class="flex-1 truncate rounded-md border bg-muted px-3 py-2 font-mono text-sm"
              >{{ matrixCommand }}</code>
              <Button
                variant="outline"
                size="sm"
                class="shrink-0 rounded-[21px]"
                @click="copyMatrixCommand"
              >
                Копировать
              </Button>
            </div>
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <span class="inline-block size-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
              Ожидание привязки
            </div>
            <button
              type="button"
              class="text-xs text-muted-foreground underline underline-offset-2 hover:text-foreground"
              @click="stopMatrixPolling(); matrixError = ''; matrixCommand = ''; matrixBotId = ''"
            >
              Отмена
            </button>
          </div>

          <!-- Idle state -->
          <div v-else class="space-y-2">
            <p class="text-[11px] text-muted-foreground">
              Привяжите Matrix, чтобы получать уведомления об объятиях.
            </p>
            <Button
              variant="yellow"
              size="sm"
              class="rounded-[21px]"
              :disabled="matrixLoading"
              @click="linkMatrix"
            >
              {{ matrixLoading ? 'Загрузка...' : 'Привязать Matrix' }}
            </Button>
          </div>

          <p v-if="matrixError" class="text-xs text-destructive">
            {{ matrixError }}
          </p>
        </div>

        <Separator />

        <!-- Password section -->
        <div class="space-y-3">
          <Label class="text-sm font-medium">Смена пароля</Label>
          <div class="grid gap-3">
            <div class="grid gap-1.5">
              <Label for="old-password" class="text-xs text-muted-foreground">Текущий пароль</Label>
              <Input
                id="old-password"
                v-model="oldPassword"
                type="password"
                placeholder="********"
                :class="{
                  'border-destructive': passwordSubmitted && passwordErrorFor('oldPassword'),
                }"
                @input="passwordSubmitted && validatePasswordForm()"
              />
              <p
                v-if="passwordSubmitted && passwordErrorFor('oldPassword')"
                class="text-xs text-destructive"
              >
                {{ passwordErrorFor('oldPassword') }}
              </p>
            </div>
            <div class="grid gap-1.5">
              <Label for="new-password" class="text-xs text-muted-foreground">Новый пароль</Label>
              <Input
                id="new-password"
                v-model="newPassword"
                type="password"
                placeholder="********"
                :class="{
                  'border-destructive': passwordSubmitted && passwordErrorFor('newPassword'),
                }"
                @input="passwordSubmitted && validatePasswordForm()"
              />
              <p
                v-if="passwordSubmitted && passwordErrorFor('newPassword')"
                class="text-xs text-destructive"
              >
                {{ passwordErrorFor('newPassword') }}
              </p>
              <PasswordRequirements :password="newPassword" />
            </div>
            <div class="grid gap-1.5">
              <Label for="new-password-confirm" class="text-xs text-muted-foreground"
                >Подтвердите новый пароль</Label
              >
              <Input
                id="new-password-confirm"
                v-model="newPasswordConfirm"
                type="password"
                placeholder="********"
                :class="{
                  'border-destructive': passwordSubmitted && passwordErrorFor('newPasswordConfirm'),
                }"
                @input="passwordSubmitted && validatePasswordForm()"
              />
              <p
                v-if="passwordSubmitted && passwordErrorFor('newPasswordConfirm')"
                class="text-xs text-destructive"
              >
                {{ passwordErrorFor('newPasswordConfirm') }}
              </p>
            </div>
          </div>
          <p v-if="passwordServerError" class="text-sm text-destructive">
            {{ passwordServerError }}
          </p>
          <Button
            variant="yellow"
            size="sm"
            class="rounded-[21px]"
            :disabled="savingPassword"
            @click="savePassword"
          >
            {{ savingPassword ? 'Сохранение...' : 'Сменить пароль' }}
          </Button>
        </div>

        <Separator />

        <!-- Blocked users section -->
        <div class="space-y-3">
          <Label class="text-sm font-medium">Заблокированные пользователи</Label>
          <div v-if="loadingBlocked" class="py-4 text-center text-sm text-muted-foreground">
            Загрузка...
          </div>
          <div
            v-else-if="blockedUsers.length === 0"
            class="py-4 text-center text-sm text-muted-foreground"
          >
            Нет заблокированных пользователей
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="user in blockedUsers"
              :key="user.id"
              class="flex items-center gap-3 rounded-md border px-3 py-2"
            >
              <Avatar class="size-7 shrink-0">
                <AvatarFallback class="text-[10px]">
                  {{ (user.display_name || user.username).slice(0, 2).toUpperCase() }}
                </AvatarFallback>
              </Avatar>
              <RouterLink
                :to="profileLink(user.username, user.id)"
                class="min-w-0 flex-1 truncate text-sm font-medium hover:underline"
                @click="open = false"
              >
                {{ user.display_name || user.username }}
              </RouterLink>
              <Button
                variant="ghost"
                size="sm"
                class="shrink-0 gap-1 text-xs"
                :disabled="unblockingId === user.id"
                @click="unblock(user.id)"
              >
                <ShieldX class="size-3.5" />
                Разблокировать
              </Button>
            </div>
          </div>
        </div>
      </div>
    </DialogContent>
  </Dialog>
</template>
