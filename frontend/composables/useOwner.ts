import type { BranchOwned } from '~/types/api'
import { branchLabel, useBranchOptions } from '~/composables/useBranchOptions'

/**
 * Products, product categories, customers, suppliers and payment methods are shared by every
 * branch (only the owner or an admin adds and changes those) or one branch's own (only that branch
 * sees and uses it; its people manage it). This gives the form's owner choices, who may change an
 * item, and the list filter.
 */
export function useOwner() {
  const { hasFullAccess } = usePermissions()
  const { activeBranchId } = useActiveBranch()
  // The user's branches (only the one picked in the top bar, when there is one).
  const { branches } = useBranchOptions()
  const SHARED = ''

  /** "Who uses it" on a form: shared (owner/admin only) or one of the user's branches. */
  const ownerOptions = computed(() => [
    ...(hasFullAccess.value ? [{ value: SHARED, label: 'Shared — every branch' }] : []),
    ...branches.value.map(b => ({ value: b.id, label: `${branchLabel(b)} only` })),
  ])

  /** A new item's owner: the document's branch, else the top-bar branch, else shared (owner/admin) or their first branch. */
  function defaultOwner(branchId = '') {
    return branchId || activeBranchId.value || (hasFullAccess.value ? SHARED : branches.value[0]?.id || SHARED)
  }

  /** Shared items are the owner/admin's to change; a branch's own, its people's (others never see them). */
  function canChange(item: BranchOwned) {
    return hasFullAccess.value || !!item.branch_id
  }

  /** A list filter: everything the user sees, shared only, or one branch's own. */
  const filterOptions = computed(() => [
    { value: '', label: 'Shared and branch' },
    { value: 'shared', label: 'Shared only' },
    ...branches.value.map(b => ({ value: b.id, label: `${b.name} only` })),
  ])

  /** Whether an item fits a filter value from filterOptions. */
  function matchesFilter(item: BranchOwned, filter: string) {
    if (!filter) return true
    if (filter === 'shared') return !item.branch_id
    return item.branch_id === filter
  }

  return { ownerOptions, defaultOwner, canChange, filterOptions, matchesFilter, hasFullAccess }
}
