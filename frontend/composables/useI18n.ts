/**
 * useI18n — Centralized internationalization composable.
 *
 * Translation files are stored in ~/locales/ directory:
 *   - locales/zh-CN.ts  (简体中文)
 *   - locales/en-US.ts  (English)
 *   - locales/index.ts  (aggregator)
 *
 * Language preference is persisted in localStorage.
 * All UI labels should go through t() for translation.
 *
 * To add a new language:
 *   1. Create locales/xx-YY.ts with same key structure
 *   2. Register in locales/index.ts
 *   3. Add to availableLocales below
 */

import { messages } from '~/locales'

export type Locale = 'zh-CN' | 'en-US'

/** Global reactive locale state (shared across the app) */
const currentLocale = ref<Locale>('zh-CN')
let initialized = false

export const useI18n = () => {
  // Initialize from localStorage once
  if (!initialized && import.meta.client) {
    const saved = localStorage.getItem('app_locale') as Locale | null
    if (saved && (saved === 'zh-CN' || saved === 'en-US')) {
      currentLocale.value = saved
    }
    initialized = true
  }

  /** Translate a key, with optional fallback */
  /** Translate a key, with optional interpolation values or fallback */
  /** Translate a key, with optional interpolation values or fallback */
  const t = (key: string, values?: string | number | (string | number)[]): string => {
    let text = messages[currentLocale.value]?.[key]

    // If text not found
    if (!text) {
      // If values is a string, treat it as fallback
      if (typeof values === 'string') return values
      return key
    }

    // Determine arguments for interpolation
    let args: (string | number)[] = []

    if (Array.isArray(values)) {
      args = values
    } else if (values !== undefined && values !== null) {
      // If single value provided and text has placeholders, use it for interpolation
      // Otherwise ignore it (it might be a fallback string provided by mistake)
      if (text.includes('{0}')) {
        args = [values]
      }
    }

    // Perform interpolation
    if (args.length > 0) {
      args.forEach((val, idx) => {
        text = text.replace(new RegExp(`\\{${idx}\\}`, 'g'), String(val))
      })
    }

    return text
  }

  /** Set locale and persist */
  const setLocale = (locale: Locale) => {
    currentLocale.value = locale
    if (import.meta.client) {
      localStorage.setItem('app_locale', locale)
    }
  }

  /** Get the current locale */
  const locale = computed(() => currentLocale.value)

  /** Get available locales */
  const availableLocales: { value: Locale; label: string; flag: string }[] = [
    { value: 'zh-CN', label: '简体中文', flag: '🇨🇳' },
    { value: 'en-US', label: 'English', flag: '🇺🇸' },
  ]

  return {
    t,
    locale,
    setLocale,
    currentLocale,
    availableLocales,
  }
}
