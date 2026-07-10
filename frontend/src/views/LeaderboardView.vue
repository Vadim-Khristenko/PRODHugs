<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useHugsStore, type IntimacyLeaderboardEntry } from '@/stores/hugs'
import { profileLink } from '@/lib/profileLink'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import UserTag from '@/components/UserTag.vue'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import RankBadge from '@/components/RankBadge.vue'

const hugsStore = useHugsStore()

const tabs = ['users', 'pairs'] as const
type Tab = (typeof tabs)[number]
const activeTab = ref<Tab>('users')
const slideDirection = ref<'left' | 'right'>('left')

const pairsLoading = ref(false)
const pairs = ref<IntimacyLeaderboardEntry[]>([])
const pairsLoaded = ref(false)

async function loadPairs() {
  if (pairsLoaded.value) return
  pairsLoading.value = true
  try {
    pairs.value = await hugsStore.getIntimacyLeaderboard(50, 0)
    pairsLoaded.value = true
  } finally {
    pairsLoading.value = false
  }
}

function switchTab(tab: Tab) {
  if (tab === activeTab.value) return
  const oldIndex = tabs.indexOf(activeTab.value)
  const newIndex = tabs.indexOf(tab)
  slideDirection.value = newIndex > oldIndex ? 'left' : 'right'
  activeTab.value = tab
  if (tab === 'pairs') loadPairs()
}

function onTabChange(val: string | number) {
  switchTab(val as Tab)
}

// ── Swipe detection ──
const touchStartX = ref(0)
const touchStartY = ref(0)
const swiping = ref(false)

function onTouchStart(e: TouchEvent) {
  const touch = e.touches[0]
  if (!touch) return
  touchStartX.value = touch.clientX
  touchStartY.value = touch.clientY
  swiping.value = true
}

function onTouchEnd(e: TouchEvent) {
  if (!swiping.value) return
  swiping.value = false
  const touch = e.changedTouches[0]
  if (!touch) return
  const dx = touch.clientX - touchStartX.value
  const dy = touch.clientY - touchStartY.value

  // Only trigger if horizontal swipe is dominant and exceeds threshold
  if (Math.abs(dx) < 50 || Math.abs(dy) > Math.abs(dx)) return

  const currentIndex = tabs.indexOf(activeTab.value)
  if (dx < 0 && currentIndex < tabs.length - 1) {
    // Swipe left → next tab
    switchTab(tabs[currentIndex + 1]!)
  } else if (dx > 0 && currentIndex > 0) {
    // Swipe right → previous tab
    switchTab(tabs[currentIndex - 1]!)
  }
}

const transitionName = computed(() =>
  slideDirection.value === 'left' ? 'tab-slide-left' : 'tab-slide-right',
)

function displayName(username: string, display_name?: string | null): string {
  return display_name || username
}

onMounted(() => {
  hugsStore.fetchLeaderboard(50, 0)
})
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-6">
    <div>
      <h1 class="text-2xl font-semibold tracking-tight">Рейтинг</h1>
      <p class="text-muted-foreground">Топ пользователей и пар</p>
    </div>

    <Tabs :model-value="activeTab" @update:model-value="onTabChange">
      <TabsList class="w-full grid grid-cols-2">
        <TabsTrigger value="users">Пользователи</TabsTrigger>
        <TabsTrigger value="pairs">Пары</TabsTrigger>
      </TabsList>

      <div
        class="mt-4 overflow-hidden"
        @touchstart.passive="onTouchStart"
        @touchend.passive="onTouchEnd"
      >
        <Transition :name="transitionName" mode="out-in">
          <!-- Users tab -->
          <div v-if="activeTab === 'users'" key="users">
            <div v-if="hugsStore.leaderboardLoading" class="space-y-3">
              <Skeleton v-for="i in 10" :key="i" class="h-12 w-full" />
            </div>

            <div
              v-else-if="hugsStore.leaderboard.length === 0"
              class="py-12 text-center text-muted-foreground"
            >
              Пока нет данных
            </div>

            <div v-else class="rounded-[10px] border border-[#75988e33]">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead class="w-8 sm:w-12">#</TableHead>
                    <TableHead>Пользователь</TableHead>
                    <TableHead class="hidden sm:table-cell">Ранг</TableHead>
                    <TableHead class="text-right">Всего</TableHead>
                    <TableHead class="hidden md:table-cell text-right">Отправлено</TableHead>
                    <TableHead class="hidden md:table-cell text-right">Получено</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow
                    v-for="(entry, index) in hugsStore.leaderboard"
                    :key="entry.user_id"
                    class="cursor-pointer hover:bg-[#002D20]"
                    @click="$router.push(profileLink(entry.username, entry.user_id))"
                  >
                    <TableCell
                      class="font-medium tabular-nums text-xs sm:text-sm"
                      :class="index === 0 ? 'text-prod-yellow' : ''"
                    >
                      {{ index + 1 }}
                    </TableCell>
                    <TableCell>
                      <div class="flex items-center gap-2">
                        <Avatar class="size-6 sm:size-7">
                          <AvatarFallback class="text-[9px] sm:text-[10px]">
                            {{ (entry.display_name || entry.username).slice(0, 2).toUpperCase() }}
                          </AvatarFallback>
                        </Avatar>
                        <div class="min-w-0">
                          <div class="flex items-center gap-1.5">
                            <span class="truncate text-xs font-medium sm:text-sm">{{
                              entry.display_name || entry.username
                            }}</span>
                            <UserTag :tag="entry.tag" />
                            <span
                              v-if="entry.special_tag"
                              class="shrink-0 text-[9px] text-prod-yellow"
                            >{{ entry.special_tag }}</span>
                          </div>
                          <span
                            v-if="entry.display_name"
                            class="block truncate text-[10px] text-muted-foreground"
                            >@{{ entry.username }}</span
                          >
                          <RankBadge :rank="entry.rank" class="mt-0.5 sm:hidden" />
                        </div>
                      </div>
                    </TableCell>
                    <TableCell class="hidden sm:table-cell">
                      <RankBadge :rank="entry.rank" />
                    </TableCell>
                    <TableCell
                      class="text-right font-bold tabular-nums text-xs sm:text-sm"
                      :class="index === 0 ? 'text-prod-yellow' : ''"
                    >
                      {{ entry.total_hugs }}
                    </TableCell>
                    <TableCell
                      class="hidden md:table-cell text-right tabular-nums text-muted-foreground"
                    >
                      {{ entry.hugs_given }}
                    </TableCell>
                    <TableCell
                      class="hidden md:table-cell text-right tabular-nums text-muted-foreground"
                    >
                      {{ entry.hugs_received }}
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          </div>

          <!-- Pairs tab -->
          <div v-else-if="activeTab === 'pairs'" key="pairs">
            <div v-if="pairsLoading" class="space-y-3">
              <Skeleton v-for="i in 10" :key="i" class="h-14 w-full" />
            </div>

            <div
              v-else-if="pairs.length === 0 && pairsLoaded"
              class="py-12 text-center text-muted-foreground"
            >
              <p class="text-base font-medium">Пока нет пар</p>
              <p class="mt-1 text-sm">Обнимайтесь, чтобы попасть в рейтинг</p>
            </div>

            <div v-else-if="pairs.length > 0" class="divide-y rounded-md border">
              <div
                v-for="(entry, idx) in pairs"
                :key="`${entry.user_a_id}-${entry.user_b_id}`"
                class="flex items-center gap-3 px-3 py-3 sm:px-4"
              >
                <span
                  class="flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-bold tabular-nums"
                  :class="
                    idx < 3
                      ? 'bg-prod-yellow/15 text-prod-yellow'
                      : 'bg-muted text-muted-foreground'
                  "
                >
                  {{ idx + 1 }}
                </span>

                <div class="min-w-0 flex-1 text-sm">
                  <RouterLink
                    :to="profileLink(entry.user_a_username, entry.user_a_id)"
                    class="truncate font-medium hover:underline"
                  >
                    {{ displayName(entry.user_a_username, entry.user_a_display_name) }}
                  </RouterLink>
                  <span class="mx-1.5 text-muted-foreground">&amp;</span>
                  <RouterLink
                    :to="profileLink(entry.user_b_username, entry.user_b_id)"
                    class="truncate font-medium hover:underline"
                  >
                    {{ displayName(entry.user_b_username, entry.user_b_display_name) }}
                  </RouterLink>
                </div>

                <div class="flex shrink-0 flex-col items-end gap-0.5">
                  <Badge
                    variant="secondary"
                    class="text-[10px] bg-prod-yellow/15 text-prod-yellow border-prod-yellow/20"
                  >
                    {{ entry.tier_name }}
                  </Badge>
                  <span class="text-xs tabular-nums text-muted-foreground">
                    {{ entry.raw_score }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </Tabs>
  </div>
</template>

<style scoped>
.tab-slide-left-enter-active,
.tab-slide-left-leave-active,
.tab-slide-right-enter-active,
.tab-slide-right-leave-active {
  transition: all 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}

.tab-slide-left-enter-from {
  opacity: 0;
  transform: translateX(16px);
}

.tab-slide-left-leave-to {
  opacity: 0;
  transform: translateX(-16px);
}

.tab-slide-right-enter-from {
  opacity: 0;
  transform: translateX(-16px);
}

.tab-slide-right-leave-to {
  opacity: 0;
  transform: translateX(16px);
}
</style>
