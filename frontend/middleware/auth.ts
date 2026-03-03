import type { PermissionGroup } from '~/types/auth'

/**
 * Check if a path is accessible for the given system-level permission groups.
 * This replaces the old hasPagePermission from useMockData.
 */
function hasPagePermission(path: string, perms: PermissionGroup[]): boolean {
  // /overview and /settings are always accessible
  if (path === '/overview' || path === '/settings' || path === '/login') return true

  if (path.startsWith('/admin/system')) return perms.includes('system_admin')
  if (path.startsWith('/admin/tenant')) return perms.includes('tenant_admin')

  // Business pages: /dashboard, /cron, /archive
  if (['/dashboard', '/cron', '/archive'].includes(path)) return perms.includes('business')

  return true
}

function getDefaultPage(perms: PermissionGroup[]): string {
  return '/overview'
}

export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/login') return

  const { isAuthenticated, restore, userPermissions, currentUser, activeRole } = useAuth()
  restore()

  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }

  // Check system role level permissions (business/tenant_admin/system_admin)
  if (!hasPagePermission(to.path, userPermissions.value)) {
    return navigateTo(getDefaultPage(userPermissions.value))
  }

  // For business users, also check org role page_permissions
  const role = activeRole.value?.role
  if (role === 'business') {
    const { members, roles } = useOrgApi()
    const uname = currentUser.value?.username
    if (uname && members.value.length > 0) {
      const member = members.value.find(m => m.username === uname)
      if (member) {
        const rIds = member.role_ids
        const pagePerms = new Set<string>()
        roles.value.filter(r => rIds.includes(r.id)).forEach(r => r.page_permissions.forEach(p => pagePerms.add(p)))
        // /overview and /settings are always accessible
        if (to.path !== '/overview' && to.path !== '/settings' && !pagePerms.has(to.path)) {
          return navigateTo('/overview')
        }
      }
    }
  }
})
