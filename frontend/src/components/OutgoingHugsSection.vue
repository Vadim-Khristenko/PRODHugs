<script setup lang="ts">
import { ref, computed } from 'vue'
import { Heart, X, Plus, Coins } from 'lucide-vue-next'
import { toast } from 'vue-sonner'
import { useHugsStore } from '@/stores/hugs'
import { useAuthStore } from '@/stores/auth'
import { suggestVerb } from '@/lib/utils'
import { profileLink } from '@/lib/profileLink'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'

const hugsStore = useHugsStore()
const auth = useAuthStore()
const cancellingId = ref<string | null>(null)
const buying = ref(false)

const outgoing = computed(() => hugsStore.outgoingHugs)
const slots = computed(() => hugsStore.slotInfo)
// The backend reports next_slot_cost as null once the user is at MaxHugSlots.
// Use a loose `!= null` check so an *undefined* (e.g. while the first
// fetch is still in flight) doesn't render the button either — otherwise
// users at max briefly see "Купить слот" before the response lands.
const canBuy = computed(() => slots.value.next_slot_cost != null)

// Build a fixed-length array of slots. Each slot is either filled (has a hug) or empty.
const slotItems = computed(() => {
  const items: Array<{ index: number; hug: (typeof outgoing.value)[number] | null }> = []
  for (let i = 0; i < slots.value.total_slots; i++) {
    items.push({ index: i, hug: outgoing.value[i] ?? null })
  }
  return items
})

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

async function cancel(hugId: string) {
  if (cancellingId.value) return
  cancellingId.value = hugId
  try {
    await hugsStore.cancelOutgoing(hugId)
    toast('Предложение отменено')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err.response?.data?.message || 'Не удалось отменить')
  } finally {
    cancellingId.value = null
  }
}

async function buySlot() {
  if (buying.value) return
  buying.value = true
  try {
    await hugsStore.buySlot()
    toast.success('Слот куплен!')
  } catch (e: unknown) {
    const err = e as { response?: { data?: { message?: string } } }
    toast.error(err.response?.data?.message || 'Не удалось купить слот')
  } finally {
    buying.value = false
  }
}
</script>

<template>
  <Card>
    <CardHeader class="pb-3">
      <CardTitle class="flex items-center justify-between text-base">
        <span>Исходящие обнимашки</span>
        <span class="text-xs font-normal text-muted-foreground tabular-nums">
          {{ outgoing.length }}/{{ slots.total_slots }} слотов
        </span>
      </CardTitle>
    </CardHeader>
    <CardContent class="space-y-2">
      <!-- Fixed slot grid: each slot transitions between filled/empty in place -->
      <div
        v-for="slot in slotItems"
        :key="slot.index"
        class="slot-row rounded-md border px-3 py-2 transition-all duration-200"
        :class="
          slot.hug
            ? 'border-prod-yellow/20 bg-prod-yellow/5'
            : 'border-dashed border-muted-foreground/20'
        "
      >
        <!-- Filled slot -->
        <div v-if="slot.hug" class="flex items-center gap-2.5">
          <Heart class="size-3.5 shrink-0 text-prod-yellow" />
          <div class="min-w-0 flex-1 text-sm">
            <span class="text-muted-foreground">
              Ты {{ suggestVerb(auth.user?.gender) }} обняться
            </span>
            <RouterLink
              :to="profileLink(slot.hug.receiver_username, slot.hug.receiver_id)"
              class="truncate font-medium hover:underline"
            >
              {{ slot.hug.receiver_display_name || slot.hug.receiver_username }}
            </RouterLink>
            <span class="ml-1.5 text-[10px] text-muted-foreground">
              {{ relativeTime(slot.hug.created_at) }}
            </span>
          </div>
          <Button
            variant="ghost"
            size="sm"
            class="size-7 shrink-0 p-0"
            :disabled="cancellingId === slot.hug.id"
            @click="cancel(slot.hug.id)"
          >
            <X class="size-3.5" />
          </Button>
        </div>

        <!-- Empty slot -->
        <div v-else class="flex items-center gap-2.5 text-sm text-muted-foreground">
          <div
            class="size-3.5 shrink-0 rounded-full border border-dashed border-muted-foreground/30"
          />
          <span>Свободный слот</span>
        </div>
      </div>

      <!-- Buy slot button -->
      <Button
        v-if="canBuy"
        variant="outline"
        size="sm"
        class="mt-2 w-full gap-1.5 rounded-[21px]"
        :disabled="buying"
        @click="buySlot"
      >
        <Plus class="size-3.5" />
        Купить слот
        <span class="inline-flex items-center gap-0.5 text-muted-foreground">
          · <Coins class="size-3" /> {{ slots.next_slot_cost }}
        </span>
      </Button>
    </CardContent>
  </Card>
</template>

<style scoped>
.slot-row {
  min-height: 2.5rem;
}
</style>
