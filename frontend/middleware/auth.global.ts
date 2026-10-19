import { useAuth } from '~/composables/useAuth'

export default defineNuxtRouteMiddleware(async (to) => {
  const publicPaths = ['/login', '/access-denied']
  if (publicPaths.includes(to.path)) return

  // Auth lives in the browser and its refresh token is an HttpOnly cookie.
  // Let the client restore it; an SSR fetch to the external API has no browser cookie.
  if (import.meta.server) return

  const { initAuth, user, authReady } = useAuth()
  if (!authReady.value) {
    await initAuth()
  }
  if (!user.value) {
    return navigateTo('/login')
  }
})
