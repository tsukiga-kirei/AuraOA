import type { LoginRequest, LoginResponse, SwitchRoleResponse, MenuItem, UserRole, PermissionGroup, RoleInfo } from '~/types/auth'


// --- Unified API response format ---
interface ApiResponse<T> {
  code: number
  message: string
  data: T
  trace_id: string
}

// --- Error code → user-friendly message mapping ---
const ERROR_CODE_MAP: Record<number, string> = {
  40103: '用户名或密码错误',
  40104: '账户已锁定，请稍后重试',
  40105: '账户已被禁用',
  40106: '租户不存在或已停用',
  40300: '权限不足',
  40400: '资源不存在',
  50000: '服务器错误，请稍后重试',
}

// --- Token refresh queue (module-level singleton) ---
let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach(cb => cb(newToken))
  refreshSubscribers = []
}

function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

// LoginRequest and LoginResponse are imported from ~/types/auth

export const useAuth = () => {
  const config = useRuntimeConfig()
  const token = useState<string | null>('auth_token', () => null)
  const refreshToken = useState<string | null>('auth_refresh', () => null)
  const menus = useState<MenuItem[]>('auth_menus', () => [])
  const userRole = useState<UserRole>('auth_role', () => 'business')

  /** All role assignments this user has (never modified after login) */
  const allRoles = useState<RoleInfo[]>('auth_all_roles', () => [])
  /** Currently active role assignment */
  const activeRole = useState<RoleInfo | null>('auth_active_role', () => null)
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

  const setAllRoles = (roles: RoleInfo[]) => {
    allRoles.value = roles
    if (import.meta.client) localStorage.setItem('all_roles', JSON.stringify(roles))
  }

  const setActiveRole = (role: RoleInfo) => {
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
    try {
      const data = await authFetch<SwitchRoleResponse>('/api/auth/switch-role', {
        method: 'PUT',
        body: { role_id: roleId },
      })

      // Atomic update — only apply changes after API call succeeds
      token.value = data.access_token
      if (import.meta.client) localStorage.setItem('token', data.access_token)

      const mappedActiveRole: RoleInfo = {
        id: data.active_role.id,
        role: data.active_role.role,
        tenant_id: data.active_role.tenant_id,
        tenant_name: data.active_role.tenant_name,
        label: data.active_role.label,
      }
      setActiveRole(mappedActiveRole)

      userPermissions.value = data.permissions as PermissionGroup[]
      if (import.meta.client) localStorage.setItem('user_permissions', JSON.stringify(data.permissions))

      menus.value = data.menus

      return true
    } catch {
      // On failure, all state remains unchanged
      return false
    }
  }

  const login = async (req: LoginRequest): Promise<boolean> => {
    try {
      const res = await $fetch<ApiResponse<LoginResponse>>(`${config.public.apiBase}/api/auth/login`, {
        method: 'POST',
        body: req,
      })

      if (res.code !== 0 || !res.data) return false

      const data = res.data

      // Store tokens
      token.value = data.access_token
      refreshToken.value = data.refresh_token

      // Map roles from LoginResponse
      const mappedRoles: RoleInfo[] = data.roles.map(r => ({
        id: r.id,
        role: r.role,
        tenant_id: r.tenant_id,
        tenant_name: r.tenant_name,
        label: r.label,
      }))
      setAllRoles(mappedRoles)

      // Map active_role
      const mappedActiveRole: RoleInfo = {
        id: data.active_role.id,
        role: data.active_role.role,
        tenant_id: data.active_role.tenant_id,
        tenant_name: data.active_role.tenant_name,
        label: data.active_role.label,
      }
      setActiveRole(mappedActiveRole)

      // Map user info
      currentUser.value = {
        username: data.user.username,
        display_name: data.user.display_name,
        tenant_id: data.active_role.tenant_id || '',
        role_label: data.active_role.label,
      }

      // Compute full permissions (all unique role types) for backward compat
      const allPerms = [...new Set(data.roles.map(r => r.role))] as PermissionGroup[]
      setFullPermissions(allPerms)

      // Store permissions from backend
      userPermissions.value = data.permissions as PermissionGroup[]

      if (import.meta.client) {
        localStorage.setItem('token', data.access_token)
        localStorage.setItem('refresh_token', data.refresh_token)
        localStorage.setItem('current_user', JSON.stringify(currentUser.value))
        localStorage.setItem('user_permissions', JSON.stringify(data.permissions))
      }

      return true
    } catch {
      return false
    }
  }

  const getMenu = async (): Promise<MenuItem[]> => {
    try {
      const data = await authFetch<{ menus: MenuItem[] }>('/api/auth/menu')
      menus.value = data.menus
      return data.menus
    } catch {
      return []
    }
  }

  const logout = async (): Promise<void> => {
    // Best-effort server-side token invalidation
    try {
      await authFetch('/api/auth/logout', { method: 'POST' })
    } catch {
      // Ignore errors — always proceed with local cleanup
    }

    // Clear all useState auth state
    token.value = null
    refreshToken.value = null
    menus.value = []
    userRole.value = 'business'
    allRoles.value = []
    activeRole.value = null
    fullPermissions.value = ['business']
    userPermissions.value = ['business']
    currentUser.value = null

    // Clear all localStorage auth keys
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

  /**
   * Refresh the access token using the stored refresh_token.
   */
  const doRefreshToken = async (): Promise<boolean> => {
    const rt = refreshToken.value || (import.meta.client ? localStorage.getItem('refresh_token') : null)
    if (!rt) return false

    try {
      const res = await $fetch<ApiResponse<{ access_token: string }>>(`${config.public.apiBase}/api/auth/refresh`, {
        method: 'POST',
        body: { refresh_token: rt },
      })
      if (res.code === 0 && res.data?.access_token) {
        token.value = res.data.access_token
        if (import.meta.client) localStorage.setItem('token', res.data.access_token)
        return true
      }
      return false
    } catch {
      return false
    }
  }

  /**
   * Authenticated fetch wrapper.
   * - Auto-injects Bearer token
   * - Parses unified response { code, message, data }; returns data when code=0
   * - On 401: auto-refreshes token and retries; queues concurrent requests during refresh
   * - On refresh failure: clears auth state and redirects to login
   * - Maps known error codes to user-friendly messages
   */
  async function authFetch<T>(path: string, options?: Record<string, any>): Promise<T> {
    const baseUrl = String(config.public.apiBase)
    const url = path.startsWith('http') ? path : `${baseUrl}${path}`

    const doRequest = (accessToken: string | null) => {
      const headers: Record<string, string> = {
        ...(options?.headers || {}),
      }
      if (accessToken) {
        headers['Authorization'] = `Bearer ${accessToken}`
      }
      return $fetch<ApiResponse<T>>(url, {
        ...options,
        headers,
      })
    }

    try {
      const res = await doRequest(token.value)

      // Unified response: code=0 means success
      if (res.code === 0) return res.data
      // Non-zero code → throw with mapped or original message
      const friendlyMsg = ERROR_CODE_MAP[res.code] || res.message || '请求失败'
      const err = new Error(friendlyMsg) as any
      err.code = res.code
      throw err
    } catch (error: any) {
      // If we threw it ourselves (from code != 0), re-throw as-is
      if (error.code && ERROR_CODE_MAP[error.code]) {
        throw error
      }

      // Network error (no response from server)
      if (error.name === 'FetchError' || error.message === 'fetch failed' || (!error.statusCode && !error.status && error.cause)) {
        throw new Error('网络连接失败，请检查网络')
      }

      // Handle 401 — token expired, attempt refresh
      const statusCode = error.statusCode || error.status
      if (statusCode === 401) {
        // If already refreshing, queue this request
        if (isRefreshing) {
          return new Promise<T>((resolve, reject) => {
            addRefreshSubscriber(async (newToken: string) => {
              try {
                const retryRes = await doRequest(newToken)
                if (retryRes.code === 0) {
                  resolve(retryRes.data)
                } else {
                  const msg = ERROR_CODE_MAP[retryRes.code] || retryRes.message || '请求失败'
                  const e = new Error(msg) as any
                  e.code = retryRes.code
                  reject(e)
                }
              } catch (retryErr) {
                reject(retryErr)
              }
            })
          })
        }

        // Start refresh
        isRefreshing = true
        const refreshOk = await doRefreshToken()
        isRefreshing = false

        if (refreshOk) {
          const newToken = token.value!
          // Notify all queued requests
          onTokenRefreshed(newToken)
          // Retry the original request
          const retryRes = await doRequest(newToken)
          if (retryRes.code === 0) return retryRes.data
          const msg = ERROR_CODE_MAP[retryRes.code] || retryRes.message || '请求失败'
          const e = new Error(msg) as any
          e.code = retryRes.code
          throw e
        } else {
          // Refresh failed — clear state, redirect to login
          refreshSubscribers = []
          await logout()
          throw new Error('登录已过期，请重新登录')
        }
      }

      // Other HTTP errors — try to extract code from response body
      if (error.data && typeof error.data.code === 'number') {
        const friendlyMsg = ERROR_CODE_MAP[error.data.code] || error.data.message || '请求失败'
        const e = new Error(friendlyMsg) as any
        e.code = error.data.code
        throw e
      }

      // Fallback: re-throw original error
      throw error
    }
  }

  const changePassword = async (req: { current_password: string; new_password: string }): Promise<boolean> => {
    try {
      await authFetch('/api/auth/change-password', {
        method: 'PUT',
        body: req,
      })
      return true
    } catch {
      return false
    }
  }

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
    login, getMenu, logout, isAuthenticated, restore,
    setUserRole, setUserPermissions, setFullPermissions, setAllRoles, setActiveRole, switchRole,
    authFetch, doRefreshToken, changePassword,
  }
}
