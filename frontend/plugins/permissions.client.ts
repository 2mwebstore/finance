/**
 * After auth is ready, keep the permissions store in sync with /auth/me
 * and optionally refresh the catalog from GET /permissions.
 */
export default defineNuxtPlugin(async () => {
  const { user, authReady, initAuth } = useAuth()
  const store = usePermissionsStore()

  await initAuth()

  watch(
    user,
    async (u) => {
      if (!u) {
        store.clearUser()
        return
      }
      store.setUserGrants([...(u.permissions || [])], !!u.is_super_admin)

      // Catalog sync when user can view roles/permissions
      const canSync =
        u.is_super_admin ||
        (u.permissions || []).includes('roles.view') ||
        (u.permissions || []).includes('permissions.manage')
      if (canSync) {
        const { api } = useApi()
        await store.syncCatalogFromApi(api)
      }
    },
    { immediate: true },
  )

  // Ensure authReady does not block forever
  if (!authReady.value) {
    await initAuth()
  }
})
