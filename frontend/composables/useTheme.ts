/**
 * useTheme — 主题模式管理（亮色/暗色切换）。
 *
 * 切换时通过全屏遮罩动画实现平滑过渡效果，
 * 主题偏好持久化到 localStorage，页面刷新后自动恢复。
 */

type ThemeMode = 'light' | 'dark'

export const useTheme = () => {
  /** 当前主题模式（响应式，全局单例） */
  const mode = useState<ThemeMode>('theme_mode', () => 'light')
  /** 主题切换动画进行中标志（防止重复触发） */
  const transitioning = useState<boolean>('theme_transitioning', () => false)

  /**
   * 切换亮色/暗色主题，带全屏遮罩淡入淡出动画。
   * 动画进行中时忽略重复调用。
   */
  const toggle = () => {
    if (transitioning.value) return

    transitioning.value = true
    const next: ThemeMode = mode.value === 'light' ? 'dark' : 'light'

    // 创建全屏遮罩层，实现平滑的色彩过渡效果
    const overlay = document.createElement('div')
    overlay.style.cssText = `
      position: fixed; inset: 0; z-index: 99999;
      pointer-events: none;
      background: ${next === 'dark' ? 'rgba(15, 23, 42, 0.45)' : 'rgba(248, 250, 252, 0.55)'};
      opacity: 0;
      transition: opacity 0.45s cubic-bezier(0.4, 0, 0.2, 1);
    `
    document.body.appendChild(overlay)

    // 触发遮罩淡入
    requestAnimationFrame(() => {
      overlay.style.opacity = '1'
    })

    // 遮罩不透明度峰值时切换主题（视觉上无突变感）
    setTimeout(() => {
      mode.value = next
      localStorage.setItem('theme', next)
      document.documentElement.setAttribute('data-theme', next)
    }, 200)

    // 开始遮罩淡出
    setTimeout(() => {
      overlay.style.opacity = '0'
    }, 350)

    // 动画结束后移除遮罩 DOM 节点，重置过渡标志
    setTimeout(() => {
      overlay.remove()
      transitioning.value = false
    }, 800)
  }

  /**
   * 从 localStorage 恢复主题偏好，并同步到 HTML 根元素的 data-theme 属性。
   * 应在应用初始化时调用，避免页面刷新后出现主题闪烁。
   */
  const restore = () => {
    const saved = localStorage.getItem('theme') as ThemeMode | null
    if (saved) {
      mode.value = saved
      document.documentElement.setAttribute('data-theme', saved)
    }
  }

  /** 是否为暗色模式（响应式计算属性） */
  const isDark = computed(() => mode.value === 'dark')

  return { mode, isDark, toggle, restore, transitioning }
}
