type ThemeMode = 'light' | 'dark'

export const useTheme = () => {
  const mode = useState<ThemeMode>('theme_mode', () => 'light')
  const transitioning = useState<boolean>('theme_transitioning', () => false)

  const toggle = () => {
    if (transitioning.value) return

    transitioning.value = true
    const next: ThemeMode = mode.value === 'light' ? 'dark' : 'light'

    //创建全屏叠加以实现平滑的色彩清洗
    const overlay = document.createElement('div')
    overlay.style.cssText = `
      position: fixed; inset: 0; z-index: 99999;
      pointer-events: none;
      background: ${next === 'dark' ? 'rgba(15, 23, 42, 0.45)' : 'rgba(248, 250, 252, 0.55)'};
      opacity: 0;
      transition: opacity 0.45s cubic-bezier(0.4, 0, 0.2, 1);
    `
    document.body.appendChild(overlay)

    //触发叠加淡入
    requestAnimationFrame(() => {
      overlay.style.opacity = '1'
    })

    //在叠加的高峰期，交换主题
    setTimeout(() => {
      mode.value = next
      localStorage.setItem('theme', next)
      document.documentElement.setAttribute('data-theme', next)
    }, 200)

    //淡出叠加
    setTimeout(() => {
      overlay.style.opacity = '0'
    }, 350)

    //清理
    setTimeout(() => {
      overlay.remove()
      transitioning.value = false
    }, 800)
  }

  const restore = () => {
    const saved = localStorage.getItem('theme') as ThemeMode | null
    if (saved) {
      mode.value = saved
      document.documentElement.setAttribute('data-theme', saved)
    }
  }

  const isDark = computed(() => mode.value === 'dark')

  return { mode, isDark, toggle, restore, transitioning }
}
