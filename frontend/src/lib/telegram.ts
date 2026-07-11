// The signed user object Telegram's Login Widget passes to its auth callback.
// See https://core.telegram.org/widgets/login
export interface TelegramWidgetUser {
  id: number
  first_name: string
  last_name?: string
  username?: string
  photo_url?: string
  auth_date: number
  hash: string
}
