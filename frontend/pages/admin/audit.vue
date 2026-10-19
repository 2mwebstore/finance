<template>
  <section class="audit-page">
    <header class="page-heading">
      <div>
        <p class="eyebrow">SECURITY</p>
        <h1>Audit log</h1>
        <p class="heading-copy">Sign-ins, money changes, and admin actions across the system.</p>
      </div>
      <button v-if="canCleanup" type="button" class="cleanup-button" @click="openCleanup"><Trash2 :size="16" /> Clean up</button>
    </header>

    <FilterCard :summary="filterSummary" :active="filtersActive">
      <div class="filter-field date-range-filter">
        <span>Date range</span>
        <DateRangePicker v-model="dateRange" />
      </div>
      <div v-if="showBranchFilter" class="filter-field filter-wide">
        <span>Branch</span>
        <SearchSelect v-model="branchFilter" :options="branchOptions" placeholder="All branches" search-placeholder="Search branches…" aria-label="Branch filter" />
      </div>
      <div class="filter-field action-filter">
        <span>Action</span>
        <SearchSelect v-model="actionFilter" :options="actionOptions" placeholder="All actions" search-placeholder="Search actions…" aria-label="Filter audit actions" />
      </div>
      <button type="button" class="filter-reset" :disabled="!filtersActive" @click="resetFilters"><RotateCcw :size="15" /> Reset</button>
    </FilterCard>

    <p v-if="notice" class="notice" role="status"><CheckCircle2 :size="17" /> {{ notice }}</p>

    <div v-if="loading" class="audit-state">Loading audit events…</div>
    <div v-else-if="err" class="audit-state is-error">{{ err }}</div>
    <template v-else>
      <ul class="audit-list">
        <li v-for="entry in items" :key="entry.id" class="audit-row">
          <span class="audit-icon" :class="entry.outcome === 'success' ? toneFor(entry.action) : 'tone-failure'">
            <component :is="iconFor(entry.action)" :size="18" />
          </span>
          <div class="audit-info">
            <div class="audit-title-line">
              <strong>{{ actionLabel(entry.action) }}</strong>
              <span class="outcome-badge" :class="entry.outcome === 'success' ? 'is-success' : 'is-failure'">{{ entry.outcome }}</span>
            </div>
            <p class="audit-meta">
              <span>{{ formatTime(entry.created_at) }}</span>
              <i>•</i><span>{{ actorLabel(entry) }}</span>
              <template v-if="entry.ip_address"><i>•</i><span>{{ entry.ip_address }}</span></template>
            </p>
            <p v-if="entry.entity_type" class="audit-entity">{{ entry.entity_type }}<template v-if="entry.entity_id"> #{{ entry.entity_id.slice(0, 8) }}</template></p>
          </div>
        </li>
        <li v-if="!items.length" class="audit-empty">
          <ShieldCheck :size="22" />
          <strong>No audit events</strong>
          <span>Try a wider date range or another action.</span>
        </li>
      </ul>
      <div v-if="items.length" class="audit-footer">
        <span>Showing {{ items.length }} of {{ total }}</span>
        <button v-if="items.length < total" type="button" class="load-more" :disabled="loadingMore" @click="loadMore">{{ loadingMore ? 'Loading…' : 'Load more' }}</button>
      </div>
    </template>

    <div v-if="cleanupOpen" class="cleanup-backdrop" @click.self="closeCleanup" @keydown.esc="closeCleanup">
      <section class="cleanup-dialog" role="alertdialog" aria-modal="true" aria-labelledby="cleanup-title">
        <span class="sheet-handle" aria-hidden="true"></span>
        <button type="button" class="dialog-close" aria-label="Close" :disabled="cleaning" @click="closeCleanup"><X :size="18" /></button>
        <div class="cleanup-icon"><Trash2 :size="21" /></div>
        <p class="cleanup-eyebrow">RETENTION</p>
        <h2 id="cleanup-title">Clean up the audit log</h2>
        <p class="cleanup-copy">Choose how much history to keep. Older events are deleted permanently and can't be restored.</p>

        <div class="keep-options" role="radiogroup" aria-label="History to keep">
          <button
            v-for="option in keepOptions"
            :key="option.months"
            type="button"
            role="radio"
            class="keep-option"
            :class="{ 'is-selected': keepMonths === option.months }"
            :aria-checked="keepMonths === option.months"
            :disabled="cleaning"
            @click="selectKeep(option.months)"
          >
            <span class="keep-radio"></span>
            <span class="keep-copy"><strong>Keep last {{ option.label }}</strong><small>Deletes events before {{ option.cutoffLabel }}</small></span>
          </button>
        </div>

        <p class="cleanup-preview" :class="{ 'has-events': (preview?.would_delete || 0) > 0 }">
          <template v-if="previewLoading">Counting events…</template>
          <template v-else-if="preview && preview.would_delete > 0"><strong>{{ preview.would_delete.toLocaleString() }}</strong> {{ preview.would_delete === 1 ? 'event' : 'events' }} will be deleted.</template>
          <template v-else-if="preview">Nothing to delete. Every event is newer than {{ selectedOption.cutoffLabel }}.</template>
        </p>
        <p v-if="cleanupError" class="cleanup-error" role="alert">{{ cleanupError }}</p>

        <div class="cleanup-actions">
          <button type="button" class="cancel-button" :disabled="cleaning" @click="closeCleanup">Cancel</button>
          <button type="button" class="confirm-button" :disabled="cleaning || previewLoading || !preview || preview.would_delete === 0" @click="runCleanup">
            <Trash2 :size="15" />{{ cleaning ? 'Deleting…' : preview && preview.would_delete > 0 ? `Delete ${preview.would_delete.toLocaleString()} ${preview.would_delete === 1 ? 'event' : 'events'}` : 'Delete events' }}
          </button>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import {
  Building2,
  CheckCircle2,
  FileDown,
  KeyRound,
  ReceiptText,
  RotateCcw,
  Settings,
  ShieldCheck,
  Trash2,
  UserRound,
  Wallet,
  X,
} from 'lucide-vue-next'
import type { AuditEntry } from '~/types/api'
import { currentMonthRange, formatDateKey } from '~/utils/dateRanges'
import { useBranchOptions } from '~/composables/useBranchOptions'

const PAGE_SIZE = 50

const { api } = useApi()
const { P, canAll, hasFullAccess } = usePermissions()
const items = ref<AuditEntry[]>([])
const total = ref(0)
const page = ref(1)
const loading = ref(true)
const loadingMore = ref(false)
const err = ref('')
const notice = ref('')
const actionFilter = ref('')
const branchFilter = ref('')
const dateRange = ref(currentMonthRange())
const { options: branchOptions, showFilter: showBranchFilter, labelFor: branchLabel } = useBranchOptions()
let requestSequence = 0

// Deleting history is limited to people who run the whole system (owner, admin, super admin).
const canCleanup = computed(() => hasFullAccess.value && canAll(P.AuditLogsView, P.SettingsManage))

const ACTION_LABELS: Record<string, string> = {
  'auth.login': 'Signed in',
  'auth.logout': 'Signed out',
  'auth.login_failed': 'Sign-in failed',
  'auth.setup_admin': 'Super admin set up',
  'auth.password_change': 'Password changed',
  'transaction.create': 'Transaction created',
  'transaction.update': 'Transaction updated',
  'transaction.void': 'Transaction deleted',
  'transaction.approve': 'Transaction approved',
  'transaction.reject': 'Transaction rejected',
  'transaction.reverse': 'Transaction reversed',
  'user.create': 'User created',
  'user.update': 'User updated',
  'user.assign_roles': 'User roles changed',
  'user.assign_branches': 'User branches changed',
  'branch.create': 'Branch created',
  'branch.update': 'Branch updated',
  'cash.opening_balance': 'Opening balance set',
  'cash.reconciliation': 'Cash reconciled',
  'report.export': 'Report exported',
  'settings.change': 'Settings changed',
  'audit.cleanup': 'Audit log cleaned up',
}
const actionOptions = [
  { value: '', label: 'All actions' },
  ...Object.entries(ACTION_LABELS).map(([value, label]) => ({ value, label })),
]

const isThisMonth = computed(() => {
  const month = currentMonthRange()
  return dateRange.value.from === month.from && dateRange.value.to === month.to
})
const filtersActive = computed(() => !isThisMonth.value || !!actionFilter.value || !!branchFilter.value)
const filterSummary = computed(() => {
  const period = isThisMonth.value
    ? 'This month'
    : dateRange.value.from ? `${formatDateKey(dateRange.value.from)} – ${formatDateKey(dateRange.value.to)}` : 'Any date'
  const parts = [period, actionFilter.value ? actionLabel(actionFilter.value) : 'All actions']
  if (showBranchFilter.value) parts.push(branchLabel(branchFilter.value))
  return parts.join(' · ')
})

function actionLabel(action: string) {
  return ACTION_LABELS[action] || action
}

function iconFor(action: string): Component {
  const area = action.split('.')[0]
  if (action === 'auth.password_change') return KeyRound
  return ({ auth: KeyRound, transaction: ReceiptText, user: UserRound, branch: Building2, cash: Wallet, report: FileDown, settings: Settings, audit: Trash2 } as Record<string, Component>)[area] || ShieldCheck
}

function toneFor(action: string) {
  const area = action.split('.')[0]
  if (action === 'transaction.void' || area === 'audit') return 'tone-warning'
  return area === 'auth' ? 'tone-neutral' : 'tone-info'
}

function actorLabel(entry: AuditEntry) {
  if (entry.actor_email) return entry.actor_email
  return entry.actor_id ? `User ${entry.actor_id.slice(0, 8)}` : 'System'
}

function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.valueOf())
    ? value
    : new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' }).format(date)
}

async function fetchPage(pageNumber: number) {
  return api<{ items: AuditEntry[]; total: number }>('/audit-logs', {
    query: {
      action: actionFilter.value || undefined,
      branch_id: branchFilter.value || undefined,
      date_from: dateRange.value.from || undefined,
      date_to: dateRange.value.to || undefined,
      page: pageNumber,
      page_size: PAGE_SIZE,
    },
  })
}

async function load() {
  const sequence = ++requestSequence
  loading.value = true
  err.value = ''
  page.value = 1
  try {
    const res = await fetchPage(1)
    if (sequence !== requestSequence) return
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    if (sequence === requestSequence) err.value = e?.data?.error || 'Failed to load audit log'
  } finally {
    if (sequence === requestSequence) loading.value = false
  }
}

async function loadMore() {
  const sequence = requestSequence
  loadingMore.value = true
  try {
    const res = await fetchPage(page.value + 1)
    if (sequence !== requestSequence) return
    page.value += 1
    items.value = [...items.value, ...(res.items || []).filter(item => !items.value.some(existing => existing.id === item.id))]
    total.value = res.total || total.value
  } catch (e: any) {
    err.value = e?.data?.error || 'Failed to load more events'
  } finally {
    loadingMore.value = false
  }
}

function resetFilters() {
  dateRange.value = currentMonthRange()
  actionFilter.value = ''
  branchFilter.value = ''
}

// Retention cleanup
type KeepMonths = 1 | 2 | 3
const cleanupOpen = ref(false)
const keepMonths = ref<KeepMonths>(3)
const preview = ref<{ would_delete: number; cutoff: string } | null>(null)
const previewLoading = ref(false)
const cleaning = ref(false)
const cleanupError = ref('')
let previewSequence = 0

const keepOptions = computed(() => ([1, 2, 3] as KeepMonths[]).map((months) => {
  const cutoff = new Date()
  cutoff.setMonth(cutoff.getMonth() - months)
  return {
    months,
    label: months === 1 ? '1 month' : `${months} months`,
    cutoffLabel: new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric' }).format(cutoff),
  }
}))
const selectedOption = computed(() => keepOptions.value.find(option => option.months === keepMonths.value)!)

async function loadPreview() {
  const sequence = ++previewSequence
  previewLoading.value = true
  cleanupError.value = ''
  try {
    const res = await api<{ would_delete: number; cutoff: string }>('/audit-logs/cleanup', {
      method: 'POST',
      body: { keep_months: keepMonths.value, dry_run: true },
    })
    if (sequence === previewSequence) preview.value = res
  } catch (e: any) {
    if (sequence === previewSequence) {
      preview.value = null
      cleanupError.value = e?.data?.error || 'Could not count old events.'
    }
  } finally {
    if (sequence === previewSequence) previewLoading.value = false
  }
}

function openCleanup() {
  keepMonths.value = 3
  preview.value = null
  cleanupOpen.value = true
  void loadPreview()
}

function closeCleanup() {
  if (!cleaning.value) cleanupOpen.value = false
}

function selectKeep(months: KeepMonths) {
  if (keepMonths.value === months) return
  keepMonths.value = months
  preview.value = null
  void loadPreview()
}

async function runCleanup() {
  cleaning.value = true
  cleanupError.value = ''
  try {
    const res = await api<{ deleted: number }>('/audit-logs/cleanup', {
      method: 'POST',
      body: { keep_months: keepMonths.value },
    })
    notice.value = `Deleted ${res.deleted.toLocaleString()} ${res.deleted === 1 ? 'event' : 'events'} older than ${selectedOption.value.cutoffLabel}.`
    cleanupOpen.value = false
    await load()
  } catch (e: any) {
    cleanupError.value = e?.data?.error || 'Could not clean up the audit log.'
  } finally {
    cleaning.value = false
  }
}

watch(() => [actionFilter.value, branchFilter.value, dateRange.value.from, dateRange.value.to], () => { void load() })
onMounted(load)
</script>

<style scoped>
.audit-page { color: #183054; }
.page-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; margin-bottom: 20px; }
.eyebrow { margin: 0 0 6px; color: #6180ae; font-size: 11px; font-weight: 800; letter-spacing: .13em; }
.page-heading h1 { margin: 0; color: #102348; font-size: 26px; line-height: 1.15; font-weight: 750; letter-spacing: -.035em; }
.heading-copy { margin: 7px 0 0; color: #7083a2; font-size: 13px; }
.cleanup-button { display: inline-flex; flex: 0 0 auto; min-height: 42px; white-space: nowrap; align-items: center; gap: 7px; border: 1px solid #f3cdd3; border-radius: 10px; padding: 0 16px; color: #c93449; background: #fff; font-size: 13px; font-weight: 700; cursor: pointer; transition: background .15s ease; }
.cleanup-button:hover { background: #fff4f5; }
.notice { display: flex; align-items: center; gap: 8px; margin: 0 0 14px; border: 1px solid #c9ecdc; border-radius: 12px; padding: 11px 14px; color: #12795a; background: #effaf5; font-size: 13px; font-weight: 600; }
.audit-state { border: 1px solid #e2eaf4; border-radius: 14px; padding: 36px 16px; color: #8091a9; background: #fff; font-size: 13px; text-align: center; }
.audit-state.is-error { color: #c33d50; }
.audit-list { margin: 0; overflow: hidden; border: 1px solid #e2eaf4; border-radius: 14px; padding: 0; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); list-style: none; }
.audit-row { display: flex; align-items: flex-start; gap: 13px; padding: 14px 16px; border-bottom: 1px solid #edf1f7; }
.audit-row:last-child { border-bottom: 0; }
.audit-icon { display: grid; flex: 0 0 auto; width: 38px; height: 38px; place-items: center; border-radius: 11px; }
.tone-info { color: #1769e8; background: #eef5ff; }
.tone-neutral { color: #5c7396; background: #f1f4f9; }
.tone-warning { color: #b7791f; background: #fff6e0; }
.tone-failure { color: #d83d52; background: #fff0f2; }
.audit-info { flex: 1; min-width: 0; }
.audit-title-line { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.audit-title-line strong { overflow: hidden; color: #1d365c; font-size: 14px; font-weight: 700; text-overflow: ellipsis; white-space: nowrap; }
.outcome-badge { flex: 0 0 auto; border-radius: 999px; padding: 3px 9px; font-size: 11px; font-weight: 700; text-transform: capitalize; }
.outcome-badge.is-success { color: #14835f; background: #e4f7ee; }
.outcome-badge.is-failure { color: #bd3e50; background: #ffeaed; }
.audit-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 4px 7px; margin: 5px 0 0; color: #71839f; font-size: 12px; }
.audit-meta span { overflow: hidden; max-width: 260px; text-overflow: ellipsis; white-space: nowrap; }
.audit-meta i { color: #b5c1d1; font-style: normal; }
.audit-entity { margin: 4px 0 0; color: #94a3b8; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 11px; }
.audit-empty { display: flex; flex-direction: column; align-items: center; gap: 6px; padding: 40px 16px; color: #8193ab; font-size: 13px; text-align: center; }
.audit-empty svg { color: #9db5d6; }
.audit-empty strong { color: #2c4468; font-size: 14px; }
.audit-footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 12px; color: #8191a8; font-size: 12px; }
.load-more { min-height: 38px; border: 1px solid #dfe7f1; border-radius: 9px; padding: 0 16px; color: #3d5a82; background: #fff; font-size: 13px; font-weight: 650; cursor: pointer; }
.load-more:disabled { opacity: .6; cursor: wait; }

/* Cleanup dialog */
.cleanup-backdrop { position: fixed; inset: 0; z-index: 110; display: flex; align-items: center; justify-content: center; padding: 18px; background: rgb(8 24 48 / 56%); backdrop-filter: blur(4px); }
.cleanup-dialog { position: relative; width: min(460px, 100%); max-height: calc(100dvh - 36px); overflow-y: auto; border: 1px solid #e5eaf2; border-radius: 18px; padding: 24px; background: #fff; box-shadow: 0 24px 72px rgb(8 24 48 / 28%); }
.sheet-handle { display: none; }
.dialog-close { position: absolute; top: 16px; right: 16px; display: grid; width: 34px; height: 34px; place-items: center; border: 0; border-radius: 10px; color: #73849d; background: #f3f6fb; cursor: pointer; }
.cleanup-icon { display: grid; width: 44px; height: 44px; margin-bottom: 14px; place-items: center; border-radius: 13px; color: #d83d52; background: #fff0f2; }
.cleanup-eyebrow { margin: 0 0 5px; color: #9a6b73; font-size: 10px; font-weight: 800; letter-spacing: .13em; }
.cleanup-dialog h2 { margin: 0; color: #172b4e; font-size: 19px; font-weight: 750; letter-spacing: -.02em; }
.cleanup-copy { margin: 8px 0 16px; color: #6f819c; font-size: 13px; line-height: 1.55; }
.keep-options { display: flex; flex-direction: column; gap: 8px; }
.keep-option { display: flex; align-items: center; gap: 12px; border: 1px solid #e1e8f2; border-radius: 12px; padding: 12px 14px; background: #fff; text-align: left; cursor: pointer; transition: border-color .15s ease, background .15s ease; }
.keep-option:hover:not(:disabled) { border-color: #bcd3f5; }
.keep-option.is-selected { border-color: #1769e8; background: #f3f8ff; }
.keep-radio { position: relative; flex: 0 0 auto; width: 18px; height: 18px; border: 2px solid #b8c6d9; border-radius: 50%; }
.keep-option.is-selected .keep-radio { border-color: #1769e8; }
.keep-option.is-selected .keep-radio::after { content: ''; position: absolute; inset: 3px; border-radius: 50%; background: #1769e8; }
.keep-copy { display: flex; flex-direction: column; gap: 3px; }
.keep-copy strong { color: #1d365c; font-size: 14px; font-weight: 700; }
.keep-copy small { color: #7b8ca5; font-size: 12px; }
.cleanup-preview { min-height: 20px; margin: 14px 0 0; border-radius: 10px; padding: 10px 12px; color: #5f7391; background: #f5f8fc; font-size: 13px; }
.cleanup-preview.has-events { color: #a8323f; background: #fff3f5; }
.cleanup-error { margin: 10px 0 0; color: #c33d50; font-size: 13px; }
.cleanup-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; padding-top: 16px; border-top: 1px solid #edf1f7; }
.cleanup-actions button { display: inline-flex; min-height: 44px; align-items: center; justify-content: center; gap: 7px; border-radius: 10px; padding: 0 16px; font-size: 14px; font-weight: 700; cursor: pointer; }
.cancel-button { border: 1px solid #dfe6ef; color: #536984; background: #fff; }
.confirm-button { border: 0; color: #fff; background: #dc3f55; }
.confirm-button:hover:not(:disabled) { background: #c93449; }
.cleanup-actions button:disabled { opacity: .5; cursor: not-allowed; }

@media (max-width: 700px) {
  .page-heading { align-items: flex-start; margin-bottom: 16px; }
  .page-heading h1 { font-size: 24px; }
  .cleanup-button { min-height: 40px; padding: 0 13px; }
  .audit-row { gap: 11px; padding: 13px 12px; }
  .audit-icon { width: 34px; height: 34px; border-radius: 10px; }
  .audit-meta span { max-width: 200px; }
  .cleanup-backdrop { align-items: flex-end; padding: 0; }
  .cleanup-dialog { width: 100%; max-height: 92dvh; border: 0; border-radius: 22px 22px 0 0; padding: 22px 18px calc(16px + env(safe-area-inset-bottom)); animation: cleanup-sheet-up .22s ease-out; }
  .sheet-handle { position: absolute; top: 8px; left: 50%; display: block; width: 38px; height: 4px; border-radius: 999px; background: #d6dfeb; transform: translateX(-50%); }
  .cleanup-actions { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr); }
}
@keyframes cleanup-sheet-up { from { transform: translateY(100%); } }
</style>
