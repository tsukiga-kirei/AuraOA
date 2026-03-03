import { hasPagePermission, getDefaultPage } from '~/composables/useMockData'

export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/login') return

  const { isAuthenticated, restore, userPermissions, currentUser, activeRole } = useAuth()
  restore()

  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }

  //检查系统角色级权限（business/tenant_admin/system_admin）
  if (!hasPagePermission(to.path, userPermissions.value)) {
    return navigateTo(getDefaultPage(userPermissions.value))
  }

  //对于业务用户，还可以从组织数据中检查业务角色 page_permissions
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
        ///overview 和 /settings 始终可访问
        if (to.path !== '/overview' && to.path !== '/settings' && !pagePerms.has(to.path)) {
          return navigateTo('/overview')
        }
      }
    }
  }
})
