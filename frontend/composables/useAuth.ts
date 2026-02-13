import { MOCK_USERS, getMockMenusByRole, getMockMenusByPermissions, hasPagePermission, getDefaultPage } from './useMockData'
import type { MockUser, MockMenuItem, UserRole, PermissionGroup } from './useMockData'

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

export type { MockUser, MockMenuItem, UserRole, PermissionGroup }
export { hasPagePermission, getDefaultPage }

export const useAuth = () => {
  const config = useRuntimeConfig()
  const token = useState<string | null>('auth_token', () => null)
  const refreshToken = useState<string | null>('auth_refresh', () => null)
  const menus = useState<MockMenuItem[]>('auth_menus', () => [])
  const userRole = useState<UserRole>('auth_role', () => 'business')
  const userPermissions = useState<PermissionGroup[]>('auth_permissions', () => ['business'])
  const currentUser = useState<{
    username: string
    display_name: string
    tenant_id: string
    role_label: string
  } | null>('auth_user', () => null)

  const isMockMode = computed(() => String(config.public.mockMode) === 'true')

  const setUserRole = (role: UserRole) => {
    userRole.value = role
    if (import.meta.client) localStorage.setItem('user_role', role)
  }

  const setUserPermissions = (perms: PermissionGroup[]) => {
    userPermissions.value = perms
    if (import.meta.client) localStorage.setItem('user_permissions', JSON.stringify(perms))
  }

  const login = async (req: LoginRequest): Promise<boolean> => {
    if (isMockMode.value) {
      const matched = MOCK_USERS.find(
        u => u.username === req.username && u.password === req.password
          && (req.tenant_id === u.tenant_id || req.tenant_id === 'default'),
      )
      if (!matched) return false

      const mockToken = 'mock_token_' + Date.now()
      token.value = mockToken
      refreshToken.value = 'mock_refresh_' + Date.now()
      currentUser.value = {
        username: matched.username,
        display_name: matched.display_name,
        tenant_id: matched.tenant_id,
        role_label: matched.role_label,
      }
      // Store permissions from the matched user
      setUserPermissions(matched.permissions)
      if (import.meta.client) {
        localStorage.setItem('token', mockToken)
        localStorage.setItem('refresh_token', refreshToken.value!)
        localStorage.setItem('current_user', JSON.stringify(currentUser.value))
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

  const getMenu = async (): Promise<MockMenuItem[]> => {
    if (isMockMode.value) {
      const m = getMockMenusByPermissions(userPermissions.value)
      menus.value = m
      return m
    }
    try {
      const data = await $fetch<{ menus: MockMenuItem[] }>(`${config.public.apiBase}/api/auth/menu`, {
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
    userPermissions.value = ['business']
    currentUser.value = null
    if (import.meta.client) {
      localStorage.removeItem('token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('user_role')
      localStorage.removeItem('user_permissions')
      localStorage.removeItem('current_user')
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
      const savedPerms = localStorage.getItem('user_permissions')
      if (savedPerms) {
        try { userPermissions.value = JSON.parse(savedPerms) } catch { /* ignore */ }
      }
      const savedUser = localStorage.getItem('current_user')
      if (savedUser) {
        try { currentUser.value = JSON.parse(savedUser) } catch { /* ignore */ }
      }
    }
  }

  return {
    token, refreshToken, menus, userRole, userPermissions, currentUser,
    login, getMenu, logout, isAuthenticated, restore, isMockMode, setUserRole, setUserPermissions,
    MOCK_USERS,
  }
}
