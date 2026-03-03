/**
 * useI18n — 集中国际化可组合项。
 *
 * 翻译文件存放在~/locales/目录下：
 * - locales/zh-CN.ts (简体中文)
 * - locales/en-US.ts（英语）
 * - locales/index.ts（聚合器）
 *
 * 语言首选项保留在 localStorage 中。
 * ��有UI标签都要经过t()进行翻译。
 *
 * 添加新语言：
 * 1. 创建具有相同密钥结构的 locales/xx-YY.ts
 * 2.在locales/index.ts中注册
 * 3.添加下面的availableLocales*/

import { messages } from '~/locales'

export type Locale = 'zh-CN' | 'en-US'

/** 全局响应式语言环境状态（在应用程序中共享）*/
const currentLocale = ref<Locale>('zh-CN')
let initialized = false

export const useI18n = () => {
  //从 localStorage 初始化一次
  if (!initialized && import.meta.client) {
    const saved = localStorage.getItem('app_locale') as Locale | null
    if (saved && (saved === 'zh-CN' || saved === 'en-US')) {
      currentLocale.value = saved
    }
    initialized = true
  }

  /** 翻译一个键，带有可选的后备*/
  /** 转换键，使用可选的插值或后备*/
  /** 转换键，使用可选的插值或后备*/
  const t = (key: string, values?: string | number | (string | number)[]): string => {
    let text = messages[currentLocale.value]?.[key]

    //如果找不到文本
    if (!text) {
      //如果值是字符串，则将其视为后备
      if (typeof values === 'string') return values
      return key
    }

    //确定插值参数
    let args: (string | number)[] = []

    if (Array.isArray(values)) {
      args = values
    } else if (values !== undefined && values !== null) {
      //如果提供单个值并且文本具有占位符，则将其用于插值
      //否则忽略它（它可能是错误提供的后备字符串）
      if (text.includes('{0}')) {
        args = [values]
      }
    }

    //执行插值
    if (args.length > 0) {
      args.forEach((val, idx) => {
        text = text.replace(new RegExp(`\\{${idx}\\}`, 'g'), String(val))
      })
    }

    return text
  }

  /** 设置语言环境并保留*/
  const setLocale = (locale: Locale) => {
    currentLocale.value = locale
    if (import.meta.client) {
      localStorage.setItem('app_locale', locale)
    }
  }

  /** 获取当前区域设置*/
  const locale = computed(() => currentLocale.value)

  /** 获取可用的语言环境*/
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
