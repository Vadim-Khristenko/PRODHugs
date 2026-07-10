<script setup lang="ts">
import { ref, computed } from 'vue'
import { toast } from 'vue-sonner'
import { useHugsStore } from '@/stores/hugs'
import { hugSuggestionPhrase } from '@/lib/utils'
import { profileLink } from '@/lib/profileLink'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import HugExplosion from '@/components/HugExplosion.vue'

const hugsStore = useHugsStore()
const acceptingId = ref<string | null>(null)
const decliningId = ref<string | null>(null)

// Explosion state
const showExplosion = ref(false)
const explosionStyle = ref<Record<string, string>>({})

const inbox = computed(() => hugsStore.inbox)
const count = computed(() => hugsStore.inboxCount)

function relativeTime(dateStr: string): string {
  const diff = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000)
  if (diff < 60) return 'только что'
  if (diff < 3600) return `${Math.floor(diff / 60)} мин. назад`
  if (diff < 86400) return `${Math.floor(diff / 3600)} ч. назад`
  return new Date(dateStr).toLocaleDateString('ru-RU', {
    day: 'numeric',
    month: 'short',
  })
}

function initials(username: string): string {
  return username.slice(0, 2).toUpperCase()
}

async function accept(item: (typeof inbox.value)[number], event: MouseEvent) {
  if (acceptingId.value || decliningId.value) return
  acceptingId.value = item.id

  // Capture button position synchronously BEFORE the await removes the DOM element.
  const target = event.currentTarget as HTMLElement
  let explosionPos: { top: string; left: string; transform: string } | null = null
  if (target) {
    const rect = target.getBoundingClientRect()
    explosionPos = {
      top: `${rect.top + rect.height / 2}px`,
      left: `${rect.left + rect.width / 2}px`,
      transform: 'translate(-50%, -50%)',
    }
  }

  try {
    await hugsStore.acceptHug(item.id)
    // Trigger explosion at the pre-captured position
    if (explosionPos) {
      explosionStyle.value = explosionPos
      showExplosion.value = true
    }
    toast.success(`Обнимашка с ${item.giver_display_name || item.giver_username} принята!`)
  } catch (e: unknown) {
    const err = e as { response?: { data?: { code?: string; message?: string } } }
    const code = err.response?.data?.code
    if (code === 'HUG_EXPIRED') {
      toast.error('Это объятие истекло')
    } else if (code === 'HUG_NOT_PENDING') {
      toast.error('Это объятие было отменено')
    } else {
      toast.error(err.response?.data?.message || 'Не удалось принять обнимашку')
    }
  } finally {
    acceptingId.value = null
  }
}

async function decline(item: (typeof inbox.value)[number]) {
  if (acceptingId.value || decliningId.value) return
  decliningId.value = item.id
  try {
    await hugsStore.declineHug(item.id)
    toast('Обнимашка отклонена')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err.response?.data?.message || 'Не удалось отклонить обнимашку')
  } finally {
    decliningId.value = null
  }
}

function onExplosionDone() {
  showExplosion.value = false
}
</script>

<template>
  <Card>
    <CardHeader>
      <CardTitle class="flex items-center gap-2 text-base">
        Предложения обняться
        <Badge
          v-if="count > 0"
          variant="secondary"
          class="bg-prod-yellow/15 text-prod-yellow border-prod-yellow/20"
        >
          {{ count }}
        </Badge>
      </CardTitle>
    </CardHeader>
    <CardContent>
      <div v-if="inbox.length === 0" class="py-4 text-center text-sm text-muted-foreground">
        Никто не хочет тебя обнять прямо сейчас(
      </div>

      <TransitionGroup v-else name="inbox" tag="div" class="space-y-2">
        <div
          v-for="item in inbox"
          :key="item.id"
          class="flex items-center gap-3 rounded-lg border p-3"
        >
          <Avatar class="size-9 shrink-0">
            <AvatarFallback class="text-xs">
              {{ initials(item.giver_display_name || item.giver_username) }}
            </AvatarFallback>
          </Avatar>
          <div class="min-w-0 flex-1">
            <div class="text-sm">
              <RouterLink :to="profileLink(item.giver_username, item.giver_id)" class="truncate font-medium hover:underline">
                {{ item.giver_display_name || item.giver_username }}
              </RouterLink>
              <span class="text-muted-foreground">
                {{ ' ' + hugSuggestionPhrase(item.hug_type) }}
              </span>
            </div>
            <p
              v-if="item.comment"
              class="mt-1 max-h-20 overflow-y-auto rounded bg-muted/50 px-2 py-1 text-xs leading-relaxed text-muted-foreground whitespace-pre-wrap [overflow-wrap:anywhere]"
            >
              {{ item.comment }}
            </p>
            <span class="text-[10px] text-muted-foreground">
              {{ relativeTime(item.created_at) }}
            </span>
          </div>
          <div class="flex shrink-0 gap-1.5">
            <Button
              variant="yellow"
              size="sm"
              class="rounded-[21px]"
              :disabled="acceptingId === item.id"
              @click="accept(item, $event)"
            >
              {{ acceptingId === item.id ? '...' : 'Обняться' }}
            </Button>
            <Button
              variant="ghost"
              size="sm"
              class="rounded-[21px]"
              :disabled="decliningId === item.id"
              @click="decline(item)"
            >
              Отклонить
            </Button>
          </div>
        </div>
      </TransitionGroup>
    </CardContent>

    <Teleport to="body">
      <div v-if="showExplosion" class="pointer-events-none fixed z-[100]" :style="explosionStyle">
        <HugExplosion @done="onExplosionDone" />
      </div>
    </Teleport>
  </Card>
</template>

<style scoped>
.inbox-enter-active {
  transition: all 0.4s cubic-bezier(0.22, 1, 0.36, 1);
}

.inbox-leave-active {
  transition: all 0.3s ease-out;
}

.inbox-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.inbox-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}

.inbox-move {
  transition: transform 0.4s cubic-bezier(0.22, 1, 0.36, 1);
}
</style>
