<template>
  <div class="dashboard-page">
    <div class="dashboard-heading">
      <div>
        <p class="eyebrow">BUSINESS OVERVIEW</p>
        <h1>Dashboard</h1>
        <p class="heading-copy">A clear view of your income and expenses.</p>
      </div>
    </div>

    <div v-if="loading" class="dashboard-message">
      <div class="loading-mark"><span></span><span></span><span></span></div>
      <span>Loading your dashboard…</span>
    </div>
    <div v-else-if="err" class="dashboard-message error-message">
      <div>
        <p class="font-semibold">We couldn’t load your dashboard.</p>
        <p class="mt-1 text-sm">{{ err }}</p>
      </div>
      <button class="retry-button" @click="load">Try again</button>
    </div>

    <div v-else-if="data" class="dashboard-layout">
      <main class="dashboard-main">
        <section class="metric-grid" aria-label="Financial summary">
          <article v-for="metric in metrics" :key="metric.label" class="metric-card">
            <div class="metric-topline">
              <div class="metric-icon" :class="metric.colorClass">
                <component :is="metric.icon" :size="21" :stroke-width="2.1" />
              </div>
              <span class="metric-period">{{ periodLabel }}</span>
            </div>
            <p class="metric-label">{{ metric.label }}</p>
            <p class="metric-value">{{ metric.value }}</p>
            <p class="metric-note">{{ metric.note }}</p>
          </article>
        </section>

        <section class="chart-grid">
          <article class="panel trend-panel">
            <div class="panel-heading">
              <div>
                <h2>Income vs Expense</h2>
                <p>Approved transactions in the selected period</p>
              </div>
              <div class="chart-legend">
                <span><i class="legend-dot income-dot"></i>Income</span>
                <span><i class="legend-dot expense-dot"></i>Expense</span>
              </div>
            </div>
            <div class="trend-chart-wrap">
              <svg class="trend-chart" :viewBox="`0 0 ${chartWidth} 252`" role="img" aria-label="Income and expense chart">
                <g v-for="tick in chartTicks" :key="tick.value">
                  <line x1="56" :y1="tick.y" :x2="chartWidth - 16" :y2="tick.y" class="chart-grid-line" />
                  <text x="46" :y="tick.y + 4" text-anchor="end" class="chart-axis-label">{{ tick.label }}</text>
                </g>
                <g v-for="bar in chartBars" :key="bar.label">
                  <rect :x="bar.incomeX" :y="bar.incomeY" :width="bar.width" :height="bar.incomeHeight" rx="4" class="income-bar" />
                  <rect :x="bar.expenseX" :y="bar.expenseY" :width="bar.width" :height="bar.expenseHeight" rx="4" class="expense-bar" />
                  <text :x="bar.labelX" y="238" text-anchor="middle" class="chart-axis-label">{{ bar.label }}</text>
                </g>
              </svg>
            </div>
            <div v-if="!chartBars.some((bar) => bar.incomeHeight || bar.expenseHeight)" class="chart-empty">
              No approved activity in this period yet.
            </div>
          </article>

          <article class="panel category-panel">
            <div class="panel-heading">
              <div>
                <h2>Expense by Category</h2>
                <p>Breakdown in {{ currency }}</p>
              </div>
              <span class="period-pill">{{ periodLabel }}</span>
            </div>
            <div class="category-content">
              <div class="donut-wrap">
                <div class="donut-chart" :style="donutStyle" role="img" aria-label="Expenses by category">
                  <div class="donut-hole">
                    <span>Total expense</span>
                    <strong>{{ selectedExpenseDisplay }}</strong>
                    <small>{{ currency }}</small>
                  </div>
                </div>
              </div>
              <div v-if="categorySegments.length" class="category-legend">
                <div v-for="category in categorySegments" :key="category.name" class="category-row">
                  <span class="category-name"><i class="legend-dot" :style="{ backgroundColor: category.color }"></i>{{ category.name }}</span>
                    <span class="category-total">{{ formatNumber(category.amount) }}</span>
                  <span class="category-percent">{{ category.percent }}%</span>
                </div>
              </div>
              <div v-else class="category-empty">No expense categories for this period.</div>
            </div>
          </article>
        </section>

        <section class="table-grid">
          <article class="panel table-panel transactions-panel">
            <div class="panel-heading">
              <div>
                <h2>Recent Transactions</h2>
                <p>Latest approved activity</p>
              </div>
              <NuxtLink v-if="canViewTransactions" to="/transactions" class="view-link">View all <ArrowUpRight :size="15" /></NuxtLink>
            </div>
            <div v-if="data.recent_transactions?.length" class="table-scroll">
              <table class="data-table transactions-table">
                <thead>
                  <tr><th>Date</th><th>Type</th><th>Description</th><th>Branch</th><th class="amount-cell">Amount</th><th>Status</th></tr>
                </thead>
                <tbody>
                  <tr v-for="transaction in data.recent_transactions.slice(0, 6)" :key="transaction.id">
                    <td class="date-cell">{{ formatTransactionDate(transaction.transaction_date) }}</td>
                    <td class="type-cell"><span class="type-pill" :class="transaction.type === 'income' ? 'type-income' : 'type-expense'">{{ transaction.type }}</span></td>
                    <td class="description-cell">{{ richTextToPlain(transaction.description) || (transaction.type === 'expense' ? transaction.category?.name : '') || transaction.transaction_number }}</td>
                    <td class="branch-cell">{{ transaction.branch?.name || transaction.branch?.code || '—' }}</td>
                    <td class="amount-cell">{{ formatMoney(transaction.amount, transaction.currency) }}</td>
                    <td class="status-cell"><span class="status-pill"><Check :size="11" /> {{ formatStatus(transaction.status) }}</span></td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="table-empty">No approved transactions in this period.</div>
          </article>

          <article class="panel table-panel branch-panel">
            <div class="panel-heading">
              <div>
                <h2>Branch Summary</h2>
                <p>Income, expense and net by branch</p>
              </div>
              <NuxtLink v-if="canViewBranches" to="/admin/branches" class="view-link">View all <ArrowUpRight :size="15" /></NuxtLink>
            </div>
            <div v-if="data.by_branch?.length" class="table-scroll">
              <table class="data-table branch-table">
                <thead><tr><th>Branch</th><th class="amount-cell">Income</th><th class="amount-cell">Expense</th><th class="amount-cell">Net</th></tr></thead>
                <tbody>
                  <tr v-for="branch in data.by_branch" :key="branch.branch_id">
                    <td><span class="branch-name">{{ branch.branch_name }}</span><small>{{ branch.branch_code }}</small></td>
                    <td class="amount-cell" data-label="Income">{{ formatBranchAmount(branch, 'income') }}</td>
                    <td class="amount-cell" data-label="Expense">{{ formatBranchAmount(branch, 'expense') }}</td>
                    <td class="amount-cell net-cell" data-label="Net" :class="branchNet(branch) < 0 ? 'negative-net' : ''">{{ formatBranchAmount(branch, 'net') }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
            <div v-else class="table-empty">No branch activity in this period.</div>
          </article>
        </section>

        <section v-if="quickLinks.length" class="quick-link-grid" aria-label="Quick links">
          <NuxtLink v-for="link in quickLinks" :key="link.label" :to="link.to" class="quick-link-card">
            <span class="quick-link-icon"><component :is="link.icon" :size="20" /></span>
            <span class="quick-link-copy"><strong>{{ link.label }}</strong><small>{{ link.description }}</small></span>
            <ArrowUpRight class="quick-link-arrow" :size="16" />
          </NuxtLink>
        </section>
      </main>

      <aside class="filter-panel panel" :class="{ 'is-collapsed': !mobileFiltersOpen }">
        <button type="button" class="filter-toggle" :aria-expanded="mobileFiltersOpen" @click="mobileFiltersOpen = !mobileFiltersOpen">
          <span class="filter-icon"><ListFilter :size="18" /></span>
          <span class="filter-toggle-copy"><strong>Filter</strong><small>{{ periodLabel }} · {{ branchLabel }} · {{ currency }}</small></span>
          <ChevronDown :size="17" class="filter-toggle-chevron" :class="{ 'is-open': mobileFiltersOpen }" />
        </button>
        <div class="filter-body">
          <div class="filter-heading">
            <span class="filter-icon"><ListFilter :size="18" /></span>
            <div><h2>Filter</h2><p>Update your overview</p></div>
          </div>

          <div class="filter-group">
            <label class="filter-label">Date Range</label>
            <div class="period-options mb-2">
              <button v-for="option in periodOptions" :key="option.value" class="period-option" :class="period === option.value ? 'selected' : ''" @click="selectPeriod(option.value)">{{ option.label }}</button>
            </div>
            <DateRangePicker v-model="dateRange" @update:model-value="period = 'custom'" />
          </div>

          <div class="filter-group">
            <label class="filter-label">Branch</label>
            <SearchSelect v-model="branchId" :options="branchOptions" placeholder="All accessible branches" search-placeholder="Search branches…" aria-label="Branch filter" />
          </div>

          <div class="filter-group">
            <label class="filter-label">Currency</label>
            <div class="currency-options">
              <button v-for="c in dashboardCurrencies" :key="c.code" :class="currency === c.code ? 'selected' : ''" @click="currency = c.code">{{ c.code }} <small>{{ c.symbol.trim() }}</small></button>
            </div>
          </div>

          <div v-if="dateValidationError" class="filter-error">{{ dateValidationError }}</div>
          <button class="apply-button" :disabled="loading" @click="applyFilters"><ListFilter :size="15" /> Apply Filter</button>
          <button class="reset-button" @click="resetFilters">Reset</button>
          <p class="filter-footnote">Totals include approved transactions only.</p>
        </div>
      </aside>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  ArrowUpRight,
  Building2,
  CalendarDays,
  Check,
  ChevronDown,
  CircleDollarSign,
  FileChartColumnIncreasing,
  ListFilter,
  ReceiptText,
  ShieldCheck,
  TrendingDown,
  TrendingUp,
  UsersRound,
} from 'lucide-vue-next'
import { useMediaQuery } from '@vueuse/core'
import type { Branch, DashboardSummary } from '~/types/api'
import { currencyMeta, formatMoney } from '~/utils/money'
import { useCurrency } from '~/composables/useCurrency'
import { richTextToPlain } from '~/utils/richText'
import { currentMonthRange } from '~/utils/dateRanges'

const { api } = useApi()
const { user } = useAuth()
const { canViewBranches, canViewTransactions, canViewUsers, canViewRoles, canViewReports } = usePermissions()
const data = ref<DashboardSummary | null>(null)
const branches = ref<Branch[]>([])
const period = ref('this_month')
const branchId = ref('')
const currency = ref('USD')
// Active currencies (Manage → Currency), plus any the period has amounts in.
const { currencies: currencyList } = useCurrency()
const dashboardCurrencies = computed(() => currencyList.value.filter(c => c.active || flowOf(data.value?.by_currency, c.code)))

// A currency's income/expense/net from a by_currency map (older USD/KHR fields as a fallback).
function flowOf(by: Record<string, { income: string; expense: string; net: string }> | undefined, code: string) {
  return by?.[code]
}
function amountOf(item: any, metric: 'income' | 'expense' | 'net') {
  const flow = flowOf(item?.by_currency, currency.value)
  if (flow) return flow[metric]
  return item?.[`${metric}_${currency.value.toLowerCase()}`]
}
const dateRange = ref(currentMonthRange())
const mobileFiltersOpen = ref(false)
// A narrower chart canvas on phones keeps the SVG axis labels at a readable size.
const isMobile = useMediaQuery('(max-width: 760px)')
const chartWidth = computed(() => isMobile.value ? 360 : 720)
const branchOptions = computed(() => [
  { value: '', label: 'All accessible branches' },
  ...branches.value.map(branch => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })),
])
const loading = ref(true)
const err = ref('')
const dateValidationError = ref('')

const periodOptions = [
  { value: 'this_month', label: 'Monthly' },
  { value: 'this_week', label: 'Weekly' },
  { value: 'today', label: 'Today' },
]
const chartColors = ['#1769e8', '#58a0ff', '#86bcff', '#22a77a', '#f3b544', '#8b7cf6', '#a8b5c7']

const periodLabel = computed(() => ({ today: 'Today', this_week: 'This week', this_month: 'This month', custom: 'Custom range' })[period.value] || 'Selected period')
const currencyMark = computed(() => currencyMeta(currency.value).symbol.trim())
const branchLabel = computed(() => {
  const selected = branches.value.find(branch => branch.id === branchId.value)
  return selected ? selected.code : 'All branches'
})
const selectedIncomeDisplay = computed(() => formatMoney(amountOf(data.value, 'income'), currency.value))
const selectedExpenseDisplay = computed(() => formatMoney(amountOf(data.value, 'expense'), currency.value))
const selectedNetDisplay = computed(() => formatMoney(amountOf(data.value, 'net'), currency.value))
const selectedNetValue = computed(() => numberValue(amountOf(data.value, 'net')))

const metrics = computed(() => [
  {
    label: 'Total Income',
    value: selectedIncomeDisplay.value,
    note: `${currency.value} · selected period`,
    icon: TrendingUp,
    colorClass: 'metric-blue',
  },
  {
    label: 'Total Expense',
    value: selectedExpenseDisplay.value,
    note: `${currency.value} · selected period`,
    icon: TrendingDown,
    colorClass: 'metric-red',
  },
  {
    label: 'Net Profit',
    value: selectedNetDisplay.value,
    note: `${currency.value} · after expenses`,
    icon: CircleDollarSign,
    colorClass: 'metric-teal',
  },
  {
    label: 'Total Transactions',
    value: new Intl.NumberFormat('en-US').format(data.value?.transaction_count || 0),
    note: 'Approved in this period',
    icon: ReceiptText,
    colorClass: 'metric-violet',
  },
])

const dailyItems = computed(() => {
  if (!data.value) return []
  const byDate = new Map((data.value.daily || []).map((item) => [item.date, item] as const))
  const start = parseDate(data.value.date_from)
  const end = parseDate(data.value.date_to)
  const days: { date: string; income: number; expense: number }[] = []
  if (!start || !end) return []
  for (const day = new Date(start); day <= end; day.setUTCDate(day.getUTCDate() + 1)) {
    const key = day.toISOString().slice(0, 10)
    const item = byDate.get(key)
    days.push({
      date: key,
      income: numberValue(amountOf(item, 'income')),
      expense: numberValue(amountOf(item, 'expense')),
    })
  }
  if (days.length <= 7) return days

  const count = 7
  return Array.from({ length: count }, (_, index) => {
    const from = Math.floor(index * days.length / count)
    const to = Math.floor((index + 1) * days.length / count)
    const group = days.slice(from, to)
    return {
      date: group[0]?.date || days[from]?.date || '',
      income: group.reduce((sum, day) => sum + day.income, 0),
      expense: group.reduce((sum, day) => sum + day.expense, 0),
    }
  })
})

const chartMaximum = computed(() => Math.max(1, ...dailyItems.value.flatMap((item) => [item.income, item.expense])))
const chartBars = computed(() => {
  const count = dailyItems.value.length
  if (!count) return []
  const slot = (chartWidth.value - 72) / count
  const width = Math.min(28, Math.max(8, slot * 0.25))
  const chartHeight = 170
  return dailyItems.value.map((item, index) => {
    const center = 56 + slot * index + slot / 2
    const incomeHeight = item.income / chartMaximum.value * chartHeight
    const expenseHeight = item.expense / chartMaximum.value * chartHeight
    const date = parseDate(item.date)
    return {
      label: date ? new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', timeZone: 'UTC' }).format(date) : '',
      width,
      incomeHeight,
      expenseHeight,
      incomeX: center - width - 2,
      incomeY: 204 - incomeHeight,
      expenseX: center + 2,
      expenseY: 204 - expenseHeight,
      labelX: center,
    }
  })
})
const chartTicks = computed(() => [0, 0.25, 0.5, 0.75, 1].map((ratio) => {
  const value = chartMaximum.value * ratio
  return { value, y: 204 - ratio * 170, label: formatAxisAmount(value) }
}))

const categorySegments = computed(() => {
  const rows = data.value?.expense_by_category
    ?.filter((item) => item.currency === currency.value)
    .map((item) => ({ name: item.category_name || 'Uncategorized', amount: numberValue(item.total) }))
    .filter((item) => item.amount > 0)
    .sort((a, b) => b.amount - a.amount) || []
  const visible = rows.slice(0, 6)
  if (rows.length > 6) visible.push({ name: 'Other', amount: rows.slice(6).reduce((sum, item) => sum + item.amount, 0) })
  const total = visible.reduce((sum, item) => sum + item.amount, 0)
  return visible.map((item, index) => ({
    ...item,
    color: chartColors[index % chartColors.length],
    percent: total ? Math.round(item.amount / total * 100) : 0,
  }))
})
const donutStyle = computed(() => {
  if (!categorySegments.value.length) return { background: '#e8eef7' }
  let cursor = 0
  const slices = categorySegments.value.map((item) => {
    const start = cursor
    cursor += item.percent
    return `${item.color} ${start}% ${cursor}%`
  })
  return { background: `conic-gradient(${slices.join(', ')})` }
})

const quickLinks = computed(() => {
  const links = []
  if (canViewBranches.value) links.push({ label: 'Branches', description: 'Manage business locations', to: '/admin/branches', icon: Building2 })
  if (canViewRoles.value) links.push({ label: 'Roles & Permissions', description: 'Review team access', to: '/admin/roles', icon: ShieldCheck })
  if (canViewUsers.value) links.push({ label: 'Users', description: 'Manage your team', to: '/admin/users', icon: UsersRound })
  if (canViewReports.value) links.push({ label: 'Reports', description: 'Explore detailed reports', to: '/reports', icon: FileChartColumnIncreasing })
  return links
})

function localDate() {
  const now = new Date()
  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function parseDate(value?: string) {
  if (!value) return null
  const parsed = new Date(`${value.slice(0, 10)}T00:00:00Z`)
  return Number.isNaN(parsed.valueOf()) ? null : parsed
}

function numberValue(value?: string | number) {
  const parsed = Number(value || 0)
  return Number.isFinite(parsed) ? parsed : 0
}

function formatNumber(value: number) {
  return formatMoney(value, currency.value)
}

function formatAxisAmount(value: number) {
  return `${currencyMark.value}${new Intl.NumberFormat('en-US', { notation: 'compact', maximumFractionDigits: 1 }).format(value)}`
}

function formatDate(value: string) {
  const date = parseDate(value)
  return date ? new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' }).format(date) : value
}

function formatTransactionDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.valueOf())) return value
  return new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' }).format(date)
}

function formatStatus(value: string) {
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : 'Approved'
}

function branchValue(branch: NonNullable<DashboardSummary['by_branch']>[number], metric: 'income' | 'expense' | 'net') {
  return numberValue(amountOf(branch, metric))
}

function formatBranchAmount(branch: NonNullable<DashboardSummary['by_branch']>[number], metric: 'income' | 'expense' | 'net') {
  return formatNumber(branchValue(branch, metric))
}

function branchNet(branch: NonNullable<DashboardSummary['by_branch']>[number]) {
  return branchValue(branch, 'net')
}

async function load() {
  loading.value = true
  err.value = ''
  dateValidationError.value = ''
  try {
    const query: Record<string, string | undefined> = {
      period: period.value,
      branch_id: branchId.value || undefined,
    }
    if (period.value === 'custom') {
      query.date_from = dateRange.value.from
      query.date_to = dateRange.value.to
    }
    data.value = await api<DashboardSummary>('/dashboard/summary', { query })
  } catch (e: any) {
    err.value = e?.data?.error || 'Please check your connection and try again.'
  } finally {
    loading.value = false
  }
}

function selectPeriod(value: string) {
  period.value = value
  if (value === 'this_month') {
    dateRange.value = currentMonthRange()
  } else if (value === 'today') {
    const today = localDate()
    dateRange.value = { from: today, to: today }
  } else if (value === 'this_week') {
    const now = new Date()
    const start = new Date(now.getFullYear(), now.getMonth(), now.getDate() - ((now.getDay() + 6) % 7))
    const end = new Date(start.getFullYear(), start.getMonth(), start.getDate() + 6)
    const dateKey = (date: Date) => `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
    dateRange.value = { from: dateKey(start), to: dateKey(end) }
  }
}

function applyFilters() {
  if (period.value === 'custom') {
    if (!dateRange.value.from || !dateRange.value.to) {
      dateValidationError.value = 'Choose both a start and end date.'
      return
    }
    if (dateRange.value.to < dateRange.value.from) {
      dateValidationError.value = 'The end date must be on or after the start date.'
      return
    }
  }
  load()
}

function resetFilters() {
  period.value = 'this_month'
  branchId.value = ''
  currency.value = 'USD'
  dateRange.value = currentMonthRange()
  load()
}

async function loadBranches() {
  if (!canViewBranches.value) {
    branches.value = user.value?.branches || []
    return
  }
  try {
    const result = await api<{ items: Branch[] }>('/branches', { query: { page_size: 100, is_active: true } })
    branches.value = result.items || []
  } catch {
    branches.value = user.value?.branches || []
  }
}

onMounted(() => {
  loadBranches()
  load()
})
</script>

<style scoped>
.dashboard-page { color: #122448; }
.dashboard-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 22px; }
.eyebrow { margin: 0 0 5px; color: #6180ae; font-size: 10px; font-weight: 800; letter-spacing: .13em; }
.dashboard-heading h1 { margin: 0; color: #102348; font-size: 26px; line-height: 1.15; font-weight: 750; letter-spacing: -.035em; }
.heading-copy { margin: 6px 0 0; color: #7083a2; font-size: 13px; }
.date-summary { display: flex; align-items: center; gap: 9px; color: #31517f; background: #fff; border: 1px solid #e1e9f4; border-radius: 10px; padding: 10px 13px; font-size: 12px; font-weight: 600; box-shadow: 0 2px 8px rgb(24 60 110 / 4%); white-space: nowrap; }
.date-summary svg { color: #3076ed; }
.date-arrow { padding: 0 5px; color: #8fa2be; }
.dashboard-layout { display: grid; grid-template-columns: minmax(0, 1fr) 270px; align-items: start; gap: 16px; }
.dashboard-main { min-width: 0; display: flex; flex-direction: column; gap: 15px; }
.metric-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; }
.metric-card, .panel { background: #fff; border: 1px solid #e2eaf5; border-radius: 12px; box-shadow: 0 3px 12px rgb(24 60 110 / 4%); }
.metric-card { min-width: 0; padding: 15px 16px 14px; }
.metric-topline { display: flex; justify-content: space-between; align-items: center; gap: 7px; }
.metric-icon { display: flex; align-items: center; justify-content: center; width: 39px; height: 39px; border-radius: 11px; }
.metric-blue { color: #fff; background: linear-gradient(140deg, #3388ff, #1461e7); box-shadow: 0 6px 12px rgb(37 105 232 / 18%); }
.metric-red { color: #fff; background: linear-gradient(140deg, #fa6473, #e63854); box-shadow: 0 6px 12px rgb(230 56 84 / 15%); }
.metric-teal { color: #fff; background: linear-gradient(140deg, #27b88d, #109b74); box-shadow: 0 6px 12px rgb(16 155 116 / 15%); }
.metric-violet { color: #fff; background: linear-gradient(140deg, #8574f6, #6251df); box-shadow: 0 6px 12px rgb(98 81 223 / 15%); }
.metric-period { color: #8c9cb5; font-size: 10px; font-weight: 600; }
.metric-label { margin: 14px 0 3px; color: #51688e; font-size: 12px; font-weight: 550; }
.metric-value { margin: 0; overflow: hidden; color: #122448; font-size: clamp(17px, 1.55vw, 22px); line-height: 1.25; font-weight: 750; letter-spacing: -.03em; text-overflow: ellipsis; white-space: nowrap; }
.metric-note { margin: 7px 0 0; color: #8293ad; font-size: 10px; }
.chart-grid { display: grid; grid-template-columns: minmax(0, 1.22fr) minmax(0, .98fr); gap: 13px; }
.panel { min-width: 0; padding: 16px; }
.panel-heading { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; margin-bottom: 9px; }
.panel-heading h2, .filter-heading h2 { margin: 0; color: #162a4d; font-size: 14px; font-weight: 730; letter-spacing: -.015em; }
.panel-heading p, .filter-heading p { margin: 4px 0 0; color: #8a9ab3; font-size: 10px; }
.chart-legend { display: flex; gap: 12px; padding-top: 2px; color: #5f7394; font-size: 10px; white-space: nowrap; }
.chart-legend > span, .category-name { display: inline-flex; align-items: center; gap: 6px; }
.legend-dot { flex: 0 0 auto; width: 8px; height: 8px; border-radius: 3px; background: #1769e8; }
.income-dot { background: #1769e8; }
.expense-dot { background: #8abaff; }
.trend-chart-wrap { width: 100%; overflow: hidden; }
.trend-chart { display: block; width: 100%; height: auto; min-height: 176px; }
.chart-grid-line { stroke: #eaf0f7; stroke-width: 1; }
.chart-axis-label { fill: #7184a3; font-size: 10px; }
.income-bar { fill: #1769e8; }
.expense-bar { fill: #8abaff; }
.chart-empty { margin-top: -2px; color: #8293ad; font-size: 11px; text-align: center; }
.period-pill { border: 1px solid #e7edf6; border-radius: 999px; padding: 5px 8px; color: #7185a4; font-size: 9px; font-weight: 650; white-space: nowrap; }
.category-content { display: flex; align-items: center; gap: 18px; min-height: 197px; }
.donut-wrap { display: flex; flex: 0 0 47%; justify-content: center; }
.donut-chart { display: flex; align-items: center; justify-content: center; width: min(188px, 100%); aspect-ratio: 1; border-radius: 50%; }
.donut-hole { display: flex; flex-direction: column; align-items: center; justify-content: center; width: 58%; height: 58%; border-radius: 50%; background: #fff; text-align: center; }
.donut-hole span { color: #7e90ac; font-size: 9px; }
.donut-hole strong { max-width: 100%; margin-top: 4px; overflow: hidden; color: #13274a; font-size: 16px; font-weight: 760; text-overflow: ellipsis; white-space: nowrap; }
.donut-hole small { margin-top: 2px; color: #9ba9bd; font-size: 9px; }
.category-legend { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 11px; }
.category-row { display: grid; grid-template-columns: minmax(0, 1fr) auto 30px; align-items: center; gap: 7px; color: #5c7193; font-size: 10px; }
.category-name { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.category-name .legend-dot { width: 7px; height: 7px; }
.category-total { color: #344d74; font-variant-numeric: tabular-nums; white-space: nowrap; }
.category-percent { color: #899ab2; text-align: right; }
.category-empty, .table-empty { display: flex; flex: 1; align-items: center; justify-content: center; color: #8495ad; font-size: 11px; text-align: center; }
.table-grid { display: grid; grid-template-columns: minmax(0, 1.6fr) minmax(290px, 1fr); gap: 13px; }
.table-panel { padding: 15px 14px 13px; }
.table-panel .panel-heading { margin: 0 0 12px; }
.view-link { display: inline-flex; align-items: center; gap: 4px; padding: 3px 0; color: #1269e9; font-size: 10px; font-weight: 650; text-decoration: none; white-space: nowrap; }
.view-link:hover { color: #0e4fb9; }
.table-scroll { width: 100%; overflow-x: auto; }
.data-table { width: 100%; border-collapse: separate; border-spacing: 0; color: #425a7d; font-size: 10px; }
.data-table th { padding: 9px 8px; background: #f5f8fd; color: #7184a2; font-size: 9px; font-weight: 700; text-align: left; white-space: nowrap; }
.data-table th:first-child { border-radius: 7px 0 0 7px; }
.data-table th:last-child { border-radius: 0 7px 7px 0; }
.data-table td { padding: 9px 8px; border-bottom: 1px solid #edf1f7; white-space: nowrap; }
.data-table tr:last-child td { border-bottom: 0; }
.date-cell { color: #687d9f; font-variant-numeric: tabular-nums; }
.description-cell { max-width: 160px; overflow: hidden; color: #344d74; text-overflow: ellipsis; }
.amount-cell { text-align: right !important; font-variant-numeric: tabular-nums; }
.type-pill, .status-pill { display: inline-flex; align-items: center; gap: 4px; border-radius: 999px; padding: 4px 7px; font-size: 9px; font-weight: 650; }
.type-income { color: #138e69; background: #e2f8ef; }
.type-expense { color: #df4658; background: #ffe8eb; }
.status-pill { color: #16845f; background: #e4f7ee; font-size: 8px; }
.branch-table th { padding-left: 7px; padding-right: 7px; }
.branch-table td { padding-left: 7px; padding-right: 7px; }
.branch-name { display: block; max-width: 118px; overflow: hidden; color: #263f66; text-overflow: ellipsis; }
.branch-table td:first-child small { display: block; margin-top: 2px; color: #97a5b8; font-size: 8px; }
.net-cell { color: #10956b; font-weight: 700; }
.negative-net { color: #df4658; }
.quick-link-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.quick-link-card { position: relative; display: flex; align-items: flex-start; gap: 10px; min-width: 0; min-height: 81px; padding: 12px; border: 1px solid #e2eaf5; border-radius: 11px; background: #fff; box-shadow: 0 3px 12px rgb(24 60 110 / 4%); text-decoration: none; transition: transform .16s ease, box-shadow .16s ease; }
.quick-link-card:hover { transform: translateY(-2px); box-shadow: 0 7px 18px rgb(24 60 110 / 9%); }
.quick-link-icon { display: flex; flex: 0 0 auto; align-items: center; justify-content: center; width: 33px; height: 33px; border-radius: 9px; background: #eef5ff; color: #1769e8; }
.quick-link-copy { display: flex; min-width: 0; flex-direction: column; gap: 5px; padding-top: 2px; }
.quick-link-copy strong { color: #1a3155; font-size: 10px; font-weight: 720; }
.quick-link-copy small { color: #8292ab; font-size: 9px; line-height: 1.4; }
.quick-link-arrow { position: absolute; top: 13px; right: 12px; color: #9aacbf; }
.filter-panel { position: sticky; top: 84px; padding: 16px; }
.filter-heading { display: flex; align-items: center; gap: 10px; padding-bottom: 15px; border-bottom: 1px solid #edf1f7; }
.filter-icon { display: flex; align-items: center; justify-content: center; width: 34px; height: 34px; border-radius: 9px; background: #edf5ff; color: #1769e8; }
.filter-group { padding: 15px 0 0; }
.filter-label { display: block; margin: 0 0 8px; color: #2c456c; font-size: 11px; font-weight: 700; }
.period-options { display: grid; grid-template-columns: repeat(3, 1fr); padding: 3px; border: 1px solid #e4ebf5; border-radius: 9px; background: #f7f9fc; }
.period-option { border: 0; border-radius: 6px; padding: 7px 4px; color: #6d809e; background: transparent; font-size: 9px; font-weight: 600; cursor: pointer; }
.period-option.selected { color: #fff; background: #1769e8; box-shadow: 0 3px 8px rgb(23 105 232 / 22%); }
.currency-options { display: grid; grid-template-columns: 1fr 1fr; gap: 7px; }
.currency-options button { display: flex; align-items: center; justify-content: space-between; border: 1px solid #e1e8f2; border-radius: 8px; padding: 8px 10px; color: #657a9b; background: #fff; font-size: 10px; font-weight: 650; cursor: pointer; }
.currency-options button small { color: #94a2b7; font-size: 10px; }
.currency-options button.selected { border-color: #a8c8fb; color: #1769e8; background: #f0f6ff; }
.filter-error { margin-top: 12px; color: #d8495b; font-size: 10px; line-height: 1.4; }
.apply-button, .reset-button, .retry-button { display: flex; align-items: center; justify-content: center; gap: 7px; width: 100%; border: 0; border-radius: 8px; padding: 10px; font-size: 10px; font-weight: 700; cursor: pointer; }
.apply-button { margin-top: 17px; color: #fff; background: linear-gradient(100deg, #267df2, #1261e6); box-shadow: 0 4px 9px rgb(23 105 232 / 20%); }
.apply-button:hover { background: #1259ce; }
.apply-button:disabled { opacity: .65; cursor: wait; }
.reset-button { margin-top: 8px; border: 1px solid #e2eaf5; color: #657a9b; background: #fff; }
.reset-button:hover { background: #f7f9fc; }
.filter-footnote { margin: 13px 0 0; color: #96a4b9; font-size: 9px; line-height: 1.4; text-align: center; }
.dashboard-message { display: flex; align-items: center; gap: 12px; min-height: 160px; padding: 24px; border: 1px solid #e2eaf5; border-radius: 12px; color: #6d809e; background: #fff; box-shadow: 0 3px 12px rgb(24 60 110 / 4%); font-size: 12px; }
.loading-mark { display: flex; align-items: center; gap: 4px; }
.loading-mark span { width: 6px; height: 6px; border-radius: 50%; background: #267df2; animation: pulse .8s ease-in-out infinite alternate; }
.loading-mark span:nth-child(2) { animation-delay: .15s; }
.loading-mark span:nth-child(3) { animation-delay: .3s; }
@keyframes pulse { to { opacity: .25; transform: translateY(-3px); } }
.error-message { justify-content: space-between; color: #a53e4e; }
.retry-button { width: auto; padding-inline: 18px; color: #fff; background: #1769e8; }

.filter-toggle, .filter-body { width: 100%; }
.filter-toggle { display: none; }
.filter-body { display: block; }
@media (max-width: 1350px) {
  .dashboard-layout { grid-template-columns: minmax(0, 1fr) 250px; gap: 12px; }
  .dashboard-main { gap: 12px; }
  .metric-grid { gap: 9px; }
  .metric-card { padding-inline: 12px; }
  .category-content { gap: 10px; }
  .donut-wrap { flex-basis: 42%; }
  .donut-chart { width: min(160px, 100%); }
  .table-grid { grid-template-columns: minmax(0, 1.45fr) minmax(265px, 1fr); gap: 10px; }
  .data-table th, .data-table td { padding-left: 6px; padding-right: 6px; }
}
@media (max-width: 1080px) {
  .dashboard-layout { grid-template-columns: minmax(0, 1fr); }
  .filter-panel { position: static; grid-row: 1; display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); align-items: end; gap: 0 15px; }
  .filter-heading { grid-column: 1 / -1; }
  .filter-group { padding-top: 12px; }
  .filter-panel .apply-button { margin-top: 12px; }
  .filter-panel .reset-button { margin-top: 12px; }
  .filter-footnote { grid-column: 1 / -1; }
}
@media (max-width: 760px) {
  .dashboard-heading { align-items: flex-start; margin-bottom: 15px; }
  .eyebrow { font-size: 11px; }
  .dashboard-heading h1 { font-size: 24px; }
  .heading-copy { max-width: none; font-size: 13px; }
  .date-summary { align-self: flex-end; padding: 8px 9px; font-size: 10px; }
  .date-summary svg { width: 14px; }
  .metric-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .metric-card { padding: 13px; }
  .metric-period { font-size: 11px; }
  .metric-label { font-size: 13px; }
  .metric-value { font-size: 19px; }
  .metric-note { font-size: 11px; }
  .panel { padding: 16px 14px; }
  .panel-heading h2 { font-size: 16px; }
  .panel-heading p { font-size: 12px; }
  .chart-legend { gap: 10px; font-size: 12px; }
  .chart-axis-label { font-size: 11px; }
  .chart-empty, .category-empty, .table-empty { font-size: 12px; }
  .period-pill { font-size: 11px; }
  .donut-hole span, .donut-hole small { font-size: 11px; }
  .category-row { font-size: 12px; }
  .view-link { font-size: 12px; }
  .chart-grid, .table-grid { grid-template-columns: minmax(0, 1fr); }
  .category-content { min-height: 175px; }
  .donut-chart { width: min(170px, 100%); }
  .table-grid { gap: 12px; }
  .data-table thead { display: none; }
  .data-table, .data-table tbody { display: block; }
  .data-table tr { display: grid; padding: 12px 2px; border-bottom: 1px solid #edf1f7; }
  .data-table tr:last-child { border-bottom: 0; }
  .data-table td { padding: 0 !important; border: 0; }
  .transactions-table tr { grid-template-columns: auto minmax(0, 1fr) auto; grid-template-areas: "desc desc amount" "date branch type"; align-items: center; gap: 6px 8px; }
  .transactions-table .description-cell { grid-area: desc; max-width: none; color: #1b3156; font-size: 13px; font-weight: 600; }
  .transactions-table .amount-cell { grid-area: amount; color: #1b3156; font-size: 13px; font-weight: 700; }
  .transactions-table .date-cell { grid-area: date; font-size: 11px; }
  .transactions-table .branch-cell { grid-area: branch; overflow: hidden; color: #8293ad; font-size: 11px; text-overflow: ellipsis; }
  .transactions-table .branch-cell::before { content: '·'; margin-right: 6px; }
  .transactions-table .type-cell { grid-area: type; justify-self: end; }
  .transactions-table .status-cell { display: none; }
  .type-pill { font-size: 10px; }
  .branch-table tr { grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; }
  .branch-table td { font-size: 13px; text-align: left !important; }
  .branch-table td:first-child { display: flex; grid-column: 1 / -1; align-items: baseline; gap: 6px; }
  .branch-table .branch-name { max-width: none; font-weight: 650; }
  .branch-table td:first-child small { display: inline; margin: 0; font-size: 11px; }
  .branch-table td[data-label]::before { content: attr(data-label); display: block; margin-bottom: 3px; color: #8a9ab3; font-size: 11px; font-weight: 600; }
  .quick-link-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .quick-link-copy strong { font-size: 13px; }
  .quick-link-copy small { font-size: 11px; }
  .filter-panel { display: block; grid-template-columns: none; padding: 0; overflow: hidden; }
  .filter-panel.is-collapsed .filter-body { display: none; }
  .filter-toggle { display: flex; align-items: center; gap: 10px; min-height: 62px; border: 0; padding: 10px 14px; color: #162a4d; background: #fff; text-align: left; cursor: pointer; }
  .filter-toggle .filter-icon { width: 34px; height: 34px; }
  .filter-toggle-copy { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 4px; }
  .filter-toggle-copy strong { font-size: 14px; }
  .filter-toggle-copy small { overflow: hidden; color: #8a9ab3; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
  .filter-toggle-chevron { color: #8293ad; transition: transform .18s ease; }
  .filter-toggle-chevron.is-open { transform: rotate(180deg); }
  .filter-body { padding: 0 14px 14px; }
  .filter-heading { display: none; }
  .filter-panel .filter-group { padding-top: 13px; grid-column: auto; }
  .filter-panel .apply-button { margin-top: 14px; }
  .filter-panel .reset-button { margin-top: 8px; }
  .filter-label { font-size: 12px; }
  .period-option { padding: 9px 4px; font-size: 12px; }
  .currency-options button, .currency-options button small { font-size: 13px; }
  .currency-options button { padding: 10px 12px; }
  .apply-button, .reset-button { padding: 12px; font-size: 13px; }
  .filter-error, .filter-footnote { font-size: 12px; }
}
@media (max-width: 430px) {
  .dashboard-heading { display: block; }
  .date-summary { display: inline-flex; margin-top: 11px; }
  .metric-period { font-size: 10px; }
  .metric-icon { width: 35px; height: 35px; }
  .metric-value { font-size: 17px; }
  .category-content { flex-direction: column; align-items: stretch; gap: 16px; }
  .donut-wrap { flex-basis: auto; }
  .donut-chart { width: min(180px, 70%); }
  .donut-hole strong { font-size: 15px; }
  .category-legend { gap: 11px; }
  .category-row { grid-template-columns: minmax(0, 1fr) auto 36px; gap: 8px; }
  .quick-link-card { min-height: 76px; padding: 10px; }
  .filter-panel { grid-template-columns: minmax(0, 1fr); }
  .filter-heading, .filter-group:first-of-type, .filter-footnote { grid-column: auto; }
  .filter-panel .apply-button, .filter-panel .reset-button { margin-top: 8px; }
}
</style>
