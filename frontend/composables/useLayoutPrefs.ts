/**
 * useLayoutPrefs — 布局个性化偏好管理。
 *
 * 将侧边栏折叠状态持久化到 localStorage，
 * 确保页面刷新和路由切换后状态不丢失。
 */
export const useLayoutPrefs = () => {
  const STORAGE_KEY = 'layout_prefs'

  interface LayoutPrefs {
    sidebarCollapsed: boolean
  }

  /** 布局偏好默认值 */
  const defaults: LayoutPrefs = {
    sidebarCollapsed: false,
  }

  /** 布局偏好响应式状态（全局单例） */
  const prefs = useState<LayoutPrefs>('layout_prefs', () => ({ ...defaults }))

  /** 从 localStorage 恢复布局偏好，解析失败时静默忽略 */
  const restore = () => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      if (raw) {
        const saved = JSON.parse(raw) as Partial<LayoutPrefs>
        prefs.value = { ...defaults, ...saved }
      }
    } catch { /* 数据损坏时忽略，使用默认值 */ }
  }

  /** 将当前布局偏好持久化到 localStorage */
  const persist = () => {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs.value))
  }

  /**
   * 侧边栏折叠状态（双向绑定）。
   * 写入时自动持久化到 localStorage。
   */
  const sidebarCollapsed = computed({
    get: () => prefs.value.sidebarCollapsed,
    set: (v: boolean) => {
      prefs.value.sidebarCollapsed = v
      persist()
    },
  })

  return {
    sidebarCollapsed,
    restore,
  }
}
