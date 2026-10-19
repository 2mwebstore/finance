<template>
  <section class="reports-page">
    <header class="reports-heading">
      <div>
        <p class="eyebrow">FINANCIAL INSIGHTS</p>
        <h1>Reports</h1>
        <p class="heading-copy">Summarized income and expenses for any date range.</p>
      </div>
      <button v-if="canExportReports" class="export-button" :disabled="exporting" @click="doExport">
        <Download :size="16" /> {{ exporting ? 'Exporting…' : 'Export CSV' }}
      </button>
    </header>

    <section class="reports-controls">
      <div class="control-fields">
        <div class="control-field">
          <span>Report date range</span>
          <DateRangePicker v-model="dateRange" />
        </div>
        <div v-if="showBranchFilter" class="control-field branch-field">
          <span>Branch</span>
          <SearchSelect v-model="branchFilter" :options="branchOptions" :searchable="branchOptions.length > 7" placeholder="All branches" aria-label="Report branch" />
        </div>
      </div>
      <p class="control-hint">Updates automatically when the dates or branch change.</p>
    </section>

    <div v-if="loading" class="state-card"><span class="mini-spinner"></span> Loading report…</div>
    <div v-else-if="err" class="state-card state-error"><p>{{ err }}</p><button @click="load">Try again</button></div>
    <div v-else-if="data" class="reports-grid">
      <article class="summary-card usd-card">
        <div class="summary-top"><span class="summary-icon">USD</span><small>United States Dollar</small></div>
        <strong>{{ formatMoney(data.net_usd, 'USD') }}</strong>
        <span>Net balance</span>
      </article>
      <article class="summary-card khr-card">
        <div class="summary-top"><span class="summary-icon">KHR</span><small>Cambodian Riel</small></div>
        <strong>{{ formatMoney(data.net_khr, 'KHR') }}</strong>
        <span>Net balance</span>
      </article>

      <article class="breakdown-card">
        <h2>Income and expense breakdown</h2>
        <div class="breakdown-row"><span>Income · USD</span><strong class="income-text">{{ formatMoney(data.income_usd, 'USD') }}</strong></div>
        <div class="breakdown-row"><span>Expense · USD</span><strong class="expense-text">{{ formatMoney(data.expense_usd, 'USD') }}</strong></div>
        <div class="breakdown-row"><span>Income · KHR</span><strong class="income-text">{{ formatMoney(data.income_khr, 'KHR') }}</strong></div>
        <div class="breakdown-row"><span>Expense · KHR</span><strong class="expense-text">{{ formatMoney(data.expense_khr, 'KHR') }}</strong></div>
        <div class="breakdown-row total-row"><span>Approved transactions</span><strong>{{ data.transaction_count }}</strong></div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Download } from 'lucide-vue-next'
import type { DashboardSummary } from '~/types/api'
import { formatMoney } from '~/utils/money'
import { currentMonthRange } from '~/utils/dateRanges'
import { useBranchOptions } from '~/composables/useBranchOptions'

const { api, apiBase } = useApi()
const { getToken } = useAuth()
const { canExportReports } = usePermissions()
const dateRange = ref(currentMonthRange())
const branchFilter = ref('')
const { options: branchOptions, showFilter: showBranchFilter } = useBranchOptions()
const data = ref<DashboardSummary | null>(null)
const loading = ref(true)
const err = ref('')
const exporting = ref(false)

async function load() {
  loading.value = true
  err.value = ''
  try {
    data.value = await api<DashboardSummary>('/dashboard/summary', {
      query: { period: 'custom', date_from: dateRange.value.from, date_to: dateRange.value.to, branch_id: branchFilter.value || undefined },
    })
  } catch (e: any) {
    err.value = e?.data?.error || 'Failed to load report'
  } finally {
    loading.value = false
  }
}

async function doExport() {
  exporting.value = true
  try {
    const token = getToken()
    const params = new URLSearchParams({ period: 'custom', date_from: dateRange.value.from, date_to: dateRange.value.to })
    if (branchFilter.value) params.set('branch_id', branchFilter.value)
    const res = await fetch(`${apiBase}/reports/export?${params.toString()}`, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
      credentials: 'include',
    })
    if (!res.ok) throw new Error('Export denied')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `report-${dateRange.value.from}-to-${dateRange.value.to}.csv`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e: any) {
    err.value = e?.message || 'Export failed'
  } finally {
    exporting.value = false
  }
}

watch(branchFilter, () => { void load() })
watch(() => [dateRange.value.from, dateRange.value.to], ([from, to]) => {
  if (!from || !to) {
    dateRange.value = currentMonthRange()
    return
  }
  void load()
})

onMounted(load)
</script>

<style scoped>
.reports-page { color: #183054; }
.reports-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 20px; }
.eyebrow { margin: 0 0 5px; color: #6180ae; font-size: 10px; font-weight: 800; letter-spacing: .13em; }
.reports-heading h1 { margin: 0; color: #102348; font-size: 26px; font-weight: 760; letter-spacing: -.035em; }
.heading-copy { margin: 6px 0 0; color: #7083a2; font-size: 12px; }
.export-button { display: inline-flex; align-items: center; gap: 7px; min-height: 40px; border: 1px solid #dbe5f3; border-radius: 10px; padding: 0 14px; color: #31517f; background: #fff; font-size: 11px; font-weight: 700; cursor: pointer; box-shadow: 0 3px 10px rgb(24 60 110 / 5%); }
.export-button:hover:not(:disabled) { border-color: #b9d2f7; color: #1769e8; background: #f4f9ff; }
.export-button:disabled { opacity: .65; cursor: wait; }
.reports-controls { display: flex; flex-wrap: wrap; align-items: flex-end; justify-content: space-between; gap: 12px; margin-bottom: 16px; border: 1px solid #e2eaf5; border-radius: 14px; padding: 15px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.control-fields { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; }
.control-field { display: flex; width: min(360px, 100%); flex-direction: column; gap: 7px; color: #51688e; font-size: 10px; font-weight: 700; }
.branch-field { width: min(240px, 100%); }
.branch-field :deep(.search-select-trigger) { font-size: 12px; }
.control-hint { margin: 0; color: #96a4b9; font-size: 10px; }
.state-card { display: flex; align-items: center; justify-content: space-between; gap: 12px; border: 1px solid #e2eaf5; border-radius: 14px; padding: 22px; color: #7184a2; background: #fff; font-size: 12px; }
.state-error { color: #bd3e50; }
.state-error button { border: 0; border-radius: 8px; padding: 9px 13px; color: #fff; background: #1769e8; font-size: 10px; font-weight: 700; cursor: pointer; }
.mini-spinner { width: 16px; height: 16px; border: 2px solid #dbe8f8; border-top-color: #2678ed; border-radius: 50%; animation: spin .7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.reports-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; }
.summary-card { position: relative; overflow: hidden; border: 1px solid #e2eaf5; border-radius: 14px; padding: 18px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.summary-card::after { content: ''; position: absolute; top: 0; right: 0; left: 0; height: 4px; }
.usd-card::after { background: linear-gradient(100deg, #267df2, #1261e6); }
.khr-card::after { background: linear-gradient(100deg, #27b88d, #109b74); }
.summary-top { display: flex; align-items: center; gap: 9px; margin-bottom: 10px; }
.summary-icon { display: inline-flex; align-items: center; justify-content: center; min-width: 48px; height: 28px; border-radius: 8px; padding: 0 8px; color: #fff; background: linear-gradient(140deg, #102f5d, #1d4b91); font-size: 10px; font-weight: 800; letter-spacing: .04em; }
.summary-top small { color: #8a9ab3; font-size: 10px; }
.summary-card strong { display: block; color: #102348; font-size: clamp(22px, 2vw, 28px); font-weight: 780; letter-spacing: -.04em; font-variant-numeric: tabular-nums; }
.summary-card span:last-child { display: block; margin-top: 4px; color: #8494ab; font-size: 10px; }
.breakdown-card { grid-column: 1 / -1; border: 1px solid #e2eaf5; border-radius: 14px; padding: 18px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.breakdown-card h2 { margin: 0 0 8px; color: #162a4d; font-size: 14px; font-weight: 730; }
.breakdown-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 12px 0; border-bottom: 1px solid #edf1f7; font-size: 12px; }
.breakdown-row span { color: #7588a4; }
.breakdown-row strong { font-variant-numeric: tabular-nums; }
.total-row { border-bottom: 0; }
.income-text { color: #12865f; }
.expense-text { color: #d94357; }
@media (max-width: 700px) {
  .control-fields, .control-field, .branch-field { width: 100%; }
  .reports-heading { display: block; margin-bottom: 18px; }
  .export-button { width: 100%; justify-content: center; margin-top: 14px; }
  .reports-grid { grid-template-columns: 1fr; }
}
</style>
