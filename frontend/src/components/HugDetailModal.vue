<script setup lang="ts">
import { ref, watch } from 'vue'
import { MessageSquare, Heart, Loader2 } from 'lucide-vue-next'
import { useHugsStore, type HugDetail } from '@/stores/hugs'
import { hugTypeLabel } from '@/lib/utils'
import { profileLink } from '@/lib/profileLink'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import StreakBadge from '@/components/StreakBadge.vue'

const props = defineProps<{
  hugId: string | null
}>()

const open = defineModel<boolean>('open', { default: false })

const hugsStore = useHugsStore()
const detail = ref<HugDetail | null>(null)
const loading = ref(false)
const error = ref(false)

watch(
  () => [open.value, props.hugId] as const,
  async ([isOpen, id]) => {
    if (isOpen && id) {
      loading.value = true
      error.value = false
      detail.value = null
      try {
        detail.value = await hugsStore.getHugDetail(id)
      } catch {
        error.value = true
      } finally {
        loading.value = false
      }
    }
  },
)

function formatDate(dateStr: string): string {
  return new Date(dateStr).toLocaleString('ru-RU', {
    day: 'numeric',
    month: 'long',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function statusLabel(status: string): string {
  switch (status) {
    case 'completed':
      return 'Завершено'
    case 'pending':
      return 'Ожидание'
    case 'declined':
      return 'Отклонено'
    case 'expired':
      return 'Истекло'
    case 'cancelled':
      return 'Отменено'
    default:
      return status
  }
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent class="sm:max-w-md">
      <DialogHeader>
        <DialogTitle class="flex items-center gap-2">
          <Heart class="size-4 text-prod-yellow" />
          Детали обнимашки
        </DialogTitle>
      </DialogHeader>

      <div v-if="loading" class="flex items-center justify-center py-8">
        <Loader2 class="size-6 animate-spin text-muted-foreground" />
      </div>

      <div v-else-if="error" class="py-6 text-center text-sm text-muted-foreground">
        Не удалось загрузить детали
      </div>

      <div v-else-if="detail" class="min-w-0 space-y-4">
        <!-- Participants -->
        <div class="space-y-2">
          <div class="flex min-w-0 items-center justify-between text-sm">
            <span class="text-muted-foreground">Отправитель</span>
            <RouterLink
              :to="profileLink(detail.giver_username, detail.giver_id)"
              class="min-w-0 truncate font-medium hover:underline"
              @click="open = false"
            >
              {{ detail.giver_display_name || detail.giver_username }}
            </RouterLink>
          </div>
          <div class="flex min-w-0 items-center justify-between text-sm">
            <span class="text-muted-foreground">Получатель</span>
            <RouterLink
              :to="profileLink(detail.receiver_username, detail.receiver_id)"
              class="min-w-0 truncate font-medium hover:underline"
              @click="open = false"
            >
              {{ detail.receiver_display_name || detail.receiver_username }}
            </RouterLink>
          </div>
        </div>

        <Separator />

        <!-- Meta -->
        <div class="flex flex-wrap gap-2">
          <Badge variant="secondary">{{ hugTypeLabel(detail.hug_type) }}</Badge>
          <Badge variant="outline">{{ statusLabel(detail.status) }}</Badge>
          <StreakBadge v-if="detail.streak_tier" :tier-key="detail.streak_tier" />
        </div>

        <div class="space-y-1 text-xs text-muted-foreground">
          <div>Отправлено: {{ formatDate(detail.created_at) }}</div>
          <div v-if="detail.accepted_at">Принято: {{ formatDate(detail.accepted_at) }}</div>
        </div>

        <!-- Comment -->
        <template v-if="detail.comment">
          <Separator />
          <div class="space-y-1.5">
            <div class="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
              <MessageSquare class="size-3" />
              Комментарий
            </div>
            <p class="max-h-40 overflow-y-auto rounded-lg bg-muted/50 p-3 text-sm leading-relaxed whitespace-pre-wrap [overflow-wrap:anywhere]">
              {{ detail.comment }}
            </p>
          </div>
        </template>
      </div>
    </DialogContent>
  </Dialog>
</template>
