import { defineStore } from 'pinia'
import { PERMISSION_CATALOG, type PermissionCode } from '~/utils/permissions'

export interface CatalogPermission {
  id?: string
  code: string
  name: string
  module: string
  description?: string
}

/**
 * Permissions store.
 * - userCodes: from GET /auth/me (always after login)
 * - catalog: static fallback + auto-sync from GET /api/v1/permissions when allowed
 */
export const usePermissionsStore = defineStore('permissions', {
  state: () => ({
    /** Permission codes granted to the current user */
    userCodes: [] as string[],
    isSuperAdmin: false,
    /** Full system catalog (static seed, then API overwrite) */
    catalog: PERMISSION_CATALOG.map((c) => ({
      code: c.code,
      name: c.name,
      module: c.module,
      description: c.description,
    })) as CatalogPermission[],
    catalogSynced: false,
    catalogSyncError: null as string | null,
    lastSyncedAt: null as string | null,
  }),

  getters: {
    codeSet: (s) => new Set(s.userCodes),
    hasAny: (s) => {
      return (...codes: string[]) => {
        if (s.isSuperAdmin) return true
        if (!codes.length) return true
        return codes.some((c) => s.userCodes.includes(c))
      }
    },
    hasAll: (s) => {
      return (...codes: string[]) => {
        if (s.isSuperAdmin) return true
        return codes.every((c) => s.userCodes.includes(c))
      }
    },
    modules: (s) => {
      const set = new Set(s.catalog.map((c) => c.module))
      return Array.from(set).sort()
    },
  },

  actions: {
    /** Apply user grants from /auth/me */
    setUserGrants(codes: string[], isSuperAdmin: boolean) {
      this.userCodes = [...codes]
      this.isSuperAdmin = isSuperAdmin
    },

    clearUser() {
      this.userCodes = []
      this.isSuperAdmin = false
    },

    /**
     * Fetch full permission catalog from API.
     * Requires roles.view (or super admin). Fails softly for other roles.
     */
    async syncCatalogFromApi(api: <T>(path: string, opts?: object) => Promise<T>) {
      try {
        const res = await api<{ items: CatalogPermission[] }>('/permissions')
        if (res.items?.length) {
          this.catalog = res.items.map((p) => ({
            id: p.id,
            code: p.code,
            name: p.name,
            module: p.module || 'Other',
            description: p.description || '',
          }))
          this.catalogSynced = true
          this.catalogSyncError = null
          this.lastSyncedAt = new Date().toISOString()
        }
      } catch (e: any) {
        // Keep static catalog; note error for admin UI
        this.catalogSyncError = e?.data?.error || e?.message || 'sync failed'
        this.catalogSynced = false
      }
    },
  },
})
