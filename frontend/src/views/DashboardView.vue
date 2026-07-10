<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useTicker } from '@/composables/useTicker'
import {
  Heart,
  ArrowUp,
  ArrowDown,
  Gift,
  Coins,
  Users,
  Trophy,
  Newspaper,
  MessageSquare,
  Flame,
} from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { RouterLink } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import {
  useHugsStore,
  type DailyRewardResponse,
  type DailyRewardStatus,
  type UserProfile,
  type TopStreakEntry,
} from '@/stores/hugs'
import { formatRemainingTime } from '@/lib/utils'
import { profileLink } from '@/lib/profileLink'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import RankBadge from '@/components/RankBadge.vue'
import StreakBadge from '@/components/StreakBadge.vue'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import { plural, hugVerb } from '@/lib/utils'
import InboxSection from '@/components/InboxSection.vue'
import HugDetailModal from '@/components/HugDetailModal.vue'

const auth = useAuthStore()
const hugs = useHugsStore()

const profile = ref<UserProfile | null>(null)
const topStreaks = ref<TopStreakEntry[]>([])
const dailyResult = ref<DailyRewardResponse | null>(null)
const dailyStatus = ref<DailyRewardStatus | null>(null)
const claimingDaily = ref(false)
const loading = ref(true)
let unmounted = false

const { now: tickerNow } = useTicker()

// Disable the claim button when the backend reports the reward has already
// been claimed today (returned via /api/v2/daily-reward/status).
const dailyAlreadyClaimed = computed(() => {
  if (dailyResult.value?.already_claimed) return true
  return dailyStatus.value ? !dailyStatus.value.can_claim : false
})

// Live countdown to the next claim window (next UTC midnight after the last
// claim). Updates via the global ticker.
const dailyNextClaimText = computed(() => {
  if (!dailyAlreadyClaimed.value) return ''
  const iso = dailyStatus.value?.next_claim_at
  if (!iso) return ''
  const diffMs = new Date(iso).getTime() - tickerNow.value
  const seconds = Math.max(0, Math.floor(diffMs / 1000))
  return formatRemainingTime(seconds)
})

const dailyStreakDays = computed(() => {
  return dailyResult.value?.streak_days ?? dailyStatus.value?.streak_days ?? 0
})

async function refreshDailyStatus() {
  try {
    dailyStatus.value = await hugs.getDailyRewardStatus()
  } catch {
    // Non-fatal: the claim button stays in its default state if status fails.
  }
}

// Hug detail modal
const detailHugId = ref<string | null>(null)
const showDetail = ref(false)

function openHugDetail(hugId: string) {
  detailHugId.value = hugId
  showDetail.value = true
}

const rankThresholds = [
  { name: 'Новичок', min: 0 },
  { name: 'Обнимашка', min: 10 },
  { name: 'Дружелюбный', min: 50 },
  { name: 'Мастер обнимашек', min: 200 },
  { name: 'Легенда', min: 500 },
  { name: 'Тактильный маньяк', min: 1000 },
]

function getRankProgress(totalHugs: number) {
  const currentIdx = rankThresholds.findLastIndex(
    (r: { name: string; min: number }) => totalHugs >= r.min,
  )
  const nextIdx = currentIdx + 1
  if (nextIdx >= rankThresholds.length) return { progress: 100, nextRank: null, needed: 0 }
  const current = rankThresholds[currentIdx]!
  const next = rankThresholds[nextIdx]!
  const progress = ((totalHugs - current.min) / (next.min - current.min)) * 100
  return { progress: Math.min(progress, 100), nextRank: next.name, needed: next.min - totalHugs }
}

onMounted(async () => {
  // Fetch balance, inbox, outgoing in parallel
  const balancePromise = hugs.fetchBalance()
  const inboxPromise = hugs.fetchInbox()
  const outgoingPromise = hugs.fetchOutgoing()
  // Daily-reward status is fire-and-forget — it updates the button label and
  // countdown when it lands, but the rest of the dashboard doesn't depend
  // on it.
  refreshDailyStatus()

  let profilePromise: Promise<UserProfile> | undefined
  let historyPromise: ReturnType<typeof hugs.getHugHistory> | undefined

  if (auth.user) {
    profilePromise = hugs.getUserProfile(auth.user.id)
    historyPromise = hugs.getHugHistory()
  }

  await balancePromise

  const [p] = await Promise.all([profilePromise, historyPromise, inboxPromise, outgoingPromise])
  if (unmounted) return
  if (p) profile.value = p

  hugs.getTopStreaks().catch(() => [] as TopStreakEntry[]).then((data) => {
    if (!unmounted) topStreaks.value = data
  })

  loading.value = false
})

onUnmounted(() => {
  unmounted = true
})

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const isToday =
    date.getDate() === now.getDate() &&
    date.getMonth() === now.getMonth() &&
    date.getFullYear() === now.getFullYear()

  if (isToday) {
    return date.toLocaleString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  }

  return date.toLocaleString('ru-RU', {
    day: 'numeric',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

async function claimDaily() {
  if (dailyAlreadyClaimed.value || claimingDaily.value) return
  claimingDaily.value = true
  try {
    dailyResult.value = await hugs.claimDailyReward()
    if (dailyResult.value.already_claimed) {
      toast.info('Вы уже получили награду сегодня')
    } else {
      toast.success(`Получено +${plural(dailyResult.value.amount, 'обниманя', 'обнимани', 'обнимань')}!`)
    }
    // Pull a fresh status so the button locks and the countdown begins.
    refreshDailyStatus()
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err.response?.data?.message || 'Ошибка')
  } finally {
    claimingDaily.value = false
  }
}

const rankInfo = () => getRankProgress(profile.value?.total_hugs ?? 0)
</script>

<template>
  <div class="mx-auto max-w-4xl space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">
        Привет, <span class="inline-block max-w-full truncate align-bottom text-prod-yellow">{{ auth.user?.display_name || auth.user?.username }}</span>
      </h1>
      <p class="text-muted-foreground">Твоя панель управления обнимашками</p>
    </div>

    <!-- Stats -->
    <div class="grid gap-4 sm:grid-cols-3">
      <Card v-if="!loading">
        <CardHeader class="flex flex-row items-center justify-between pb-2">
          <CardDescription>Всего обнимашек</CardDescription>
          <Heart class="size-4 text-prod-yellow" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">{{ profile?.total_hugs ?? 0 }}</div>
        </CardContent>
      </Card>
      <Card v-if="!loading">
        <CardHeader class="flex flex-row items-center justify-between pb-2">
          <CardDescription>Инициировано</CardDescription>
          <ArrowUp class="size-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">{{ profile?.hugs_given ?? 0 }}</div>
        </CardContent>
      </Card>
      <Card v-if="!loading">
        <CardHeader class="flex flex-row items-center justify-between pb-2">
          <CardDescription>Принято</CardDescription>
          <ArrowDown class="size-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div class="text-2xl font-bold">{{ profile?.hugs_received ?? 0 }}</div>
        </CardContent>
      </Card>
      <template v-if="loading">
        <Card v-for="i in 3" :key="i">
          <CardHeader class="pb-2"><Skeleton class="h-4 w-24" /></CardHeader>
          <CardContent><Skeleton class="h-8 w-16" /></CardContent>
        </Card>
      </template>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <!-- Rank & Progress -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Ваш ранг</CardTitle>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="flex items-center gap-3">
            <RankBadge :rank="profile?.rank ?? 'Новичок'" />
            <div class="flex items-center gap-1.5 text-sm text-muted-foreground">
              <Coins class="size-3.5" />
              {{ plural(hugs.balance?.amount ?? 0, 'обниманя', 'обнимани', 'обнимань') }}
            </div>
          </div>
          <div v-if="rankInfo().nextRank" class="space-y-2">
            <div class="flex justify-between text-xs text-muted-foreground">
              <span>{{ profile?.rank }}</span>
              <span>{{ rankInfo().nextRank }}</span>
            </div>
            <Progress :model-value="rankInfo().progress" class="h-2" />
            <p class="text-xs text-muted-foreground">
              Ещё
              {{ plural(rankInfo().needed, 'обнимашка', 'обнимашки', 'обнимашек') }} до следующего
              ранга
            </p>
          </div>
          <p v-else class="text-xs text-muted-foreground">Максимальный ранг достигнут</p>
        </CardContent>
      </Card>

      <!-- Daily reward -->
      <Card>
        <CardHeader>
          <CardTitle class="text-base">Ежедневная награда</CardTitle>
          <CardDescription
            >Заходите каждый день для бонуса. Серия увеличивает награду.</CardDescription
          >
        </CardHeader>
        <CardContent class="flex flex-1 flex-col justify-end space-y-3">
          <div v-if="dailyAlreadyClaimed" class="text-sm space-y-0.5">
            <p class="text-muted-foreground">
              Награда уже получена. Серия: {{ dailyStreakDays }} дн.
            </p>
            <p v-if="dailyNextClaimText" class="text-xs text-muted-foreground">
              Следующая через
              <span class="font-mono tabular-nums">{{ dailyNextClaimText }}</span>
            </p>
          </div>
          <div v-else-if="dailyResult" class="text-sm">
            <p class="text-prod-yellow">
              +{{ plural(dailyResult.amount, 'обниманя', 'обнимани', 'обнимань') }}! Серия:
              {{ dailyResult.streak_days }} дн.
            </p>
          </div>
          <Button
            @click="claimDaily"
            :disabled="claimingDaily || dailyAlreadyClaimed"
            variant="yellow"
            class="w-full rounded-[21px]"
          >
            <Gift class="size-4" />
            {{
              dailyAlreadyClaimed
                ? 'Награда уже получена'
                : claimingDaily
                  ? 'Загрузка...'
                  : 'Забрать награду'
            }}
          </Button>
        </CardContent>
      </Card>

      <!-- Active streaks -->
      <Card class="md:col-span-2">
        <CardHeader>
          <CardTitle class="flex items-center gap-2 text-base">
            <Flame class="size-4 text-prod-yellow" />
            Активные серии
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p v-if="topStreaks.length === 0" class="text-sm text-muted-foreground">
            Нет активных серий
          </p>
          <div v-else class="space-y-2">
            <RouterLink
              v-for="entry in topStreaks"
              :key="entry.user_id"
              :to="profileLink(entry.username, entry.user_id)"
              class="flex items-center justify-between rounded-md px-2 py-1.5 transition-colors hover:bg-muted/50"
            >
              <span class="text-sm font-medium">
                {{ entry.display_name || entry.username }}
              </span>
              <StreakBadge
                v-if="entry.tier_key"
                :tier-key="entry.tier_key"
                :tier-name="entry.tier_name"
                :streak-days="entry.current_streak"
              />
              <span v-else class="text-sm tabular-nums text-muted-foreground">
                {{ plural(entry.current_streak, 'день', 'дня', 'дней') }}
              </span>
            </RouterLink>
          </div>
        </CardContent>
      </Card>
    </div>

    <!-- Inbox section -->
    <InboxSection />

    <!-- Hug history -->
    <Card>
      <CardHeader>
        <CardTitle class="text-base">История обнимашек</CardTitle>
        <CardDescription>Последние обнимашки</CardDescription>
      </CardHeader>
      <CardContent>
        <div v-if="loading" class="space-y-3">
          <Skeleton v-for="i in 3" :key="i" class="h-8 w-full rounded" />
        </div>
        <div
          v-else-if="hugs.history.length === 0"
          class="py-6 text-center text-sm text-muted-foreground"
        >
          Пока нет обнимашек
        </div>
        <div v-else class="max-h-96 space-y-1 overflow-y-auto">
          <div v-for="(hug, i) in hugs.history" :key="hug.id">
            <Separator v-if="i > 0" class="my-1" />
            <div
              class="flex items-center justify-between py-2"
              :class="{ 'cursor-pointer rounded-md transition-colors hover:bg-muted/50': hug.has_comment }"
              @click="hug.has_comment ? openHugDetail(hug.id) : undefined"
            >
              <div class="flex items-center gap-2 text-sm">
                <ArrowUp
                  v-if="hug.giver_id === auth.user?.id"
                  class="size-3.5 text-muted-foreground"
                />
                <ArrowDown v-else class="size-3.5 text-muted-foreground" />
                <span v-if="hug.giver_id === auth.user?.id" class="text-muted-foreground">
                  Ты {{ hugVerb(auth.user?.gender) }}
                  <RouterLink
                    :to="profileLink(hug.receiver_username, hug.receiver_id)"
                    class="font-medium text-foreground hover:underline"
                    @click.stop
                    >{{ hug.receiver_display_name || hug.receiver_username }}</RouterLink
                  >
                </span>
                <span v-else class="text-muted-foreground">
                  <RouterLink
                    :to="profileLink(hug.giver_username, hug.giver_id)"
                    class="font-medium text-foreground hover:underline"
                    @click.stop
                    >{{ hug.giver_display_name || hug.giver_username }}</RouterLink
                  >
                  {{ hugVerb(hug.giver_gender) }} тебя
                </span>
                <MessageSquare v-if="hug.has_comment" class="size-3 text-prod-yellow" />
              </div>
              <span class="text-xs text-muted-foreground tabular-nums">
                {{ formatDate(hug.created_at) }}
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>

    <!-- Quick links -->
    <div class="grid gap-4 sm:grid-cols-3">
      <RouterLink to="/users">
        <Card class="h-full transition-colors hover:bg-[#002D20]">
          <CardHeader class="flex flex-row items-center gap-3">
            <Users class="size-5 text-prod-yellow" />
            <div>
              <CardTitle class="text-sm">Пользователи</CardTitle>
              <CardDescription>Найти и обнять</CardDescription>
            </div>
          </CardHeader>
        </Card>
      </RouterLink>
      <RouterLink to="/feed">
        <Card class="h-full transition-colors hover:bg-[#002D20]">
          <CardHeader class="flex flex-row items-center gap-3">
            <Newspaper class="size-5 text-prod-yellow" />
            <div>
              <CardTitle class="text-sm">Лента</CardTitle>
              <CardDescription>Обнимашки в реальном времени</CardDescription>
            </div>
          </CardHeader>
        </Card>
      </RouterLink>
      <RouterLink to="/leaderboard">
        <Card class="h-full transition-colors hover:bg-[#002D20]">
          <CardHeader class="flex flex-row items-center gap-3">
            <Trophy class="size-5 text-prod-yellow" />
            <div>
              <CardTitle class="text-sm">Рейтинг</CardTitle>
              <CardDescription>Топ пользователей</CardDescription>
            </div>
          </CardHeader>
        </Card>
      </RouterLink>
    </div>

    <HugDetailModal v-model:open="showDetail" :hug-id="detailHugId" />
  </div>
</template>
