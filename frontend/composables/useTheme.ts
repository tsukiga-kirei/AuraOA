type ThemeMode = 'light' | 'dark'

export const useTheme = () => {
  const mode = useState<ThemeMode>('theme_mode', () => 'light')

  const toggle = () => {
    mode.value = mode.value === 'light' ? 'dark' : 'light'
    if (import.meta.client) {
      localStorage.setItem('theme', mode.value)
      document.documentElement.setAttribute('data-theme', mode.value)
    }
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

  return { mode, isDark, toggle, restore }
}
