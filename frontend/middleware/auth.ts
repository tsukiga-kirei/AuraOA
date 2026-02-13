import { hasPagePermission, getDefaultPage } from '~/composables/useMockData'

export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/login') return

  const { isAuthenticated, restore, userPermissions } = useAuth()
  restore()

  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }

  // Check page-level permission using the user's actual permissions array
  if (!hasPagePermission(to.path, userPermissions.value)) {
    // Redirect to the first page the user has access to
    return navigateTo(getDefaultPage(userPermissions.value))
  }
})
