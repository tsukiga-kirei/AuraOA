import type { LoginRequest, LoginResponse, SwitchRoleResponse, MenuItem, UserRole, PermissionGroup, RoleInfo, MeResponse } from '~/types/auth'

// 统一 API 响应格式
interface ApiResponse<T> {
  code: number
  message: string
  data: T
  trace_id: string
}

// 后端业务错误码到用户友好提示的映射表
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

/**
 * 解析 JWT payload 中的 exp 字段（秒级时间戳），不验证签名。
 * 用于在本地判断 token 是否已过期，避免不必要的网络请求。
 */
function parseJwtExp(token: string): number | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')))
    return typeof payload.exp === 'number' ? payload.exp : null
  } catch { return null }
}

/**
 * 从 access_token 解析 active_role.role 字段，与网关/后端 TenantContext 保持一致。
 * 优先使用 token 中的角色信息，避免持久化状态与 token 不同步导致调用错误的租户/平台接口。
 */
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

// 令牌刷新队列（模块级单例），防止并发请求同时触发多次刷新
let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

// 刷新成功后通知所有等待中的请求使用新 token 重试
function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach(cb => cb(newToken))
  refreshSubscribers = []
}

// 将等待刷新的请求回调加入队列
function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

// 合并存储到 localStorage 的认证状态结构（单 key 减少读写次数）
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

// localStorage 中认证状态的存储键名
const AUTH_STATE_KEY = 'auth_state'

export const useAuth = () => {
  const config = useRuntimeConfig()

  // 当前有效的 access_token（响应式，用于请求头注入）
  const token = useState<string | null>('auth_token', () => null)
  // 用于静默刷新 access_token 的 refresh_token
  const refreshToken = useState<string | null>('auth_refresh', () => null)
  // 后端返回的菜单权限列表（用于侧边栏渲染和路由守卫细粒度校验）
  const menus = useState<MenuItem[]>('auth_menus', () => [])
  // 当前用户的系统角色（business / tenant_admin / system_admin）
  const userRole = useState<UserRole>('auth_role', () => 'business')
  // 用户拥有的所有角色列表（多租户场景下可切换）
  const allRoles = useState<RoleInfo[]>('auth_all_roles', () => [])
  // 当前激活的角色（决定请求时使用哪个租户上下文）
  const activeRole = useState<RoleInfo | null>('auth_active_role', () => null)

  /**
   * 与后端 JWT 声明一致的有效角色标识。
   * 优先从 token payload 解析，避免持久化状态与 token 不同步导致调用错误的接口。
   */
  const effectiveActiveRoleForApi = computed(() => {
    return parseJwtActiveRoleRole(token.value) ?? activeRole.value?.role ?? null
  })

  // 当前用户的权限组列表（用于路由守卫粗粒度检查）
  const userPermissions = useState<PermissionGroup[]>('auth_permissions', () => ['business'])
  // 当前登录用户的基本信息（用于页面展示）
  const currentUser = useState<PersistedAuthState['current_user']>('auth_user', () => null)
  // 当前用户的语言偏好（zh-CN / en-US）
  const userLocale = useState<string>('auth_locale', () => 'zh-CN')

  // =========================================================================
  // localStorage 统一读写
  // =========================================================================

  /** 将当前响应式状态序列化到 localStorage（单个 key，减少存储碎片） */
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

  /** 从 localStorage 恢复状态到响应式变量，返回是否成功读取到数据 */
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

  /** 清除所有持久化的认证数据（token 和状态） */
  const clearStorage = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('refresh_token')
    localStorage.removeItem(AUTH_STATE_KEY)
  }

  /** 持久化 token 对到独立 key（高频读写，与状态分离） */
  const persistTokens = () => {
    if (token.value) localStorage.setItem('token', token.value)
    if (refreshToken.value) localStorage.setItem('refresh_token', refreshToken.value)
  }

  // =========================================================================
  // 状态设置器（同步更新响应式变量并持久化）
  // =========================================================================

  /** 更新用户系统角色并持久化 */
  const setUserRole = (role: UserRole) => {
    userRole.value = role
    persistState()
  }

  /** 更新用户权限组列表并持久化 */
  const setUserPermissions = (perms: PermissionGroup[]) => {
    userPermissions.value = perms
    persistState()
  }

  /** 更新所有角色列表并持久化 */
  const setAllRoles = (roles: RoleInfo[]) => {
    allRoles.value = roles
    persistState()
  }

  /** 切换激活角色，同步更新权限组并持久化 */
  const setActiveRole = (role: RoleInfo) => {
    activeRole.value = role
    userPermissions.value = [role.role]
    persistState()
  }

  // =========================================================================
  // 核心认证方法
  // =========================================================================

  /**
   * 切换当前激活角色（多租户场景）。
   * 后端返回新的 access_token，前端更新 token 和权限状态。
   * @param roleId 目标角色 ID
   * @returns 操作结果，失败时包含错误信息
   */
  const switchRole = async (roleId: string): Promise<{ ok: boolean; errorMsg?: string }> => {
    try {
      const data = await authFetch<SwitchRoleResponse>('/api/auth/switch-role', {
        method: 'PUT',
        body: { role_id: roleId },
      })

      // 更新 access_token（切换角色后 token 中的 active_role 声明已变更）
      token.value = data.access_token
      localStorage.setItem('token', data.access_token)

      // 更新激活角色信息
      activeRole.value = {
        id: data.active_role.id,
        role: data.active_role.role,
        tenant_id: data.active_role.tenant_id,
        tenant_name: data.active_role.tenant_name,
        label: data.active_role.label,
      }

      // 更新权限组（优先使用后端返回的 permissions，降级使用角色标识）
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

  /**
   * 用户登录，成功后初始化所有认证状态。
   * @param req 登录请求（用户名、密码、租户等）
   * @returns 操作结果，失败时包含用户友好的错误信息
   */
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

      // 持久化 token 对
      token.value = data.access_token
      refreshToken.value = data.refresh_token
      persistTokens()

      // 初始化角色列表
      allRoles.value = data.roles.map(r => ({
        id: r.id, role: r.role, tenant_id: r.tenant_id,
        tenant_name: r.tenant_name, label: r.label,
      }))

      // 设置当前激活角色
      activeRole.value = {
        id: data.active_role.id, role: data.active_role.role,
        tenant_id: data.active_role.tenant_id, tenant_name: data.active_role.tenant_name,
        label: data.active_role.label,
      }

      // 初始化用户基本信息
      currentUser.value = {
        username: data.user.username,
        display_name: data.user.display_name,
        tenant_id: data.active_role.tenant_id || '',
        role_label: data.active_role.label,
        email: data.user.email || '',
        phone: data.user.phone || '',
      }

      // 初始化权限组（优先使用后端返回的 permissions）
      userPermissions.value = data.permissions && data.permissions.length > 0
        ? data.permissions as PermissionGroup[]
        : [data.active_role.role] as PermissionGroup[]

      // 从后端用户数据同步语言偏好
      if (data.user.locale) userLocale.value = data.user.locale

      // 异步拉取菜单权限（失败不影响登录流程）
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

  /**
   * 拉取当前用户的菜单权限列表并更新本地状态。
   * 用于角色切换后刷新侧边栏菜单。
   */
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

  /**
   * 仅清除本地登录态，不调用注销接口、不触发页面跳转。
   * 用于令牌失效或用户已被删除时的中间件静默处理。
   */
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

  /**
   * /me 接口校验结果的短期缓存（同 token 前缀 10 秒内复用）。
   * 避免每次路由切换都发起后端请求，减少不必要的网络开销。
   */
  const meValidateCache = useState<{ tokenPrefix: string; at: number; ok: boolean } | null>(
    'auth_me_validate_cache',
    () => null,
  )

  /**
   * 用当前 access_token 请求 /api/auth/me，确认用户仍存在且令牌可用。
   * 网络异常时返回 true，避免因短暂断网把用户踢下线。
   * 仅 401 响应才判定为 token 无效。
   */
  const validateAccessToken = async (): Promise<boolean> => {
    const t = token.value || localStorage.getItem('token')
    if (!t) return false
    // 使用 token 前 48 字符作为缓存键（避免存储完整 token）
    const prefix = t.slice(0, 48)
    const now = Date.now()
    const c = meValidateCache.value
    // 命中缓存则直接返回，避免重复请求
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
      // 非 401 错误（网络超时等）视为校验通过，避免误踢用户
      return true
    }
  }

  /**
   * 用户主动登出：调用后端注销接口使 token 失效，然后清除本地状态并跳转登录页。
   */
  const logout = async (): Promise<void> => {
    try {
      const baseUrl = String(config.public.apiBase)
      const currentToken = token.value || localStorage.getItem('token')
      await $fetch(`${baseUrl}/api/auth/logout`, {
        method: 'POST',
        headers: currentToken ? { Authorization: `Bearer ${currentToken}` } : {},
      })
    } catch { /* 忽略注销接口错误，继续清除本地状态 */ }

    clearLocalSession()
    navigateTo('/login')
  }

  // 是否已登录（基于 access_token 是否存在）
  const isAuthenticated = computed(() => !!token.value)

  /**
   * 使用 refresh_token 向后端换取新的 access_token。
   * 成功后更新本地 token 状态，失败时返回 false。
   */
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

  /**
   * 带认证头的统一请求方法，自动处理 token 刷新和错误映射。
   * - 请求前注入 Authorization 头
   * - 遇到 401 时自动尝试刷新 token 并重试（使用队列防止并发刷新）
   * - 刷新失败则触发登出流程
   * - 业务错误码映射为用户友好的中文提示
   * @param path 请求路径（相对路径或完整 URL）
   * @param options fetch 选项（method、body、headers 等）
   */
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

      // 网络连接失败（非 HTTP 错误）
      const statusCode = error.statusCode || error.status
      if (!statusCode && (error.name === 'FetchError' || error.message === 'fetch failed' || error.cause)) {
        throw new Error('网络连接失败，请检查网络')
      }

      if (statusCode === 401) {
        // 业务层面的认证错误（账户禁用、锁定等），不走 token 刷新流程
        if (error.data && typeof error.data.code === 'number' && ERROR_CODE_MAP[error.data.code]) {
          const e = new Error(ERROR_CODE_MAP[error.data.code]) as any
          e.code = error.data.code
          throw e
        }

        // 已有刷新请求在进行中，将当前请求加入等待队列
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

        // 发起 token 刷新，刷新期间其他 401 请求进入队列
        isRefreshing = true
        const refreshOk = await doRefreshToken()
        isRefreshing = false

        if (refreshOk) {
          // 刷新成功，通知队列中的请求使用新 token 重试
          const newToken = token.value!
          onTokenRefreshed(newToken)
          const retryRes = await doRequest(newToken)
          if (retryRes.code === 0) return retryRes.data
          const msg = ERROR_CODE_MAP[retryRes.code] || retryRes.message || '请求失败'
          const e = new Error(msg) as any; e.code = retryRes.code; throw e
        } else {
          // 刷新失败，清空队列并触发登出
          refreshSubscribers = []
          await logout()
          throw new Error('登录已过期，请重新登录')
        }
      }

      // 其他 HTTP 错误，尝试提取后端业务错误信息
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

  /** 修改当前用户密码，成功返回 true */
  const changePassword = async (req: { current_password: string; new_password: string }): Promise<boolean> => {
    try {
      await authFetch('/api/auth/change-password', { method: 'PUT', body: req })
      return true
    } catch { return false }
  }

  /** 获取当前用户的完整个人信息（含角色、权限等） */
  const getProfile = async (): Promise<MeResponse | null> => {
    try { return await authFetch<MeResponse>('/api/auth/me') }
    catch { return null }
  }

  /**
   * 更新当前用户的个人资料（显示名、邮箱、手机号）。
   * 成功后同步更新本地 currentUser 状态。
   */
  const updateProfile = async (req: { display_name: string; email: string; phone: string }): Promise<{ ok: boolean; errorMsg?: string }> => {
    try {
      await authFetch('/api/auth/profile', { method: 'PUT', body: req })
      // 同步更新本地用户信息缓存
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

  /**
   * 更新当前用户的语言偏好并持久化到后端。
   * @param locale 语言代码（zh-CN / en-US）
   */
  const updateLocale = async (locale: string): Promise<boolean> => {
    try {
      await authFetch('/api/auth/locale', { method: 'PUT', body: { locale } })
      userLocale.value = locale
      persistState()
      return true
    } catch { return false }
  }

  /**
   * 从后端 /api/auth/me 刷新角色列表，过滤掉已停用租户的角色。
   * 在页面恢复时异步调用，不阻塞页面渲染。
   */
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
  // 状态恢复（页面刷新时从 localStorage 重建响应式状态）
  // =========================================================================

  /**
   * 同步恢复认证状态：从 localStorage 读取 token 和用户状态。
   * 在路由守卫最早阶段调用，确保后续逻辑能访问到正确的认证状态。
   * 恢复成功后异步刷新角色列表（过滤已停用租户），不阻塞页面。
   */
  const restore = () => {
    // 独立恢复 token（高频读写，与状态分离存储）
    const savedToken = localStorage.getItem('token')
    if (savedToken) token.value = savedToken
    const savedRefresh = localStorage.getItem('refresh_token')
    if (savedRefresh) refreshToken.value = savedRefresh

    // 从合并 key 恢复其余状态
    loadState()

    // 恢复成功后，异步刷新角色列表，不阻塞页面
    if (savedToken) refreshRoles()
  }

  /** 仅在本地更新语言偏好并持久化（不调用后端接口） */
  const setUserLocale = (locale: string) => {
    userLocale.value = locale
    persistState()
  }

  /**
   * 异步恢复：当 access_token 丢失但 refresh_token 仍未过期时，
   * 尝试用 refresh_token 换取新 token。
   * 返回 true 表示恢复成功（token 已可用），false 表示无法恢复。
   */
  const tryRestoreAsync = async (): Promise<boolean> => {
    if (token.value) return true
    const rt = refreshToken.value || localStorage.getItem('refresh_token')
    if (!rt) return false
    // 本地预检 refresh_token 是否已过期，避免无效请求
    const exp = parseJwtExp(rt)
    if (exp && exp < Date.now() / 1000) return false
    return doRefreshToken()
  }

  /**
   * 判断 refresh_token 是否仍然有效（本地解析 exp，不发网络请求）。
   * 用于路由守卫决定是否尝试静默刷新。
   */
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
