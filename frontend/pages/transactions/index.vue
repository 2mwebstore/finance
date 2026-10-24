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
      <!-- Kept in the URL (?search=), like the top bar's search, so both stay in sync. -->
      <ListSearch :model-value="searchTerm" class="transaction-search" placeholder="Search by number or description…" @update:model-value="setSearch" />
    </div>

    <FilterCard :summary="filterSummary" :active="filtersActive">
      <div class="filter-field date-range-filter">
        <span>Date range</span>
        <DateRangePicker v-model="dateRange" />
      </div>
      <div v-if="showBranchFilter" class="filter-field filter-wide branch-filter">
        <span>Branch</span>
        <SearchSelect v-model="branchFilter" :options="branchOptions" placeholder="All branches" search-placeholder="Search branches…" aria-label="Branch filter" />
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
      <article v-for="transaction in items" :key="transaction.id" class="transaction-row" :class="{ 'is-pending': transaction.status === 'pending' }">
        <div class="transaction-type-icon" :class="transaction.type === 'income' ? 'income-tone' : 'expense-tone'">
          <ArrowDownLeft v-if="transaction.type === 'income'" :size="18" />
          <ArrowUpRight v-else :size="18" />
        </div>
        <div class="transaction-info">
          <div class="transaction-title-line">
            <strong>{{ transactionTitle(transaction) }}</strong>
            <span class="status-badge wide-only" :class="statusClass(transaction.status)">{{ displayStatus(transaction.status) }}</span>
            <button v-if="transaction.attachments?.length" type="button" class="attachment-pill wide-only" :title="attachmentLabel(transaction)" :aria-label="attachmentLabel(transaction)" @click="openAttachment(transaction)">
              <ImageIcon v-if="transaction.attachments[0]!.type.startsWith('image/')" :size="12" /><FileText v-else :size="12" />
              <span>{{ transaction.attachments.length > 1 ? `${transaction.attachments.length} files` : 'View' }}</span>
            </button>
          </div>
          <div class="transaction-meta">
            <span class="meta-number">{{ transaction.transaction_number }}</span>
            <span v-if="transaction.type === 'expense'">{{ transaction.category?.name || 'Uncategorized' }}</span>
            <span>{{ transaction.branch?.name || transaction.branch?.code || 'No branch' }}</span>
            <span class="meta-date">{{ formatDate(transaction.transaction_date) }}</span>
          </div>
          <p v-if="transaction.status === 'pending'" class="pending-hint" :class="{ 'is-yours': isMyDecision(transaction) }">
            <Clock3 :size="12" :stroke-width="2.4" aria-hidden="true" />
            <strong>{{ isMyDecision(transaction) ? 'Needs your approval' : 'Waiting for approval' }}</strong>
            <span v-if="pendingSource(transaction)" class="pending-by">{{ pendingSource(transaction) }}</span>
          </p>
          <!-- Who entered it, and who approved, rejected or deleted it. -->
          <p v-else-if="transaction.created_by?.name" class="people-line" :class="transaction.outcome ? `is-${transaction.outcome.action}` : ''">
            <span class="people-created"><UserRound :size="12" :stroke-width="2.4" aria-hidden="true" />{{ createdText(transaction) }}</span>
            <span v-if="outcomeText(transaction)" class="people-outcome" :title="outcomeTitle(transaction)">
              <Check v-if="transaction.outcome?.action === 'approved'" :size="12" :stroke-width="2.8" aria-hidden="true" />
              <X v-else-if="transaction.outcome?.action === 'rejected'" :size="12" :stroke-width="2.8" aria-hidden="true" />
              <Trash2 v-else :size="12" :stroke-width="2.4" aria-hidden="true" />
              {{ outcomeText(transaction) }}
            </span>
          </p>
        </div>
        <div class="transaction-amount" :class="transaction.type === 'income' ? 'income-text' : 'expense-text'">
          <span>{{ transaction.type === 'income' ? '+' : '−' }}{{ formatMoney(transaction.amount, transaction.currency) }}</span>
          <small class="amount-type">{{ transaction.type }}</small>
          <small class="amount-date">{{ formatDate(transaction.transaction_date) }}</small>
        </div>
        <!-- On wide screens this wrapper disappears (display: contents); on phones it's the row's bottom bar. -->
        <div class="row-footer">
          <div class="row-tags">
            <span class="status-badge" :class="statusClass(transaction.status)">{{ displayStatus(transaction.status) }}</span>
            <button v-if="transaction.attachments?.length" type="button" class="attachment-pill" :aria-label="attachmentLabel(transaction)" @click="openAttachment(transaction)">
              <ImageIcon v-if="transaction.attachments[0]!.type.startsWith('image/')" :size="12" /><FileText v-else :size="12" />
              <span>{{ transaction.attachments.length > 1 ? `${transaction.attachments.length} files` : 'View file' }}</span>
            </button>
          </div>
          <div class="row-actions">
            <button v-if="transaction.status === 'pending' && transaction.can_approve" class="row-action approve-action" :aria-label="`Approve ${transaction.transaction_number}`" title="Approve" @click="openApproval(transaction, 'approve')"><Check :size="16" :stroke-width="2.6" /></button>
            <button v-if="transaction.status === 'pending' && transaction.can_reject" class="row-action reject-action" :aria-label="`Reject ${transaction.transaction_number}`" title="Reject" @click="openApproval(transaction, 'reject')"><X :size="16" :stroke-width="2.6" /></button>
            <button v-if="canEdit(transaction)" class="row-action edit-action" :aria-label="`Edit ${transaction.transaction_number}`" title="Edit" @click="openEdit(transaction)"><Pencil :size="15" /></button>
            <button v-if="canDelete(transaction)" class="row-action delete-action" :aria-label="`Delete ${transaction.transaction_number}`" title="Delete" @click="openDelete(transaction)"><Trash2 :size="15" /></button>
          </div>
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
    <ApprovalDialog
      v-model="approvalOpen"
      :transaction="approvalTransaction"
      :action="approvalAction"
      @done="onApprovalDone"
      @files-changed="onFilesChanged"
      @preview="(index) => approvalTransaction && openAttachment(approvalTransaction, index)"
    />
    <!-- After the approval dialog so a preview opened from it shows on top. -->
    <AttachmentViewer v-model="attachmentViewerOpen" :transaction="attachmentTransaction" :start-index="attachmentStart" @changed="onFilesChanged" />
  </div>
</template>

<script setup lang="ts">
import { ArrowDownLeft, ArrowUpRight, Check, CircleDollarSign, Clock3, FileText, ImageIcon, Pencil, Plus, ReceiptText, RotateCcw, Trash2, UserRound, X } from 'lucide-vue-next'
import { useIntersectionObserver, useMediaQuery } from '@vueuse/core'
import type { Transaction } from '~/types/api'
import { formatMoney } from '~/utils/money'
import { richTextToPlain } from '~/utils/richText'
import { currentMonthRange, formatDateKey } from '~/utils/dateRanges'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { useToast } from '~/composables/useToast'

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
const { user } = useAuth()
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
const emptyCopy = computed(() => {
  if (searchTerm.value) return `Nothing matches “${searchTerm.value}”. Try other words or a wider date range.`
  return routeType.value
    ? `New ${routeType.value} records will appear here.`
    : 'New income and expense records will appear here.'
})
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

function timeAgo(value?: string) {
  if (!value) return ''
  const seconds = (Date.now() - new Date(value).valueOf()) / 1000
  if (Number.isNaN(seconds)) return ''
  if (seconds < 60) return 'just now'
  if (seconds < 3600) return `${Math.floor(seconds / 60)} min ago`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)} h ago`
  if (seconds < 7 * 86400) return `${Math.floor(seconds / 86400)} d ago`
  return formatDate(value)
}

function isMyDecision(transaction: Transaction) {
  return !!(transaction.can_approve || transaction.can_reject)
}

function who(person?: { id: string; name: string }) {
  if (!person?.name) return 'someone'
  return person.id === user.value?.id ? 'you' : person.name
}

// "Created by Dara Cashier", or "Created & approved by Sokha" when the same person did both.
function createdText(transaction: Transaction) {
  const outcome = transaction.outcome
  if (outcome?.action === 'approved' && outcome.by.id === transaction.created_by?.id) return `Created & approved by ${who(transaction.created_by)}`
  return `Created by ${who(transaction.created_by)}`
}

// "Approved by Sokha", "Rejected by Sokha: “no receipt”", "Deleted by Sokha: “duplicate”".
function outcomeText(transaction: Transaction) {
  const outcome = transaction.outcome
  if (!outcome || (outcome.action === 'approved' && outcome.by.id === transaction.created_by?.id)) return ''
  const verb = { approved: 'Approved', rejected: 'Rejected', voided: 'Deleted' }[outcome.action]
  const reason = outcome.action !== 'approved' && outcome.notes ? `: “${outcome.notes}”` : ''
  return `${verb} by ${who(outcome.by)}${reason}`
}

function outcomeTitle(transaction: Transaction) {
  const outcome = transaction.outcome
  if (!outcome) return ''
  const date = new Date(outcome.at)
  const when = Number.isNaN(date.valueOf()) ? '' : ` on ${new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' }).format(date)}`
  return `${outcomeText(transaction) || createdText(transaction)}${when}${outcome.notes && outcome.action === 'approved' ? ` · Note: ${outcome.notes}` : ''}`
}

// "by Dara Cashier · 2 h ago" ("you" for your own submissions).
function pendingSource(transaction: Transaction) {
  const creator = transaction.created_by
  const who = !creator?.name ? '' : creator.id === user.value?.id ? 'you' : creator.name
  return [who && `by ${who}`, timeAgo(transaction.created_at)].filter(Boolean).join(' · ')
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

function setSearch(value: string) {
  void navigateTo({ path: route.path, query: { ...route.query, search: value || undefined } }, { replace: true })
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

function transactionTitle(transaction: Transaction) {
  return richTextToPlain(transaction.description)
    || (transaction.type === 'expense' ? transaction.category?.name : '')
    || transaction.transaction_number
}

// Attachment popup (images and PDFs; switches between a transaction's files)
const attachmentViewerOpen = ref(false)
const attachmentTransaction = ref<Transaction | null>(null)
const attachmentStart = ref(0)
function openAttachment(transaction: Transaction, index = 0) {
  attachmentTransaction.value = transaction
  attachmentStart.value = index
  attachmentViewerOpen.value = true
}

function attachmentLabel(transaction: Transaction) {
  const files = transaction.attachments || []
  return files.length > 1 ? `View ${files.length} files` : `View ${files[0]?.name || 'file'}`
}

// A file was added or deleted from the viewer or the approval dialog: update that row in place
// (the API's answer has no list-only fields such as can_approve, so keep the row's).
function onFilesChanged(updated: Transaction) {
  const merge = (txn: Transaction | null) => txn && txn.id === updated.id ? { ...txn, attachments: updated.attachments } : txn
  items.value = items.value.map(txn => merge(txn)!)
  attachmentTransaction.value = merge(attachmentTransaction.value)
  approvalTransaction.value = merge(approvalTransaction.value)
}

// Approve / reject (only shown when the API says this user may decide)
const approvalOpen = ref(false)
const approvalTransaction = ref<Transaction | null>(null)
const approvalAction = ref<'approve' | 'reject'>('approve')
function openApproval(transaction: Transaction, action: 'approve' | 'reject') {
  approvalTransaction.value = transaction
  approvalAction.value = action
  approvalOpen.value = true
}
const toast = useToast()

async function onApprovalDone(updated: Transaction) {
  const approved = updated.status === 'approved'
  toast.success(
    approved ? `${updated.transaction_number} now counts toward totals and reports.` : `${updated.transaction_number} stays in the history but won’t count.`,
    { title: approved ? 'Transaction approved' : 'Transaction rejected' },
  )
  approvalTransaction.value = null
  await load()
}

async function onTransactionSaved(saved: Transaction) {
  const label = saved.type === 'income' ? 'Income' : 'Expense'
  const amount = formatMoney(saved.amount, saved.currency)
  if (modalMode.value === 'edit' && saved.status === 'pending' && selectedTransaction.value?.status === 'approved') {
    toast.info(`${saved.transaction_number} counts again once a manager approves the change.`, { title: 'Sent for approval again' })
  } else if (modalMode.value === 'edit') toast.success(`${saved.transaction_number} is up to date.`, { title: 'Changes saved' })
  else if (saved.status === 'pending') toast.info(`${amount} ${saved.type} counts once a manager approves it.`, { title: 'Sent for approval' })
  else toast.success(`${amount} recorded as ${saved.transaction_number}.`, { title: `${label} saved` })
  selectedTransaction.value = null
  await load()
}

async function onTransactionDeleted(deleted: Transaction) {
  toast.success(`${deleted.transaction_number} no longer counts toward totals.`, { title: 'Transaction deleted' })
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
.transaction-search { flex: 0 1 340px; margin-left: auto; }
.transaction-list { overflow: hidden; border: 1px solid #e2eaf4; border-radius: 14px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.transaction-row { display: flex; flex-wrap: wrap; align-items: center; gap: 14px; min-width: 0; padding: 14px 16px; border-bottom: 1px solid #edf1f7; transition: background .15s ease; }
.attachment-pill { display: inline-flex; flex: 0 0 auto; align-items: center; gap: 4px; border: 1px solid #d6e4fb; border-radius: 999px; padding: 3px 9px; color: #1769e8; background: #f2f7ff; font-size: 11px; font-weight: 700; cursor: pointer; transition: background .15s ease; }
.attachment-pill:hover { background: #e4eeff; }
/* One quiet line under the details: "Waiting for approval · by Dara · 2 h ago". */
.pending-hint { display: flex; align-items: center; gap: 5px; min-width: 0; margin: 5px 0 0; overflow: hidden; color: #a0732a; font-size: 11px; line-height: 1.3; white-space: nowrap; }
.pending-hint svg { flex: 0 0 auto; }
.pending-hint strong { flex: 0 0 auto; font-weight: 700; }
.pending-by { overflow: hidden; color: #9a8a6c; text-overflow: ellipsis; }
.pending-by::before { content: '·'; margin-right: 5px; color: #c9b890; }
.people-line { display: flex; flex-wrap: wrap; align-items: center; gap: 2px 10px; min-width: 0; margin: 5px 0 0; color: #8294b0; font-size: 11px; line-height: 1.35; }
.people-line > span { display: inline-flex; min-width: 0; align-items: center; gap: 4px; }
.people-line svg { flex: 0 0 auto; }
.people-outcome { overflow: hidden; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.people-line.is-approved .people-outcome { color: #14835f; }
.people-line.is-rejected .people-outcome { color: #bd3e50; }
.people-line.is-voided .people-outcome { color: #8a96a8; }
.pending-hint.is-yours { color: #c2570c; }
.pending-hint.is-yours strong { color: #b4480a; }
.row-footer { display: contents; }
.row-tags, .amount-date { display: none; }
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
.transaction-meta span + span::before { content: ''; display: inline-block; width: 3px; height: 3px; margin: 0 8px 0 0; border-radius: 50%; background: #bbc5d2; vertical-align: middle; }
.transaction-amount { display: flex; flex: 0 0 130px; flex-direction: column; align-items: flex-end; gap: 4px; text-align: right; font-variant-numeric: tabular-nums; }
.transaction-amount > span { font-size: 14px; font-weight: 750; white-space: nowrap; }
.transaction-amount small { color: #8796ab; font-size: 10px; text-transform: capitalize; }
.income-text { color: #12865f; }
.expense-text { color: #d94357; }
.row-actions { display: flex; flex: 0 0 auto; gap: 5px; }
/* Every row action has a tinted background so it reads as a button. */
.row-action { display: flex; align-items: center; justify-content: center; width: 34px; height: 34px; border: 0; border-radius: 10px; cursor: pointer; transition: background .15s ease, transform .12s ease; }
.row-action:active { transform: scale(.92); }
.edit-action { color: #1769e8; background: #eaf2ff; }
.edit-action:hover { background: #d9e8ff; }
.delete-action { color: #c93449; background: #fdecef; }
.delete-action:hover { background: #fbdbe1; }
.approve-action { color: #fff; background: #13966e; box-shadow: 0 3px 8px rgb(19 150 110 / 25%); }
.approve-action:hover { background: #0f8562; }
.reject-action { color: #c2410c; background: #fff1e6; }
.reject-action:hover { background: #ffe2cc; }
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
  .transaction-search { margin-top: 10px; }
  .type-filters { display: flex; width: 100%; }
  .type-filters button { flex: 1; padding: 10px 8px; font-size: 13px; }
  .totals-summary { grid-template-columns: 1fr; }
  .totals-equation strong { font-size: 13px; }
  .pagination-controls { width: 100%; justify-content: space-between; gap: 4px; }
  .page-size-control span { display: none; }
  .transaction-list { border-radius: 14px; }
  /* Phone row: icon | title + details (+ pending line) | amount + date, then a bar with status and actions. */
  .transaction-row { display: grid; grid-template-columns: 38px minmax(0, 1fr) auto; column-gap: 12px; row-gap: 0; align-items: start; padding: 14px 14px 12px; }
  .transaction-row.is-pending { background: #fffdf7; }
  .transaction-type-icon { grid-column: 1; grid-row: 1; width: 38px; height: 38px; border-radius: 11px; }
  .transaction-info { grid-column: 2; grid-row: 1; padding-top: 1px; }
  .transaction-title-line { display: block; }
  .transaction-title-line strong { display: -webkit-box; overflow: hidden; color: #1d365c; font-size: 14px; line-height: 1.3; white-space: normal; overflow-wrap: anywhere; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
  .wide-only, .amount-type, .meta-number, .meta-date, .transaction-meta .meta-number + span::before { display: none; }
  /* One line that trims at the end, rather than trimming every part. */
  .transaction-meta { display: block; margin-top: 5px; overflow: hidden; font-size: 12px; text-overflow: ellipsis; }
  .transaction-meta span { overflow: visible; }
  .transaction-meta span + span::before { margin: 0 7px; }
  .transaction-amount { grid-column: 3; grid-row: 1; max-width: 132px; gap: 4px; padding-top: 1px; }
  .transaction-amount > span { font-size: 15px; }
  .amount-date { display: block; color: #8796ab; font-size: 11px; white-space: nowrap; }
  /* Narrow column: "by … · time" drops under the status, lined up with its text. */
  .pending-hint { flex-wrap: wrap; row-gap: 2px; margin-top: 6px; font-size: 11.5px; }
  /* Stays beside its icon and wraps when a long amount leaves little room. */
  .pending-hint strong { flex: 1 1 0; min-width: 0; white-space: normal; }
  /* No hover on phones, so show the whole reason instead of cutting it off. */
  .people-line { margin-top: 6px; font-size: 11.5px; }
  .people-outcome { align-items: flex-start; white-space: normal; overflow-wrap: anywhere; }
  .people-outcome svg { margin-top: 2px; }
  .pending-by { flex: 1 0 100%; padding-left: 17px; white-space: normal; overflow-wrap: break-word; }
  .pending-by::before { content: none; }
  /* The bar uses the full row width; when status, file and actions still don't fit (pending rows
     with files on narrow phones), the actions wrap to their own line, on the right. */
  .row-footer { display: flex; grid-column: 1 / -1; grid-row: 2; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px; margin-top: 10px; }
  .row-tags { display: flex; flex-wrap: wrap; min-width: 0; align-items: center; gap: 6px; }
  .row-tags .attachment-pill { padding: 5px 10px; }
  .row-actions { gap: 6px; margin-left: auto; }
  .row-action { width: 34px; height: 34px; }
}
</style>
