export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/login') return

  const { isAuthenticated, restore } = useAuth()
  restore()

  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }
})
