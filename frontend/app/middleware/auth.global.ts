const PUBLIC_ROUTES = new Set(['/login', '/auth/callback'])

function normalizePath(path: string) {
  return path.length > 1 ? path.replace(/\/+$/, '') : path
}

export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthStore()
  const path = normalizePath(to.path)

  if (!auth.initialized) {
    await auth.fetchMe()
  }

  if (!auth.isAuthenticated && !PUBLIC_ROUTES.has(path)) {
    return navigateTo('/login')
  }

  if (auth.isAuthenticated && path === '/login') {
    return navigateTo('/')
  }
})
