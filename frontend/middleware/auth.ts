import { hasPagePermission } from '~/composables/useMockData'
import type { UserRole } from '~/composables/useMockData'

export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/login') return

  const { isAuthenticated, restore, userRole } = useAuth()
  restore()

  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }

  // Check page-level permission
  if (!hasPagePermission(to.path, userRole.value as UserRole)) {
    return navigateTo('/dashboard')
  }
})
