<template>
  <section class="rp-compact space-y-4">
    <!-- The entries: when each was made and its title -->
    <template v-if="!form">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">RP</p>
          <h1 class="text-xl font-bold text-navy">Data entry</h1>
        </div>
        <div class="flex gap-2">
          <NuxtLink v-if="can('winlose-reports.view')" to="/winlose/reports/weekly" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy hover:bg-canvas"><FileSpreadsheet :size="16" /> Report</NuxtLink>
          <button v-if="canCreate" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openNew"><Plus :size="16" /> New entry</button>
        </div>
      </header>

      <div class="grid grid-cols-1 gap-2" :class="rp.show.value ? 'md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1fr)]' : 'md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]'">
        <ListSearch v-model="search" placeholder="Search by title…" />
        <!-- Only for people who work in more than one branch (chosen once for all the RP pages) -->
        <SearchSelect v-if="rp.show.value" v-model="rp.branchId.value" :options="rp.optionsFor(true)" :searchable="false" aria-label="Branch" />
        <!-- The days the entries were created (empty: any day) -->
        <WinloseDateFilter v-model="created" allow-all label="Created" />
      </div>

      <div v-if="listLoading && !entries.length" class="text-sm text-muted">Loading entries…</div>
      <AppAlert v-else-if="listError && !entries.length" title="Couldn’t load entries" :message="listError"><template #action><button type="button" @click="resetEntries">Try again</button></template></AppAlert>
      <div v-else class="overflow-hidden rounded-xl border border-border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-2 py-2 font-bold sm:w-52 sm:px-3">Created at</th>
              <th class="px-2 py-2 font-bold sm:px-3">Title</th>
              <th class="px-2 py-2 sm:px-3"><span class="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="e in entries" :key="e.id" class="text-navy">
              <td class="px-2 py-2.5 text-[10px] text-muted sm:whitespace-nowrap sm:px-3 sm:text-xs"><span class="block sm:inline">{{ formatDocDate(e.created_at) }}</span><template v-if="e.created_by"><span class="hidden sm:inline"> · </span><span class="block whitespace-nowrap font-medium text-navy sm:inline">{{ e.created_by }}</span></template></td>
              <td class="px-2 py-2.5 sm:px-3">
                <span class="block font-semibold">{{ e.title }}</span>
                <span v-if="rp.show.value && e.branch_code" class="block text-[11px] text-muted">{{ e.branch_name || e.branch_code }}</span>
                <!-- ✅ / ❌: hovering (or tapping) one shows who -->
                <div class="mt-1"><WinloseEntryReactions :url="`/winlose/entries/${e.id}/reactions`" :reactions="e.reactions || []" /></div>
              </td>
              <td class="px-2 py-2 sm:px-3">
                <!-- On phones the buttons are icons only. -->
                <div class="flex justify-end gap-1 sm:flex-wrap sm:gap-1.5">
                  <NuxtLink :to="`/winlose/entries/${e.id}`" class="action-chip" title="View"><Eye :size="13" /> <span class="max-sm:sr-only">View</span></NuxtLink>
                  <button v-if="canCreate || canEdit" type="button" class="action-chip action-edit" title="Update" @click="openEdit(e)"><Pencil :size="13" /> <span class="max-sm:sr-only">Update</span></button>
                  <button v-if="canDelete" type="button" class="action-chip action-danger" title="Delete" @click="deleting = e"><Trash2 :size="13" /> <span class="max-sm:sr-only">Delete</span></button>
                </div>
              </td>
            </tr>
            <tr v-if="!entries.length"><td colspan="3" class="px-3 py-8 text-center text-sm text-muted">{{ search || created.from ? 'No entries match these filters.' : 'No entries yet.' }}</td></tr>
          </tbody>
        </table>
      </div>
      <InfiniteLoader :has-more="hasMore" :loading="loadingMore" :count="entries.length" @more="loadMore" />
    </template>

    <!-- New entry / edit: the week, a title and each agent's amounts -->
    <template v-else>
      <header class="space-y-2">
        <!-- Back to the entries (asks first when something isn't saved) -->
        <button type="button" class="inline-flex items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="closeForm"><ChevronLeft :size="14" /> Back</button>
        <h1 class="text-xl font-bold text-navy">{{ form.mode === 'new' && !sheet?.entry ? 'New entry' : canSave ? 'Update entry' : 'Entry' }}</h1>
      </header>

      <!-- A new entry is for this week; one being updated keeps its own. The title saves on its own
           too (each agent's Save also saves it). -->
      <!-- Only with several branches: a new entry picks its branch (its agents change with it); a saved one keeps its own -->
      <div v-if="rp.show.value" class="space-y-1">
        <span class="block text-xs font-medium text-muted">Branch</span>
        <SearchSelect v-if="form.mode === 'new' && !sheet?.entry" v-model="form.branch_id" :options="rp.optionsFor(false)" :searchable="false" aria-label="Entry branch" />
        <p v-else class="h-10 rounded-lg border border-border bg-canvas px-3 py-2.5 text-sm text-muted">{{ sheet?.branch_name || sheet?.branch_code }}</p>
      </div>
      <div class="space-y-1">
        <label for="entry-title" class="block text-xs font-medium text-muted">Title</label>
        <div class="flex items-center gap-2">
          <input id="entry-title" v-model="title" maxlength="128" :disabled="!canSave" :aria-invalid="titleMissing ? 'true' : undefined" class="h-10 min-w-0 flex-1 rounded-lg border border-border px-3 text-sm text-navy disabled:bg-canvas disabled:text-muted" placeholder="e.g. Week (01-06)" />
          <button v-if="canSave && titleChanged && title.trim()" type="button" class="action-chip action-success" :disabled="saving" @click="trySave(null)"><Save :size="13" /> {{ saving && !only ? 'Saving…' : 'Save' }}</button>
        </div>
      </div>

      <div v-if="loading" class="text-sm text-muted">Loading the week…</div>
      <AppAlert v-else-if="error" title="Couldn’t load the week" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
      <template v-else-if="sheet">
        <!-- Finds an agent by user or name, and puts each currency's agents A to Z or Z to A; what was
             typed in the others stays -->
        <div v-if="sheet.rows.length" class="flex flex-wrap items-center gap-2">
          <div class="min-w-0 flex-1 sm:max-w-sm"><ListSearch v-model="agentSearch" :delay="150" placeholder="Search agent…" /></div>
          <div class="flex items-center gap-0.5 rounded-lg bg-canvas p-1" role="radiogroup" aria-label="Sort agents">
            <span class="px-1.5 text-[11px] font-semibold text-muted">Sort</span>
            <button v-for="o in SORTS" :key="o.value" type="button" role="radio" :aria-checked="sort === o.value" class="rounded-md px-2.5 py-1 text-xs font-semibold" :class="sort === o.value ? 'bg-white text-primary shadow-sm' : 'text-muted hover:text-navy'" @click="sort = o.value">{{ o.label }}</button>
          </div>
        </div>
        <!-- Only the agents in one currency, or whose code starts with SBK_, SBU_, 12K_ … (All: every one) -->
        <div v-if="currencies.length || prefixes.length" class="flex flex-wrap items-center gap-x-3 gap-y-1.5">
          <div v-if="currencies.length" class="flex flex-wrap items-center gap-1.5" role="radiogroup" aria-label="Currency">
            <button v-for="c in [{ value: '', label: 'All', count: sheet.rows.length }, ...currencies]" :key="c.value || 'all'" type="button" role="radio" :aria-checked="currency === c.value" class="inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-semibold" :class="currency === c.value ? 'border-primary bg-blue-50 text-primary' : 'border-border bg-white text-muted hover:text-navy'" @click="chosenCurrency = c.value">
              <span>{{ c.label }}</span><span class="font-normal opacity-70">{{ c.count }}</span>
            </button>
          </div>
          <span v-if="currencies.length && prefixes.length" class="h-5 w-px bg-border" aria-hidden="true" />
          <div v-if="prefixes.length" class="flex flex-wrap items-center gap-1.5" role="radiogroup" aria-label="Agents">
            <button v-for="p in [{ value: '', label: 'All', count: sheet.rows.length }, ...prefixes]" :key="p.value || 'all'" type="button" role="radio" :aria-checked="prefix === p.value" class="inline-flex items-center gap-1 rounded-full border px-3 py-1 text-xs font-semibold" :class="prefix === p.value ? 'border-primary bg-blue-50 text-primary' : 'border-border bg-white text-muted hover:text-navy'" @click="chosenPrefix = p.value">
              <span :class="p.value ? 'font-mono' : ''">{{ p.label }}</span><span class="font-normal opacity-70">{{ p.count }}</span>
            </button>
          </div>
        </div>
        <!-- Like the Report: one table per currency, USD first then KHR, each numbered from 1 (one
             table with the message when no agent is shown) -->
        <div v-for="part in agentTables.length ? agentTables : [{ currency: '', rows: [] }]" :key="part.currency || 'none'" class="overflow-hidden rounded-xl border border-border bg-white">
          <h2 v-if="part.currency" class="border-b border-border px-4 py-2.5 text-sm font-semibold text-navy">{{ part.currency }}</h2>
          <table class="wl-table w-full text-left text-sm">
            <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
              <tr>
                <th class="w-12 px-3 py-2 text-center font-bold">No</th>
                <th class="px-3 py-2 font-bold">Agent</th>
                <th class="w-56 px-2 py-2 font-bold">W/L</th>
                <th class="w-56 px-2 py-2 font-bold">COMM</th>
                <th v-if="canSave" class="w-24 px-2 py-2"><span class="sr-only">Save</span></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <tr v-for="{ row, no } in part.rows" :key="row.platform_id" :class="changed(row) ? 'bg-amber-50/40' : ''">
                <td class="row-no px-3 py-2.5 text-center text-sm tabular-nums text-muted">{{ no }}</td>
                <td class="px-3 py-2.5">
                  <!-- The currency the amounts are entered in (on phones the row's number comes first) -->
                  <p class="font-mono text-sm font-semibold text-navy"><span class="row-no-inline font-sans text-muted">{{ no }}. </span>{{ row.platform_code }} <span class="font-sans text-xs font-normal text-muted">· {{ row.currency }}</span></p>
                  <p v-if="row.platform_name && row.platform_name !== row.platform_code" class="text-[11px] text-muted">{{ row.platform_name }}</p>
                  <p class="text-[11px]" :class="changed(row) ? 'font-semibold text-amber-700' : 'text-muted'">{{ statusOf(row) }}</p>
                  <!-- ✅ / ❌ on this agent in the entry, once saved -->
                  <div v-if="row.result_id" class="mt-1"><WinloseEntryReactions :url="`/winlose/results/${row.result_id}/reactions`" :reactions="row.reactions || []" /></div>
                </td>
                <td class="px-2 py-2" data-label="W/L">
                  <MoneyInput v-model="edits[row.platform_id]!.win_lose" :currency="row.currency" signed class="w-full rounded-lg border border-border px-2.5 py-1.5 text-right text-sm tabular-nums disabled:bg-canvas disabled:text-muted" :class="Number(edits[row.platform_id]!.win_lose) < 0 ? 'text-expense' : 'text-navy'" :aria-invalid="invalid(edits[row.platform_id]!.win_lose) ? 'true' : undefined" :disabled="!editable(row)" :aria-label="`Win/lose for ${row.platform_code}`" />
                </td>
                <td class="px-2 py-2" data-label="COMM">
                  <MoneyInput v-model="edits[row.platform_id]!.commission" :currency="row.currency" signed class="w-full rounded-lg border border-border px-2.5 py-1.5 text-right text-sm tabular-nums disabled:bg-canvas disabled:text-muted" :class="Number(edits[row.platform_id]!.commission) < 0 ? 'text-expense' : 'text-navy'" :aria-invalid="invalid(edits[row.platform_id]!.commission) ? 'true' : undefined" :disabled="!editable(row)" :aria-label="`Commission for ${row.platform_code}`" />
                </td>
                <!-- Saves just this agent: the others keep what was typed -->
                <td v-if="canSave" class="row-save px-2 py-2 text-right">
                  <button v-if="editable(row)" type="button" class="action-chip action-success" :disabled="!changed(row) || rowInvalid(row) || saving" :aria-label="`Save ${row.platform_code}`" @click="trySave(row)"><Save :size="13" /> {{ saving && only?.platform_id === row.platform_id ? 'Saving…' : 'Save' }}</button>
                </td>
              </tr>
              <tr v-if="!sheet.rows.length" class="empty-row"><td :colspan="canSave ? 5 : 4" class="px-3 py-8 text-center text-sm text-muted">No agents are turned on.<NuxtLink v-if="can('platforms.create')" to="/admin/platforms" class="ml-1 font-semibold text-primary">Agents</NuxtLink></td></tr>
              <tr v-else-if="!part.currency" class="empty-row"><td :colspan="canSave ? 5 : 4" class="px-3 py-8 text-center text-sm text-muted">{{ noMatch }}</td></tr>
            </tbody>
          </table>
        </div>
        <p v-if="hasInvalid" class="text-xs font-medium text-expense" role="alert">Enter amounts as numbers, e.g. -1500 or 250.75.</p>

        <AppAlert v-if="saveError" title="Couldn’t save the entry" :message="saveError" dismissible @dismiss="saveError = ''" />
        <div class="flex flex-wrap items-center justify-end gap-2">
          <span v-if="dirty" class="mr-auto text-xs text-amber-700">Unsaved changes</span>
          <button type="button" class="rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-muted disabled:opacity-50" :disabled="saving" @click="closeForm">{{ canSave ? 'Cancel' : 'Back' }}</button>
        </div>
      </template>
    </template>

    <!-- Deleting removes the entry's amounts too (kept in the audit log with the reason) -->
    <InventoryReasonDialog :model-value="!!deleting" :title="deleting ? `Delete “${deleting.title}”?` : ''" eyebrow="DELETE ENTRY" subtitle="Its amounts are removed too; the audit log keeps them with your reason." label="Reason for deleting" confirm-label="Delete entry" danger :busy="deleteBusy" :server-error="deleteError" @update:model-value="(open) => { if (!open) { deleting = null; deleteError = '' } }" @confirm="deleteEntry" />

    <InventoryReasonDialog v-model="reasonOpen" title="Change saved results?" eyebrow="CORRECTION" subtitle="The old and new figures and your reason go into the audit log." label="Reason for the change" confirm-label="Save changes" :busy="saving" :server-error="saveError" @confirm="(reason) => save(reason)" />
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, Eye, FileSpreadsheet, Pencil, Plus, Save, Trash2 } from 'lucide-vue-next'
import type { WeekSheet, WeekSheetRow, WinLoseEntry } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useInfiniteList } from '~/composables/useInfiniteList'
import { useRpBranch } from '~/composables/useRpBranch'
import { currentMonthRange } from '~/utils/dateRanges'
import { formatDocDate, todayISO } from '~/utils/inventory'
import { byAgentCode, byCurrency, currencyTables, mondayOf } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canCreate = computed(() => can('winlose.create'))
const canEdit = computed(() => can('winlose.edit'))
const canDelete = computed(() => can('winlose.delete'))
const today = todayISO()

// --- the list ---
// The branch the RP pages work in (only for people with several).
const rp = useRpBranch()
const search = ref('')
const created = ref(currentMonthRange()) // this month; All dates in the filter
// Entries load a page at a time as the list is scrolled; a save or delete fetches the pages
// already shown again (loadEntries).
const { items: entries, loading: listLoading, loadingMore, error: listError, hasMore, reset: resetEntries, loadMore, refresh: loadEntries } = useInfiniteList<WinLoseEntry>(
  async (page, size) => await api<{ items: WinLoseEntry[]; total: number }>('/winlose/entries', {
    query: { search: search.value.trim() || undefined, created_from: created.value.from || undefined, created_to: created.value.to || undefined, branch_id: rp.branchId.value || undefined, page, page_size: size },
  }),
  entry => entry.id, 30, 'Could not load entries.',
)
// Another search or dates start from the top again.
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(resetEntries, 300)
})
watch(created, resetEntries, { deep: true })
watch(() => rp.branchId.value, resetEntries)
onMounted(resetEntries)

// --- delete (Correct Win/Lose, with a reason) ---
const deleting = ref<WinLoseEntry | null>(null)
const deleteBusy = ref(false)
const deleteError = ref('')
async function deleteEntry(reason: string) {
  const e = deleting.value
  if (!e) return
  deleteBusy.value = true
  deleteError.value = ''
  try {
    await api(`/winlose/entries/${e.id}`, { method: 'DELETE', body: { reason } })
    deleting.value = null
    toast.success('Its amounts are removed too.', { title: `“${e.title}” deleted` })
    await loadEntries()
  } catch (err: any) {
    deleteError.value = err?.data?.error || 'Could not delete the entry.'
  } finally {
    deleteBusy.value = false
  }
}

// --- the form: a new entry is for this week, in the person's own group (the API works it out) ---
// A new entry has no id until its first save; the saves after change only that entry.
const form = ref<{ mode: 'new' | 'edit'; week_start: string; entry_id: string; branch_id: string } | null>(null)
const sheet = ref<WeekSheet | null>(null)
const loading = ref(false)
const error = ref('')
const title = ref('')
const edits = reactive<Record<string, { win_lose: string; commission: string }>>({})
// The agents shown, like the Report: a table per currency (USD first, then KHR), its agents in the
// order chosen and numbered in it; then only those of the currency and the code start chosen (All:
// every one) matching the search by user or name, each keeping its number.
const agentSearch = ref('')
const SORTS = [
  { value: 'name', label: 'A to Z' },
  { value: 'desc', label: 'Z to A' },
] as const
const sort = ref<(typeof SORTS)[number]['value']>('name')
// The currencies the entry's agents are in (USD first), each with how many: buttons only with two.
const currencies = computed(() => {
  const counts = new Map<string, number>()
  for (const row of sheet.value?.rows || []) counts.set(row.currency, (counts.get(row.currency) || 0) + 1)
  if (counts.size < 2) return []
  return [...counts].sort(([a], [b]) => byCurrency(a, b)).map(([value, count]) => ({ value, label: value, count }))
})
const chosenCurrency = ref('')
const currency = computed(() => (currencies.value.some(c => c.value === chosenCurrency.value) ? chosenCurrency.value : ''))
// A code's start: up to its first "_" (SBK_, SBU_, 12K_ …), in capitals; '' when it has none.
const prefixOf = (code: string) => {
  const at = code.indexOf('_')
  return at > 0 ? code.slice(0, at + 1).toUpperCase() : ''
}
// The starts the entry's agents have, A to Z, each with how many: buttons only when they split the
// agents (not when every agent has the same one).
const prefixes = computed(() => {
  const rows = sheet.value?.rows || []
  const counts = new Map<string, number>()
  for (const row of rows) {
    const p = prefixOf(row.platform_code)
    if (p) counts.set(p, (counts.get(p) || 0) + 1)
  }
  if (counts.size === 1 && [...counts.values()][0] === rows.length) return []
  return [...counts].sort(([a], [b]) => byAgentCode(a, b)).map(([value, count]) => ({ value, label: value, count }))
})
const chosenPrefix = ref('')
// What was chosen, while the entry's agents still have it.
const prefix = computed(() => (prefixes.value.some(p => p.value === chosenPrefix.value) ? chosenPrefix.value : ''))
const agentTables = computed(() => {
  const list = [...(sheet.value?.rows || [])].sort((a, b) => byAgentCode(a.platform_code, b.platform_code))
  if (sort.value === 'desc') list.reverse()
  const q = agentSearch.value.trim().toLowerCase()
  const shown = (row: WeekSheetRow) => (!currency.value || row.currency === currency.value)
    && (!prefix.value || prefixOf(row.platform_code) === prefix.value)
    && (!q || row.platform_code.toLowerCase().includes(q) || (row.platform_name || '').toLowerCase().includes(q))
  return currencyTables(list, row => row.currency)
    .map(t => ({ currency: t.currency, rows: t.rows.map((row, i) => ({ row, no: i + 1 })).filter(({ row }) => shown(row)) }))
    .filter(t => t.rows.length)
})
const noMatch = computed(() => {
  const q = agentSearch.value.trim()
  const within = [currency.value, prefix.value].filter(Boolean).join(' · ')
  return `No agent matches${q ? ` “${q}”` : ''}${within ? ` in ${within}` : ''}.`
})
// Each New entry or Update starts on every agent, A to Z.
function resetAgentFilters() {
  agentSearch.value = ''
  chosenPrefix.value = ''
  chosenCurrency.value = ''
  sort.value = 'name'
}

function openNew() {
  // A new entry is in the branch chosen (else the person's own; with one branch the API picks it).
  resetAgentFilters()
  form.value = { mode: 'new', week_start: mondayOf(today), entry_id: '', branch_id: rp.effective.value }
  load()
}
// Another branch for a new entry has other agents: the sheet is loaded again.
watch(() => form.value?.branch_id, () => { if (form.value?.mode === 'new' && !form.value.entry_id) load() })
function openEdit(e: WinLoseEntry) {
  resetAgentFilters()
  form.value = { mode: 'edit', week_start: e.week_start, entry_id: e.id, branch_id: e.branch_id }
  load()
}
function closeForm() {
  if (dirty.value && !window.confirm('Leave this entry without saving your changes?')) return
  form.value = null
  sheet.value = null
  saveError.value = ''
}

// The inputs only let numbers through (with the currency's decimals); "1500." while typing is fine.
const AMOUNT = /^-?\d+(\.\d{0,4})?$/
function invalid(value: string) {
  return !!value && !AMOUNT.test(value)
}
const asText = (value: string | null) => value === null ? '' : String(Number(value))
// Amounts compare as numbers: 1500 and 1500.00 are the same, and an empty box is 0.
const same = (typed: string, saved: string | null) => Number(typed || 0) === Number(saved || 0)

function resetEdits() {
  for (const key of Object.keys(edits)) delete edits[key]
  for (const row of sheet.value?.rows || []) edits[row.platform_id] = { win_lose: asText(row.win_lose), commission: asText(row.commission) }
  title.value = sheet.value?.entry?.title || ''
}

let loadRequest = 0
async function load() {
  if (!form.value) return
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  saveError.value = ''
  try {
    const result = await api<WeekSheet>('/winlose/entry', { query: form.value.entry_id ? { entry_id: form.value.entry_id } : { week_start: form.value.week_start, branch_id: form.value.branch_id || undefined } })
    if (request !== loadRequest) return
    sheet.value = result
    resetEdits()
  } catch (e: any) {
    if (request === loadRequest) { sheet.value = null; error.value = e?.data?.error || 'Could not load the week.' }
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
onBeforeRouteLeave(() => !dirty.value || window.confirm('Leave without saving your changes?'))

function changed(row: WeekSheetRow) {
  const e = edits[row.platform_id]
  if (!e) return false
  return row.saved ? !same(e.win_lose, row.win_lose) || !same(e.commission, row.commission) : !!(e.win_lose || e.commission)
}
const titleChanged = computed(() => !!sheet.value && title.value.trim() !== (sheet.value.entry?.title || ''))
const titleMissing = computed(() => !!saveError.value && !title.value.trim())
const dirty = computed(() => !!sheet.value && (titleChanged.value || sheet.value.rows.some(changed)))
const hasInvalid = computed(() => Object.values(edits).some(e => invalid(e.win_lose) || invalid(e.commission)))
// Saving needs Enter Win/Lose for a new entry or new rows, Correct Win/Lose for saved ones.
const canSave = computed(() => !!sheet.value && (canCreate.value || canEdit.value))

function editable(row: WeekSheetRow) {
  if (!canSave.value) return false
  return row.saved ? canEdit.value : canCreate.value
}

function statusOf(row: WeekSheetRow) {
  if (changed(row)) return row.saved ? 'Changed — not saved' : 'New — not saved'
  if (row.saved) return `Saved${row.updated_by ? ` by ${row.updated_by}` : ''}`
  return 'Not entered'
}

function rowInvalid(row: WeekSheetRow) {
  const e = edits[row.platform_id]
  return !!e && (invalid(e.win_lose) || invalid(e.commission))
}

// A saved agent's empty box is saved as 0 (an unsaved agent left empty isn't sent).
function rowBody(row: WeekSheetRow) {
  const clean = (value: string) => value.replace(/\.$/, '') || (row.saved ? '0' : '')
  const e = edits[row.platform_id]!
  return { platform_id: row.platform_id, win_lose: clean(e.win_lose), commission: clean(e.commission) }
}

// --- saving: one agent at a time (with the title), or the title on its own ---
const saving = ref(false)
const saveError = ref('')
const reasonOpen = ref(false)
// The agent being saved (null: the title alone).
const only = ref<WeekSheetRow | null>(null)
const NEED_TITLE = 'Give the entry a title.'
watch(title, (value) => { if (value.trim() && saveError.value === NEED_TITLE) saveError.value = '' })

function trySave(row: WeekSheetRow | null) {
  saveError.value = ''
  if (row && rowInvalid(row)) return
  if (!title.value.trim()) { saveError.value = NEED_TITLE; return }
  only.value = row
  // Changing a saved row needs a reason (audit log).
  if (row?.saved && changed(row)) {
    reasonOpen.value = true
    return
  }
  save('')
}

async function save(reason: string) {
  if (!sheet.value || !form.value) return
  const row = only.value
  saving.value = true
  saveError.value = ''
  try {
    const saved = await api<WeekSheet>('/winlose/entry', {
      method: 'PUT',
      body: { entry_id: form.value.entry_id || undefined, branch_id: form.value.entry_id ? undefined : form.value.branch_id || undefined, week_start: form.value.week_start, title: title.value.trim(), rows: row ? [rowBody(row)] : [], reason },
    })
    reasonOpen.value = false
    // Stay on the entry: the agent saved shows what was saved, the others keep what was typed.
    sheet.value = saved
    form.value = { mode: 'edit', week_start: saved.week_start, entry_id: saved.entry?.id || form.value.entry_id, branch_id: saved.branch_id }
    for (const r of saved.rows) {
      if (r.platform_id === row?.platform_id || !edits[r.platform_id]) edits[r.platform_id] = { win_lose: asText(r.win_lose), commission: asText(r.commission) }
    }
    toast.success(`Week of ${formatDocDate(saved.week_start)}`, { title: row ? `${row.platform_code} saved` : `“${saved.entry?.title || title.value.trim()}” saved` })
    loadEntries()
  } catch (e: any) {
    saveError.value = e?.data?.error || 'Could not save the entry.'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
/* The row's number shows in its own column, or before the agent on phones. */
.row-no-inline { display: none; }

/* Phones: each agent becomes a card: its number and name, then the two amounts side by side. */
@media (max-width: 639px) {
  .row-no-inline { display: inline; }
  .wl-table thead { display: none; }
  .wl-table, .wl-table tbody, .wl-table tr, .wl-table td { display: block; width: 100%; }
  .wl-table tbody tr:not(.empty-row) { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 10px; padding: 12px; }
  .wl-table tbody tr:not(.empty-row) td { padding: 0; text-align: left; }
  .wl-table tbody tr:not(.empty-row) td.row-no { display: none; }
  .wl-table tbody tr:not(.empty-row) td:nth-child(2) { grid-column: 1 / -1; }
  .wl-table tbody tr:not(.empty-row) td.row-save { grid-column: 1 / -1; text-align: right; }
  .wl-table tbody tr:not(.empty-row) td.row-save:empty { display: none; }
  .wl-table td[data-label]::before { content: attr(data-label); display: block; margin-bottom: 3px; color: #64748b; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; }
}
</style>
