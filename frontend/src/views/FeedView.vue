<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { Wifi, WifiOff, ChevronUp, MessageSquare, Loader2 } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useHugsStore, type HugFeedItem, type HugActivityItem } from '@/stores/hugs'
import { useWebSocket } from '@/composables/useWebSocket'
import { hugFeedPhrase, streakTierLabel } from '@/lib/utils'
import { profileLink } from '@/lib/profileLink'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import HugDetailModal from '@/components/HugDetailModal.vue'
import StreakBadge from '@/components/StreakBadge.vue'
import { VisArea, VisAxis, VisXYContainer } from '@unovis/vue'
import type { ChartConfig } from '@/components/ui/chart'
import {
  ChartContainer,
  ChartCrosshair,
  ChartTooltip,
  ChartTooltipContent,
  componentToString,
} from '@/components/ui/chart'

const PAGE_SIZE = 50

const auth = useAuthStore()
const hugsStore = useHugsStore()
const ws = useWebSocket()
const feed = ref<HugFeedItem[]>([])
const initialLoading = ref(true)
const loadingMore = ref(false)
const hasMore = ref(true)
const now = ref(Date.now())
let tick: ReturnType<typeof setInterval> | null = null

// Hug detail modal
const detailHugId = ref<string | null>(null)
const showDetail = ref(false)

function openHugDetail(hugId: string) {
  detailHugId.value = hugId
  showDetail.value = true
}

function canViewComment(item: HugFeedItem): boolean {
  if (!auth.user) return false
  if (auth.user.role === 'admin') return true
  return item.giver_id === auth.user.id || item.receiver_id === auth.user.id
}

/** IDs of items that arrived via WebSocket (for highlight effect) */
const newItemIds = ref(new Set<string>())
/** Items that arrived via WebSocket while the user is scrolled down */
const pendingItems = ref<HugFeedItem[]>([])
/** Whether the user has scrolled away from the top */
const isScrolledAway = ref(false)
/** Scroll container ref */
const scrollContainer = ref<HTMLElement | null>(null)

const pendingCount = computed(() => pendingItems.value.length)

/** Chart data */
const activity = ref<HugActivityItem[]>([])
const chartLoading = ref(true)

const chartConfig = {
  count: {
    label: 'Обнимашки',
    color: '#ffdd2d',
  },
} satisfies ChartConfig

const totalHugs24h = computed(() => activity.value.reduce((sum, d) => sum + d.count, 0))

const SCROLL_THRESHOLD = 80

function timeAgo(dateStr: string): string {
  const diff = Math.floor((now.value - new Date(dateStr).getTime()) / 1000)
  if (diff < 5) return 'только что'
  if (diff < 60) return `${diff} сек. назад`
  if (diff < 3600) return `${Math.floor(diff / 60)} мин. назад`
  if (diff < 86400) return `${Math.floor(diff / 3600)} ч. назад`
  return new Date(dateStr).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })
}

function onScroll() {
  const container = scrollContainer.value
  if (!container) return
  isScrolledAway.value = container.scrollTop > SCROLL_THRESHOLD
}

function prependItem(item: HugFeedItem) {
  newItemIds.value.add(item.id)
  feed.value.unshift(item)
}

function flushPending() {
  const items = pendingItems.value.splice(0)
  for (const item of items) {
    prependItem(item)
  }
  nextTick(() => {
    scrollContainer.value?.scrollTo({ top: 0, behavior: 'smooth' })
  })
}

function bumpActivityBucket(createdAt: string) {
  const eventHour = new Date(createdAt)
  eventHour.setMinutes(0, 0, 0)
  const bucket = activity.value.find((b) => new Date(b.timestamp).getTime() === eventHour.getTime())
  if (bucket) {
    bucket.count++
  }
}

// Infinite scroll
const sentinel = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

async function loadMore() {
  if (loadingMore.value || !hasMore.value || initialLoading.value) return
  loadingMore.value = true
  try {
    const result = await hugsStore.fetchFeedPage(PAGE_SIZE, feed.value.length)
    feed.value.push(...result)
    hasMore.value = result.length >= PAGE_SIZE
  } finally {
    loadingMore.value = false
  }
}

function observeSentinel() {
  if (observer) observer.disconnect()
  observer = new IntersectionObserver(
    (entries) => {
      if (entries[0]?.isIntersecting && hasMore.value && !loadingMore.value) {
        loadMore()
      }
    },
    { rootMargin: '200px' },
  )
  if (sentinel.value) observer.observe(sentinel.value)
}

// WebSocket subscription cleanup
const cleanups: Array<() => void> = []
let isUnmounted = false

onMounted(async () => {
  hugsStore
    .getHugActivity()
    .then((data) => {
      activity.value = data
    })
    .finally(() => {
      chartLoading.value = false
    })

  await hugsStore.fetchFeed(PAGE_SIZE)
  feed.value = [...hugsStore.feed]
  hasMore.value = feed.value.length >= PAGE_SIZE
  initialLoading.value = false

  nextTick(() => {
    if (!isUnmounted) observeSentinel()
  })

  // Subscribe to hug_completed events via the shared composable
  cleanups.push(
    ws.on<HugFeedItem>('hug_completed', (item) => {
      if (isScrolledAway.value) {
        pendingItems.value.unshift(item)
      } else {
        prependItem(item)
      }
      // Update the chart bucket for this hug in real-time
      bumpActivityBucket(item.created_at)
    }),
  )

  tick = setInterval(() => {
    now.value = Date.now()
  }, 1000)

  // Find the nearest scrollable ancestor for scroll detection
  nextTick(() => {
    // Guard against the component being unmounted before nextTick resolves.
    if (isUnmounted) return
    let el: HTMLElement | null = document.querySelector('.feed-scroll-root')
    if (!el) {
      // Fallback: walk up to find the scrollable container
      el = document.querySelector('main') ?? document.documentElement
    }
    scrollContainer.value = el
    el.addEventListener('scroll', onScroll, { passive: true })
  })
})

onUnmounted(() => {
  isUnmounted = true
  cleanups.forEach((fn) => fn())
  if (tick) {
    clearInterval(tick)
    tick = null
  }
  if (observer) {
    observer.disconnect()
    observer = null
  }
  scrollContainer.value?.removeEventListener('scroll', onScroll)
})
</script>

<template>
  <div class="mx-auto max-w-2xl space-y-4 sm:space-y-6">
    <div class="flex items-center justify-between gap-2">
      <div class="min-w-0">
        <h1 class="text-2xl font-semibold tracking-tight">Лента</h1>
        <p class="text-muted-foreground">Обнимашки в реальном времени</p>
      </div>
      <Badge
        :variant="ws.connected.value ? 'secondary' : 'destructive'"
        class="shrink-0 gap-1.5"
        :class="
          ws.connected.value ? 'bg-prod-yellow/15 text-prod-yellow border-prod-yellow/20' : ''
        "
      >
        <Wifi v-if="ws.connected.value" class="size-3" />
        <WifiOff v-else class="size-3" />
        <span class="hidden xs:inline">{{ ws.connected.value ? 'Подключено' : 'Отключено' }}</span>
      </Badge>
    </div>

    <!-- Activity chart — last 24 hours -->
    <div class="rounded-md border p-3 sm:p-4">
      <div class="mb-3 flex items-center justify-between gap-2">
        <div class="min-w-0">
          <h2 class="text-base font-medium">Активность за 24 часа</h2>
          <p class="hidden text-xs text-muted-foreground sm:block">Обнимашки по часам</p>
        </div>
        <div v-if="!chartLoading && activity.length > 0" class="shrink-0 text-right">
          <p class="text-lg font-semibold tabular-nums">{{ totalHugs24h }}</p>
          <p class="text-xs text-muted-foreground">всего</p>
        </div>
      </div>

      <Skeleton v-if="chartLoading" class="h-[140px] w-full sm:h-[180px]" />
      <div
        v-else-if="activity.length === 0"
        class="flex h-[140px] items-center justify-center text-sm text-muted-foreground sm:h-[180px]"
      >
        Нет данных
      </div>
      <ChartContainer v-else :config="chartConfig" class="h-[140px] w-full sm:h-[180px]">
        <VisXYContainer :data="activity" :padding="{ top: 8 }">
          <VisArea
            :x="(_d: HugActivityItem, i: number) => i"
            :y="(d: HugActivityItem) => d.count"
            :color="chartConfig.count.color"
            :opacity="0.15"
            curve-type="monotoneX"
          />
          <VisAxis
            type="x"
            :x="(_d: HugActivityItem, i: number) => i"
            :tick-line="false"
            :domain-line="false"
            :grid-line="false"
            :num-ticks="5"
            :tick-format="
              (i: number) => {
                const item = activity[Math.round(i)]
                if (!item) return ''
                return new Date(item.timestamp).toLocaleTimeString('ru-RU', {
                  hour: '2-digit',
                  minute: '2-digit',
                })
              }
            "
          />
          <VisAxis
            type="y"
            :tick-line="false"
            :domain-line="false"
            :grid-line="true"
            :num-ticks="3"
          />
          <ChartTooltip />
          <ChartCrosshair
            :template="componentToString(chartConfig, ChartTooltipContent)"
            :color="chartConfig.count.color"
          />
        </VisXYContainer>
      </ChartContainer>
    </div>

    <div v-if="initialLoading" class="space-y-3">
      <Skeleton v-for="i in 8" :key="i" class="h-12 w-full" />
    </div>

    <div
      v-else-if="feed.length === 0 && pendingCount === 0"
      class="py-12 text-center text-muted-foreground sm:py-16"
    >
      <p class="text-base font-medium sm:text-lg">Пока нет обнимашек</p>
      <p class="mt-1 text-sm">Будьте первыми!</p>
    </div>

    <div v-else class="relative">
      <!-- New events indicator -->
      <Transition name="indicator">
        <button
          v-if="pendingCount > 0"
          class="sticky top-2 z-20 mx-auto flex cursor-pointer items-center gap-1.5 rounded-full border border-prod-yellow/30 bg-card/90 px-3 py-1.5 text-sm font-medium text-prod-yellow shadow-lg backdrop-blur-sm transition-all hover:bg-card hover:shadow-xl active:scale-95 sm:px-4"
          @click="flushPending"
        >
          <ChevronUp class="size-3.5" />
          {{ pendingCount }} {{ pendingCount === 1 ? 'новая обнимашка' : 'новых обнимашек' }}
        </button>
      </Transition>

      <div class="divide-y rounded-md border">
        <TransitionGroup name="feed">
          <div
            v-for="item in feed"
            :key="item.id"
            class="flex items-center gap-2 px-3 py-2.5 sm:gap-3 sm:px-4 sm:py-3"
            :class="{
              'feed-new': newItemIds.has(item.id),
              'cursor-pointer transition-colors hover:bg-muted/50': item.has_comment && canViewComment(item),
            }"
            @animationend="newItemIds.delete(item.id)"
            @click="item.has_comment && canViewComment(item) ? openHugDetail(item.id) : undefined"
          >
            <div class="min-w-0 flex-1 text-sm">
              <RouterLink
                :to="profileLink(item.giver_username, item.giver_id)"
                class="truncate font-medium hover:underline"
                @click.stop
              >{{ item.giver_display_name || item.giver_username }}</RouterLink>
              <span class="mx-1 text-muted-foreground">{{
                hugFeedPhrase(item.giver_gender, item.hug_type)
              }}</span>
              <RouterLink
                :to="profileLink(item.receiver_username, item.receiver_id)"
                class="truncate font-medium hover:underline"
                @click.stop
              >{{ item.receiver_display_name || item.receiver_username }}</RouterLink>
              <MessageSquare
                v-if="item.has_comment && canViewComment(item)"
                class="ml-1 inline size-3 text-prod-yellow"
              />
              <StreakBadge
                v-if="item.streak_tier"
                :tier-key="item.streak_tier"
                :tier-name="streakTierLabel(item.streak_tier)"
                class="ml-1 inline-flex scale-90"
              />
            </div>
            <span class="shrink-0 text-xs text-muted-foreground tabular-nums">
              {{ timeAgo(item.created_at) }}
            </span>
          </div>
        </TransitionGroup>
      </div>

      <!-- Infinite scroll sentinel -->
      <div v-if="hasMore" ref="sentinel" class="flex justify-center py-4">
        <Loader2 v-if="loadingMore" class="size-5 animate-spin text-muted-foreground" />
      </div>
      <p
        v-if="!hasMore && feed.length > 0"
        class="py-4 text-center text-sm text-muted-foreground"
      >
        Вся лента загружена
      </p>
    </div>

    <HugDetailModal v-model:open="showDetail" :hug-id="detailHugId" />
  </div>
</template>

<style scoped>
/* Enter — slide down + fade in */
.feed-enter-active {
  transition: all 0.4s cubic-bezier(0.22, 1, 0.36, 1);
}
.feed-enter-from {
  opacity: 0;
  transform: translateY(-12px);
}

/* Leave — fade out */
.feed-leave-active {
  transition: all 0.25s ease-out;
  position: absolute;
  width: 100%;
}
.feed-leave-to {
  opacity: 0;
}

/* Move — smooth repositioning when new items are prepended */
.feed-move {
  transition: transform 0.4s cubic-bezier(0.22, 1, 0.36, 1);
}

/* New-events indicator enter/leave */
.indicator-enter-active {
  transition: all 0.3s cubic-bezier(0.22, 1, 0.36, 1);
}
.indicator-leave-active {
  transition: all 0.2s ease-out;
}
.indicator-enter-from,
.indicator-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.95);
}
</style>
