import { ROUTE_PERMISSIONS } from '~/utils/permissions'

/**
 * After auth, enforce route-level permissions.
 * Unauthenticated users are handled by auth.global.ts.
 */
export default defineNuxtRouteMiddleware((to) => {
  if (to.path === '/login' || to.path === '/access-denied') return

  const { user, authReady } = useAuth()
  if (!authReady.value || !user.value) return // auth middleware will redirect

  const { canAccessRoute } = usePermissions()

  // Match longest route prefix in ROUTE_PERMISSIONS
  const matched = Object.keys(ROUTE_PERMISSIONS)
    .filter((r) => to.path === r || to.path.startsWith(r + '/'))
    .sort((a, b) => b.length - a.length)[0]

  if (matched && !canAccessRoute(matched)) {
    return navigateTo('/access-denied')
  }
})
