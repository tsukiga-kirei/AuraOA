import type { LoginRequest, LoginResponse, SwitchRoleResponse, MenuItem, UserRole, PermissionGroup, RoleInfo, MeResponse } from '~/types/auth'

// --- 统一 API 响应格式 ---
interface ApiResponse<T> {
  code: number
  message: string
  data: T
  trace_id: string
}

// --- 错误代码 → 用户友好的消息映射 ---
const ERROR_CODE_MAP: Record<number, string> = {
  40103: '用户名或密码错误',
  40104: '账户已锁定，请稍后重试',
  40105: '账户已被禁用',
  40106: '租户不存在或已停用',
  40300: '权限不足',
  40400: '资源不存在',
  50000: '服务器错误，请稍后重试',
}

// --- 令牌刷新队列（模块级单例）---
let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach(cb => cb(newToken))
  refreshSubscribers = []
}

function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

// --- 合并的 localStorage 状态结构 ---
interface PersistedAuthState {
  user_role: UserRole
  user_permissions: PermissionGroup[]
  all_roles: RoleInfo[]
  active_role: RoleInfo | null
  current_user: {
    username: string
    display_name: string
    tenant_id: string
    role_label: string
    email: string
    phone: string
  } | null
  menus: MenuItem[]
  locale: string
}

const AUTH_STATE_KEY = 'auth_state'

export const useAuth = () => {
  const config = useRuntimeConfig()
  const token = useState<string | null>('auth_token', () => null)
  const refreshToken = useState<string | null>('auth_refresh', () => null)
  const menus = useState<MenuItem[]>('auth_menus', () => [])
  const userRole = useState<UserRole>('auth_role', () => 'business')
  const allRoles = useState<RoleInfo[]>('auth_all_roles', () => [])
  const activeRole = useState<RoleInfo | null>('auth_active_role', () => null)
  const userPermissions = useState<PermissionGroup[]>('auth_permissions', () => ['business'])
  const currentUser = useState<PersistedAuthState['current_user']>('auth_user', () => null)
  const userLocale = useState<string>('auth_locale', () => 'zh-CN')

  // =========================================================================
  // 统一 localStorage 读写
  // =========================================================================

  /** 将当前响应式状态序列化到 localStorage（单个 key） */
  const persistState = () => {
    if (!import.meta.client) return
    const state: PersistedAuthState = {
      user_role: userRole.value,
      user_permissions: userPermissions.value,
      all_roles: allRoles.value,
      active_role: activeRole.value,
      current_user: currentUser.value,
      menus: menus.value,
      locale: userLocale.value,
    }
    localStorage.setItem(AUTH_STATE_KEY, JSON.stringify(state))
  }

  /** 从 localStorage 恢复状态到响应式变量 */
  const loadState = (): boolean => {
    if (!import.meta.client) return false
    const raw = localStorage.getItem(AUTH_STATE_KEY)
    if (!raw) return false
    try {
      const state: PersistedAuthState = JSON.parse(raw)
      if (state.user_role) userRole.value = state.user_role
      if (state.user_permissions) userPermissions.value = state.user_permissions
      if (state.all_roles) allRoles.value = state.all_roles
      if (state.active_role !== undefined) activeRole.value = state.active_role
      if (state.current_user !== undefined) currentUser.value = state.current_user
      if (state.menus) menus.value = state.menus
      if (state.locale) userLocale.value = state.locale
      return true
    } catch { return false }
  }

  /** 清除所有持久化的认证数据 */
  const clearStorage = () => {
    if (!import.meta.client) return
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem(AUTH_STATE_KEY)
    // 兼容：清除旧版分散 key（升级过渡期）
    ;['user_role', 'user_permissions', 'all_roles', 'active_role',
      'current_user', 'auth_menus', 'app_locale'].forEach(k => localStorage.removeItem(k))
  }

  /** 持久化 token 对（独立 key，高频读写） */
  const persistTokens = () => {
    if (!import.meta.client) return
    if (token.value) localStorage.setItem('token', token.value)
    if (refreshToken.value) localStorage.setItem('refresh_token', refreshToken.value)
  }

  // =========================================================================
  // 状态设置器（更新响应式 + 持久化）
  // =========================================================================

  const setUserRole = (role: UserRole) => {
    userRole.value = role
    persistState()
  }

  const setUserPermissions = (perms: PermissionGroup[]) => {
    userPermissions.value = perms
    persistState()
  }

  const setAllRoles = (roles: RoleInfo[]) => {
    allRoles.value = roles
    persistState()
  }

  const setActiveRole = (role: RoleInfo) => {
    activeRole.value = role
    userPermissions.value = [role.role]
    persistState()
  }

  // =========================================================================
  // 核心认证方法
  // =========================================================================

  const switchRole = async (roleId: string): Promise<boolean> => {
    try {
      const data = await authFetch<SwitchRoleResponse>('/api/auth/switch-role', {
        method: 'PUT',
        body: { role_id: roleId },
      })

      token.value = data.access_token
      if (import.meta.client) localStorage.setItem('token', data.access_token)

      activeRole.value = {
        id: data.active_role.id,
        role: data.active_role.role,
        tenant_id: data.active_role.tenant_id,
        tenant_name: data.active_role.tenant_name,
        label: data.active_role.label,
      }

      const switchPerms = data.permissions && data.permissions.length > 0
        ? data.permissions as PermissionGroup[]
        : [data.active_role.role] as PermissionGroup[]
      userPermissions.value = switchPerms

      menus.value = data.menus
      persistState()
      return true
    } catch {
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

      // 令牌
      token.value = data.access_token
      refreshToken.value = data.refresh_token
      persistTokens()

      // 角色
      allRoles.value = data.roles.map(r => ({
        id: r.id, role: r.role, tenant_id: r.tenant_id,
        tenant_name: r.tenant_name, label: r.label,
      }))

      activeRole.value = {
        id: data.active_role.id, role: data.active_role.role,
        tenant_id: data.active_role.tenant_id, tenant_name: data.active_role.tenant_name,
        label: data.active_role.label,
      }

      // 用户信息
      currentUser.value = {
        username: data.user.username,
        display_name: data.user.display_name,
        tenant_id: data.active_role.tenant_id || '',
        role_label: data.active_role.label,
        email: data.user.email || '',
        phone: data.user.phone || '',
      }

      // 权限
      userPermissions.value = data.permissions && data.permissions.length > 0
        ? data.permissions as PermissionGroup[]
        : [data.active_role.role] as PermissionGroup[]

      // locale 从后端用户数据获取
      if (data.user.locale) userLocale.value = data.user.locale

      // 拉取菜单
      try {
        const menuData = await authFetch<{ menus: MenuItem[] }>('/api/auth/menu')
        menus.value = menuData.menus
      } catch { /* 菜单加载失败不影响登录 */ }

      persistState()
      return true
    } catch {
      return false
    }
  }

  const getMenu = async (): Promise<MenuItem[]> => {
    try {
      const data = await authFetch<{ menus: MenuItem[] }>('/api/auth/menu')
      menus.value = data.menus
      persistState()
      return data.menus
    } catch {
      return []
    }
  }

  const logout = async (): Promise<void> => {
    try {
      const baseUrl = String(config.public.apiBase)
      const currentToken = token.value || (import.meta.client ? localStorage.getItem('token') : null)
      await $fetch(`${baseUrl}/api/auth/logout`, {
        method: 'POST',
        headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {},
      })
    } catch { /* 忽略 */ }

    // 清除响应式状态
    token.value = null
    refreshToken.value = null
    menus.value = []
    userRole.value = 'business'
    allRoles.value = []
    activeRole.value = null
    userPermissions.value = ['business']
    currentUser.value = null
    userLocale.value = 'zh-CN'

    clearStorage()
    navigateTo('/login')
  }

  const isAuthenticated = computed(() => !!token.value)

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
      console.warn('[auth] refresh token response not ok:', res.code, res.message)
      return false
    } catch (e) {
      console.warn('[auth] refresh token failed:', e)
      return false
    }
  }

  // =========================================================================
  // authFetch — 带自动刷新的认证请求包装器
  // =========================================================================

  async function authFetch<T>(path: string, options?: Record<string, any>): Promise<T> {
    const baseUrl = String(config.public.apiBase)
    const url = path.startsWith('http') ? path : `${baseUrl}${path}`

    const doRequest = (accessToken: string | null) => {
      const headers: Record<string, string> = { ...(options?.headers || {}) }
      if (accessToken) headers['Authorization'] = `Bearer ${accessToken}`
      return $fetch<ApiResponse<T>>(url, { ...options, headers })
    }

    try {
      const res = await doRequest(token.value)
      if (res.code === 0) return res.data
      const friendlyMsg = ERROR_CODE_MAP[res.code] || res.message || '请求失败'
      const err = new Error(friendlyMsg) as any
      err.code = res.code
      throw err
    } catch (error: any) {
      if (error.code && ERROR_CODE_MAP[error.code]) throw error

      const statusCode = error.statusCode || error.status
      if (!statusCode && (error.name === 'FetchError' || error.message === 'fetch failed' || error.cause)) {
        throw new Error('网络连接失败，请检查网络')
      }

      if (statusCode === 401) {
        if (isRefreshing) {
          return new Promise<T>((resolve, reject) => {
            addRefreshSubscriber(async (newToken: string) => {
              try {
                const retryRes = await doRequest(newToken)
                if (retryRes.code === 0) resolve(retryRes.data)
                else {
                  const msg = ERROR_CODE_MAP[retryRes.code] || retryRes.message || '请求失败'
                  const e = new Error(msg) as any; e.code = retryRes.code; reject(e)
                }
              } catch (retryErr) { reject(retryErr) }
            })
          })
        }

        isRefreshing = true
        const refreshOk = await doRefreshToken()
        isRefreshing = false

        if (refreshOk) {
          const newToken = token.value!
          onTokenRefreshed(newToken)
          const retryRes = await doRequest(newToken)
          if (retryRes.code === 0) return retryRes.data
          const msg = ERROR_CODE_MAP[retryRes.code] || retryRes.message || '请求失败'
          const e = new Error(msg) as any; e.code = retryRes.code; throw e
        } else {
          refreshSubscribers = []
          await logout()
          throw new Error('登录已过期，请重新登录')
        }
      }

      if (error.data && typeof error.data.code === 'number') {
        const friendlyMsg = ERROR_CODE_MAP[error.data.code] || error.data.message || '请求失败'
        const e = new Error(friendlyMsg) as any; e.code = error.data.code; throw e
      }

      throw error
    }
  }

  // =========================================================================
  // 辅助方法
  // =========================================================================

  const changePassword = async (req: { current_password: string; new_password: string }): Promise<boolean> => {
    try {
      await authFetch('/api/auth/change-password', { method: 'PUT', body: req })
      return true
    } catch { return false }
  }

  const getProfile = async (): Promise<MeResponse | null> => {
    try { return await authFetch<MeResponse>('/api/auth/me') }
    catch { return null }
  }

  const updateLocale = async (locale: string): Promise<boolean> => {
    try {
      await authFetch('/api/auth/locale', { method: 'PUT', body: { locale } })
      userLocale.value = locale
      persistState()
      return true
    } catch { return false }
  }

  // =========================================================================
  // 恢复（页面刷新时从 localStorage 重建状态）
  // =========================================================================

  const restore = () => {
    if (!import.meta.client) return

    // 令牌独立恢复
    const savedToken = localStorage.getItem('token')
    if (savedToken) token.value = savedToken
    const savedRefresh = localStorage.getItem('refresh_token')
    if (savedRefresh) refreshToken.value = savedRefresh

    // 尝试从合并 key 恢复
    if (loadState()) return

    // 兼容旧版分散 key（升级过渡期）
    const savedRole = localStorage.getItem('user_role') as UserRole | null
    if (savedRole) userRole.value = savedRole
    try { const v = localStorage.getItem('all_roles'); if (v) allRoles.value = JSON.parse(v) } catch {}
    try { const v = localStorage.getItem('active_role'); if (v) activeRole.value = JSON.parse(v) } catch {}
    try { const v = localStorage.getItem('user_permissions'); if (v) userPermissions.value = JSON.parse(v) } catch {}
    try { const v = localStorage.getItem('current_user'); if (v) currentUser.value = JSON.parse(v) } catch {}
    try { const v = localStorage.getItem('auth_menus'); if (v) menus.value = JSON.parse(v) } catch {}
    const savedLocale = localStorage.getItem('app_locale')
    if (savedLocale) userLocale.value = savedLocale

    // 迁移：写入合并 key，清除旧 key
    persistState()
    ;['user_role', 'user_permissions', 'all_roles', 'active_role',
      'current_user', 'auth_menus', 'app_locale'].forEach(k => localStorage.removeItem(k))
  }

  /** 设置 locale 并持久化到 auth_state（不调用后端） */
  const setUserLocale = (locale: string) => {
    userLocale.value = locale
    persistState()
  }

  /**
   * 异步恢复：当 token 丢失但 refresh_token 仍在时，尝试用 refresh_token 换取新 token。
   * 返回 true 表示恢复成功（token 已可用），false 表示无法恢复。
   */
  const tryRestoreAsync = async (): Promise<boolean> => {
    if (token.value) return true
    const rt = refreshToken.value || (import.meta.client ? localStorage.getItem('refresh_token') : null)
    if (!rt) return false
    return doRefreshToken()
  }

  return {
    token, refreshToken, menus, userRole, userPermissions, currentUser,
    allRoles, activeRole, userLocale,
    login, getMenu, logout, isAuthenticated, restore, tryRestoreAsync,
    setUserRole, setUserPermissions, setAllRoles, setActiveRole, switchRole,
    authFetch, doRefreshToken, changePassword, getProfile, updateLocale, setUserLocale,
  }
}
