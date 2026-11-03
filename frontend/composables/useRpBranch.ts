import { useBranchOptions } from '~/composables/useBranchOptions'

const STORAGE_KEY = 'rp-branch'

/**
 * The branch the RP pages (Data Entry, Report, Group Summary, Group Agent Assignments) work in.
 * It is chosen once and kept for all four, and is only offered to people who work in more than one
 * branch: with one, the API takes it by itself and no page shows a branch choice.
 *
 * `''` is "every branch I work in" where a page can show that (lists, reports for those with
 * consolidated access); anywhere that needs one branch (a new entry, a new group) `effective` is
 * the chosen branch, else the person's default one.
 */
export function useRpBranch() {
  const { user } = useAuth()
  const { branches } = useBranchOptions('All branches')
  const chosen = useState<string>('rp-branch', () => {
    try { return localStorage.getItem(STORAGE_KEY) || '' } catch { return '' }
  })

  // Several branches to choose from.
  const show = computed(() => branches.value.length > 1)
  // What was chosen, if the person still works in it.
  const branchId = computed({
    get: () => (show.value && branches.value.some(b => b.id === chosen.value) ? chosen.value : ''),
    set: (value: string) => {
      chosen.value = value
      try { localStorage.setItem(STORAGE_KEY, value) } catch { /* private window: it just isn't remembered */ }
    },
  })
  // The person's own branch: their default if they work in it, else the first.
  const defaultBranchId = computed(() => {
    const own = (user.value as { default_branch_id?: string | null } | null)?.default_branch_id
    return branches.value.find(b => b.id === own)?.id || branches.value[0]?.id || ''
  })
  /** The one branch to work in: the chosen one, else the person's own ('' with a single branch: the API picks it). */
  const effective = computed(() => (show.value ? branchId.value || defaultBranchId.value : ''))
  /** Options for a Branch select; allowAll adds "All branches" ('' = every branch in scope). */
  const optionsFor = (allowAll: boolean) => [
    ...(allowAll ? [{ value: '', label: 'All branches' }] : []),
    ...branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })),
  ]
  const labelFor = (id: string) => branches.value.find(b => b.id === id)?.code || ''

  return { branches, show, branchId, effective, defaultBranchId, optionsFor, labelFor }
}
