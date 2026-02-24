import { MOCK_USERS, getMockMenusByRole, getMockMenusByPermissions, getMockMenusByActiveRole, hasPagePermission, getDefaultPage, getDefaultPageForRole } from './useMockData'
import type { MockUser, MockMenuItem, UserRole, PermissionGroup, UserRoleAssignment } from './useMockData'

interface LoginRequest {
  username: string
  password: string
  tenant_id: string
  /** 用户在登录页选择的入口类型，用于决定默认激活哪个角色 */
  preferred_role?: UserRole
}

interface TokenResponse {
  access_token: string
  refresh_token: string
  expires_in: number
}

export type { MockUser, MockMenuItem, UserRole, PermissionGroup, UserRoleAssignment }
export { hasPagePermission, getDefaultPage, getDefaultPageForRole }

export const useAuth = () => {
  const config = useRuntimeConfig()
  const token = useState<string | null>('auth_token', () => null)
  const refreshToken = useState<string | null>('auth_refresh', () => null)
  const menus = useState<MockMenuItem[]>('auth_menus', () => [])
  const userRole = useState<UserRole>('auth_role', () => 'business')

  /** All role assignments this user has (never modified after login) */
  const allRoles = useState<UserRoleAssignment[]>('auth_all_roles', () => [])
  /** Currently active role assignment */
  const activeRole = useState<UserRoleAssignment | null>('auth_active_role', () => null)
  /** Active permission group (derived from activeRole) */
  const userPermissions = useState<PermissionGroup[]>('auth_permissions', () => ['business'])
  /** Full permissions — kept for backward compat but now derived from allRoles */
  const fullPermissions = useState<PermissionGroup[]>('auth_full_permissions', () => ['business'])

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

  const setFullPermissions = (perms: PermissionGroup[]) => {
    fullPermissions.value = perms
    if (import.meta.client) localStorage.setItem('full_permissions', JSON.stringify(perms))
  }

  const setAllRoles = (roles: UserRoleAssignment[]) => {
    allRoles.value = roles
    if (import.meta.client) localStorage.setItem('all_roles', JSON.stringify(roles))
  }

  const setActiveRole = (role: UserRoleAssignment) => {
    activeRole.value = role
    // Derive permissions: only the active role's permission group
    userPermissions.value = [role.role]
    if (import.meta.client) {
      localStorage.setItem('active_role', JSON.stringify(role))
      localStorage.setItem('user_permissions', JSON.stringify([role.role]))
    }
  }

  /** Switch to a specific role by its assignment ID */
  const switchRole = async (roleId: string): Promise<boolean> => {
    const target = allRoles.value.find(r => r.id === roleId)
    if (!target) return false
    setActiveRole(target)
    // Regenerate menus based on the new active role
    if (isMockMode.value) {
      menus.value = getMockMenusByActiveRole(target)
    }
    return true
  }

  const login = async (req: LoginRequest): Promise<boolean> => {
    if (isMockMode.value) {
      const matched = MOCK_USERS.find(
        u => u.username === req.username && u.password === req.password,
      )
      if (!matched) return false

      const mockToken = 'mock_token_' + Date.now()
      token.value = mockToken
      refreshToken.value = 'mock_refresh_' + Date.now()

      // Store all roles from the user
      setAllRoles(matched.roles)

      // 根据用户在登录页选择的入口类型，优先激活匹配的角色
      // 例：用户选了"租户管理员"入口 → 优先激活 tenant_admin 角色
      let defaultRole: UserRoleAssignment | undefined
      if (req.preferred_role) {
        defaultRole = matched.roles.find(r => r.role === req.preferred_role)
      }
      // 回退：如果没有匹配的角色，按优先级选择
      if (!defaultRole) {
        const sysRole = matched.roles.find(r => r.role === 'system_admin')
        const tenantRole = matched.roles.find(r => r.role === 'tenant_admin')
        defaultRole = sysRole || tenantRole || matched.roles[0]
      }
      setActiveRole(defaultRole)

      // Set current user info
      currentUser.value = {
        username: matched.username,
        display_name: matched.display_name,
        tenant_id: defaultRole.tenant_id || '',
        role_label: defaultRole.label,
      }

      // Compute full permissions (all unique role types) for backward compat
      const allPerms = [...new Set(matched.roles.map(r => r.role))] as PermissionGroup[]
      setFullPermissions(allPerms)

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
      // Use activeRole for menu generation
      const role = activeRole.value
      const m = role
        ? getMockMenusByActiveRole(role)
        : getMockMenusByPermissions(userPermissions.value)
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
    allRoles.value = []
    activeRole.value = null
    fullPermissions.value = ['business']
    userPermissions.value = ['business']
    currentUser.value = null
    if (import.meta.client) {
      localStorage.removeItem('token')
      localStorage.removeItem('refresh_token')
      localStorage.removeItem('user_role')
      localStorage.removeItem('user_permissions')
      localStorage.removeItem('full_permissions')
      localStorage.removeItem('all_roles')
      localStorage.removeItem('active_role')
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
      const savedAllRoles = localStorage.getItem('all_roles')
      if (savedAllRoles) {
        try { allRoles.value = JSON.parse(savedAllRoles) } catch { /* ignore */ }
      }
      const savedActiveRole = localStorage.getItem('active_role')
      if (savedActiveRole) {
        try { activeRole.value = JSON.parse(savedActiveRole) } catch { /* ignore */ }
      }
      const savedFullPerms = localStorage.getItem('full_permissions')
      if (savedFullPerms) {
        try { fullPermissions.value = JSON.parse(savedFullPerms) } catch { /* ignore */ }
      }
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
    token, refreshToken, menus, userRole, fullPermissions, userPermissions, currentUser,
    allRoles, activeRole,
    login, getMenu, logout, isAuthenticated, restore, isMockMode,
    setUserRole, setUserPermissions, setFullPermissions, setAllRoles, setActiveRole, switchRole,
    MOCK_USERS,
  }
}
