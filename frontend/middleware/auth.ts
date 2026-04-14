import type { PermissionGroup } from '~/types/auth'

// 后端引导状态接口响应格式
interface ApiBootstrapRes {
  code: number
  message: string
  data?: { needs_setup: boolean }
  trace_id?: string
}

/**
 * 系统角色级别粗粒度检查：判断当前路径是否对该角色开放。
 * - /overview、/settings、/login、/setup 对所有人开放
 * - /admin/system 仅系统管理员可访问
 * - /admin/tenant 仅租户管理员可访问
 * - /dashboard、/cron、/archive 仅业务用户可访问
 */
function hasRoleAccess(path: string, perms: PermissionGroup[]): boolean {
  if (path === '/overview' || path === '/settings' || path === '/login' || path === '/setup') return true
  if (path.startsWith('/admin/system')) return perms.includes('system_admin')
  if (path.startsWith('/admin/tenant')) return perms.includes('tenant_admin')
  if (['/dashboard', '/cron', '/archive'].includes(path)) return perms.includes('business')
  return true
}

/**
 * 全局路由守卫中间件，处理以下逻辑：
 * 1. 从 localStorage 恢复本地登录态
 * 2. access_token 缺失但 refresh_token 有效时，尝试静默刷新
 * 3. 已登录时向后端校验 token 有效性（用户被删、令牌吊销等场景）
 * 4. 处理 /setup 和 /login 页面的跳转逻辑
 * 5. 未登录用户重定向到登录或初始化页面
 * 6. 第一层：系统角色级别权限检查
 * 7. 第二层：基于后端菜单权限的细粒度页面访问控制
 */
export default defineNuxtRouteMiddleware(async (to) => {
  const config = useRuntimeConfig()
  const {
    isAuthenticated,
    restore,
    tryRestoreAsync,
    isRefreshTokenValid,
    validateAccessToken,
    clearLocalSession,
    userPermissions,
    menus,
  } = useAuth()

  // 缓存系统初始化状态，避免同一次导航多次请求后端
  let cachedNeedsSetup: boolean | null = null
  const ensureNeedsSetup = async (): Promise<boolean> => {
    if (cachedNeedsSetup !== null) return cachedNeedsSetup
    try {
      const res = await $fetch<ApiBootstrapRes>(`${String(config.public.apiBase)}/api/auth/bootstrap-status`)
      cachedNeedsSetup = res.code === 0 && res.data?.needs_setup === true
    } catch {
      cachedNeedsSetup = false
    }
    return cachedNeedsSetup
  }

  // 第一步：从 localStorage 同步恢复 token 和用户状态
  restore()

  // 第二步：access_token 缺失但 refresh_token 仍有效时，尝试静默换取新 token
  if (!isAuthenticated.value && isRefreshTokenValid()) {
    await tryRestoreAsync()
  }

  // 第三步：本地有 JWT 时向后端校验（用户被删、令牌失效等情况应清掉本地态，避免仍停留在管理台）
  if (isAuthenticated.value) {
    let ok = await validateAccessToken()
    if (!ok) {
      // 校验失败时尝试用 refresh_token 换取新 access_token 后再次校验
      const refreshed = await tryRestoreAsync()
      if (refreshed) {
        ok = await validateAccessToken()
      }
      // 刷新后仍无效，清除本地登录态
      if (!ok) {
        clearLocalSession()
      }
    }
  }

  // 处理 /setup 页面：已登录则跳转概览，系统未初始化才允许访问
  if (to.path === '/setup') {
    if (isAuthenticated.value) {
      return navigateTo('/overview')
    }
    if (!(await ensureNeedsSetup())) {
      return navigateTo('/login')
    }
    return
  }

  // 处理 /login 页面：已登录则跳转概览，系统未初始化则跳转 setup
  if (to.path === '/login') {
    if (isAuthenticated.value) {
      return navigateTo('/overview')
    }
    if (await ensureNeedsSetup()) {
      return navigateTo('/setup')
    }
    return
  }

  // 未登录用户访问受保护页面：根据系统状态决定跳转目标
  if (!isAuthenticated.value) {
    if (await ensureNeedsSetup()) {
      return navigateTo('/setup')
    }
    return navigateTo('/login')
  }

  // 第一层：系统角色级别检查（粗粒度，基于 JWT 权限组）
  if (!hasRoleAccess(to.path, userPermissions.value)) {
    return navigateTo('/overview')
  }

  // 第二层：基于后端 menus（org_roles.page_permissions）的细粒度检查
  // /overview 和 /settings 为通用页面，无需菜单权限校验
  if (to.path === '/overview' || to.path === '/settings') return

  // 菜单尚未加载时放行（等待后续加载后再校验）
  if (menus.value.length === 0) return

  // 构建允许访问的路径集合，不在集合内则重定向到概览
  const allowed = new Set(menus.value.map((m: any) => m.path).filter(Boolean))
  if (!allowed.has(to.path)) {
    return navigateTo('/overview')
  }
})
