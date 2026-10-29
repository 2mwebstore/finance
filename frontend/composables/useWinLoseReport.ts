import type { Partner, WinLoseReport } from '~/types/api'

/**
 * Loads a win/lose report for a date range and partner. There's no group choice: the API covers
 * every group the user works in for people who may see consolidated totals, else their own group.
 * `byWeek` gives a line per week (the partner statement); `needPartner` waits until a partner is
 * chosen.
 */
export function useWinLoseReport(range: () => { from: string; to: string }, options: { byWeek?: boolean; needPartner?: boolean } = {}) {
  const { api } = useApi()
  const { can } = usePermissions()
  const route = useRoute()
  const consolidated = computed(() => can('winlose-reports.consolidated'))

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
    if (!from || !to || (options.needPartner && !partnerId.value)) return
    const mine = ++request
    loading.value = true
    error.value = ''
    try {
      const result = await api<WinLoseReport>('/winlose/reports', {
        query: { date_from: from, date_to: to, partner_id: partnerId.value || undefined, by_week: options.byWeek ? 'true' : undefined },
      })
      if (mine === request) report.value = result
    } catch (e: any) {
      if (mine === request) error.value = e?.data?.error || 'Could not load the report.'
    } finally {
      if (mine === request) loading.value = false
    }
  }
  watch([range, partnerId], load, { deep: true })
  onMounted(load)

  return { report, loading, error, load, consolidated, partners, partnerId, partnerOptions }
}
