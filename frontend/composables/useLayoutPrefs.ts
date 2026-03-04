/**
 * useLayoutPrefs — 集中布局个性化状态。
 *
 * 在 localStorage 中保留侧边栏折叠状态，以便它能够生存
 * 页面导航和重新加载。*/
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
        try {
            const raw = localStorage.getItem(STORAGE_KEY)
            if (raw) {
                const saved = JSON.parse(raw) as Partial<LayoutPrefs>
                prefs.value = { ...defaults, ...saved }
            }
        } catch { /*忽略损坏的数据*/ }
    }

    const persist = () => {
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
