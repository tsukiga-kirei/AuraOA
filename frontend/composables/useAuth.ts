interface LoginRequest {
  username: string
  password: string
  tenant_id: string
}

interface TokenResponse {
  access_token: string
  refresh_token: string
  expires_in: number
}

interface MenuItem {
  key: string
  label: string
  icon?: string
  path: string
  children?: MenuItem[]
}

type UserRole = 'business' | 'tenant_admin' | 'system_admin'

export const useAuth = () => {
  const config = useRuntimeConfig()
  const token = useState<string | null>('auth_token', () => null)
  const refreshToken = useState<string | null>('auth_refresh', () => null)
  const menus = useState<MenuItem[]>('auth_menus', () => [])
  const userRole = useState<UserRole>('auth_role', () => 'business')

  const isMockMode = computed(() => config.public.mockMode === true || config.public.mockMode === 'true')

  const setUserRole = (role: UserRole) => {
    userRole.value = role
    if (import.meta.client) {
      localStorage.setItem('user_role', role)
    }
  }

  const login = async (req: LoginRequest): Promise<boolean> => {
    if (isMockMode.value) {
      const mockToken = 'mock_token_' + Date.now()
      token.value = mockToken
      refreshToken.value = 'mock_refresh_' + Date.now()
      if (import.meta.client) {
        localStorage.setItem('token', mockToken)
        localStorage.setItem('refresh_token', refreshToken.value)
      }
      return true
    }

    try {
      const data = await $fetch<TokenResponse>(`${config.public.apiBase}/api/auth/login`, {
        method: 'POST',
        body: req,
      })
      token.value = data.access_token
      refreshToken.value = data.refresh_token
      if (import.meta.client) {
        localStorage.setItem('token', data.access_token)
        localStorage.setItem('refresh_token', data.refresh_token)
      }
      return true
    } catch {
      return false
    }
  }

  const getMenu = async (): Promise<MenuItem[]> => {
    if (isMockMode.value) {
      const mockMenus: MenuItem[] = [
        { key: 'dashboard', label: '审核工作台', path: '/dashboard' },
        { key: 'cron', label: '定时任务', path: '/cron' },
        { key: 'archive', label: '归档复盘', path: '/archive' },
        { key: 'tenant', label: '租户配置', path: '/admin/tenant' },
        { key: 'system', label: '系统管理', path: '/admin/system' },
        { key: 'monitor', label: '全局监控', path: '/admin/monitor' },
      ]
      menus.value = mockMenus
      return mockMenus
    }

    try {
      const data = await $fetch<{ menus: MenuItem[] }>(`${config.public.apiBase}/api/auth/menu`, {
        headers: { Authorization: `Bearer ${token.value}` },
      })
      menus.value = data.menus
      return data.menus
    } catch {
      return []
    }
  }

  const logout = () => {
    token.value = null
    refreshToken.value = null
    menus.value = []
    userRole.value = 'business'
    if (import.meta.client) {
      localStorage.removeItem('token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('user_role')
    }
    navigateTo('/login')
  }

  const isAuthenticated = computed(() => !!token.value)

  const restore = () => {
    if (import.meta.client) {
      const saved = localStorage.getItem('token')
      if (saved) token.value = saved
      const savedRefresh = localStorage.getItem('refresh_token')
      if (savedRefresh) refreshToken.value = savedRefresh
      const savedRole = localStorage.getItem('user_role') as UserRole | null
      if (savedRole) userRole.value = savedRole
    }
  }

  return {
    token, refreshToken, menus, userRole,
    login, getMenu, logout, isAuthenticated, restore, isMockMode, setUserRole,
  }
}
