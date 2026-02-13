type ThemeMode = 'light' | 'dark'

export const useTheme = () => {
  const mode = useState<ThemeMode>('theme_mode', () => 'light')
  const transitioning = useState<boolean>('theme_transitioning', () => false)

  const toggle = () => {
    if (transitioning.value) return
    if (!import.meta.client) return

    transitioning.value = true
    const next: ThemeMode = mode.value === 'light' ? 'dark' : 'light'

    // Create a full-screen overlay for smooth color wash
    const overlay = document.createElement('div')
    overlay.style.cssText = `
      position: fixed; inset: 0; z-index: 99999;
      pointer-events: none;
      background: ${next === 'dark' ? 'rgba(15, 23, 42, 0.45)' : 'rgba(248, 250, 252, 0.55)'};
      opacity: 0;
      transition: opacity 0.45s cubic-bezier(0.4, 0, 0.2, 1);
    `
    document.body.appendChild(overlay)

    // Trigger overlay fade-in
    requestAnimationFrame(() => {
      overlay.style.opacity = '1'
    })

    // At peak of overlay, swap the theme
    setTimeout(() => {
      mode.value = next
      localStorage.setItem('theme', next)
      document.documentElement.setAttribute('data-theme', next)
    }, 200)

    // Fade out overlay
    setTimeout(() => {
      overlay.style.opacity = '0'
    }, 350)

    // Cleanup
    setTimeout(() => {
      overlay.remove()
      transitioning.value = false
    }, 800)
  }

  const restore = () => {
    if (import.meta.client) {
      const saved = localStorage.getItem('theme') as ThemeMode | null
      if (saved) {
        mode.value = saved
        document.documentElement.setAttribute('data-theme', saved)
      }
    }
  }

  const isDark = computed(() => mode.value === 'dark')

  return { mode, isDark, toggle, restore, transitioning }
}
