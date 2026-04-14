/**
 * useI18n — 国际化工具函数封装。
 *
 * 语言偏好由 useAuth 的 userLocale 统一管理（持久化在 auth_state 中）。
 * 所有 UI 文案均通过 t() 函数翻译，支持占位符替换。
 */

import { messages } from '~/locales'

export type Locale = 'zh-CN' | 'en-US'

export const useI18n = () => {
  const { userLocale } = useAuth()

  /**
   * 根据当前语言获取指定键的翻译文案，支持 {0}、{1} 占位符替换。
   * @param key 翻译键名
   * @param values 占位符替换值（单个值或数组）
   * @returns 翻译后的文案，键不存在时返回 key 本身
   */
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

  /**
   * 检查当前语言下是否存在指定翻译键（避免回退到原始 key 时的误判）。
   * @param key 翻译键名
   * @returns 存在且非空则返回 true
   */
  const te = (key: string): boolean => {
    const loc = (userLocale.value || 'zh-CN') as Locale
    const text = messages[loc]?.[key]
    return typeof text === 'string' && text.length > 0
  }

  /** 切换当前语言 */
  const setLocale = (locale: Locale) => {
    userLocale.value = locale
  }

  /** 当前语言（响应式计算属性） */
  const locale = computed(() => (userLocale.value || 'zh-CN') as Locale)

  /** 支持的语言列表（含显示名和国旗） */
  const availableLocales: { value: Locale; label: string; flag: string }[] = [
    { value: 'zh-CN', label: '简体中文', flag: '🇨🇳' },
    { value: 'en-US', label: 'English', flag: '🇺🇸' },
  ]

  return { t, te, locale, setLocale, currentLocale: userLocale, availableLocales }
}
