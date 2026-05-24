import { defineStore } from 'pinia'
import { ref } from 'vue'
import {
  hugsApi,
  balanceApi,
  balanceApiV2,
  leaderboardApi,
  notesApiV2,
  usersApi,
  usersApiV2,
  intimacyApi,
  streaksApi,
  type UserNoteDTO,
} from '@/api/client'
import { useAuthStore } from '@/stores/auth'

export type HugType = 'standard' | 'bear' | 'group' | 'warm' | 'soul' | 'agressive'

export interface HugFeedItem {
  id: string
  giver_id: string
  receiver_id: string
  giver_username: string
  receiver_username: string
  giver_gender?: string | null
  giver_display_name?: string | null
  receiver_display_name?: string | null
  hug_type: HugType
  has_comment?: boolean
  streak_tier?: string
  created_at: string
}

export interface IntimacyInfo {
  raw_score: number
  tier: number
  tier_name: string
  next_tier_at?: number | null
  cooldown_reduction_pct: number
  available_hug_types: HugType[]
  bonus_coins: number
}

export interface ConnectionItem {
  user_id: string
  username: string
  gender?: string | null
  display_name?: string | null
  intimacy: IntimacyInfo
}

export interface IntimacyLeaderboardEntry {
  user_a_id: string
  user_a_username: string
  user_a_display_name?: string | null
  user_b_id: string
  user_b_username: string
  user_b_display_name?: string | null
  raw_score: number
  tier: number
  tier_name: string
}

export interface PendingHugInboxItem {
  id: string
  giver_id: string
  receiver_id: string
  giver_username: string
  giver_gender?: string | null
  giver_display_name?: string | null
  hug_type: HugType
  comment?: string | null
  created_at: string
}

export interface OutgoingPendingHug {
  id: string
  giver_id: string
  receiver_id: string
  receiver_username: string
  receiver_gender?: string | null
  receiver_display_name?: string | null
  hug_type: HugType
  comment?: string | null
  created_at: string
}

export interface HugDetail {
  id: string
  giver_id: string
  receiver_id: string
  giver_username: string
  receiver_username: string
  giver_gender?: string | null
  giver_display_name?: string | null
  receiver_display_name?: string | null
  status: string
  hug_type: HugType
  comment?: string | null
  streak_tier?: string
  created_at: string
  accepted_at?: string | null
}

export interface SlotInfo {
  total_slots: number
  used_slots: number
  next_slot_cost: number | null
}

export interface LeaderboardEntry {
  user_id: string
  username: string
  display_name?: string | null
  tag?: string | null
  special_tag?: string | null
  total_hugs: number
  hugs_given: number
  hugs_received: number
  rank: string
}

export interface UserProfile {
  id: string
  username: string
  display_name?: string | null
  tag?: string | null
  special_tag?: string | null
  role: string
  gender?: string | null
  hugs_given: number
  hugs_received: number
  total_hugs: number
  rank: string
  balance?: number
  mutual_total?: number
  mutual_given?: number
  mutual_received?: number
  is_blocked?: boolean
  intimacy?: IntimacyInfo | null
  captcha_type: 'none' | 'sudoku' | 'casino'
  captcha_cooldown_until?: string | null
}

export interface BlockedUser {
  id: string
  username: string
  display_name?: string | null
  gender?: string | null
  tag?: string | null
  special_tag?: string | null
  created_at: string
}

export interface CooldownInfo {
  giver_id: string
  receiver_id: string
  cooldown_seconds: number
  remaining_seconds: number
  can_hug: boolean
  effective_cooldown_seconds: number
  intimacy_reduction_pct: number
}

export interface Balance {
  user_id: string
  amount: number
}

export interface HugActivityItem {
  timestamp: string
  count: number
}

export interface DailyRewardResponse {
  amount: number
  streak_days: number
  new_balance: number
  already_claimed?: boolean
}

export interface DailyRewardStatus {
  can_claim: boolean
  next_claim_at: string
  streak_days: number
  last_claimed_at?: string | null
}

export interface StreakInfo {
  current_streak: number
  best_streak: number
  tier_level: number
  tier_name: string
  tier_key: string
  next_tier_at?: number | null
  a_hugged_today: boolean
  b_hugged_today: boolean
}

export interface TopStreakEntry {
  user_id: string
  username: string
  display_name?: string | null
  gender?: string | null
  current_streak: number
  best_streak: number
  tier_level: number
  tier_name: string
  tier_key: string
}

export interface StreakCalendarDay {
  date: string
  hug_count: number
  completed: boolean
}

// SearchUser mirrors the v1 UserListItem schema (and the equivalent v2 one).
// Used both for the user-search results and the VIP slot list.
export interface SearchUser {
  id: string
  username: string
  role: string
  gender?: string | null
  display_name?: string | null
  tag?: string | null
  special_tag?: string | null
  is_telegram_linked?: boolean
  is_recently_active?: boolean
  avg_response_time?: number | null
  promoted_until?: string | null
  promotion_message?: string | null
  promotion_bid?: number
  vip_remaining_seconds?: number
  vip_cooldown_until?: string | null
}

export interface PairStreakResponse {
  streak: StreakInfo
  calendar: StreakCalendarDay[]
}

export const useHugsStore = defineStore('hugs', () => {
  const balance = ref<Balance | null>(null)
  const feed = ref<HugFeedItem[]>([])
  const leaderboard = ref<LeaderboardEntry[]>([])
  const history = ref<HugFeedItem[]>([])
  const vips = ref<SearchUser[]>([])
  const loading = ref(false)
  const feedLoading = ref(false)
  const leaderboardLoading = ref(false)

  // Inbox / outgoing state
  const inbox = ref<PendingHugInboxItem[]>([])
  const outgoingHugs = ref<OutgoingPendingHug[]>([])
  const slotInfo = ref<SlotInfo>({ total_slots: 1, used_slots: 0, next_slot_cost: 10 })
  const inboxCount = ref(0)

  // Track timestamps of when a specific user's cooldown needs to be refreshed by HugButton components
  const cooldownRefreshes = ref<Record<string, number>>({})

  function triggerCooldownRefresh(userId: string) {
    cooldownRefreshes.value[userId] = Date.now()
  }

  async function fetchBalance() {
    try {
      const res = await balanceApi.get()
      balance.value = res.data
    } catch {
      // Ignore
    }
  }

  async function claimDailyReward(): Promise<DailyRewardResponse> {
    const res = await balanceApi.claimDaily()
    await fetchBalance()
    return res.data
  }

  async function suggestHug(userId: string, hugType?: string, comment?: string, captchaToken?: string) {
    const res = await hugsApi.suggest(userId, hugType, comment, captchaToken)
    // The suggest endpoint returns receiver_username/receiver_gender directly.
    outgoingHugs.value.unshift({
      id: res.data.id,
      giver_id: res.data.giver_id,
      receiver_id: res.data.receiver_id,
      receiver_username: res.data.receiver_username,
      receiver_gender: res.data.receiver_gender,
      hug_type: res.data.hug_type || 'standard',
      comment: res.data.comment || null,
      created_at: res.data.created_at,
    })
    slotInfo.value.used_slots = outgoingHugs.value.length
    return res.data
  }

  async function acceptHug(hugId: string) {
    // Capture inbox item before it gets removed so we can build a history entry
    const inboxItem = inbox.value.find((h) => h.id === hugId)
    const res = await hugsApi.accept(hugId)
    // Remove from inbox
    inbox.value = inbox.value.filter((h) => h.id !== hugId)
    inboxCount.value = Math.max(0, inboxCount.value - 1)
    // Prepend to history immediately so the user sees the hug right away
    if (inboxItem) {
      const auth = useAuthStore()
      const historyItem: HugFeedItem = {
        id: hugId,
        giver_id: inboxItem.giver_id,
        receiver_id: inboxItem.receiver_id,
        giver_username: inboxItem.giver_username,
        receiver_username: auth.user?.username ?? '',
        giver_gender: inboxItem.giver_gender,
        giver_display_name: inboxItem.giver_display_name,
        receiver_display_name: auth.user?.display_name ?? null,
        hug_type: (res.data.hug_type as HugType) || 'standard',
        has_comment: !!inboxItem.comment,
        created_at: new Date().toISOString(),
      }
      history.value.unshift(historyItem)
    }
    // Refresh balance (both users get +1 coin)
    await fetchBalance()
    return res.data
  }

  async function declineHug(hugId: string) {
    const res = await hugsApi.decline(hugId)
    // Remove from inbox
    inbox.value = inbox.value.filter((h) => h.id !== hugId)
    inboxCount.value = Math.max(0, inboxCount.value - 1)
    return res.data
  }

  async function cancelOutgoing(hugId: string) {
    const res = await hugsApi.cancel(hugId)
    outgoingHugs.value = outgoingHugs.value.filter((h) => h.id !== hugId)
    slotInfo.value.used_slots = outgoingHugs.value.length
    return res.data
  }

  async function fetchInbox() {
    const res = await hugsApi.getInbox()
    inbox.value = res.data || []
    inboxCount.value = inbox.value.length
    return inbox.value
  }

  async function fetchOutgoing() {
    const res = await hugsApi.getOutgoing()
    outgoingHugs.value = res.data?.hugs || []
    if (res.data?.slots) {
      slotInfo.value = res.data.slots
    }
    return outgoingHugs.value
  }

  async function buySlot() {
    const res = await hugsApi.buySlot()
    slotInfo.value = res.data.slots
    balance.value = balance.value
      ? { ...balance.value, amount: res.data.new_balance }
      : { user_id: '', amount: res.data.new_balance }
    return res.data
  }

  async function fetchInboxCount() {
    const res = await hugsApi.getInboxCount()
    inboxCount.value = res.data.count
    return res.data.count
  }

  async function getCooldown(userId: string): Promise<CooldownInfo> {
    const res = await hugsApi.getCooldown(userId)
    return res.data
  }

  async function upgradeCooldown(userId: string): Promise<CooldownInfo> {
    const res = await hugsApi.upgradeCooldown(userId)
    await fetchBalance()
    return res.data
  }

  async function fetchFeed(limit = 50, offset = 0) {
    feedLoading.value = true
    try {
      const res = await hugsApi.getFeed(limit, offset)
      feed.value = res.data || []
    } finally {
      feedLoading.value = false
    }
  }

  async function fetchFeedPage(limit: number, offset: number): Promise<HugFeedItem[]> {
    const res = await hugsApi.getFeed(limit, offset)
    return res.data || []
  }

  async function fetchLeaderboard(limit = 20, offset = 0) {
    leaderboardLoading.value = true
    try {
      const res = await leaderboardApi.get(limit, offset)
      leaderboard.value = res.data || []
    } finally {
      leaderboardLoading.value = false
    }
  }

  async function getHugDetail(hugId: string): Promise<HugDetail> {
    const res = await hugsApi.getDetail(hugId)
    return res.data
  }

  async function getHugHistory() {
    const res = await hugsApi.getHistory()
    history.value = res.data || []
    return history.value
  }

  async function getHugActivity(): Promise<HugActivityItem[]> {
    const res = await hugsApi.getActivity()
    return res.data || []
  }

  async function searchUsers(q = '', limit = 20, offset = 0) {
    // v2 understands the "@username" prefix; v1 stays as legacy.
    const res = await usersApiV2.search(q, limit, offset)
    return res.data || []
  }

  async function fetchVIPs() {
    const res = await usersApi.getVIPs()
    vips.value = res.data || []
    return vips.value
  }

  async function getUserProfile(userId: string): Promise<UserProfile> {
    // v2 accepts a UUID or a username (with or without a leading "@").
    const res = await usersApiV2.getProfile(userId)
    return res.data
  }

  async function getDailyRewardStatus(): Promise<DailyRewardStatus> {
    const res = await balanceApiV2.getDailyRewardStatus()
    return res.data
  }

  // ── Notes (private per-pair, 1 note per (author, target), ≤256 chars) ──

  // Returns null when the user has no note about the target (HTTP 404 is the
  // "no note" signal, not an error worth surfacing).
  async function getNote(usernameOrId: string): Promise<UserNoteDTO | null> {
    try {
      const res = await notesApiV2.get(usernameOrId)
      return res.data
    } catch (e: unknown) {
      const status = (e as { response?: { status?: number } }).response?.status
      if (status === 404) return null
      throw e
    }
  }

  async function upsertNote(usernameOrId: string, content: string): Promise<UserNoteDTO> {
    const res = await notesApiV2.upsert(usernameOrId, content)
    return res.data
  }

  async function deleteNote(usernameOrId: string): Promise<void> {
    await notesApiV2.remove(usernameOrId)
  }

  async function listNotes(limit = 50, offset = 0): Promise<UserNoteDTO[]> {
    const res = await notesApiV2.list(limit, offset)
    return res.data || []
  }

  async function blockUser(userId: string) {
    await usersApi.blockUser(userId)
  }

  async function unblockUser(userId: string) {
    await usersApi.unblockUser(userId)
  }

  async function getBlockedUsers(): Promise<BlockedUser[]> {
    const res = await usersApi.getBlockedUsers()
    return res.data || []
  }

  async function getPairIntimacy(userId: string): Promise<IntimacyInfo> {
    const res = await intimacyApi.getPairIntimacy(userId)
    return res.data
  }

  async function getConnections(limit = 20, offset = 0): Promise<ConnectionItem[]> {
    const res = await intimacyApi.getConnections(limit, offset)
    return res.data || []
  }

  async function getIntimacyLeaderboard(
    limit = 20,
    offset = 0,
  ): Promise<IntimacyLeaderboardEntry[]> {
    const res = await intimacyApi.getLeaderboard(limit, offset)
    return res.data || []
  }

  async function getPairStreak(userId: string): Promise<PairStreakResponse> {
    const res = await streaksApi.getPairStreak(userId)
    return res.data
  }

  async function getTopStreaks(): Promise<TopStreakEntry[]> {
    const res = await streaksApi.getTopStreaks()
    return res.data || []
  }

  return {
    balance,
    feed,
    leaderboard,
    history,
    loading,
    feedLoading,
    leaderboardLoading,
    inbox,
    outgoingHugs,
    slotInfo,
    inboxCount,
    vips,
    cooldownRefreshes,
    triggerCooldownRefresh,
    fetchBalance,
    claimDailyReward,
    suggestHug,
    getHugDetail,
    acceptHug,
    declineHug,
    cancelOutgoing,
    fetchInbox,
    fetchOutgoing,
    buySlot,
    fetchInboxCount,
    getCooldown,
    upgradeCooldown,
    fetchFeed,
    fetchFeedPage,
    fetchLeaderboard,
    getHugHistory,
    getHugActivity,
    searchUsers,
    fetchVIPs,
    getUserProfile,
    getDailyRewardStatus,
    getNote,
    upsertNote,
    deleteNote,
    listNotes,
    blockUser,
    unblockUser,
    getBlockedUsers,
    getPairIntimacy,
    getConnections,
    getIntimacyLeaderboard,
    getPairStreak,
    getTopStreaks,
  }
})
