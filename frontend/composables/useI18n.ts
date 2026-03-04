/**
 * useI18n — 集中国际化可组合项。
 *
 * 语言偏好由 useAuth 的 userLocale 统一管理（持久化在 auth_state 中）。
 * 所有 UI 标签都要经过 t() 进行翻译。
 */

import { messages } from '~/locales'

export type Locale = 'zh-CN' | 'en-US'

export const useI18n = () => {
  const { userLocale } = useAuth()

  const t = (key: string, values?: string | number | (string | number)[]): string => {
    const loc = (userLocale.value || 'zh-CN') as Locale
    let text = messages[loc]?.[key]

    if (!text) {
      if (typeof values === 'string') return values
      return key
    }

    let args: (string | number)[] = []
    if (Array.isArray(values)) {
      args = values
    } else if (values !== undefined && values !== null) {
      if (text.includes('{0}')) args = [values]
    }

    if (args.length > 0) {
      args.forEach((val, idx) => {
        text = text.replace(new RegExp(`\\{${idx}\\}`, 'g'), String(val))
      })
    }

    return text
  }

  const setLocale = (locale: Locale) => {
    userLocale.value = locale
  }

  const locale = computed(() => (userLocale.value || 'zh-CN') as Locale)

  const availableLocales: { value: Locale; label: string; flag: string }[] = [
    { value: 'zh-CN', label: '简体中文', flag: '🇨🇳' },
    { value: 'en-US', label: 'English', flag: '🇺🇸' },
  ]

  return { t, locale, setLocale, currentLocale: userLocale, availableLocales }
}
