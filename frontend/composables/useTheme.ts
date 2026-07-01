/**
 * useTheme — 主题模式管理（亮色 / 暗色 / 暖色三主题切换）。
 *
 * 切换时通过全屏遮罩动画实现平滑过渡效果，
 * 主题偏好持久化到 localStorage，页面刷新后自动恢复。
 */

export type ThemeMode = 'light' | 'dark' | 'warm'

const THEME_CYCLE: ThemeMode[] = ['light', 'dark', 'warm']

const THEME_OVERLAY: Record<ThemeMode, string> = {
  light: 'rgba(244, 246, 251, 0.32)',
  dark: 'rgba(12, 18, 34, 0.48)',
  warm: 'rgba(250, 248, 244, 0.58)',
}

export const useTheme = () => {
  /** 当前主题模式（响应式，全局单例） */
  const mode = useState<ThemeMode>('theme_mode', () => 'light')
  /** 主题切换动画进行中标志（防止重复触发） */
  const transitioning = useState<boolean>('theme_transitioning', () => false)

  const applyTheme = (next: ThemeMode) => {
    mode.value = next
    localStorage.setItem('theme', next)
    document.documentElement.setAttribute('data-theme', next)
  }

  const nextTheme = (current: ThemeMode): ThemeMode => {
    const idx = THEME_CYCLE.indexOf(current)
    return THEME_CYCLE[(idx + 1) % THEME_CYCLE.length]
  }

  /**
   * 循环切换 light → dark → warm → light，带全屏遮罩淡入淡出动画。
   */
  const toggle = () => {
    if (transitioning.value) return

    transitioning.value = true
    const next = nextTheme(mode.value)

    const overlay = document.createElement('div')
    overlay.style.cssText = `
      position: fixed; inset: 0; z-index: 99999;
      pointer-events: none;
      background: ${THEME_OVERLAY[next]};
      opacity: 0;
      transition: opacity 0.45s cubic-bezier(0.4, 0, 0.2, 1);
    `
    document.body.appendChild(overlay)

    requestAnimationFrame(() => {
      overlay.style.opacity = '1'
    })

    setTimeout(() => {
      applyTheme(next)
    }, 200)

    setTimeout(() => {
      overlay.style.opacity = '0'
    }, 350)

    setTimeout(() => {
      overlay.remove()
      transitioning.value = false
    }, 800)
  }

  /**
   * 直接设置指定主题（无动画），用于主题选择器。
   */
  const setTheme = (next: ThemeMode) => {
    if (mode.value === next) return
    applyTheme(next)
  }

  /**
   * 从 localStorage 恢复主题偏好，并同步到 HTML 根元素的 data-theme 属性。
   */
  const restore = () => {
    const saved = localStorage.getItem('theme') as ThemeMode | null
    if (saved && THEME_CYCLE.includes(saved)) {
      applyTheme(saved)
    }
  }

  /** 是否为暗色模式（响应式计算属性） */
  const isDark = computed(() => mode.value === 'dark')

  /** 是否为暖色主题 */
  const isWarm = computed(() => mode.value === 'warm')

  return { mode, isDark, isWarm, toggle, setTheme, restore, transitioning }
}
