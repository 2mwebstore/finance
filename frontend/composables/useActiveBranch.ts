const STORAGE_KEY = 'active-branch'

function remembered() {
  try { return localStorage.getItem(STORAGE_KEY) || '' } catch { return '' }
}

// Shared like useAuth's state: useApi reads it on every request.
const chosen = ref(remembered())

/** A new sign-in starts on "All branches". */
export function forgetActiveBranch() {
  chosen.value = ''
  try { localStorage.removeItem(STORAGE_KEY) } catch { /* private window: nothing was remembered */ }
}

/**
 * The branch picked in the top bar, for people with several: the whole app shows that branch.
 * Reads through useApi carry it as branch_id (the API still checks access), and useBranchOptions
 * offers only this branch, so page filters hide and forms preselect it.
 *
 * `''` is all of their branches. With a single branch there is nothing to pick.
 */
export function useActiveBranch() {
  const { user } = useAuth()
  const myBranches = computed(() => user.value?.scope_branches || user.value?.branches || [])
  const canSwitch = computed(() => myBranches.value.length > 1)
  // What was chosen, if the person still works in it.
  const activeBranch = computed(() => (canSwitch.value ? myBranches.value.find(b => b.id === chosen.value) || null : null))
  const activeBranchId = computed(() => activeBranch.value?.id || '')

  function setActiveBranch(id: string) {
    chosen.value = id
    try { localStorage.setItem(STORAGE_KEY, id) } catch { /* private window: it just isn't remembered */ }
  }

  return { myBranches, canSwitch, activeBranch, activeBranchId, setActiveBranch }
}
