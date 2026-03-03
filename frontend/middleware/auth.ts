import { hasPagePermission, getDefaultPage } from '~/composables/useMockData'

export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/login') return

  const { isAuthenticated, restore, userPermissions, currentUser, activeRole } = useAuth()
  restore()

  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }

  // Check system-role-level permission (business / tenant_admin / system_admin)
  if (!hasPagePermission(to.path, userPermissions.value)) {
    return navigateTo(getDefaultPage(userPermissions.value))
  }

  // For business users, also check business-role page_permissions from org data
  const role = activeRole.value?.role
  if (role === 'business') {
    const { members, roles } = useOrgApi()
    const uname = currentUser.value?.username
    if (uname) {
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
