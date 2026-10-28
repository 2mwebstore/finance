import type { WinLoseReport } from '~/types/api'
import { useBranchOptions } from '~/composables/useBranchOptions'

/**
 * Loads a win/lose report for a date range and HQ. HQ choices: every HQ the user works in, plus
 * "All my HQs" for people who may see consolidated totals (Main HQ); others start on their first HQ.
 */
export function useWinLoseReport(range: () => { from: string; to: string }) {
  const { api } = useApi()
  const { can } = usePermissions()
  const route = useRoute()
  const consolidated = computed(() => can('winlose-reports.consolidated'))
  const { branches } = useBranchOptions()
  const hqOptions = computed(() => [
    ...(consolidated.value ? [{ value: '', label: 'All my HQs' }] : []),
    ...branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })),
  ])
  const branchId = ref(String(route.query.branch || ''))
  watch(branches, (list) => {
    if (!branchId.value && !consolidated.value && list.length) branchId.value = list[0]!.id
  }, { immediate: true })

  const report = ref<WinLoseReport | null>(null)
  const loading = ref(false)
  const error = ref('')
  let request = 0
  async function load() {
    const { from, to } = range()
    if (!from || !to || (!branchId.value && !consolidated.value)) return
    const mine = ++request
    loading.value = true
    error.value = ''
    try {
      const result = await api<WinLoseReport>('/winlose/reports', { query: { date_from: from, date_to: to, branch_id: branchId.value || undefined } })
      if (mine === request) report.value = result
    } catch (e: any) {
      if (mine === request) error.value = e?.data?.error || 'Could not load the report.'
    } finally {
      if (mine === request) loading.value = false
    }
  }
  watch([range, branchId], load, { deep: true })
  onMounted(load)

  return { report, loading, error, load, branchId, hqOptions, branches, consolidated }
}
