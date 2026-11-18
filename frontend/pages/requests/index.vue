<template>
  <div class="requests-page">
    <div class="page-heading">
      <div>
        <p class="eyebrow">FINANCIAL ACTIVITY</p>
        <h1>Requests</h1>
        <p class="heading-copy">Expenses waiting for approval. Once approved, a request moves to Expense.</p>
      </div>
      <button v-if="canCreateRequest" type="button" class="create-button" @click="formOpen = true"><Plus :size="16" /> New request</button>
    </div>

    <div class="list-toolbar">
      <div class="status-filters" role="group" aria-label="Request status">
        <button v-for="chip in chips" :key="chip.value" type="button" :aria-pressed="status === chip.value" :class="[{ active: status === chip.value }, `chip-${chip.value || 'all'}`]" @click="setStatus(chip.value)">
          <span>{{ chip.label }}</span>
          <span v-if="counts" class="chip-count">{{ chip.count }}</span>
        </button>
      </div>
      <ListSearch v-model="search" class="request-search" placeholder="Search by number or description…" />
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

    <div v-if="loading" class="list-state"><span class="mini-spinner"></span> Loading requests…</div>
    <div v-else-if="error && !items.length" class="list-state list-error"><p>{{ error }}</p><button type="button" @click="reset()">Try again</button></div>
    <div v-else-if="!items.length" class="list-state empty-state">
      <span class="empty-icon"><ClipboardCheck :size="22" /></span>
      <strong>{{ empty.title }}</strong>
      <span>{{ empty.copy }}</span>
    </div>
    <div v-else class="request-list">
      <article v-for="request in items" :key="request.id" class="request-row" :class="`is-${request.status}`">
        <div class="request-icon" :class="`tone-${request.status}`" aria-hidden="true">
          <Clock3 v-if="request.status === 'pending'" :size="18" :stroke-width="2.2" />
          <Check v-else-if="request.status === 'approved'" :size="18" :stroke-width="2.6" />
          <X v-else-if="request.status === 'rejected'" :size="18" :stroke-width="2.6" />
          <Trash2 v-else :size="17" />
        </div>
        <div class="request-info">
          <div class="request-title-line">
            <strong>{{ title(request) }}</strong>
            <span class="status-badge wide-only" :class="`status-${request.status}`">{{ statusLabel(request.status) }}</span>
            <button v-if="request.attachments?.length" type="button" class="attachment-pill wide-only" :aria-label="attachmentLabel(request)" :title="attachmentLabel(request)" @click="openAttachment(request)">
              <ImageIcon v-if="request.attachments[0]!.type.startsWith('image/')" :size="12" /><FileText v-else :size="12" />
              <span>{{ request.attachments.length > 1 ? `${request.attachments.length} files` : 'View' }}</span>
            </button>
          </div>
          <div class="request-meta">
            <span class="meta-number">{{ request.transaction_number }}</span>
            <span>{{ request.category?.name || 'Uncategorized' }}</span>
            <span>{{ request.branch?.name || 'No branch' }}</span>
            <span class="meta-date">{{ formatDate(request.transaction_date) }}</span>
          </div>
          <p v-if="request.status === 'pending'" class="pending-hint" :class="{ 'is-yours': request.can_approve || request.can_reject }">
            <Clock3 :size="12" :stroke-width="2.4" aria-hidden="true" />
            <strong>{{ request.can_approve || request.can_reject ? 'Needs your approval' : 'Waiting for approval' }}</strong>
            <span class="pending-by">{{ requestedBy(request) }}</span>
          </p>
          <p v-else class="people-line" :class="request.outcome ? `is-${request.outcome.action}` : ''">
            <span class="people-created"><UserRound :size="12" :stroke-width="2.4" aria-hidden="true" />Requested by {{ who(request.created_by) }}</span>
            <span v-if="request.outcome" class="people-outcome" :title="outcomeTitle(request)">
              <Check v-if="request.outcome.action === 'approved'" :size="12" :stroke-width="2.8" aria-hidden="true" />
              <X v-else-if="request.outcome.action === 'rejected'" :size="12" :stroke-width="2.8" aria-hidden="true" />
              <Trash2 v-else :size="12" :stroke-width="2.4" aria-hidden="true" />
              {{ outcomeText(request) }}
            </span>
          </p>
          <p v-if="request.status === 'pending' && !request.attachments?.length" class="receipt-hint">No receipt yet: it's needed before approval.</p>
        </div>
        <div class="request-amount">
          <span>{{ formatMoney(request.amount, request.currency) }}</span>
          <small class="amount-date">{{ formatDate(request.transaction_date) }}</small>
        </div>
        <div class="row-footer">
          <div class="row-tags">
            <span class="status-badge" :class="`status-${request.status}`">{{ statusLabel(request.status) }}</span>
            <button v-if="request.attachments?.length" type="button" class="attachment-pill" :aria-label="attachmentLabel(request)" @click="openAttachment(request)">
              <ImageIcon v-if="request.attachments[0]!.type.startsWith('image/')" :size="12" /><FileText v-else :size="12" />
              <span>{{ request.attachments.length > 1 ? `${request.attachments.length} files` : 'View file' }}</span>
            </button>
          </div>
          <div class="row-actions">
            <template v-if="request.status === 'pending'">
              <button v-if="request.can_approve" type="button" class="row-action approve-action" :aria-label="`Approve ${request.transaction_number}`" title="Approve" @click="openDecision(request, 'approve')"><Check :size="16" :stroke-width="2.6" /><span>Approve</span></button>
              <button v-if="request.can_reject" type="button" class="row-action reject-action" :aria-label="`Reject ${request.transaction_number}`" title="Reject" @click="openDecision(request, 'reject')"><X :size="16" :stroke-width="2.6" /><span>Reject</span></button>
            </template>
            <NuxtLink v-else-if="request.status === 'approved' && canViewExpense" :to="{ path: '/transactions', query: { type: 'expense', search: request.transaction_number } }" class="row-action expense-link" :aria-label="`Open ${request.transaction_number} in Expense`">
              <span>In Expense</span><ArrowRight :size="14" />
            </NuxtLink>
          </div>
        </div>
      </article>
      <InfiniteLoader :has-more="hasMore" :loading="loadingMore" :count="items.length" @more="loadMore" />
      <p v-if="error && items.length" class="more-error">{{ error }} <button type="button" @click="refresh()">Try again</button></p>
    </div>

    <TransactionModal v-model="formOpen" request @saved="onSaved" />
    <ApprovalDialog
      v-model="decisionOpen"
      :transaction="decisionRequest"
      :action="decisionAction"
      @done="onDecided"
      @files-changed="onFilesChanged"
      @preview="(index) => decisionRequest && openAttachment(decisionRequest, index)"
    />
    <!-- After the approval dialog so a preview opened from it shows on top. -->
    <AttachmentViewer v-model="viewerOpen" :transaction="viewerRequest" :start-index="viewerStart" @changed="onFilesChanged" />
  </div>
</template>

<script setup lang="ts">
import { ArrowRight, Check, ClipboardCheck, Clock3, FileText, ImageIcon, Plus, RotateCcw, Trash2, UserRound, X } from 'lucide-vue-next'
import type { Transaction } from '~/types/api'
import { formatMoney } from '~/utils/money'
import { richTextToPlain } from '~/utils/richText'
import { formatDateKey, type DateRangeValue } from '~/utils/dateRanges'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { useCurrency } from '~/composables/useCurrency'
import { useInfiniteList } from '~/composables/useInfiniteList'
import { usePendingRequests } from '~/composables/usePendingRequests'
import { useToast } from '~/composables/useToast'

type Status = '' | 'pending' | 'approved' | 'rejected'
interface Counts { pending: number; approved: number; rejected: number; all: number }

const { api } = useApi()
const { user } = useAuth()
const { can, canCreateRequest } = usePermissions()
const toast = useToast()
const canViewExpense = computed(() => can('expense.view', 'transactions.view'))

// Pending first: it's the list people act on.
const status = ref<Status>('pending')
const search = ref('')
// Every date by default, so an old request still waiting isn't hidden.
const dateRange = ref<DateRangeValue>({ from: '', to: '' })
const currency = ref('')
const branchFilter = ref('')
const counts = ref<Counts | null>(null)
const { options: branchOptions, showFilter: showBranchFilter, labelFor: branchLabel } = useBranchOptions()
const { options: currencyChoices } = useCurrency()
const currencyOptions = computed(() => [{ value: '', label: 'All currencies' }, ...currencyChoices(currency.value)])

const chips = computed(() => [
  { value: 'pending' as Status, label: 'Pending', count: counts.value?.pending ?? 0 },
  { value: 'approved' as Status, label: 'Approved', count: counts.value?.approved ?? 0 },
  { value: 'rejected' as Status, label: 'Rejected', count: counts.value?.rejected ?? 0 },
  { value: '' as Status, label: 'All', count: counts.value?.all ?? 0 },
])
const filtersActive = computed(() => !!dateRange.value.from || !!currency.value || !!branchFilter.value)
const filterSummary = computed(() => {
  const parts = [
    dateRange.value.from ? `${formatDateKey(dateRange.value.from)} – ${formatDateKey(dateRange.value.to)}` : 'Any date',
    currency.value || 'All currencies',
  ]
  if (showBranchFilter.value) parts.push(branchLabel(branchFilter.value))
  return parts.join(' · ')
})
const empty = computed(() => {
  if (search.value) return { title: 'No requests found', copy: `Nothing matches “${search.value}”.` }
  switch (status.value) {
    case 'pending': return { title: 'Nothing waiting', copy: 'New requests show here until someone approves or rejects them.' }
    case 'approved': return { title: 'No approved requests', copy: 'Approved requests move to Expense and are listed here too.' }
    case 'rejected': return { title: 'No rejected requests', copy: 'Rejected requests stay here with the reason.' }
    default: return { title: 'No requests yet', copy: canCreateRequest.value ? 'Use “New request” to ask for an expense.' : 'Requests will show here.' }
  }
})

const { items, loading, loadingMore, error, hasMore, reset, loadMore, refresh } = useInfiniteList<Transaction>(
  async (page, size) => {
    const result = await api<{ items: Transaction[]; total: number; counts: Counts }>('/requests', {
      query: {
        status: status.value || undefined,
        search: search.value || undefined,
        date_from: dateRange.value.from || undefined,
        date_to: dateRange.value.to || undefined,
        currency: currency.value || undefined,
        branch_id: branchFilter.value || undefined,
        page,
        page_size: size,
      },
    })
    counts.value = result.counts
    return result
  },
  request => request.id,
  20,
  'Could not load requests.',
)

function setStatus(value: Status) {
  if (status.value === value) return
  status.value = value
  void reset()
}

function resetFilters() {
  dateRange.value = { from: '', to: '' }
  currency.value = ''
  branchFilter.value = ''
}

watch(() => [search.value, dateRange.value.from, dateRange.value.to, currency.value, branchFilter.value], () => { void reset() })
onMounted(reset)

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

function who(person?: { id: string; name: string }) {
  if (!person?.name) return 'someone'
  return person.id === user.value?.id ? 'you' : person.name
}

function title(request: Transaction) {
  return richTextToPlain(request.description) || request.category?.name || request.transaction_number
}

function statusLabel(value: string) {
  return { pending: 'Pending', approved: 'Approved', rejected: 'Rejected', voided: 'Deleted', reversed: 'Reversed' }[value] || value
}

// "by Dara · 2 h ago"
function requestedBy(request: Transaction) {
  return [`by ${who(request.created_by)}`, timeAgo(request.created_at)].filter(Boolean).join(' · ')
}

// "Approved by Sokha", "Rejected by Sokha: “no receipt”".
function outcomeText(request: Transaction) {
  const outcome = request.outcome
  if (!outcome) return ''
  const verb = { approved: 'Approved', rejected: 'Rejected', voided: 'Deleted' }[outcome.action]
  const reason = outcome.action !== 'approved' && outcome.notes ? `: “${outcome.notes}”` : ''
  return `${verb} by ${who(outcome.by)}${reason}`
}

function outcomeTitle(request: Transaction) {
  const outcome = request.outcome
  if (!outcome) return ''
  const date = new Date(outcome.at)
  const when = Number.isNaN(date.valueOf()) ? '' : ` on ${new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' }).format(date)}`
  return `${outcomeText(request)}${when}${outcome.notes && outcome.action === 'approved' ? ` · Note: ${outcome.notes}` : ''}`
}

function attachmentLabel(request: Transaction) {
  const files = request.attachments || []
  return files.length > 1 ? `View ${files.length} files` : `View ${files[0]?.name || 'file'}`
}

// New request
const formOpen = ref(false)
// The sidebar's pending chip follows what happens here.
const { refreshPendingRequests } = usePendingRequests()
async function onSaved(saved: Transaction) {
  toast.success(`${formatMoney(saved.amount, saved.currency)} waits here until someone approves it.`, { title: 'Request sent' })
  if (status.value !== 'pending' && status.value !== '') status.value = 'pending'
  void refreshPendingRequests()
  await reset()
}

// Approve / reject (buttons only where the API says this person may decide)
const decisionOpen = ref(false)
const decisionRequest = ref<Transaction | null>(null)
const decisionAction = ref<'approve' | 'reject'>('approve')
function openDecision(request: Transaction, action: 'approve' | 'reject') {
  decisionRequest.value = request
  decisionAction.value = action
  decisionOpen.value = true
}
async function onDecided(updated: Transaction) {
  if (updated.status === 'approved') toast.success(`${updated.transaction_number} is now an expense and counts toward totals.`, { title: 'Request approved' })
  else toast.success(`${updated.transaction_number} stays here as rejected and won’t count.`, { title: 'Request rejected' })
  decisionRequest.value = null
  void refreshPendingRequests()
  await refresh()
}

// Files: the viewer, and files added or deleted from it or the approval dialog
const viewerOpen = ref(false)
const viewerRequest = ref<Transaction | null>(null)
const viewerStart = ref(0)
function openAttachment(request: Transaction, index = 0) {
  viewerRequest.value = request
  viewerStart.value = index
  viewerOpen.value = true
}
function onFilesChanged(updated: Transaction) {
  const merge = (txn: Transaction | null) => txn && txn.id === updated.id ? { ...txn, attachments: updated.attachments } : txn
  items.value = items.value.map(txn => merge(txn)!)
  viewerRequest.value = merge(viewerRequest.value)
  decisionRequest.value = merge(decisionRequest.value)
}
</script>

<style scoped>
.requests-page { color: #183054; }
.page-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; margin-bottom: 20px; }
.eyebrow { margin: 0 0 6px; color: #6180ae; font-size: 11px; font-weight: 800; letter-spacing: .13em; }
.page-heading h1 { margin: 0; color: #102348; font-size: 26px; line-height: 1.15; font-weight: 750; letter-spacing: -.035em; }
.heading-copy { margin: 7px 0 0; color: #7083a2; font-size: 13px; }
.create-button { display: inline-flex; flex: 0 0 auto; align-items: center; justify-content: center; gap: 7px; min-height: 42px; border: 0; border-radius: 10px; padding: 0 16px; color: #fff; background: #1769e8; box-shadow: 0 5px 12px rgb(23 105 232 / 22%); font-size: 12px; font-weight: 700; cursor: pointer; transition: filter .15s ease, transform .15s ease; }
.create-button:hover { filter: brightness(.95); transform: translateY(-1px); }
.list-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 12px; }
.status-filters { display: inline-flex; gap: 3px; border: 1px solid #e2e9f3; border-radius: 11px; padding: 3px; background: #fff; }
.status-filters button { display: inline-flex; align-items: center; justify-content: center; gap: 6px; border: 0; border-radius: 8px; padding: 8px 12px; color: #70829d; background: transparent; font-size: 12px; font-weight: 650; white-space: nowrap; cursor: pointer; transition: color .15s ease, background .15s ease; }
.status-filters button:hover:not(.active) { background: #f4f7fb; }
.status-filters button.active { color: #fff; background: #1769e8; box-shadow: 0 3px 9px rgb(23 105 232 / 22%); }
.chip-count { min-width: 20px; border-radius: 999px; padding: 1px 6px; color: #5b6f8c; background: #eef2f7; font-size: 11px; font-weight: 750; font-variant-numeric: tabular-nums; }
.chip-pending:not(.active) .chip-count { color: #a46c0d; background: #fff4d7; }
.status-filters button.active .chip-count { color: #1769e8; background: #fff; }
.request-search { flex: 0 1 340px; margin-left: auto; }

.request-list { overflow: hidden; border: 1px solid #e2eaf4; border-radius: 14px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.request-row { display: flex; align-items: center; gap: 14px; min-width: 0; padding: 14px 16px; border-bottom: 1px solid #edf1f7; transition: background .15s ease; }
.request-row:hover { background: #fbfcfe; }
.request-list > .request-row:last-of-type { border-bottom: 0; }
.request-icon { display: grid; flex: 0 0 auto; width: 42px; height: 42px; place-items: center; border-radius: 12px; color: #6a7c96; background: #f0f3f7; }
.tone-pending { color: #b8770b; background: #fff4d7; }
.tone-approved { color: #11956c; background: #e4f8ef; }
.tone-rejected { color: #dd4055; background: #fff0f2; }
.request-info { flex: 1; min-width: 0; }
.request-title-line { display: flex; align-items: center; gap: 8px; min-width: 0; }
.request-title-line strong { overflow: hidden; color: #223b60; font-size: 13px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
.status-badge { flex: 0 0 auto; border-radius: 999px; padding: 4px 9px; color: #647790; background: #f0f3f7; font-size: 10px; font-weight: 700; }
.status-approved { color: #14835f; background: #e4f7ee; }
.status-pending { color: #a46c0d; background: #fff4d7; }
.status-rejected, .status-voided { color: #bd3e50; background: #ffeaed; }
.attachment-pill { display: inline-flex; flex: 0 0 auto; align-items: center; gap: 4px; border: 1px solid #d6e4fb; border-radius: 999px; padding: 3px 9px; color: #1769e8; background: #f2f7ff; font-size: 11px; font-weight: 700; cursor: pointer; }
.attachment-pill:hover { background: #e4eeff; }
.request-meta { display: flex; align-items: center; gap: 8px; min-width: 0; margin-top: 6px; overflow: hidden; color: #8294b0; font-size: 11px; white-space: nowrap; }
.request-meta span { overflow: hidden; text-overflow: ellipsis; }
.request-meta span + span::before { content: ''; display: inline-block; width: 3px; height: 3px; margin: 0 8px 0 0; border-radius: 50%; background: #bbc5d2; vertical-align: middle; }
.pending-hint { display: flex; align-items: center; gap: 5px; min-width: 0; margin: 5px 0 0; overflow: hidden; color: #a0732a; font-size: 11px; line-height: 1.3; white-space: nowrap; }
.pending-hint svg { flex: 0 0 auto; }
.pending-hint strong { flex: 0 0 auto; font-weight: 700; }
.pending-hint.is-yours { color: #c2570c; }
.pending-hint.is-yours strong { color: #b4480a; }
.pending-by { overflow: hidden; color: #9a8a6c; text-overflow: ellipsis; }
.pending-by::before { content: '·'; margin-right: 5px; color: #c9b890; }
.receipt-hint { margin: 4px 0 0; color: #8c97a8; font-size: 11px; }
.people-line { display: flex; flex-wrap: wrap; align-items: center; gap: 2px 10px; min-width: 0; margin: 5px 0 0; color: #8294b0; font-size: 11px; line-height: 1.35; }
.people-line > span { display: inline-flex; min-width: 0; align-items: center; gap: 4px; }
.people-line svg { flex: 0 0 auto; }
.people-outcome { overflow: hidden; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.people-line.is-approved .people-outcome { color: #14835f; }
.people-line.is-rejected .people-outcome { color: #bd3e50; }
.people-line.is-voided .people-outcome { color: #8a96a8; }
.request-amount { display: flex; flex: 0 0 130px; flex-direction: column; align-items: flex-end; gap: 4px; color: #d94357; text-align: right; font-variant-numeric: tabular-nums; }
.request-amount > span { font-size: 14px; font-weight: 750; white-space: nowrap; }
.is-rejected .request-amount > span, .is-voided .request-amount > span { color: #9aa6b8; text-decoration: line-through; }
.amount-date { display: none; color: #8796ab; font-size: 11px; white-space: nowrap; }
.row-footer { display: contents; }
.row-tags { display: none; }
.row-actions { display: flex; flex: 0 0 auto; gap: 6px; min-width: 0; }
.row-action { display: inline-flex; align-items: center; justify-content: center; gap: 5px; height: 34px; border: 0; border-radius: 10px; padding: 0 11px; font-size: 12px; font-weight: 700; text-decoration: none; cursor: pointer; transition: background .15s ease, transform .12s ease; }
.row-action:active { transform: scale(.95); }
.approve-action { color: #fff; background: #13966e; box-shadow: 0 3px 8px rgb(19 150 110 / 25%); }
.approve-action:hover { background: #0f8562; }
.reject-action { color: #c2410c; background: #fff1e6; }
.reject-action:hover { background: #ffe2cc; }
.expense-link { color: #1769e8; background: #eef5ff; }
.expense-link:hover { background: #dfeaff; }

.list-state { display: flex; align-items: center; justify-content: center; gap: 9px; min-height: 170px; border: 1px solid #e2eaf4; border-radius: 14px; padding: 24px; color: #8090a7; background: #fff; font-size: 12px; text-align: center; }
.mini-spinner { width: 16px; height: 16px; border: 2px solid #dbe8f8; border-top-color: #2678ed; border-radius: 50%; animation: spin .7s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
.list-error { justify-content: space-between; color: #bd3e50; }
.list-error p { margin: 0; }
.list-error button, .more-error button { border: 0; border-radius: 8px; padding: 9px 14px; color: #fff; background: #1769e8; font-size: 12px; font-weight: 650; cursor: pointer; }
.more-error { display: flex; align-items: center; justify-content: center; gap: 10px; margin: 0; padding: 12px; color: #bd3e50; font-size: 12px; }
.empty-state { flex-direction: column; gap: 8px; }
.empty-icon { display: grid; width: 48px; height: 48px; margin-bottom: 4px; place-items: center; border-radius: 14px; color: #5182c9; background: #eef5ff; }
.empty-state strong { color: #405a7e; font-size: 14px; }
.empty-state > span:last-child { font-size: 12px; }

@media (max-width: 700px) {
  .page-heading { display: block; margin-bottom: 16px; }
  .create-button { width: 100%; margin-top: 14px; }
  .list-toolbar { display: block; }
  .status-filters { display: flex; width: 100%; }
  .status-filters button { flex: 1; min-width: 0; padding: 9px 4px; font-size: 12px; }
  .request-search { margin-top: 10px; }
  /* Phone row: icon | title + details | amount, then a bar with status, files and actions. */
  .request-row { display: grid; grid-template-columns: 38px minmax(0, 1fr) auto; column-gap: 12px; align-items: start; padding: 14px 14px 12px; }
  .request-row.is-pending { background: #fffdf7; }
  .request-icon { grid-column: 1; grid-row: 1; width: 38px; height: 38px; border-radius: 11px; }
  .request-info { grid-column: 2; grid-row: 1; padding-top: 1px; }
  .request-title-line { display: block; }
  .request-title-line strong { display: -webkit-box; overflow: hidden; color: #1d365c; font-size: 14px; line-height: 1.3; white-space: normal; overflow-wrap: anywhere; -webkit-box-orient: vertical; -webkit-line-clamp: 2; }
  .wide-only, .meta-number, .meta-date { display: none; }
  .request-meta { display: block; margin-top: 5px; font-size: 12px; text-overflow: ellipsis; }
  .request-meta span { overflow: visible; }
  .request-meta span + span::before { margin: 0 7px; }
  .request-meta .meta-number + span::before { display: none; }
  .request-amount { grid-column: 3; grid-row: 1; max-width: 132px; padding-top: 1px; }
  .request-amount > span { font-size: 15px; }
  .amount-date { display: block; }
  .pending-hint { flex-wrap: wrap; row-gap: 2px; margin-top: 6px; font-size: 11.5px; }
  .pending-hint strong { flex: 1 1 0; min-width: 0; white-space: normal; }
  .pending-by { flex: 1 0 100%; padding-left: 17px; white-space: normal; overflow-wrap: break-word; }
  .pending-by::before { content: none; }
  .people-line { margin-top: 6px; font-size: 11.5px; }
  .people-outcome { align-items: flex-start; white-space: normal; overflow-wrap: anywhere; }
  .people-outcome svg { margin-top: 2px; }
  .receipt-hint { font-size: 11.5px; }
  .row-footer { display: flex; grid-column: 1 / -1; grid-row: 2; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 8px; margin-top: 10px; }
  .row-tags { display: flex; flex-wrap: wrap; min-width: 0; align-items: center; gap: 6px; }
  .row-tags .attachment-pill { padding: 5px 10px; }
  .row-actions { margin-left: auto; }
  .row-action { height: 36px; padding: 0 13px; }
}
</style>
