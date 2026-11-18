import { useApi } from '~/composables/useApi'
import { usePermissions } from '~/composables/usePermissions'

// Shared by the sidebar and the pages that change requests, like useActiveBranch's state.
const pending = ref(0)
let inFlight: Promise<void> | null = null

/**
 * The number on the sidebar's Requests item: pending requests in the top bar's branch (useApi adds
 * branch_id), the same as the Pending chip on the Requests page with no filters. Pages that make,
 * approve or reject a request call `refreshPendingRequests()` so it updates at once.
 */
export function usePendingRequests() {
  const { api } = useApi()
  const { canViewRequests } = usePermissions()

  function refreshPendingRequests(): Promise<void> {
    if (!canViewRequests.value) {
      pending.value = 0
      return Promise.resolve()
    }
    if (inFlight) return inFlight
    inFlight = api<{ pending: number }>('/requests/pending-count')
      .then((res) => { pending.value = Number(res?.pending) || 0 })
      .catch(() => { /* keep the last number; the Requests page shows its own errors */ })
      .finally(() => { inFlight = null })
    return inFlight
  }

  return { pendingRequests: readonly(pending), refreshPendingRequests }
}
