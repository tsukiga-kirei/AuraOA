/**
 * Locale index — exports all locale files as a map.
 *
 * To add a new language:
 * 1. Create a new file (e.g. ja-JP.ts) with the same key structure as zh-CN.ts
 * 2. Import and add it to the `messages` map below
 * 3. Add the locale info to `availableLocales` in useI18n.ts
 */
import zhCN from './zh-CN'
import enUS from './en-US'

import type { Locale } from '~/composables/useI18n'

export const messages: Record<Locale, Record<string, string>> = {
    'zh-CN': zhCN,
    'en-US': enUS,
}
