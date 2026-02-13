/**
 * useLayoutPrefs — centralized layout personalization state.
 *
 * Persists sidebar collapsed state (and potentially other prefs)
 * in localStorage so they survive page navigations and reloads.
 *
 * Uses Nuxt `useState` to share across components within the same
 * client-side session, and syncs writes to localStorage.
 */
export const useLayoutPrefs = () => {
    const STORAGE_KEY = 'layout_prefs'

    interface LayoutPrefs {
        sidebarCollapsed: boolean
    }

    const defaults: LayoutPrefs = {
        sidebarCollapsed: false,
    }

    // Shared state across all composable consumers
    const prefs = useState<LayoutPrefs>('layout_prefs', () => ({ ...defaults }))

    /** Read from localStorage and populate state */
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

    /** Persist current state to localStorage */
    const persist = () => {
        if (!import.meta.client) return
        localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs.value))
    }

    // --- Sidebar collapsed ---
    const sidebarCollapsed = computed({
        get: () => prefs.value.sidebarCollapsed,
        set: (v: boolean) => {
            prefs.value.sidebarCollapsed = v
            persist()
        },
    })

    // --- Source Layout Context (for Settings Page) ---
    // Use useState to share state across components immediately
    const sourceLayout = useState<string>('settings_source_layout_state', () => 'default')

    const setSourceLayout = (layout: string) => {
        sourceLayout.value = layout
        if (import.meta.client) {
            localStorage.setItem('settings_source_layout', layout)
        }
    }

    const restoreSourceLayout = () => {
        if (import.meta.client) {
            const stored = localStorage.getItem('settings_source_layout')
            if (stored) sourceLayout.value = stored
        }
    }

    return {
        sidebarCollapsed,
        restore,
        sourceLayout,
        setSourceLayout,
        restoreSourceLayout,
    }
}
