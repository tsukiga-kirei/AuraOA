import type { PermissionGroup } from '~/types/auth'

interface ApiBootstrapRes {
  code: number
  message: string
  data?: { needs_setup: boolean }
  trace_id?: string
}

/**
 * 系统角色级别粗粒度检查：该路径大类是否对当前角色开放。
 */
function hasRoleAccess(path: string, perms: PermissionGroup[]): boolean {
  if (path === '/overview' || path === '/settings' || path === '/login' || path === '/setup') return true
  if (path.startsWith('/admin/system')) return perms.includes('system_admin')
  if (path.startsWith('/admin/tenant')) return perms.includes('tenant_admin')
  if (['/dashboard', '/cron', '/archive'].includes(path)) return perms.includes('business')
  return true
}

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

  restore()

  if (!isAuthenticated.value && isRefreshTokenValid()) {
    await tryRestoreAsync()
  }

  // 本地有 JWT 时向后端校验（用户被删、令牌失效等情况应清掉本地态，避免仍停留在管理台）
  if (isAuthenticated.value) {
    let ok = await validateAccessToken()
    if (!ok) {
      const refreshed = await tryRestoreAsync()
      if (refreshed) {
        ok = await validateAccessToken()
      }
      if (!ok) {
        clearLocalSession()
      }
    }
  }

  if (to.path === '/setup') {
    if (isAuthenticated.value) {
      return navigateTo('/overview')
    }
    if (!(await ensureNeedsSetup())) {
      return navigateTo('/login')
    }
    return
  }

  if (to.path === '/login') {
    if (isAuthenticated.value) {
      return navigateTo('/overview')
    }
    if (await ensureNeedsSetup()) {
      return navigateTo('/setup')
    }
    return
  }

  if (!isAuthenticated.value) {
    if (await ensureNeedsSetup()) {
      return navigateTo('/setup')
    }
    return navigateTo('/login')
  }

  // 第一层：系统角色级别检查
  if (!hasRoleAccess(to.path, userPermissions.value)) {
    return navigateTo('/overview')
  }

  // 第二层：基于后端 menus（org_roles.page_permissions）的细粒度检查
  if (to.path === '/overview' || to.path === '/settings') return

  if (menus.value.length === 0) return

  const allowed = new Set(menus.value.map((m: any) => m.path).filter(Boolean))
  if (!allowed.has(to.path)) {
    return navigateTo('/overview')
  }
})
