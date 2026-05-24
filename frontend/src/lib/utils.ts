import type { ClassValue } from 'clsx'
import { clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Russian pluralization.
 * Returns the correct form for a number:
 *   plural(1, 'обниманя', 'обнимани', 'обнимань')  → '1 обниманя'
 *   plural(3, 'обниманя', 'обнимани', 'обнимань')  → '3 обнимани'
 *   plural(5, 'обниманя', 'обнимани', 'обнимань')  → '5 обнимань'
 *   plural(21, 'обниманя', 'обнимани', 'обнимань') → '21 обниманя'
 */
/**
 * Gender-aware verb form for Russian.
 * Returns 'обнял' for male, 'обняла' for female, 'обнял(а)' when unknown.
 */
export function hugVerb(gender?: string | null): string {
  if (gender === 'male') return 'обнял'
  if (gender === 'female') return 'обняла'
  return 'обнял(а)'
}

/**
 * Returns the full feed phrase with hug type integrated naturally.
 * e.g. "тепло обнял(а)", "обнял(а) по-медвежьи"
 */
export function hugFeedPhrase(gender?: string | null, hugType?: string): string {
  const verb = hugVerb(gender)
  switch (hugType) {
    case 'bear':
      return `${verb} по-медвежьи`
    case 'warm':
      return `тепло ${verb}`
    case 'group':
      return `${verb} вместе со всеми`
    case 'soul':
      return `по-душевному ${verb}`
    case 'agressive':
      return `агрессивно ${verb}`
    default:
      return verb
  }
}

export function suggestVerb(gender?: string | null): string {
  if (gender === 'male') return 'предложил'
  if (gender === 'female') return 'предложила'
  return 'предложил(а)'
}

/**
 * Returns a natural suggestion phrase for the inbox.
 * e.g. "хочет обнять тебя по-медвежьи", "хочет тепло тебя обнять"
 */
export function hugSuggestionPhrase(hugType?: string): string {
  switch (hugType) {
    case 'bear':
      return 'хочет обнять тебя по-медвежьи'
    case 'warm':
      return 'хочет тепло тебя обнять'
    case 'group':
      return 'хочет обнять тебя вместе со всеми'
    case 'soul':
      return 'хочет обнять тебя по-душевному'
    case 'agressive':
      return 'хочет агрессивно тебя обнять'
    default:
      return 'предлагает обняться'
  }
}

/**
 * Returns a toast message for a completed hug.
 * e.g. "Медвежьи обнимашки с X приняты!", "Обнимашки с X приняты!"
 */
export function hugCompletedToast(username: string, hugType?: string): string {
  switch (hugType) {
    case 'bear':
      return `Медвежьи обнимашки с ${username} приняты!`
    case 'warm':
      return `Тёплые обнимашки с ${username} приняты!`
    case 'group':
      return `Групповые обнимашки с ${username} приняты!`
    case 'soul':
      return `Душевные обнимашки с ${username} приняты!`
    case 'agressive':
      return `Агрессивные обнимашки с ${username} приняты!`
    default:
      return `Обнимашки с ${username} приняты!`
  }
}

/** Map hug type to a Russian label. */
export function hugTypeLabel(hugType: string): string {
  switch (hugType) {
    case 'bear':
      return 'Медвежьи'
    case 'group':
      return 'Групповые'
    case 'warm':
      return 'Тёплые'
    case 'soul':
      return 'Душевные'
    case 'agressive':
      return 'Агрессивные'
    default:
      return 'Обычные'
  }
}

export function plural(n: number, one: string, few: string, many: string): string {
  const abs = Math.abs(n)
  const mod10 = abs % 10
  const mod100 = abs % 100
  if (mod10 === 1 && mod100 !== 11) return `${n} ${one}`
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return `${n} ${few}`
  return `${n} ${many}`
}

// ── Streak tier utilities ──

export interface StreakTierDef {
  key: string
  name: string
  minDays: number
  borderClass: string
  bgClass: string
  textClass: string
  badgeClasses: string
}

export const streakTiers: StreakTierDef[] = [
  {
    key: 'legendary',
    name: 'Легендарная',
    minDays: 90,
    borderClass: 'border-amber-400',
    bgClass: 'bg-amber-400/10',
    textClass: 'text-amber-300',
    badgeClasses: 'bg-amber-400/15 text-amber-300 border-amber-400/20',
  },
  {
    key: 'obsidian',
    name: 'Обсидиановая',
    minDays: 60,
    borderClass: 'border-violet-500',
    bgClass: 'bg-violet-500/10',
    textClass: 'text-violet-400',
    badgeClasses: 'bg-violet-500/15 text-violet-400 border-violet-500/20',
  },
  {
    key: 'diamond',
    name: 'Алмазная',
    minDays: 30,
    borderClass: 'border-cyan-300',
    bgClass: 'bg-cyan-300/10',
    textClass: 'text-cyan-200',
    badgeClasses: 'bg-cyan-300/15 text-cyan-200 border-cyan-300/20',
  },
  {
    key: 'sapphire',
    name: 'Сапфировая',
    minDays: 21,
    borderClass: 'border-blue-500',
    bgClass: 'bg-blue-500/10',
    textClass: 'text-blue-400',
    badgeClasses: 'bg-blue-500/15 text-blue-400 border-blue-500/20',
  },
  {
    key: 'ruby',
    name: 'Рубиновая',
    minDays: 14,
    borderClass: 'border-rose-500',
    bgClass: 'bg-rose-500/10',
    textClass: 'text-rose-400',
    badgeClasses: 'bg-rose-500/15 text-rose-400 border-rose-500/20',
  },
  {
    key: 'emerald',
    name: 'Изумрудная',
    minDays: 7,
    borderClass: 'border-emerald-400',
    bgClass: 'bg-emerald-400/10',
    textClass: 'text-emerald-300',
    badgeClasses: 'bg-emerald-400/15 text-emerald-300 border-emerald-400/20',
  },
]

/** Look up streak tier styling by key. Returns undefined if no tier (empty key). */
export function getStreakTier(key: string | undefined | null): StreakTierDef | undefined {
  if (!key) return undefined
  return streakTiers.find((t) => t.key === key)
}

/** Get the streak tier name in Russian for a given key. */
export function streakTierLabel(key: string | undefined | null): string {
  const tier = getStreakTier(key)
  return tier?.name ?? ''
}

/** Get border class for a streak tier (for feed item left border). */
export function streakTierBorderClass(key: string | undefined | null): string {
  const tier = getStreakTier(key)
  return tier ? `border-l-2 ${tier.borderClass}` : ''
}

export function formatRemainingTime(seconds: number): string {
  if (seconds <= 0) return ''
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  
  if (h > 0) return `${h}ч ${m}м`
  return `${m}:${s.toString().padStart(2, '0')}`
}
