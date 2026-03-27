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
  40910: '系统已初始化，无法再次创建管理员',
  50000: '服务器错误，请稍后重试',
}

// --- 解析 JWT payload 中的 exp（秒级时间戳），不验证签名 ---
function parseJwtExp(token: string): number | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')))
    return typeof payload.exp === 'number' ? payload.exp : null
  } catch { return null }
}

/** 从 access token 解析 active_role.role（与网关/后端 TenantContext 一致）；不验证签名 */
function parseJwtActiveRoleRole(tokenVal: string | null): string | null {
  if (!tokenVal) return null
  try {
    const parts = tokenVal.split('.')
    if (parts.length !== 3) return null
    const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')))
    const r = payload?.active_role?.role
    return typeof r === 'string' && r.length > 0 ? r : null
  } catch {
    return null
  }
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

  /** 与后端 JWT 声明一致的身份（优先令牌，避免持久化状态与 token 不同步导致错调租户/平台接口） */
  const effectiveActiveRoleForApi = computed(() => {
    return parseJwtActiveRoleRole(token.value) ?? activeRole.value?.role ?? null
  })
  const userPermissions = useState<PermissionGroup[]>('auth_permissions', () => ['business'])
  const currentUser = useState<PersistedAuthState['current_user']>('auth_user', () => null)
  const userLocale = useState<string>('auth_locale', () => 'zh-CN')

  // =========================================================================
  // 统一 localStorage 读写
  // =========================================================================

  /** 将当前响应式状态序列化到 localStorage（单个 key） */
  const persistState = () => {
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
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem(AUTH_STATE_KEY)
  }

  /** 持久化 token 对（独立 key，高频读写） */
  const persistTokens = () => {
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

  const switchRole = async (roleId: string): Promise<{ ok: boolean; errorMsg?: string }> => {
    try {
      const data = await authFetch<SwitchRoleResponse>('/api/auth/switch-role', {
        method: 'PUT',
        body: { role_id: roleId },
      })

      token.value = data.access_token
      localStorage.setItem('token', data.access_token)

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
      return { ok: true }
    } catch (e: any) {
      return { ok: false, errorMsg: e.message || undefined }
    }
  }

  const login = async (req: LoginRequest): Promise<{ ok: boolean; errorMsg?: string }> => {
    try {
      const res = await $fetch<ApiResponse<LoginResponse>>(`${config.public.apiBase}/api/auth/login`, {
        method: 'POST',
        body: req,
      })

      if (res.code !== 0 || !res.data) {
        const msg = ERROR_CODE_MAP[res.code] || res.message || undefined
        return { ok: false, errorMsg: msg }
      }
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
      return { ok: true }
    } catch (error: any) {
      // 从 $fetch 错误中提取后端返回的业务错误信息
      if (error.data && typeof error.data.code === 'number') {
        const msg = ERROR_CODE_MAP[error.data.code] || error.data.message || undefined
        return { ok: false, errorMsg: msg }
      }
      return { ok: false }
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

  /** 仅清除本地登录态（不调注销接口、不 navigate），用于令牌失效或用户已删除时的中间件处理。 */
  const clearLocalSession = () => {
    token.value = null
    refreshToken.value = null
    menus.value = []
    userRole.value = 'business'
    allRoles.value = []
    activeRole.value = null
    userPermissions.value = ['business']
    currentUser.value = null
    userLocale.value = 'zh-CN'
    meValidateCache.value = null
    clearStorage()
  }

  /** 短期内复用 /me 校验结果，避免每次路由切换都打接口（同 token 约 10s 内只校验一次）。 */
  const meValidateCache = useState<{ tokenPrefix: string; at: number; ok: boolean } | null>(
    'auth_me_validate_cache',
    () => null,
  )

  /**
   * 用当前 access_token 请求 /api/auth/me，确认用户仍存在且令牌可用。
   * 网络异常时返回 true，避免因短暂断网把用户踢下线。
   */
  const validateAccessToken = async (): Promise<boolean> => {
    const t = token.value || localStorage.getItem('token')
    if (!t) return false
    const prefix = t.slice(0, 48)
    const now = Date.now()
    const c = meValidateCache.value
    if (c && c.tokenPrefix === prefix && now - c.at < 10_000) {
      return c.ok
    }
    try {
      const res = await $fetch<ApiResponse<unknown>>(`${config.public.apiBase}/api/auth/me`, {
        headers: { Authorization: `Bearer ${t}` },
      })
      const ok = res.code === 0
      meValidateCache.value = { tokenPrefix: prefix, at: now, ok }
      return ok
    } catch (e: any) {
      const st = e?.statusCode ?? e?.status ?? e?.response?.status
      if (st === 401) {
        meValidateCache.value = { tokenPrefix: prefix, at: now, ok: false }
        return false
      }
      return true
    }
  }

  const logout = async (): Promise<void> => {
    try {
      const baseUrl = String(config.public.apiBase)
      const currentToken = token.value || localStorage.getItem('token')
      await $fetch(`${baseUrl}/api/auth/logout`, {
        method: 'POST',
        headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {},
      })
    } catch { /* 忽略 */ }

    clearLocalSession()
    navigateTo('/login')
  }

  const isAuthenticated = computed(() => !!token.value)

  const doRefreshToken = async (): Promise<boolean> => {
    const rt = refreshToken.value || localStorage.getItem('refresh_token')
    if (!rt) return false

    try {
      const res = await $fetch<ApiResponse<{ access_token: string }>>(`${config.public.apiBase}/api/auth/refresh`, {
        method: 'POST',
        body: { refresh_token: rt },
      })
      if (res.code === 0 && res.data?.access_token) {
        token.value = res.data.access_token
        localStorage.setItem('token', res.data.access_token)
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
        // 先检查是否为业务层面的认证错误（如账户禁用、账户锁定），不走 token 刷新
        if (error.data && typeof error.data.code === 'number' && ERROR_CODE_MAP[error.data.code]) {
          const e = new Error(ERROR_CODE_MAP[error.data.code]) as any
          e.code = error.data.code
          throw e
        }

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

  const updateProfile = async (req: { display_name: string; email: string; phone: string }): Promise<{ ok: boolean; errorMsg?: string }> => {
    try {
      await authFetch('/api/auth/profile', { method: 'PUT', body: req })
      // Sync local state
      if (currentUser.value) {
        currentUser.value = {
          ...currentUser.value,
          display_name: req.display_name || currentUser.value.display_name,
          email: req.email ?? currentUser.value.email,
          phone: req.phone ?? currentUser.value.phone,
        }
        persistState()
      }
      return { ok: true }
    } catch (e: any) {
      return { ok: false, errorMsg: e.message || '更新失败' }
    }
  }

  const updateLocale = async (locale: string): Promise<boolean> => {
    try {
      await authFetch('/api/auth/locale', { method: 'PUT', body: { locale } })
      userLocale.value = locale
      persistState()
      return true
    } catch { return false }
  }

  /** 从后端 /api/auth/me 刷新角色列表，过滤掉已停用租户的角色 */
  const refreshRoles = async (): Promise<void> => {
    try {
      const me = await authFetch<MeResponse>('/api/auth/me')
      if (me.roles) {
        allRoles.value = me.roles.map(r => ({
          id: r.id, role: r.role, tenant_id: r.tenant_id,
          tenant_name: r.tenant_name, label: r.label,
        }))
        persistState()
      }
    } catch { /* 刷新角色失败不影响正常使用 */ }
  }

  // =========================================================================
  // 恢复（页面刷新时从 localStorage 重建状态）
  // =========================================================================

  const restore = () => {
    // 令牌独立恢复
    const savedToken = localStorage.getItem('token')
    if (savedToken) token.value = savedToken
    const savedRefresh = localStorage.getItem('refresh_token')
    if (savedRefresh) refreshToken.value = savedRefresh

    // 从合并 key 恢复
    loadState()

    // 恢复成功后，异步刷新角色列表（过滤已停用租户），不阻塞页面
    if (savedToken) refreshRoles()
  }

  /** 设置 locale 并持久化到 auth_state（不调用后端） */
  const setUserLocale = (locale: string) => {
    userLocale.value = locale
    persistState()
  }

  /**
   * 异步恢复：当 token 丢失但 refresh_token 仍未过期时，尝试用 refresh_token 换取新 token。
   * 返回 true 表示恢复成功（token 已可用），false 表示无法恢复。
   */
  const tryRestoreAsync = async (): Promise<boolean> => {
    if (token.value) return true
    const rt = refreshToken.value || localStorage.getItem('refresh_token')
    if (!rt) return false
    // 检查 refresh_token 是否已过期
    const exp = parseJwtExp(rt)
    if (exp && exp < Date.now() / 1000) return false
    return doRefreshToken()
  }

  /** 判断 refresh_token 是否仍然有效（未过期） */
  const isRefreshTokenValid = (): boolean => {
    const rt = refreshToken.value || localStorage.getItem('refresh_token')
    if (!rt) return false
    const exp = parseJwtExp(rt)
    if (!exp) return false
    return exp > Date.now() / 1000
  }

  return {
    token, refreshToken, menus, userRole, userPermissions, currentUser,
    allRoles, activeRole, effectiveActiveRoleForApi, userLocale,
    login, getMenu, logout, clearLocalSession, validateAccessToken,
    isAuthenticated, restore, tryRestoreAsync, isRefreshTokenValid,
    setUserRole, setUserPermissions, setAllRoles, setActiveRole, switchRole,
    authFetch, doRefreshToken, changePassword, getProfile, updateProfile, updateLocale, setUserLocale,
    refreshRoles,
  }
}
