import type { PermissionGroup } from '~/types/auth'

/**
 * 系统角色级别粗粒度检查：该路径大类是否对当前角色开放。
 */
function hasRoleAccess(path: string, perms: PermissionGroup[]): boolean {
  if (path === '/overview' || path === '/settings' || path === '/login') return true
  if (path.startsWith('/admin/system')) return perms.includes('system_admin')
  if (path.startsWith('/admin/tenant')) return perms.includes('tenant_admin')
  if (['/dashboard', '/cron', '/archive'].includes(path)) return perms.includes('business')
  return true
}

export default defineNuxtRouteMiddleware(async (to) => {
  // SSR 端没有 localStorage / token，认证检查仅在客户端执行
  if (import.meta.server) return

  const { isAuthenticated, restore, tryRestoreAsync, userPermissions, menus } = useAuth()
  restore()

  if (to.path === '/login') {
    return isAuthenticated.value ? navigateTo('/overview') : undefined
  }

  // token 不存在时，尝试用 refresh_token 恢复
  if (!isAuthenticated.value) {
    const restored = await tryRestoreAsync()
    if (!restored) return navigateTo('/login')
  }

  // 第一层：系统角色级别检查
  if (!hasRoleAccess(to.path, userPermissions.value)) {
    return navigateTo('/overview')
  }

  // 第二层：基于后端 menus（org_roles.page_permissions）的细粒度检查
  // /overview 和 /settings 始终放行，不依赖 menus
  if (to.path === '/overview' || to.path === '/settings') return

  // menus 未加载时（理论上不会，restore 会从 localStorage 恢复）放行
  if (menus.value.length === 0) return

  const allowed = new Set(menus.value.map((m: any) => m.path).filter(Boolean))
  if (!allowed.has(to.path)) {
    return navigateTo('/overview')
  }
})
