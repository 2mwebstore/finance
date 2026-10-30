<template>
  <section class="mx-auto max-w-5xl space-y-4">
    <!-- The entries: when each was made and its title -->
    <template v-if="!form">
      <header class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS</p>
          <h1 class="text-xl font-bold text-navy">Data entry</h1>
          <p class="mt-1 text-xs text-muted">Each entry is one week (Monday to Sunday) of every agent’s win/lose and commission.</p>
        </div>
        <div class="flex gap-2">
          <NuxtLink v-if="can('winlose-reports.view')" to="/winlose/reports/weekly" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy hover:bg-canvas"><FileSpreadsheet :size="16" /> Weekly report</NuxtLink>
          <button v-if="canCreate" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openNew"><Plus :size="16" /> New entry</button>
        </div>
      </header>

      <div class="grid grid-cols-1 gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
        <ListSearch v-model="search" placeholder="Search by title…" />
        <!-- The days the entries were created (empty: any day) -->
        <WinloseDateFilter v-model="created" allow-all label="Created" />
      </div>

      <div v-if="listLoading && !entries.length" class="text-sm text-muted">Loading entries…</div>
      <AppAlert v-else-if="listError" title="Couldn’t load entries" :message="listError"><template #action><button type="button" @click="loadEntries">Try again</button></template></AppAlert>
      <div v-else class="overflow-hidden rounded-xl border border-border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="w-52 px-3 py-2 font-bold">Created at</th>
              <th class="px-3 py-2 font-bold">Title</th>
              <th class="px-3 py-2"><span class="sr-only">Actions</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="e in entries" :key="e.id" class="text-navy">
              <td class="whitespace-nowrap px-3 py-2.5 text-xs text-muted">{{ formatDocDate(e.created_at) }}<template v-if="e.created_by"> · <span class="font-medium text-navy">{{ e.created_by }}</span></template></td>
              <td class="px-3 py-2.5 font-semibold">{{ e.title }}</td>
              <td class="px-3 py-2">
                <!-- A finalized week can only be viewed -->
                <div class="flex justify-end gap-1.5">
                  <button type="button" class="action-chip" @click="openView(e)"><Eye :size="13" /> View</button>
                  <button v-if="!e.locked && (canCreate || canEdit)" type="button" class="action-chip action-edit" @click="openEdit(e)"><Pencil :size="13" /> Update</button>
                  <button v-if="!e.locked && canEdit" type="button" class="action-chip action-danger" @click="deleting = e"><Trash2 :size="13" /> Delete</button>
                </div>
              </td>
            </tr>
            <tr v-if="!entries.length"><td colspan="3" class="px-3 py-8 text-center text-sm text-muted">{{ search || created.from ? 'No entries match these filters.' : 'No entries yet.' }}</td></tr>
          </tbody>
        </table>
      </div>
    </template>

    <!-- New entry / edit: the week, a title and each agent's amounts -->
    <template v-else>
      <header>
        <button type="button" class="inline-flex items-center gap-1 text-[11px] font-bold tracking-[0.14em] text-slate-400 hover:text-primary" @click="closeForm"><ChevronLeft :size="13" /> DATA ENTRY</button>
        <h1 class="text-xl font-bold text-navy">{{ form.mode === 'new' && !sheet?.entry ? 'New entry' : canSave ? 'Update entry' : 'Entry' }}</h1>
      </header>

      <!-- A new entry is for this week; one being updated keeps its own -->
      <label class="block space-y-1 text-xs font-medium text-muted">Title
        <input v-model="title" maxlength="128" :disabled="!canSave" :aria-invalid="titleMissing ? 'true' : undefined" class="h-10 w-full rounded-lg border border-border px-3 text-sm text-navy disabled:bg-canvas disabled:text-muted" placeholder="e.g. Week 44 Phnom Penh" />
      </label>

      <div v-if="loading" class="text-sm text-muted">Loading the week…</div>
      <AppAlert v-else-if="error" title="Couldn’t load the week" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
      <template v-else-if="sheet">
        <AppAlert v-if="sheet.locked" type="warning" title="This week is finalized" :message="`The week of ${formatDocDate(sheet.week_start)} – ${formatDocDate(sheet.week_end)} is settled. Reopen it on the weekly report to change it.`" />
        <p v-else-if="form.mode === 'new' && sheet.entry" class="rounded-lg bg-blue-50 px-3 py-2 text-xs text-navy">This week already has the entry “{{ sheet.entry.title }}”; saving updates it.</p>
        <div class="overflow-hidden rounded-xl border border-border bg-white">
          <table class="wl-table w-full text-left text-sm">
            <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
              <tr>
                <th class="px-3 py-2 font-bold">Agent</th>
                <th class="w-56 px-2 py-2 font-bold">Win/Lose</th>
                <th class="w-56 px-2 py-2 font-bold">Commission</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <tr v-for="row in sheet.rows" :key="row.platform_id" :class="changed(row) ? 'bg-amber-50/40' : ''">
                <td class="px-3 py-2.5">
                  <!-- The currency the amounts are entered in -->
                  <p class="font-mono text-sm font-semibold text-navy">{{ row.platform_code }} <span class="font-sans text-xs font-normal text-muted">· {{ row.currency }}</span></p>
                  <p class="text-[11px]" :class="changed(row) ? 'font-semibold text-amber-700' : 'text-muted'">{{ statusOf(row) }}</p>
                </td>
                <td class="px-2 py-2" data-label="Win/Lose">
                  <MoneyInput v-model="edits[row.platform_id]!.win_lose" :currency="row.currency" signed class="w-full rounded-lg border border-border px-2.5 py-1.5 text-right text-sm tabular-nums disabled:bg-canvas disabled:text-muted" :class="Number(edits[row.platform_id]!.win_lose) < 0 ? 'text-expense' : 'text-navy'" :aria-invalid="invalid(edits[row.platform_id]!.win_lose) ? 'true' : undefined" :disabled="!editable(row)" :aria-label="`Win/lose for ${row.platform_code}`" />
                </td>
                <td class="px-2 py-2" data-label="Commission">
                  <MoneyInput v-model="edits[row.platform_id]!.commission" :currency="row.currency" signed class="w-full rounded-lg border border-border px-2.5 py-1.5 text-right text-sm tabular-nums disabled:bg-canvas disabled:text-muted" :class="Number(edits[row.platform_id]!.commission) < 0 ? 'text-expense' : 'text-navy'" :aria-invalid="invalid(edits[row.platform_id]!.commission) ? 'true' : undefined" :disabled="!editable(row)" :aria-label="`Commission for ${row.platform_code}`" />
                </td>
              </tr>
              <tr v-if="!sheet.rows.length" class="empty-row"><td colspan="3" class="px-3 py-8 text-center text-sm text-muted">No agents are assigned to your group in this week.<NuxtLink v-if="can('platform-assignments.manage')" to="/admin/platform-assignments" class="ml-1 font-semibold text-primary">Assign agents</NuxtLink></td></tr>
            </tbody>
          </table>
        </div>
        <p v-if="hasInvalid" class="text-xs font-medium text-expense" role="alert">Enter amounts as numbers, e.g. -1500 or 250.75.</p>

        <AppAlert v-if="saveError" title="Couldn’t save the entry" :message="saveError" dismissible @dismiss="saveError = ''" />
        <div class="flex flex-wrap items-center justify-end gap-2">
          <span v-if="dirty" class="mr-auto text-xs text-amber-700">Unsaved changes</span>
          <button type="button" class="rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-muted disabled:opacity-50" :disabled="saving" @click="closeForm">{{ canSave ? 'Cancel' : 'Back' }}</button>
          <button v-if="canSave" type="button" class="rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-white disabled:opacity-50" :disabled="!dirty || saving || hasInvalid" @click="trySave">{{ saving ? 'Saving…' : 'Save' }}</button>
        </div>
      </template>
    </template>

    <!-- An entry's details: read-only -->
    <AppModal :model-value="!!viewing" :title="viewing?.title || ''" eyebrow="ENTRY" :subtitle="viewing ? `Week of ${formatDocDate(viewing.week_start)} – ${formatDocDate(viewing.week_end)}` : ''" width="600px" @update:model-value="(open) => { if (!open) viewing = null }">
      <template #icon><NotebookPen :size="20" /></template>
      <dl v-if="viewing" class="grid grid-cols-2 gap-3 text-xs">
        <div><dt class="text-muted">Created at</dt><dd class="font-semibold text-navy">{{ formatDocDate(viewing.created_at) }}</dd></div>
        <div><dt class="text-muted">Created by</dt><dd class="font-semibold text-navy">{{ viewing.created_by || '—' }}</dd></div>
        <div v-if="viewing.locked" class="col-span-2"><span class="inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-bold uppercase text-emerald-700"><Lock :size="11" /> Finalized</span></div>
      </dl>
      <p v-if="viewLoading" class="mt-4 text-sm text-muted">Loading…</p>
      <AppAlert v-else-if="viewError" class="mt-4" compact :message="viewError" />
      <div v-else-if="viewSheet" class="mt-4 overflow-hidden rounded-lg border border-border">
        <table class="w-full text-left text-xs">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-3 py-2 font-bold">Agent</th>
              <th class="px-3 py-2 text-right font-bold">Win/Lose</th>
              <th class="px-3 py-2 text-right font-bold">Commission</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="row in viewRows" :key="row.platform_id" class="text-navy">
              <td class="px-3 py-2"><span class="font-mono font-semibold">{{ row.platform_code }}</span> <span class="text-muted">· {{ row.currency }}</span><span v-if="row.updated_by" class="block text-[11px] text-muted">Saved by {{ row.updated_by }}</span></td>
              <td class="px-3 py-2 text-right tabular-nums" :class="Number(row.win_lose) < 0 ? 'text-expense' : ''">{{ formatAmount(row.win_lose || 0, row.currency) }}</td>
              <td class="px-3 py-2 text-right tabular-nums" :class="Number(row.commission) < 0 ? 'text-expense' : ''">{{ formatAmount(row.commission || 0, row.currency) }}</td>
            </tr>
            <tr v-if="!viewRows.length"><td colspan="3" class="px-3 py-6 text-center text-muted">No amounts entered yet.</td></tr>
          </tbody>
        </table>
      </div>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" @click="viewing = null">Close</button>
        <button v-if="viewing && !viewing.locked && (canCreate || canEdit)" type="button" class="inline-flex items-center justify-center gap-1.5 bg-primary text-white" @click="updateViewed"><Pencil :size="14" /> Update</button>
      </template>
    </AppModal>

    <!-- Deleting removes the week's amounts too (kept in the audit log with the reason) -->
    <InventoryReasonDialog :model-value="!!deleting" :title="deleting ? `Delete “${deleting.title}”?` : ''" eyebrow="DELETE ENTRY" subtitle="Its week’s amounts are removed too; the audit log keeps them with your reason." label="Reason for deleting" confirm-label="Delete entry" danger :busy="deleteBusy" :server-error="deleteError" @update:model-value="(open) => { if (!open) { deleting = null; deleteError = '' } }" @confirm="deleteEntry" />

    <InventoryReasonDialog v-model="reasonOpen" title="Change saved results?" eyebrow="CORRECTION" subtitle="The old and new figures and your reason go into the audit log." label="Reason for the change" confirm-label="Save changes" :busy="saving" :server-error="saveError" @confirm="(reason) => save(reason)" />
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, Eye, FileSpreadsheet, Lock, NotebookPen, Pencil, Plus, Trash2 } from 'lucide-vue-next'
import type { WeekSheet, WeekSheetRow, WinLoseEntry } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { formatDocDate, todayISO } from '~/utils/inventory'
import { formatAmount, mondayOf } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canCreate = computed(() => can('winlose.create'))
const canEdit = computed(() => can('winlose.edit'))
const today = todayISO()

// --- the list ---
const entries = ref<WinLoseEntry[]>([])
const listLoading = ref(false)
const listError = ref('')
const search = ref('')
const created = ref({ from: '', to: '' })
let listRequest = 0
async function loadEntries() {
  const request = ++listRequest
  listLoading.value = true
  listError.value = ''
  try {
    const result = await api<{ items: WinLoseEntry[] }>('/winlose/entries', {
      query: { search: search.value.trim() || undefined, created_from: created.value.from || undefined, created_to: created.value.to || undefined },
    })
    if (request === listRequest) entries.value = result.items || []
  } catch (e: any) {
    if (request === listRequest) listError.value = e?.data?.error || 'Could not load entries.'
  } finally {
    if (request === listRequest) listLoading.value = false
  }
}
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(loadEntries, 300)
})
watch(created, loadEntries, { deep: true })
onMounted(loadEntries)

// --- an entry's details (View) ---
const viewing = ref<WinLoseEntry | null>(null)
const viewSheet = ref<WeekSheet | null>(null)
const viewLoading = ref(false)
const viewError = ref('')
// Only the agents with amounts saved.
const viewRows = computed(() => (viewSheet.value?.rows || []).filter(row => row.saved))
async function openView(e: WinLoseEntry) {
  viewing.value = e
  viewSheet.value = null
  viewError.value = ''
  viewLoading.value = true
  try {
    const result = await api<WeekSheet>('/winlose/entry', { query: { week_start: e.week_start, branch_id: e.branch_id } })
    if (viewing.value?.id === e.id) viewSheet.value = result
  } catch (err: any) {
    if (viewing.value?.id === e.id) viewError.value = err?.data?.error || 'Could not load the entry.'
  } finally {
    if (viewing.value?.id === e.id) viewLoading.value = false
  }
}
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
    toast.success('Its week’s amounts are removed too.', { title: `“${e.title}” deleted` })
    await loadEntries()
  } catch (err: any) {
    deleteError.value = err?.data?.error || 'Could not delete the entry.'
  } finally {
    deleteBusy.value = false
  }
}

function updateViewed() {
  const e = viewing.value
  viewing.value = null
  if (e) openEdit(e)
}

// --- the form: a new entry goes in the person's own group (the API works it out) ---
// A new entry is for this week (the person's own group: the API works it out).
const form = ref<{ mode: 'new' | 'edit'; week_start: string; branch_id: string } | null>(null)
const sheet = ref<WeekSheet | null>(null)
const loading = ref(false)
const error = ref('')
const title = ref('')
const edits = reactive<Record<string, { win_lose: string; commission: string }>>({})

function openNew() {
  form.value = { mode: 'new', week_start: mondayOf(today), branch_id: '' }
  load()
}
function openEdit(e: WinLoseEntry) {
  form.value = { mode: 'edit', week_start: e.week_start, branch_id: e.branch_id }
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
    const result = await api<WeekSheet>('/winlose/entry', { query: { week_start: form.value.week_start, branch_id: form.value.branch_id || undefined } })
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
const canSave = computed(() => !!sheet.value && !sheet.value.locked && (canCreate.value || canEdit.value))

function editable(row: WeekSheetRow) {
  if (!canSave.value) return false
  return row.saved ? canEdit.value : canCreate.value
}

function statusOf(row: WeekSheetRow) {
  if (changed(row)) return row.saved ? 'Changed — not saved' : 'New — not saved'
  if (row.saved) return `Saved${row.updated_by ? ` by ${row.updated_by}` : ''}`
  return 'Not entered'
}

function filledRows() {
  const clean = (value: string) => value.replace(/\.$/, '')
  return (sheet.value?.rows || [])
    .map(row => ({ platform_id: row.platform_id, win_lose: clean(edits[row.platform_id]!.win_lose), commission: clean(edits[row.platform_id]!.commission) }))
    .filter(row => row.win_lose || row.commission)
}

// --- saving ---
const saving = ref(false)
const saveError = ref('')
const reasonOpen = ref(false)
const NEED_TITLE = 'Give the entry a title.'
watch(title, (value) => { if (value.trim() && saveError.value === NEED_TITLE) saveError.value = '' })

function trySave() {
  saveError.value = ''
  if (hasInvalid.value) return
  if (!title.value.trim()) { saveError.value = NEED_TITLE; return }
  // Changing a saved week needs a reason (audit log).
  if (sheet.value?.rows.some(row => row.saved && changed(row))) {
    reasonOpen.value = true
    return
  }
  save('')
}

async function save(reason: string) {
  if (!sheet.value || !form.value) return
  saving.value = true
  saveError.value = ''
  try {
    const saved = await api<WeekSheet>('/winlose/entry', {
      method: 'PUT',
      body: { week_start: form.value.week_start, branch_id: sheet.value.branch_id, title: title.value.trim(), rows: filledRows(), reason },
    })
    reasonOpen.value = false
    toast.success(`Week of ${formatDocDate(saved.week_start)}`, { title: `“${saved.entry?.title || title.value.trim()}” saved` })
    form.value = null
    sheet.value = null
    await loadEntries()
  } catch (e: any) {
    saveError.value = e?.data?.error || 'Could not save the entry.'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
/* Phones: each agent becomes a card: its name, then the two amounts side by side. */
@media (max-width: 639px) {
  .wl-table thead { display: none; }
  .wl-table, .wl-table tbody, .wl-table tr, .wl-table td { display: block; width: 100%; }
  .wl-table tbody tr:not(.empty-row) { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 10px; padding: 12px; }
  .wl-table tbody tr:not(.empty-row) td { padding: 0; text-align: left; }
  .wl-table tbody tr:not(.empty-row) td:first-child { grid-column: 1 / -1; }
  .wl-table td[data-label]::before { content: attr(data-label); display: block; margin-bottom: 3px; color: #64748b; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; }
}
</style>
