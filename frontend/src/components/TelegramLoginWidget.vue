<script setup lang="ts">
/**
 * Renders the official Telegram Login Widget button.
 *
 * Prerequisites:
 *  - Set VITE_TELEGRAM_BOT_USERNAME in your .env (or .env.local).
 *  - Register the domain with BotFather via /setdomain.
 *    On localhost use an ngrok tunnel and register that domain instead.
 *
 * If VITE_TELEGRAM_BOT_USERNAME is empty/unset the component renders nothing
 * so the login page continues to work without the widget.
 */

import { ref, onMounted, onUnmounted } from 'vue'
import type { TelegramWidgetUser } from '@/lib/telegram'

// Extend Window so TypeScript is happy with the global callback.
declare global {
  interface Window {
    onTelegramAuth?: (user: TelegramWidgetUser) => void
  }
}

const emit = defineEmits<{
  (e: 'auth', user: TelegramWidgetUser): void
}>()

const botUsername = import.meta.env.VITE_TELEGRAM_BOT_USERNAME as string | undefined

const container = ref<HTMLElement | null>(null)

onMounted(() => {
  if (!botUsername || !container.value) return

  // Register global callback — Telegram widget calls this on successful auth.
  window.onTelegramAuth = (user: TelegramWidgetUser) => {
    emit('auth', user)
  }

  // Build and inject the Telegram widget <script> tag.
  const script = document.createElement('script')
  script.src = 'https://telegram.org/js/telegram-widget.js?22'
  script.setAttribute('data-telegram-login', botUsername)
  script.setAttribute('data-size', 'large')
  script.setAttribute('data-radius', '8')
  script.setAttribute('data-request-access', 'write')
  script.setAttribute('data-onauth', 'onTelegramAuth(user)')
  script.async = true
  container.value.appendChild(script)
})

onUnmounted(() => {
  // Clean up the global callback to avoid stale references.
  delete window.onTelegramAuth
})
</script>

<template>
  <!-- Render nothing when the bot username env var is not configured. -->
  <div v-if="botUsername" ref="container" />
</template>
