import type { Partner, WinLoseReport } from '~/types/api'
import { useBranchOptions } from '~/composables/useBranchOptions'

/**
 * Loads a win/lose report for a date range, branch and partner. Branch choices: every branch the
 * user works in, plus "All my branches" for people who may see consolidated totals; others start
 * on their first branch. `byWeek` gives a line per week (the partner statement); `needPartner`
 * waits until a partner is chosen.
 */
export function useWinLoseReport(range: () => { from: string; to: string }, options: { byWeek?: boolean; needPartner?: boolean } = {}) {
  const { api } = useApi()
  const { can } = usePermissions()
  const route = useRoute()
  const consolidated = computed(() => can('winlose-reports.consolidated'))
  const { branches } = useBranchOptions()
  const hqOptions = computed(() => [
    ...(consolidated.value ? [{ value: '', label: 'All my branches' }] : []),
    ...branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })),
  ])
  const branchId = ref(String(route.query.branch || ''))
  watch(branches, (list) => {
    if (!branchId.value && !consolidated.value && list.length) branchId.value = list[0]!.id
  }, { immediate: true })

  // Partners (turned-off ones too: their old weeks still count).
  const partners = ref<Partner[]>([])
  const partnerId = ref(String(route.query.partner || ''))
  const partnerOptions = computed(() => partners.value.map(p => ({ value: p.id, label: p.is_active ? p.name : `${p.name} (off)` })))
  onMounted(async () => {
    try { partners.value = (await api<{ items: Partner[] }>('/partners', { query: { include_inactive: 'true' } })).items || [] } catch { partners.value = [] }
  })

  const report = ref<WinLoseReport | null>(null)
  const loading = ref(false)
  const error = ref('')
  let request = 0
  async function load() {
    const { from, to } = range()
    if (!from || !to || (!branchId.value && !consolidated.value) || (options.needPartner && !partnerId.value)) return
    const mine = ++request
    loading.value = true
    error.value = ''
    try {
      const result = await api<WinLoseReport>('/winlose/reports', {
        query: { date_from: from, date_to: to, branch_id: branchId.value || undefined, partner_id: partnerId.value || undefined, by_week: options.byWeek ? 'true' : undefined },
      })
      if (mine === request) report.value = result
    } catch (e: any) {
      if (mine === request) error.value = e?.data?.error || 'Could not load the report.'
    } finally {
      if (mine === request) loading.value = false
    }
  }
  watch([range, branchId, partnerId], load, { deep: true })
  onMounted(load)

  return { report, loading, error, load, branchId, hqOptions, branches, consolidated, partners, partnerId, partnerOptions }
}
