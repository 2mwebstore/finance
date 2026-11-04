import { useBranchOptions } from '~/composables/useBranchOptions'

/**
 * The branch an RP page (Data Entry, Report, Group Summary, Group Agent Assignments) works in.
 * The branch picked in the top bar decides, and the page then offers no choice of its own. With
 * "All branches" there, people with several branches get a Branch filter on the page, which starts
 * from the top bar each time (like every other page's filters); with one branch the API takes it.
 *
 * `''` is "every branch I work in" where a page can show that (lists, reports for those with
 * consolidated access); anywhere that needs one branch (a new entry, a new group, a report without
 * consolidated access) `effective` is the chosen branch, else the person's default one.
 */
export function useRpBranch() {
  const { user } = useAuth()
  const { activeBranchId } = useActiveBranch()
  const { branches } = useBranchOptions('All branches')
  const chosen = ref('')

  // Several branches to choose from.
  const show = computed(() => branches.value.length > 1)
  // What was chosen on this page, if the person works in it.
  const branchId = computed({
    get: () => activeBranchId.value || (show.value && branches.value.some(b => b.id === chosen.value) ? chosen.value : ''),
    set: (value: string) => { chosen.value = value },
  })
  // The person's own branch: their default if they work in it, else the first.
  const defaultBranchId = computed(() => {
    const own = (user.value as { default_branch_id?: string | null } | null)?.default_branch_id
    return branches.value.find(b => b.id === own)?.id || branches.value[0]?.id || ''
  })
  /** The one branch to work in: the chosen one, else the person's own ('' with a single branch: the API picks it). */
  const effective = computed(() => activeBranchId.value || (show.value ? branchId.value || defaultBranchId.value : ''))
  /** Options for a Branch select; allowAll adds "All branches" ('' = every branch in scope). */
  const optionsFor = (allowAll: boolean) => [
    ...(allowAll ? [{ value: '', label: 'All branches' }] : []),
    ...branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })),
  ]
  const labelFor = (id: string) => branches.value.find(b => b.id === id)?.code || ''

  return { branches, show, branchId, effective, defaultBranchId, optionsFor, labelFor }
}
