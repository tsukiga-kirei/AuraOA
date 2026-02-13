/**
 * useLayoutPrefs — centralized layout personalization state.
 *
 * Persists sidebar collapsed state in localStorage so it survives
 * page navigations and reloads.
 */
export const useLayoutPrefs = () => {
    const STORAGE_KEY = 'layout_prefs'

    interface LayoutPrefs {
        sidebarCollapsed: boolean
    }

    const defaults: LayoutPrefs = {
        sidebarCollapsed: false,
    }

    const prefs = useState<LayoutPrefs>('layout_prefs', () => ({ ...defaults }))

    const restore = () => {
        if (!import.meta.client) return
        try {
            const raw = localStorage.getItem(STORAGE_KEY)
            if (raw) {
                const saved = JSON.parse(raw) as Partial<LayoutPrefs>
                prefs.value = { ...defaults, ...saved }
            }
        } catch { /* ignore corrupt data */ }
    }

    const persist = () => {
        if (!import.meta.client) return
        localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs.value))
    }

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
