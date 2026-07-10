<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import {
  Coins,
  Clock,
  ArrowUpCircle,
  MoreHorizontal,
  Ban,
  ShieldCheck,
  Flame,
  StickyNote,
  Pencil,
  Trash2,
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import {
  useHugsStore,
  type UserProfile,
  type CooldownInfo,
  type PairStreakResponse,
  type TopStreakEntry,
} from '@/stores/hugs'
import { type UserNoteDTO } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Skeleton } from '@/components/ui/skeleton'
import { Textarea } from '@/components/ui/textarea'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import HugButton from '@/components/HugButton.vue'
import RankBadge from '@/components/RankBadge.vue'
import UserTag from '@/components/UserTag.vue'
import StreakCalendar from '@/components/StreakCalendar.vue'
import StreakBadge from '@/components/StreakBadge.vue'
import { plural } from '@/lib/utils'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const hugsStore = useHugsStore()

const profile = ref<UserProfile | null>(null)
const cooldown = ref<CooldownInfo | null>(null)
const pairStreak = ref<PairStreakResponse | null>(null)
const topStreaks = ref<TopStreakEntry[]>([])
const loading = ref(true)
const upgrading = ref(false)
const error = ref('')

// The route can be either /user/@:username (canonical) or /user/:id (legacy
// UUID lookup that redirects to the canonical URL once the profile loads).
const routeKey = computed(() => {
  const username = route.params.username as string | undefined
  if (username) return `@${username}`
  return (route.params.id as string) ?? ''
})

// Numeric/UUID id is only known after the profile loads (when navigating by
// username) — most code paths only need the loaded profile's id.
const userId = computed(() => profile.value?.id ?? (route.params.id as string) ?? '')
const isMe = computed(() => !!auth.user && !!profile.value && auth.user.id === profile.value.id)
const isBlocked = computed(() => profile.value?.is_blocked === true)
const blocking = ref(false)

// ── Private note state ──
// Single 256-char note authored by the viewer about this profile. Visible only
// to the author. Self-notes are allowed with an "о самом себе" easter-egg
// label per product spec.
const MAX_NOTE_LEN = 256
const note = ref<UserNoteDTO | null>(null)
const noteEditorOpen = ref(false)
const noteEditorContent = ref('')
const noteEditorSaving = ref(false)
const noteDeleteConfirmOpen = ref(false)
const noteDeleting = ref(false)

// Count Unicode code points (not UTF-16 code units) so emoji and other
// supplementary-plane chars match the backend's utf8.RuneCountInString /
// Postgres char_length validation. "AB" → 2 ; "🙂" → 1 (string.length === 2).
function codePointLength(s: string): number {
  return [...s].length
}

const noteEditorLength = computed(() => codePointLength(noteEditorContent.value))
const noteRemainingChars = computed(() => MAX_NOTE_LEN - noteEditorLength.value)
const noteEditorCanSave = computed(() => {
  const trimmed = noteEditorContent.value.trim()
  const len = codePointLength(trimmed)
  return len > 0 && len <= MAX_NOTE_LEN && !noteEditorSaving.value
})

// Enforce the limit in input ourselves: the native maxlength attribute counts
// UTF-16 code units, which would block emoji that fit in the backend's
// code-point budget. Replace `noteEditorContent` with a code-point-truncated
// version whenever the user pushes past MAX_NOTE_LEN.
function onNoteEditorInput() {
  if (noteEditorLength.value <= MAX_NOTE_LEN) return
  noteEditorContent.value = [...noteEditorContent.value].slice(0, MAX_NOTE_LEN).join('')
}

function openNoteEditor() {
  noteEditorContent.value = note.value?.content ?? ''
  noteEditorOpen.value = true
}

async function saveNote() {
  if (!noteEditorCanSave.value || !profile.value) return
  noteEditorSaving.value = true
  try {
    const saved = await hugsStore.upsertNote(routeKey.value, noteEditorContent.value.trim())
    note.value = saved
    noteEditorOpen.value = false
    toast.success('Заметка сохранена')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err.response?.data?.message || 'Не удалось сохранить заметку')
  } finally {
    noteEditorSaving.value = false
  }
}

async function deleteNote() {
  if (noteDeleting.value || !profile.value) return
  noteDeleting.value = true
  try {
    await hugsStore.deleteNote(routeKey.value)
    note.value = null
    noteDeleteConfirmOpen.value = false
    toast.success('Заметка удалена')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err.response?.data?.message || 'Не удалось удалить заметку')
  } finally {
    noteDeleting.value = false
  }
}

// Compute whether the viewer is user_a in the canonical pair (a < b by UUID string)
const viewerIsA = computed(() => {
  if (!auth.user?.id) return true
  return auth.user.id < userId.value
})

async function load() {
  loading.value = true
  pairStreak.value = null
  topStreaks.value = []
  note.value = null
  try {
    // v2 profile endpoint accepts UUID or username — pass routeKey as-is.
    const loaded = await hugsStore.getUserProfile(routeKey.value)
    profile.value = loaded

    // If the user navigated by UUID, swap the URL to the canonical
    // /user/@username form so links can be shared and the back-stack stays
    // clean. router.replace preserves the entry instead of pushing a new one.
    if (!route.params.username && loaded.username) {
      await router.replace({
        name: 'user-profile-by-username',
        params: { username: loaded.username },
      })
    }

    const meId = auth.user?.id
    const viewingSelf = meId === loaded.id
    if (!viewingSelf && !loaded.is_blocked) {
      const [cd, streak] = await Promise.all([
        hugsStore.getCooldown(loaded.id),
        hugsStore.getPairStreak(loaded.id).catch(() => null),
      ])
      cooldown.value = cd
      pairStreak.value = streak
    }
    if (viewingSelf) {
      topStreaks.value = await hugsStore.getTopStreaks().catch(() => [])
    }
    // Notes are author-private — always fetch (including self) and tolerate
    // failures (note display is non-critical, profile already loaded).
    // Capture the routeKey that initiated this request so a slow response
    // can't overwrite the note shown for a different profile if the user
    // navigates away mid-fetch.
    const noteFetchKey = routeKey.value
    hugsStore.getNote(noteFetchKey).then((n) => {
      if (noteFetchKey !== routeKey.value) return
      note.value = n
    }).catch(() => {
      // Ignore — note section just stays empty.
    })
  } catch {
    error.value = 'Пользователь не найден'
  } finally {
    loading.value = false
  }
}

async function toggleBlock() {
  if (blocking.value || !profile.value) return
  blocking.value = true
  try {
    if (isBlocked.value) {
      await hugsStore.unblockUser(userId.value)
      profile.value = { ...profile.value, is_blocked: false }
      toast.success('Пользователь разблокирован')
      // Reload cooldown now that they're unblocked
      try {
        cooldown.value = await hugsStore.getCooldown(userId.value)
      } catch {
        // Ignore
      }
    } else {
      await hugsStore.blockUser(userId.value)
      profile.value = { ...profile.value, is_blocked: true }
      cooldown.value = null
      toast.success('Пользователь заблокирован')
    }
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err.response?.data?.message || 'Ошибка')
  } finally {
    blocking.value = false
  }
}

async function upgrade() {
  upgrading.value = true
  try {
    cooldown.value = await hugsStore.upgradeCooldown(userId.value)
    toast.success('Кулдаун уменьшен!')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err.response?.data?.message || 'Недостаточно обнимань')
  } finally {
    upgrading.value = false
  }
}

async function onHugged() {
  // Suggesting a hug doesn't change stats — just refresh cooldown state
  if (!isMe.value) {
    try {
      cooldown.value = await hugsStore.getCooldown(userId.value)
    } catch {
      // Ignore
    }
  }
}

onMounted(load)

// Re-fetch when navigating between user profiles (component is reused by Vue
// Router). Watch the routeKey so both @username and UUID transitions trigger
// a reload.
watch(routeKey, (next, prev) => {
  if (next && next !== prev) load()
})
</script>

<template>
  <div class="mx-auto max-w-2xl space-y-6">
    <div v-if="loading" class="space-y-4">
      <Skeleton class="h-40 w-full rounded-lg sm:h-32" />
      <div class="grid grid-cols-3 gap-2 sm:gap-4">
        <Skeleton class="h-16 rounded-lg sm:h-20" />
        <Skeleton class="h-16 rounded-lg sm:h-20" />
        <Skeleton class="h-16 rounded-lg sm:h-20" />
      </div>
    </div>

    <div v-else-if="error" class="py-12 text-center text-muted-foreground">{{ error }}</div>

    <template v-else-if="profile">
      <!-- Profile header -->
      <Card>
        <CardContent class="p-4">
          <!-- Mobile: stacked layout -->
          <div
            class="flex flex-col items-center gap-3 text-center sm:flex-row sm:items-center sm:gap-5 sm:text-left"
          >
            <Avatar class="size-16 sm:size-16">
              <AvatarFallback class="text-lg">
                {{ (profile.display_name || profile.username).slice(0, 2).toUpperCase() }}
              </AvatarFallback>
            </Avatar>
            <div class="min-w-0 flex-1 space-y-1.5">
              <h1 class="text-lg font-semibold break-words sm:text-xl">
                {{ profile.display_name || profile.username }}
              </h1>
              <p v-if="profile.display_name" class="truncate text-xs text-muted-foreground">
                @{{ profile.username }}
              </p>
              <div class="flex items-center justify-center gap-2 sm:justify-start">
                <RankBadge :rank="profile.rank" />
                <UserTag :tag="profile.tag" size="md" />
                <span v-if="profile.special_tag" class="text-xs text-prod-yellow">
                  {{ profile.special_tag }}
                </span>
                <span v-else class="text-xs text-muted-foreground">
                  {{ profile.role === 'admin' ? 'Администратор' : 'Пользователь' }}
                </span>
              </div>
            </div>
            <div class="flex items-center gap-1.5">
              <HugButton
                v-if="!isMe && !isBlocked"
                :userId="userId"
                :username="profile.display_name || profile.username"
                size="lg"
                @hugged="onHugged"
              />
              <DropdownMenu>
                <DropdownMenuTrigger as-child>
                  <Button variant="ghost" size="icon-sm">
                    <MoreHorizontal class="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" class="w-48">
                  <DropdownMenuItem v-if="!note" @click="openNoteEditor">
                    <StickyNote class="size-4" />
                    Добавить заметку
                  </DropdownMenuItem>
                  <DropdownMenuItem v-if="!isMe" @click="toggleBlock" :disabled="blocking">
                    <template v-if="isBlocked">
                      <ShieldCheck class="size-4" />
                      Разблокировать
                    </template>
                    <template v-else>
                      <Ban class="size-4" />
                      Заблокировать
                    </template>
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </CardContent>
      </Card>

      <!-- Private note (visible only to the author). Shown only when one exists;
           the empty state is just the "Добавить заметку" dropdown entry. -->
      <div
        v-if="note"
        class="rounded-[10px] border border-prod-yellow/30 bg-prod-yellow/5 p-3 space-y-1.5"
      >
        <div class="flex items-center justify-between gap-2">
          <div class="flex items-center gap-1.5 text-xs text-muted-foreground">
            <StickyNote class="size-3.5 text-prod-yellow" />
            <span v-if="isMe">Заметка о самом себе (видна только вам)</span>
            <span v-else>Заметка (видна только вам)</span>
          </div>
          <div class="flex items-center gap-0.5">
            <Button
              variant="ghost"
              size="icon-sm"
              class="size-7"
              @click="openNoteEditor"
              title="Редактировать"
            >
              <Pencil class="size-3.5" />
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              class="size-7 text-destructive hover:text-destructive"
              @click="noteDeleteConfirmOpen = true"
              title="Удалить"
            >
              <Trash2 class="size-3.5" />
            </Button>
          </div>
        </div>
        <p class="whitespace-pre-wrap break-words text-sm">{{ note.content }}</p>
      </div>

      <!-- Stats -->
      <div class="grid grid-cols-3 gap-2 sm:gap-4">
        <div class="rounded-[10px] border bg-card p-2.5 sm:p-4">
          <p class="text-[11px] text-muted-foreground sm:text-xs">Всего</p>
          <div class="mt-1 text-xl font-bold tabular-nums sm:text-2xl">
            {{ profile.total_hugs }}
          </div>
          <p
            v-if="!isMe && profile.mutual_total != null"
            class="mt-0.5 text-[10px] text-muted-foreground sm:text-xs"
          >
            {{ profile.mutual_total }} из них с тобой
          </p>
        </div>
        <div class="rounded-[10px] border bg-card p-2.5 sm:p-4">
          <p class="text-[11px] text-muted-foreground sm:text-xs">Инициировано</p>
          <div class="mt-1 text-xl font-bold tabular-nums sm:text-2xl">
            {{ profile.hugs_given }}
          </div>
          <p
            v-if="!isMe && profile.mutual_given != null"
            class="mt-0.5 text-[10px] text-muted-foreground sm:text-xs"
          >
            {{ profile.mutual_given }} из них тебе
          </p>
        </div>
        <div class="rounded-[10px] border bg-card p-2.5 sm:p-4">
          <p class="text-[11px] text-muted-foreground sm:text-xs">Принято</p>
          <div class="mt-1 text-xl font-bold tabular-nums sm:text-2xl">
            {{ profile.hugs_received }}
          </div>
          <p
            v-if="!isMe && profile.mutual_received != null"
            class="mt-0.5 text-[10px] text-muted-foreground sm:text-xs"
          >
            {{ profile.mutual_received }} из них от тебя
          </p>
        </div>
      </div>

      <!-- Intimacy section -->
      <Card v-if="!isMe && !isBlocked && profile.intimacy">
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            Близость
          </CardTitle>
          <CardDescription>
            {{ profile.intimacy.tier_name }} (уровень {{ profile.intimacy.tier }})
          </CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          <!-- Progress bar to next tier -->
          <div v-if="profile.intimacy.next_tier_at != null">
            <div class="flex items-center justify-between text-xs text-muted-foreground mb-1">
              <span>{{ profile.intimacy.raw_score }} очков</span>
              <span>{{ profile.intimacy.next_tier_at }}</span>
            </div>
            <div class="h-2 w-full rounded-full bg-muted overflow-hidden">
              <div
                class="h-full rounded-full bg-prod-yellow transition-all"
                :style="{
                  width:
                    Math.min(
                      (profile.intimacy.raw_score / profile.intimacy.next_tier_at!) * 100,
                      100,
                    ) + '%',
                }"
              />
            </div>
          </div>
          <div v-else class="text-xs text-muted-foreground">
            {{ profile.intimacy.raw_score }} очков (максимальный уровень)
          </div>
          <!-- Bonuses summary -->
          <div class="flex flex-wrap gap-2 text-xs text-muted-foreground">
            <span v-if="profile.intimacy.cooldown_reduction_pct > 0">
              Кулдаун -{{ profile.intimacy.cooldown_reduction_pct }}%
            </span>
            <span v-if="profile.intimacy.bonus_coins > 0">
              +{{ profile.intimacy.bonus_coins }} бонусных обнимань за обнимашку
            </span>
          </div>
        </CardContent>
      </Card>

      <!-- Pair streak section (viewing another user) -->
      <Card v-if="!isMe && !isBlocked && pairStreak">
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            <Flame class="size-4" />
            Серия обнимашек
          </CardTitle>
          <CardDescription>
            Обнимайте друг друга каждый день, чтобы продлить серию
          </CardDescription>
        </CardHeader>
        <CardContent>
          <StreakCalendar
            :streak="pairStreak.streak"
            :calendar="pairStreak.calendar"
            :viewer-is-a="viewerIsA"
          />
        </CardContent>
      </Card>

      <!-- Top streaks section (viewing own profile) -->
      <Card v-if="isMe && topStreaks.length > 0">
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            <Flame class="size-4" />
            Активные серии
          </CardTitle>
          <CardDescription>Ваши лучшие текущие серии обнимашек</CardDescription>
        </CardHeader>
        <CardContent class="space-y-3">
          <RouterLink
            v-for="entry in topStreaks"
            :key="entry.user_id"
            :to="'/user/' + entry.user_id"
            class="flex items-center gap-3 rounded-lg border p-3 transition-colors hover:bg-muted/50"
          >
            <Avatar class="size-8">
              <AvatarFallback class="text-xs">
                {{ (entry.display_name || entry.username).slice(0, 2).toUpperCase() }}
              </AvatarFallback>
            </Avatar>
            <div class="flex-1 min-w-0">
              <p class="truncate text-sm font-medium">
                {{ entry.display_name || entry.username }}
              </p>
              <p class="text-xs text-muted-foreground">@{{ entry.username }}</p>
            </div>
            <div class="flex items-center gap-2">
              <StreakBadge
                v-if="entry.tier_key"
                :tier-key="entry.tier_key"
                :tier-name="entry.tier_name"
                :streak-days="entry.current_streak"
              />
              <span v-else class="text-sm font-semibold tabular-nums">{{ plural(entry.current_streak, 'день', 'дня', 'дней') }}</span>
            </div>
          </RouterLink>
        </CardContent>
      </Card>

      <!-- Blocked notice -->
      <div
        v-if="isBlocked"
        class="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-center text-sm text-muted-foreground"
      >
        Пользователь заблокирован
      </div>

      <!-- Cooldown upgrade -->
      <Card v-if="!isMe && !isBlocked && cooldown">
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            <Clock class="size-4" />
            Кулдаун
          </CardTitle>
          <CardDescription>
            <template v-if="cooldown.intimacy_reduction_pct > 0">
              Эффективный: {{ Math.floor(cooldown.effective_cooldown_seconds / 60) }} мин.
              (базовый {{ Math.floor(cooldown.cooldown_seconds / 60) }} мин.,
              -{{ cooldown.intimacy_reduction_pct }}% от близости)
            </template>
            <template v-else>
              Текущий кулдаун: {{ Math.floor(cooldown.cooldown_seconds / 60) }} мин.
            </template>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <p class="text-xs text-muted-foreground sm:text-sm">
              <Coins class="inline size-3.5 mr-1" />
              {{ plural(5, 'обниманя', 'обнимани', 'обнимань') }} = -10 мин. (мин. 5 мин.)
            </p>
            <Button
              @click="upgrade"
              :disabled="upgrading || cooldown.cooldown_seconds <= 300"
              variant="yellow"
              size="sm"
              class="w-full rounded-[21px] sm:w-auto"
            >
              <ArrowUpCircle class="size-4" />
              {{ cooldown.cooldown_seconds <= 300 ? 'Максимум' : 'Улучшить' }}
            </Button>
          </div>
        </CardContent>
      </Card>
    </template>

    <!-- Note editor (add or edit). Dialog lives outside the v-if so it can
         animate out cleanly when closed. -->
    <Dialog v-model:open="noteEditorOpen">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ note ? 'Изменить заметку' : 'Новая заметка' }}</DialogTitle>
          <DialogDescription>
            Заметка приватная — её увидите только вы. До {{ MAX_NOTE_LEN }} символов.
          </DialogDescription>
        </DialogHeader>
        <Textarea
          v-model="noteEditorContent"
          @input="onNoteEditorInput"
          rows="4"
          placeholder="Заметка о пользователе..."
        />
        <div class="flex items-center justify-between text-xs text-muted-foreground">
          <span>{{ noteRemainingChars }} символов осталось</span>
        </div>
        <DialogFooter class="gap-2 sm:gap-3">
          <Button variant="outline" @click="noteEditorOpen = false" :disabled="noteEditorSaving">
            Отмена
          </Button>
          <Button variant="yellow" :disabled="!noteEditorCanSave" @click="saveNote">
            {{ noteEditorSaving ? 'Сохранение...' : 'Сохранить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Delete confirmation -->
    <Dialog v-model:open="noteDeleteConfirmOpen">
      <DialogContent class="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Удалить заметку?</DialogTitle>
          <DialogDescription>
            Заметку нельзя будет восстановить.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter class="gap-2 sm:gap-3">
          <Button
            variant="outline"
            @click="noteDeleteConfirmOpen = false"
            :disabled="noteDeleting"
          >
            Отмена
          </Button>
          <Button variant="destructive" @click="deleteNote" :disabled="noteDeleting">
            {{ noteDeleting ? 'Удаление...' : 'Удалить' }}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  </div>
</template>
