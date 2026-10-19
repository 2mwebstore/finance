<template>
  <div class="transactions-page">
    <div class="page-heading">
      <div>
        <p class="eyebrow">FINANCIAL ACTIVITY</p>
        <h1>{{ pageTitle }}</h1>
        <p class="heading-copy">{{ headingCopy }}</p>
      </div>
      <div class="create-actions">
        <button v-if="canCreateIncome && routeType !== 'expense'" class="create-button income-create" @click="openCreate('income')"><Plus :size="16" /> Add Income</button>
        <button v-if="canCreateExpense && routeType !== 'income'" class="create-button expense-create" @click="openCreate('expense')"><Plus :size="16" /> Add Expense</button>
      </div>
    </div>

    <div class="list-toolbar">
      <div v-if="!routeType" class="type-filters" aria-label="Transaction type filter">
        <button v-for="filter in filters" :key="filter.value" :class="typeFilter === filter.value ? 'active' : ''" @click="selectType(filter.value)">{{ filter.label }}</button>
      </div>
      <p v-if="searchTerm" class="search-caption">Search results for <strong>“{{ searchTerm }}”</strong></p>
    </div>

    <FilterCard :summary="filterSummary" :active="filtersActive">
      <div class="filter-field date-range-filter">
        <span>Date range</span>
        <DateRangePicker v-model="dateRange" />
      </div>
      <div v-if="showBranchFilter" class="filter-field filter-wide branch-filter">
        <span>Branch</span>
        <SearchSelect v-model="branchFilter" :options="branchOptions" :searchable="branchOptions.length > 7" placeholder="All branches" aria-label="Branch filter" />
      </div>
      <div class="filter-field currency-filter">
        <span>Currency</span>
        <SearchSelect v-model="currency" :options="currencyOptions" :searchable="false" placeholder="All currencies" aria-label="Currency filter" />
      </div>
      <button type="button" class="filter-reset" :disabled="!filtersActive" @click="resetFilters"><RotateCcw :size="15" /> Reset</button>
    </FilterCard>

    <section v-if="summaryRows.length" class="totals-summary" aria-label="Income minus expense totals">
      <article v-for="row in summaryRows" :key="row.currency" class="totals-card">
        <div class="totals-card-heading"><CircleDollarSign :size="17" /><span>{{ row.currency }} · approved total</span></div>
        <div class="totals-equation">
          <div><small>Income</small><strong class="income-text">{{ formatMoney(row.income, row.currency) }}</strong></div>
          <span class="equation-operator">−</span>
          <div><small>Expense</small><strong class="expense-text">{{ formatMoney(row.expense, row.currency) }}</strong></div>
          <span class="equation-operator">=</span>
          <div><small>Net</small><strong :class="Number(row.net) >= 0 ? 'income-text' : 'expense-text'">{{ formatMoney(row.net, row.currency) }}</strong></div>
        </div>
      </article>
    </section>

    <div v-if="loading" class="list-state"><span class="mini-spinner"></span> Loading transactions…</div>
    <div v-else-if="err" class="list-state list-error"><p>{{ err }}</p><button @click="load()">Try again</button></div>
    <div v-else-if="!items.length" class="list-state empty-state">
      <span class="empty-icon"><ReceiptText :size="22" /></span>
      <strong>{{ emptyTitle }}</strong>
      <span>{{ emptyCopy }}</span>
    </div>
    <div v-else class="transaction-list">
      <article v-for="transaction in items" :key="transaction.id" class="transaction-row">
        <div class="transaction-type-icon" :class="transaction.type === 'income' ? 'income-tone' : 'expense-tone'">
          <ArrowDownLeft v-if="transaction.type === 'income'" :size="18" />
          <ArrowUpRight v-else :size="18" />
        </div>
        <div class="transaction-info">
          <div class="transaction-title-line">
            <strong>{{ transaction.description || (transaction.type === 'expense' ? transaction.category?.name : '') || transaction.transaction_number }}</strong>
            <span class="status-badge" :class="statusClass(transaction.status)">{{ displayStatus(transaction.status) }}</span>
          </div>
          <div class="transaction-meta">
            <span>{{ transaction.transaction_number }}</span>
            <template v-if="transaction.type === 'expense'"><i></i><span>{{ transaction.category?.name || 'Uncategorized' }}</span></template>
            <i></i>
            <span>{{ transaction.branch?.name || transaction.branch?.code || 'No branch' }}</span><i></i>
            <span>{{ formatDate(transaction.transaction_date) }}</span>
          </div>
        </div>
        <div class="transaction-amount" :class="transaction.type === 'income' ? 'income-text' : 'expense-text'">
          <span>{{ transaction.type === 'income' ? '+' : '−' }}{{ formatMoney(transaction.amount, transaction.currency) }}</span>
          <small>{{ transaction.type }}</small>
        </div>
        <div class="row-actions">
          <button v-if="canEdit(transaction)" class="row-action edit-action" :aria-label="`Edit ${transaction.transaction_number}`" title="Edit" @click="openEdit(transaction)"><Pencil :size="15" /></button>
          <button v-if="canDelete(transaction)" class="row-action delete-action" :aria-label="`Delete ${transaction.transaction_number}`" title="Delete" @click="openDelete(transaction)"><Trash2 :size="15" /></button>
        </div>
      </article>
    </div>

    <div v-if="isMobile && !loading && !err && items.length" ref="loadMoreSentinel" class="mobile-load-more" aria-live="polite">
      <span v-if="loadingMore"><span class="mini-spinner"></span> Loading more transactions…</span>
      <span v-else-if="items.length < total">Scroll for more</span>
      <span v-else>All {{ total }} transactions loaded</span>
    </div>

    <div v-if="!isMobile && !loading && !err && total > 0" class="pagination-bar">
      <span class="pagination-summary">Showing {{ firstItem }}–{{ lastItem }} of {{ total }}</span>
      <div class="pagination-controls">
        <div class="page-size-control">
          <span>Rows</span>
          <SearchSelect :model-value="String(pageSize)" :options="pageSizeOptions" :searchable="false" aria-label="Rows per page" @update:model-value="setPageSize" />
        </div>
        <button class="page-button" :disabled="page <= 1" aria-label="Previous page" @click="changePage(page - 1)">Previous</button>
        <span class="page-indicator">Page {{ page }} of {{ pageCount }}</span>
        <button class="page-button" :disabled="page >= pageCount" aria-label="Next page" @click="changePage(page + 1)">Next</button>
      </div>
    </div>

    <TransactionModal
      v-model="transactionModalOpen"
      :mode="modalMode"
      :type="modalType"
      :transaction="selectedTransaction"
      @saved="onTransactionSaved"
    />
    <DeleteTransactionModal
      v-model="deleteModalOpen"
      :transaction="transactionToDelete"
      @deleted="onTransactionDeleted"
    />
  </div>
</template>

<script setup lang="ts">
import { ArrowDownLeft, ArrowUpRight, CircleDollarSign, Pencil, Plus, ReceiptText, RotateCcw, Trash2 } from 'lucide-vue-next'
import { useIntersectionObserver, useMediaQuery } from '@vueuse/core'
import type { Transaction } from '~/types/api'
import { formatMoney } from '~/utils/money'
import { currentMonthRange, formatDateKey } from '~/utils/dateRanges'
import { useBranchOptions } from '~/composables/useBranchOptions'

const { api } = useApi()
const route = useRoute()
const {
  canCreateIncome,
  canCreateExpense,
  canEditIncome,
  canEditExpense,
  canVoidIncome,
  canVoidExpense,
} = usePermissions()
const typeFilter = ref(typeof route.query.type === 'string' ? route.query.type : '')
const searchTerm = ref(typeof route.query.search === 'string' ? route.query.search : '')
const dateRange = ref(currentMonthRange())
const currency = ref('')
const branchFilter = ref('')
const { options: branchOptions, showFilter: showBranchFilter, labelFor: branchLabel } = useBranchOptions()
const isThisMonth = computed(() => {
  const month = currentMonthRange()
  return dateRange.value.from === month.from && dateRange.value.to === month.to
})
const filtersActive = computed(() => !isThisMonth.value || !!currency.value || !!branchFilter.value)
const filterSummary = computed(() => {
  const period = isThisMonth.value
    ? 'This month'
    : `${formatDateKey(dateRange.value.from)} – ${formatDateKey(dateRange.value.to)}`
  const parts = [period, currency.value || 'All currencies']
  if (showBranchFilter.value) parts.push(branchLabel(branchFilter.value))
  return parts.join(' · ')
})
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const totals = ref<{ income_usd: string; expense_usd: string; net_usd: string; income_khr: string; expense_khr: string; net_khr: string } | null>(null)
const loadingMore = ref(false)
const loadMoreSentinel = ref<HTMLElement | null>(null)
const isMobile = useMediaQuery('(max-width: 760px)')
let requestSequence = 0
let filterTimer: ReturnType<typeof setTimeout> | undefined
const routeType = computed(() => route.query.type === 'income' || route.query.type === 'expense' ? route.query.type : '')
const pageTitle = computed(() => routeType.value === 'income' ? 'Income' : routeType.value === 'expense' ? 'Expense' : 'Transactions')
const headingCopy = computed(() => routeType.value
  ? `View, update, and manage ${routeType.value} records.`
  : 'View, update, and manage transaction records.')
const emptyTitle = computed(() => routeType.value ? `No ${routeType.value} found` : 'No transactions found')
const emptyCopy = computed(() => routeType.value
  ? `New ${routeType.value} records will appear here.`
  : 'New income and expense records will appear here.')
const items = ref<Transaction[]>([])
const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)))
const firstItem = computed(() => total.value ? ((page.value - 1) * pageSize.value) + 1 : 0)
const lastItem = computed(() => Math.min(page.value * pageSize.value, total.value))
const summaryRows = computed(() => {
  if (!totals.value) return []
  const rows = [
    { currency: 'USD' as const, income: totals.value.income_usd, expense: totals.value.expense_usd, net: totals.value.net_usd },
    { currency: 'KHR' as const, income: totals.value.income_khr, expense: totals.value.expense_khr, net: totals.value.net_khr },
  ]
  return currency.value ? rows.filter(row => row.currency === currency.value) : rows
})
const loading = ref(true)
const err = ref('')
const transactionModalOpen = ref(false)
const modalMode = ref<'create' | 'edit'>('create')
const modalType = ref<'income' | 'expense'>('income')
const selectedTransaction = ref<Transaction | null>(null)
const deleteModalOpen = ref(false)
const transactionToDelete = ref<Transaction | null>(null)
const filters = [
  { value: '', label: 'All activity' },
  { value: 'income', label: 'Income' },
  { value: 'expense', label: 'Expense' },
]
const currencyOptions = [
  { value: '', label: 'All currencies' },
  { value: 'USD', label: 'USD ($)' },
  { value: 'KHR', label: 'KHR (៛)' },
]
const pageSizeOptions = [10, 20, 50].map(value => ({ value: String(value), label: `${value} rows` }))

function formatDate(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.valueOf()) ? value : new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' }).format(date)
}

function displayStatus(value: string) {
  return value ? value.charAt(0).toUpperCase() + value.slice(1) : 'Approved'
}

function statusClass(value: string) {
  return `status-${value.toLowerCase()}`
}

function canEdit(transaction: Transaction) {
  const isEditable = transaction.status === 'approved' || transaction.status === 'pending'
  return isEditable && (transaction.type === 'income' ? canEditIncome.value : canEditExpense.value)
}

function canDelete(transaction: Transaction) {
  const isVoided = transaction.status === 'voided' || transaction.status === 'reversed'
  return !isVoided && (transaction.type === 'income' ? canVoidIncome.value : canVoidExpense.value)
}

async function load(append = false) {
  if (append && (loading.value || loadingMore.value || items.value.length >= total.value)) return
  const requestId = ++requestSequence
  if (append) loadingMore.value = true
  else {
    loading.value = true
    loadingMore.value = false
  }
  err.value = ''
  try {
    const listQuery = {
      type: routeType.value || typeFilter.value || undefined,
      search: searchTerm.value || undefined,
      date_from: dateRange.value.from || undefined,
      date_to: dateRange.value.to || undefined,
      currency: currency.value || undefined,
      branch_id: branchFilter.value || undefined,
      page: page.value,
      page_size: pageSize.value,
    }
    const [result, summary] = await Promise.all([
      api<{ items: Transaction[]; total: number }>('/transactions', { query: listQuery }),
      append
        ? Promise.resolve(null)
        : api<NonNullable<typeof totals.value>>('/transactions/totals', {
            query: {
              type: listQuery.type,
              search: listQuery.search,
              date_from: listQuery.date_from,
              date_to: listQuery.date_to,
              currency: listQuery.currency,
              branch_id: listQuery.branch_id,
            },
          }),
    ])
    if (requestId !== requestSequence) return
    items.value = append
      ? [...items.value, ...(result.items || []).filter(item => !items.value.some(existing => existing.id === item.id))]
      : result.items || []
    total.value = result.total || 0
    if (summary) totals.value = summary
  } catch (e: any) {
    if (requestId !== requestSequence) return
    err.value = e?.data?.error || 'Could not load transactions.'
  } finally {
    if (requestId === requestSequence) {
      loading.value = false
      loadingMore.value = false
    }
  }
}

function loadMore() {
  if (!isMobile.value || loading.value || loadingMore.value || items.value.length >= total.value) return
  page.value += 1
  void load(true)
}

useIntersectionObserver(loadMoreSentinel, ([entry]) => {
  if (entry?.isIntersecting) loadMore()
}, { rootMargin: '500px 0px' })

function scheduleFilterLoad() {
  if (filterTimer) clearTimeout(filterTimer)
  filterTimer = setTimeout(() => {
    page.value = 1
    items.value = []
    void load()
  }, 250)
}

function selectType(value: string) {
  typeFilter.value = value
  page.value = 1
  load()
}

function resetFilters() {
  dateRange.value = currentMonthRange()
  currency.value = ''
  branchFilter.value = ''
  scheduleFilterLoad()
}

function changePage(value: number) {
  if (value < 1 || value > pageCount.value) return
  page.value = value
  items.value = []
  load()
}

function setPageSize(value: string) {
  pageSize.value = Number(value) || 10
  page.value = 1
  load()
}

function openCreate(type: 'income' | 'expense') {
  selectedTransaction.value = null
  modalMode.value = 'create'
  modalType.value = type
  transactionModalOpen.value = true
}

function openEdit(transaction: Transaction) {
  selectedTransaction.value = transaction
  modalMode.value = 'edit'
  modalType.value = transaction.type
  transactionModalOpen.value = true
}

function openDelete(transaction: Transaction) {
  transactionToDelete.value = transaction
  deleteModalOpen.value = true
}

async function onTransactionSaved() {
  selectedTransaction.value = null
  await load()
}

async function onTransactionDeleted() {
  transactionToDelete.value = null
  await load()
}

watch(() => [route.query.type, route.query.search], ([type, search]) => {
  typeFilter.value = typeof type === 'string' ? type : ''
  searchTerm.value = typeof search === 'string' ? search : ''
  page.value = 1
  items.value = []
  load()
})

watch(() => [dateRange.value.from, dateRange.value.to, currency.value, branchFilter.value], scheduleFilterLoad)

onMounted(load)
</script>

<style scoped>
.transactions-page { color: #183054; }
.page-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; margin-bottom: 20px; }
.eyebrow { margin: 0 0 6px; color: #6180ae; font-size: 11px; font-weight: 800; letter-spacing: .13em; }
.page-heading h1 { margin: 0; color: #102348; font-size: 26px; line-height: 1.15; font-weight: 750; letter-spacing: -.035em; }
.heading-copy { margin: 7px 0 0; color: #7083a2; font-size: 13px; }
.create-actions { display: flex; align-items: center; gap: 9px; }
.create-button { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 42px; border: 0; border-radius: 10px; padding: 0 16px; color: #fff; font-size: 12px; font-weight: 700; cursor: pointer; transition: filter .15s ease, transform .15s ease; }
.income-create { background: #14946d; box-shadow: 0 5px 12px rgb(20 148 109 / 20%); }
.expense-create { background: #e14458; box-shadow: 0 5px 12px rgb(225 68 88 / 20%); }
.create-button:hover { filter: brightness(.95); transform: translateY(-1px); }
.list-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.page-button { min-height: 38px; border: 1px solid #dfe7f1; border-radius: 9px; padding: 0 14px; color: #5e7392; background: #fff; font-size: 12px; font-weight: 650; cursor: pointer; }
.page-button:hover:not(:disabled) { filter: brightness(.97); }
.totals-summary { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; margin-bottom: 14px; }
.totals-card { position: relative; min-width: 0; overflow: hidden; border: 1px solid #e2eaf4; border-radius: 13px; padding: 15px 16px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.totals-card::before { content: ''; position: absolute; top: 0; right: 0; left: 0; height: 3px; background: linear-gradient(100deg, #267df2, #1261e6); }
.totals-card-heading { display: flex; align-items: center; gap: 7px; color: #6a7f9f; font-size: 11px; font-weight: 700; }
.totals-card-heading svg { color: #2879ed; }
.totals-equation { display: grid; grid-template-columns: minmax(0, 1fr) auto minmax(0, 1fr) auto minmax(0, 1fr); align-items: center; gap: 8px; margin-top: 11px; }
.totals-equation > div { display: flex; min-width: 0; flex-direction: column; gap: 4px; }
.totals-equation small { color: #8797ad; font-size: 10px; }
.totals-equation strong { overflow: hidden; font-size: 14px; font-variant-numeric: tabular-nums; text-overflow: ellipsis; white-space: nowrap; }
.equation-operator { color: #96a4b7; font-size: 15px; font-weight: 700; }
.mobile-load-more { display: flex; min-height: 44px; align-items: center; justify-content: center; color: #8292a9; font-size: 12px; }
.mobile-load-more > span { display: inline-flex; align-items: center; gap: 7px; }
.pagination-bar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 12px; border: 1px solid #e2eaf5; border-radius: 11px; padding: 11px 13px; color: #8191a8; background: #fff; font-size: 12px; }
.pagination-controls { display: flex; align-items: center; gap: 8px; }
.page-size-control { display: flex; align-items: center; gap: 6px; }
.page-size-control :deep(.search-select) { width: 88px; }
.page-size-control :deep(.search-select-trigger) { min-height: 34px; padding: 6px 8px; font-size: 11px; }
.page-button { min-height: 34px; padding: 0 12px; }
.page-button:disabled { color: #aab5c3; background: #f7f9fb; cursor: not-allowed; }
.page-indicator { min-width: 70px; color: #526985; text-align: center; }
.type-filters { display: inline-flex; gap: 3px; border: 1px solid #e2e9f3; border-radius: 11px; padding: 3px; background: #fff; }
.type-filters button { border: 0; border-radius: 8px; padding: 9px 14px; color: #70829d; background: transparent; font-size: 12px; font-weight: 650; cursor: pointer; white-space: nowrap; transition: color .15s ease, background .15s ease; }
.type-filters button.active { color: #fff; background: #1769e8; box-shadow: 0 3px 9px rgb(23 105 232 / 22%); }
.type-scope { display: inline-flex; align-items: center; min-height: 34px; border-radius: 8px; padding: 0 12px; color: #3565a7; background: #eaf3ff; font-size: 11px; font-weight: 700; }
.search-caption { margin: 0; color: #8493a8; font-size: 12px; }
.search-caption strong { color: #496386; font-weight: 650; }
.transaction-list { overflow: hidden; border: 1px solid #e2eaf4; border-radius: 14px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.transaction-row { display: flex; align-items: center; gap: 14px; min-width: 0; padding: 14px 16px; border-bottom: 1px solid #edf1f7; transition: background .15s ease; }
.transaction-row:last-child { border-bottom: 0; }
.transaction-row:hover { background: #fbfcfe; }
.transaction-type-icon { display: flex; flex: 0 0 auto; align-items: center; justify-content: center; width: 42px; height: 42px; border-radius: 12px; }
.income-tone { color: #11956c; background: #e4f8ef; }
.expense-tone { color: #dd4055; background: #fff0f2; }
.transaction-info { flex: 1; min-width: 0; }
.transaction-title-line { display: flex; align-items: center; gap: 8px; min-width: 0; }
.transaction-title-line strong { overflow: hidden; color: #223b60; font-size: 13px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
.status-badge { flex: 0 0 auto; border-radius: 999px; padding: 4px 9px; color: #647790; background: #f0f3f7; font-size: 10px; font-weight: 700; }
.status-approved { color: #14835f; background: #e4f7ee; }
.status-pending { color: #a46c0d; background: #fff4d7; }
.status-rejected, .status-voided { color: #bd3e50; background: #ffeaed; }
.status-reversed { color: #6d59b7; background: #f0edff; }
.transaction-meta { display: flex; align-items: center; gap: 8px; min-width: 0; margin-top: 6px; overflow: hidden; color: #8294b0; font-size: 11px; white-space: nowrap; }
.transaction-meta span { overflow: hidden; text-overflow: ellipsis; }
.transaction-meta i { flex: 0 0 auto; width: 3px; height: 3px; border-radius: 50%; background: #bbc5d2; }
.transaction-amount { display: flex; flex: 0 0 130px; flex-direction: column; align-items: flex-end; gap: 4px; text-align: right; font-variant-numeric: tabular-nums; }
.transaction-amount > span { font-size: 14px; font-weight: 750; white-space: nowrap; }
.transaction-amount small { color: #8796ab; font-size: 10px; text-transform: capitalize; }
.income-text { color: #12865f; }
.expense-text { color: #d94357; }
.row-actions { display: flex; flex: 0 0 auto; gap: 5px; }
.row-action { display: flex; align-items: center; justify-content: center; width: 34px; height: 34px; border: 1px solid transparent; border-radius: 9px; background: transparent; cursor: pointer; }
.edit-action { color: #4773b6; }
.edit-action:hover { border-color: #dce9fc; background: #f0f6ff; }
.delete-action { color: #bd6872; }
.delete-action:hover { border-color: #f7dce0; color: #c93449; background: #fff2f4; }
.list-state { display: flex; align-items: center; justify-content: center; gap: 9px; min-height: 170px; border: 1px solid #e2eaf4; border-radius: 14px; padding: 24px; color: #8090a7; background: #fff; font-size: 12px; text-align: center; }
.mini-spinner { width: 16px; height: 16px; border: 2px solid #dbe8f8; border-top-color: #2678ed; border-radius: 50%; animation: spin .7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.list-error { justify-content: space-between; color: #bd3e50; }
.list-error p { margin: 0; }
.list-error button { border: 0; border-radius: 8px; padding: 9px 14px; color: #fff; background: #1769e8; font-size: 12px; font-weight: 650; cursor: pointer; }
.empty-state { flex-direction: column; gap: 8px; }
.empty-icon { display: flex; align-items: center; justify-content: center; width: 48px; height: 48px; margin-bottom: 4px; border-radius: 14px; color: #5182c9; background: #eef5ff; }
.empty-state strong { color: #405a7e; font-size: 14px; }
.empty-state > span:last-child { font-size: 12px; }
@media (max-width: 700px) {
  .page-heading { display: block; margin-bottom: 16px; }
  .create-actions { margin-top: 14px; }
  .create-button { flex: 1; }
  .list-toolbar { display: block; }
  .search-caption { margin-top: 10px; }
  .type-filters { display: flex; width: 100%; }
  .type-filters button { flex: 1; padding: 10px 8px; font-size: 13px; }
  .totals-summary { grid-template-columns: 1fr; }
  .totals-equation strong { font-size: 13px; }
  .pagination-controls { width: 100%; justify-content: space-between; gap: 4px; }
  .page-size-control span { display: none; }
  .transaction-list { border-radius: 14px; }
  .transaction-row { align-items: flex-start; gap: 10px; padding: 13px 12px; }
  .transaction-type-icon { width: 36px; height: 36px; border-radius: 10px; }
  .transaction-title-line { flex-wrap: wrap; gap: 6px; }
  .transaction-title-line strong { max-width: 100%; font-size: 13px; }
  .transaction-meta { flex-wrap: wrap; gap: 6px; line-height: 1.45; white-space: normal; }
  .transaction-meta span { max-width: 130px; }
  .transaction-amount { flex-basis: auto; margin-left: auto; padding-top: 1px; }
  .transaction-amount > span { font-size: 13px; }
  .row-actions { align-self: center; flex-direction: column; }
  .row-action { width: 32px; height: 32px; }
}
@media (max-width: 420px) {
  .transaction-meta span:nth-of-type(1), .transaction-meta i:nth-of-type(1) { display: none; }
  .transaction-amount { max-width: 92px; }
}
</style>
