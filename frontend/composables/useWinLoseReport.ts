import type { AssignmentGroup, WinLoseReport } from '~/types/api'
import { useRpBranch } from '~/composables/useRpBranch'

/**
 * Loads a win/lose report for a date range and, optionally, one titled group (Group Agent
 * Assignments). There's no branch choice: the API covers every branch the user works in for
 * people who may see consolidated totals, else their own. byEntry: the data entries too, each with
 * its lines; entryId: only one entry (both read when loading; the range or detail changes with them).
 */
export function useWinLoseReport(range: () => { from: string; to: string }, options: { byEntry?: () => boolean; entryId?: () => string } = {}) {
  const { api } = useApi()
  const { can } = usePermissions()
  const route = useRoute()
  const consolidated = computed(() => can('winlose-reports.consolidated'))

  // The branch the RP pages work in (only for people with several): "All branches" for those with
  // consolidated access, else always one branch.
  const rp = useRpBranch()
  const showBranch = rp.show
  const branchModel = computed({
    get: () => (consolidated.value ? rp.branchId.value : rp.effective.value),
    set: (value: string) => { rp.branchId.value = value },
  })
  const branchOptions = computed(() => rp.optionsFor(consolidated.value))
  /** '' = the API's own choice (every branch in scope, or the person's own group). */
  const branch = computed(() => (showBranch.value ? branchModel.value : rp.effective.value))

  // The switched-on groups (a switched-off one's weeks are left out of the reports).
  const groups = ref<AssignmentGroup[]>([])
  const groupId = ref(String(route.query.group || ''))
  // One agent: its part in each of its groups.
  const agentId = ref(String(route.query.agent || ''))
  const groupOptions = computed(() => [
    { value: '', label: 'All groups' },
    ...groups.value.map(g => ({ value: g.id, label: g.title })),
  ])
  async function loadGroups() {
    // The branch's groups; with every branch chosen, those of all of them.
    const branchQuery = branch.value || (showBranch.value ? 'all' : undefined)
    try { groups.value = (await api<{ items: AssignmentGroup[] }>('/platform-groups', { query: { status: 'active', branch_id: branchQuery } })).items || [] } catch { groups.value = [] }
    if (groupId.value && !groups.value.some(g => g.id === groupId.value)) groupId.value = ''
  }
  onMounted(loadGroups)
  watch(branch, loadGroups)

  const report = ref<WinLoseReport | null>(null)
  const loading = ref(false)
  const error = ref('')
  let request = 0
  async function load() {
    const { from, to } = range()
    if (!from || !to) return
    const mine = ++request
    loading.value = true
    error.value = ''
    try {
      const result = await api<WinLoseReport>('/winlose/reports', { query: { date_from: from, date_to: to, branch_id: branch.value || undefined, group_id: groupId.value || undefined, platform_id: agentId.value || undefined, by_entry: options.byEntry?.() ? 'true' : undefined, entry_id: options.entryId?.() || undefined } })
      if (mine === request) report.value = result
    } catch (e: any) {
      if (mine === request) error.value = e?.data?.error || 'Could not load the report.'
    } finally {
      if (mine === request) loading.value = false
    }
  }
  watch([range, groupId, agentId, branch], load, { deep: true })
  onMounted(load)

  return { report, loading, error, load, consolidated, groups, groupId, agentId, groupOptions, showBranch, branchModel, branchOptions, branch }
}
