import type { User, LoginResponse } from '~/types/api'
import { useApiBase } from '~/composables/useApiBase'

const accessToken = ref<string | null>(null)
const currentUser = ref<User | null>(null)
const authReady = ref(false)

export function useAuth() {
  const apiBase = useApiBase()

  async function login(identifier: string, password: string) {
    const res = await $fetch<LoginResponse>(`${apiBase}/auth/login`, {
      method: 'POST',
      body: { identifier, password },
      credentials: 'include',
    })
    accessToken.value = res.access_token
    // Load full profile with permissions
    await fetchMe()
    return res
  }

  async function fetchMe() {
    if (!accessToken.value) {
      currentUser.value = null
      return null
    }
    try {
      const me = await $fetch<User>(`${apiBase}/auth/me`, {
        headers: { Authorization: `Bearer ${accessToken.value}` },
        credentials: 'include',
      })
      currentUser.value = me
      try {
        const store = usePermissionsStore()
        store.setUserGrants(me.permissions || [], !!me.is_super_admin)
      } catch { /* pinia may not be ready in rare cases */ }
      return me
    } catch {
      accessToken.value = null
      currentUser.value = null
      try { usePermissionsStore().clearUser() } catch { /* ignore */ }
      return null
    }
  }

  async function logout() {
    try {
      if (accessToken.value) {
        await $fetch(`${apiBase}/auth/logout`, {
          method: 'POST',
          headers: { Authorization: `Bearer ${accessToken.value}` },
          credentials: 'include',
        })
      }
    } catch {
      // ignore
    }
    accessToken.value = null
    currentUser.value = null
    try { usePermissionsStore().clearUser() } catch { /* ignore */ }
  }

  async function tryRefresh() {
    try {
      const res = await $fetch<{ access_token: string }>(`${apiBase}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      })
      accessToken.value = res.access_token
      await fetchMe()
      return true
    } catch {
      accessToken.value = null
      currentUser.value = null
      return false
    }
  }

  function hasPermission(...codes: string[]) {
    if (!currentUser.value) return false
    if (currentUser.value.is_super_admin) return true
    const perms = currentUser.value.permissions || []
    return codes.some((c) => perms.includes(c))
  }

  function getToken() {
    return accessToken.value
  }

  async function initAuth() {
    if (authReady.value) return
    if (!accessToken.value) {
      await tryRefresh()
    } else {
      await fetchMe()
    }
    authReady.value = true
  }

  return {
    accessToken: readonly(accessToken),
    user: readonly(currentUser),
    authReady: readonly(authReady),
    login,
    logout,
    fetchMe,
    tryRefresh,
    hasPermission,
    getToken,
    initAuth,
  }
}
