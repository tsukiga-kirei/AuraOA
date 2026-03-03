import type { LoginRequest, LoginResponse, SwitchRoleResponse, MenuItem, UserRole, PermissionGroup, RoleInfo } from '~/types/auth'


//---统一API响应格式---
interface ApiResponse<T> {
  code: number
  message: string
  data: T
  trace_id: string
}

//--- 错误代码 → 用户友好的消息映射 ---
const ERROR_CODE_MAP: Record<number, string> = {
  40103: '用户名或密码错误',
  40104: '账户已锁定，请稍后重试',
  40105: '账户已被禁用',
  40106: '租户不存在或已停用',
  40300: '权限不足',
  40400: '资源不存在',
  50000: '服务器错误，请稍后重试',
}

//--- 令牌刷新队列（模块级单例）---
let isRefreshing = false
let refreshSubscribers: Array<(token: string) => void> = []

function onTokenRefreshed(newToken: string) {
  refreshSubscribers.forEach(cb => cb(newToken))
  refreshSubscribers = []
}

function addRefreshSubscriber(cb: (token: string) => void) {
  refreshSubscribers.push(cb)
}

//LoginRequest 和 LoginResponse 是从 ~/types/auth 导入的

export const useAuth = () => {
  const config = useRuntimeConfig()
  const token = useState<string | null>('auth_token', () => null)
  const refreshToken = useState<string | null>('auth_refresh', () => null)
  const menus = useState<MenuItem[]>('auth_menus', () => [])
  const userRole = useState<UserRole>('auth_role', () => 'business')

  /** 该用户拥有的所有角色分配（登录后从未修改）*/
  const allRoles = useState<RoleInfo[]>('auth_all_roles', () => [])
  /** 当前活跃的角色分配*/
  const activeRole = useState<RoleInfo | null>('auth_active_role', () => null)
  /** Active权限组（派生自activeRole）*/
  const userPermissions = useState<PermissionGroup[]>('auth_permissions', () => ['business'])
  /** 完整权限 - 保留用于向后兼容，但现在派生自 allRoles*/
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
    //派生权限：仅活动角色的权限组
    userPermissions.value = [role.role]
    if (import.meta.client) {
      localStorage.setItem('active_role', JSON.stringify(role))
      localStorage.setItem('user_permissions', JSON.stringify([role.role]))
    }
  }

  /** 通过分配ID切换到特定角色*/
  const switchRole = async (roleId: string): Promise<boolean> => {
    try {
      const data = await authFetch<SwitchRoleResponse>('/api/auth/switch-role', {
        method: 'PUT',
        body: { role_id: roleId },
      })

      //原子更新——仅在 API 调用成功后应用更改
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

      const switchPerms = data.permissions && data.permissions.length > 0
        ? data.permissions as PermissionGroup[]
        : [data.active_role.role] as PermissionGroup[]
      userPermissions.value = switchPerms
      if (import.meta.client) localStorage.setItem('user_permissions', JSON.stringify(switchPerms))

      menus.value = data.menus

      return true
    } catch {
      //失败时，所有状态保持不变
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

      //存储代币
      token.value = data.access_token
      refreshToken.value = data.refresh_token

      //从 LoginResponse 映射角色
      const mappedRoles: RoleInfo[] = data.roles.map(r => ({
        id: r.id,
        role: r.role,
        tenant_id: r.tenant_id,
        tenant_name: r.tenant_name,
        label: r.label,
      }))
      setAllRoles(mappedRoles)

      //映射 active_role
      const mappedActiveRole: RoleInfo = {
        id: data.active_role.id,
        role: data.active_role.role,
        tenant_id: data.active_role.tenant_id,
        tenant_name: data.active_role.tenant_name,
        label: data.active_role.label,
      }
      setActiveRole(mappedActiveRole)

      //映射用户信息
      currentUser.value = {
        username: data.user.username,
        display_name: data.user.display_name,
        tenant_id: data.active_role.tenant_id || '',
        role_label: data.active_role.label,
      }

      //计算向后兼容的完整权限（所有独特的角色类型）
      const allPerms = [...new Set(data.roles.map(r => r.role))] as PermissionGroup[]
      setFullPermissions(allPerms)

      // Use backend permissions if available, otherwise fall back to active role
      const effectivePerms = data.permissions && data.permissions.length > 0
        ? data.permissions as PermissionGroup[]
        : [data.active_role.role] as PermissionGroup[]
      userPermissions.value = effectivePerms

      if (import.meta.client) {
        localStorage.setItem('token', data.access_token)
        localStorage.setItem('refresh_token', data.refresh_token)
        localStorage.setItem('current_user', JSON.stringify(currentUser.value))
        localStorage.setItem('user_permissions', JSON.stringify(effectivePerms))
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
    //尽力而为的服务器端令牌失效
    try {
      await authFetch('/api/auth/logout', { method: 'POST' })
    } catch {
      //忽略错误 - 始终继续进行本地清理
    }

    //清除所有 useState 身份验证状态
    token.value = null
    refreshToken.value = null
    menus.value = []
    userRole.value = 'business'
    allRoles.value = []
    activeRole.value = null
    fullPermissions.value = ['business']
    userPermissions.value = ['business']
    currentUser.value = null

    //清除所有 localStorage 身份验证密钥
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
   * 使用存储的refresh_token刷新访问令牌。*/
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
   * 经过身份验证的获取包装器。
   * - 自动注入不记名令牌
   * - 解析统一响应 { code, message, data };当code=0时返回数据
   * - 在 401 上：自动刷新令牌并重试；刷新期间对并发请求进行排队
   * - 刷新失败时：清除身份验证状态并重定向到登录
   * - 将已知错误代码映射到用户友好的消息*/
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

      //统一响应：code=0表示成功
      if (res.code === 0) return res.data
      //非零代码 → 抛出映射或原始消息
      const friendlyMsg = ERROR_CODE_MAP[res.code] || res.message || '请求失败'
      const err = new Error(friendlyMsg) as any
      err.code = res.code
      throw err
    } catch (error: any) {
      //如果我们自己抛出它（从代码！= 0），则按原样重新抛出
      if (error.code && ERROR_CODE_MAP[error.code]) {
        throw error
      }

      //网络错误（服务器无响应）
      if (error.name === 'FetchError' || error.message === 'fetch failed' || (!error.statusCode && !error.status && error.cause)) {
        throw new Error('网络连接失败，请检查网络')
      }

      //句柄 401 — 令牌过期，尝试刷新
      const statusCode = error.statusCode || error.status
      if (statusCode === 401) {
        //如果已经刷新，则将此请求排队
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

        //开始刷新
        isRefreshing = true
        const refreshOk = await doRefreshToken()
        isRefreshing = false

        if (refreshOk) {
          const newToken = token.value!
          //通知所有排队的请求
          onTokenRefreshed(newToken)
          //重试原始请求
          const retryRes = await doRequest(newToken)
          if (retryRes.code === 0) return retryRes.data
          const msg = ERROR_CODE_MAP[retryRes.code] || retryRes.message || '请求失败'
          const e = new Error(msg) as any
          e.code = retryRes.code
          throw e
        } else {
          //刷新失败——清除状态，重定向到登录
          refreshSubscribers = []
          await logout()
          throw new Error('登录已过期，请重新登录')
        }
      }

      //其他 HTTP 错误 — 尝试从响应正文中提取代码
      if (error.data && typeof error.data.code === 'number') {
        const friendlyMsg = ERROR_CODE_MAP[error.data.code] || error.data.message || '请求失败'
        const e = new Error(friendlyMsg) as any
        e.code = error.data.code
        throw e
      }

      //Fallback：重新抛出原始错误
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
        try { allRoles.value = JSON.parse(savedAllRoles) } catch { /*忽略*/ }
      }
      const savedActiveRole = localStorage.getItem('active_role')
      if (savedActiveRole) {
        try { activeRole.value = JSON.parse(savedActiveRole) } catch { /*忽略*/ }
      }
      const savedFullPerms = localStorage.getItem('full_permissions')
      if (savedFullPerms) {
        try { fullPermissions.value = JSON.parse(savedFullPerms) } catch { /*忽略*/ }
      }
      const savedPerms = localStorage.getItem('user_permissions')
      if (savedPerms) {
        try { userPermissions.value = JSON.parse(savedPerms) } catch { /*忽略*/ }
      }
      const savedUser = localStorage.getItem('current_user')
      if (savedUser) {
        try { currentUser.value = JSON.parse(savedUser) } catch { /*忽略*/ }
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
