<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Coins, LogOut, Settings } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useHugsStore } from '@/stores/hugs'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import SettingsDialog from '@/components/SettingsDialog.vue'

const auth = useAuthStore()
const hugs = useHugsStore()
const route = useRoute()
const router = useRouter()
const settingsOpen = ref(false)

const showBack = computed(() => !!route.meta.back)

onMounted(() => {
  hugs.fetchBalance()
})
</script>

<template>
  <div class="flex flex-1 items-center justify-between">
    <!-- Mobile-only: back button or branding -->
    <div class="relative flex items-center md:hidden" style="min-width: 160px; height: 32px">
      <Transition name="header-swap">
        <button
          v-if="showBack"
          key="back"
          class="absolute inset-y-0 left-0 flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
          @click="router.back()"
        >
          <ArrowLeft class="size-4" />
          <span>Назад</span>
        </button>
        <div v-else key="brand" class="absolute inset-y-0 left-0 flex items-center gap-2">
          <img src="/logo.webp" alt="PROD" class="size-8 shrink-0 rounded-lg object-contain" />
          <span class="text-sm font-semibold text-foreground"
            ><span class="font-bold">PROD</span>нимашки</span
          >
        </div>
      </Transition>
    </div>

    <!-- Desktop: back button or empty spacer -->
    <div class="hidden md:flex md:items-center">
      <Transition name="header-swap">
        <button
          v-if="showBack"
          key="back-desktop"
          class="flex items-center gap-2 text-sm text-muted-foreground transition-colors hover:text-foreground"
          @click="router.back()"
        >
          <ArrowLeft class="size-4" />
          <span>Назад</span>
        </button>
        <div v-else key="spacer" />
      </Transition>
    </div>
    <div class="flex items-center gap-3">
      <Badge
        variant="secondary"
        class="gap-1.5 font-mono tabular-nums bg-prod-yellow/15 text-prod-yellow border-prod-yellow/20"
      >
        <Coins class="size-3.5" />
        {{ hugs.balance?.amount ?? 0 }}
      </Badge>

      <DropdownMenu>
        <DropdownMenuTrigger as-child>
          <Button variant="ghost" size="icon" class="rounded-full">
            <Avatar class="size-8">
              <AvatarFallback class="text-xs">
                {{ (auth.user?.display_name || auth.user?.username)?.slice(0, 2)?.toUpperCase() }}
              </AvatarFallback>
            </Avatar>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" class="w-48">
          <DropdownMenuLabel class="truncate">{{ auth.user?.display_name || auth.user?.username }}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem @click="settingsOpen = true">
            <Settings class="size-4" />
            Настройки
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem @click="auth.logout()" class="text-destructive">
            <LogOut class="size-4" />
            Выйти
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <SettingsDialog v-model:open="settingsOpen" />
    </div>
  </div>
</template>

<style scoped>
.header-swap-enter-active,
.header-swap-leave-active {
  transition: all 0.2s ease;
}

.header-swap-enter-from {
  opacity: 0;
  transform: translateX(-8px);
}

.header-swap-leave-to {
  opacity: 0;
  transform: translateX(8px);
}
</style>
