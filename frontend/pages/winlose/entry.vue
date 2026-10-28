<template>
  <section class="mx-auto max-w-5xl space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS</p>
        <h1 class="text-xl font-bold text-navy">Win/lose entry</h1>
        <p class="mt-1 text-xs text-muted">Enter each platform’s win/lose and commission once a week (Monday to Sunday).</p>
      </div>
      <NuxtLink v-if="can('winlose-reports.view')" :to="{ path: '/winlose/reports/weekly', query: { from: weekStart, to: weekEnd, branch: branchId } }" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy hover:bg-canvas"><FileSpreadsheet :size="16" /> Weekly report</NuxtLink>
    </header>

    <div class="grid grid-cols-1 gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
      <!-- The week: any day picks its Monday-to-Sunday week -->
      <div class="flex items-center gap-1.5">
        <button type="button" class="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-border bg-white text-navy hover:bg-canvas" aria-label="Previous week" @click="shiftWeek(-7)"><ChevronLeft :size="17" /></button>
        <div class="min-w-0 flex-1"><DatePicker :model-value="weekStart" aria-label="Week" @update:model-value="(day) => (weekStart = mondayOf(day))" /></div>
        <button type="button" class="grid h-10 w-10 shrink-0 place-items-center rounded-lg border border-border bg-white text-navy hover:bg-canvas disabled:opacity-40" aria-label="Next week" :disabled="weekStart >= thisMonday" @click="shiftWeek(7)"><ChevronRight :size="17" /></button>
      </div>
      <SearchSelect v-model="branchId" :options="hqOptions" placeholder="Choose a branch" aria-label="Branch" />
    </div>
    <p class="-mt-2 text-xs font-semibold text-navy">Week of {{ formatDocDate(weekStart) }} – {{ formatDocDate(weekEnd) }}<span v-if="weekStart === thisMonday" class="ml-1.5 rounded-full bg-blue-50 px-2 py-0.5 text-[10px] font-bold text-primary">This week</span></p>

    <p v-if="!hqOptions.length" class="rounded-xl border border-border bg-white p-6 text-center text-sm text-muted">You don’t work in any branch yet.</p>
    <div v-else-if="loading" class="text-sm text-muted">Loading the week…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the week" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <template v-else-if="sheet">
      <AppAlert v-if="sheet.locked" type="warning" title="This week is finalized" :message="`The week of ${formatDocDate(sheet.week_start)} – ${formatDocDate(sheet.week_end)} is settled for ${sheet.branch_code}. Reopen it on the weekly report to change it.`" />
      <div class="overflow-hidden rounded-xl border border-border bg-white">
        <table class="wl-table w-full text-left text-sm">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-3 py-2 font-bold">Platform</th>
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
            <tr v-if="!sheet.rows.length" class="empty-row"><td colspan="3" class="px-3 py-8 text-center text-sm text-muted">No platforms are assigned to {{ sheet.branch_code }} in this week.<NuxtLink v-if="can('platform-assignments.manage')" :to="{ path: '/admin/platform-assignments', query: { branch: branchId } }" class="ml-1 font-semibold text-primary">Assign platforms</NuxtLink></td></tr>
          </tbody>
        </table>
      </div>
      <p v-if="hasInvalid" class="text-xs font-medium text-expense" role="alert">Enter amounts as numbers, e.g. -1500 or 250.75.</p>

      <AppAlert v-if="saveError" title="Couldn’t save the week" :message="saveError" dismissible @dismiss="saveError = ''" />
      <div v-if="sheet.rows.length && !sheet.locked && (canCreate || canEdit)" class="flex flex-wrap items-center justify-end gap-2">
        <span v-if="dirty" class="mr-auto text-xs text-amber-700">Unsaved changes</span>
        <button type="button" class="rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-muted disabled:opacity-50" :disabled="!dirty || saving" @click="resetEdits">Undo changes</button>
        <button type="button" class="rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-white disabled:opacity-50" :disabled="!dirty || saving || hasInvalid" @click="trySave">{{ saving ? 'Saving…' : 'Save week' }}</button>
      </div>
    </template>

    <InventoryReasonDialog v-model="reasonOpen" title="Change saved results?" eyebrow="CORRECTION" subtitle="The old and new figures and your reason go into the audit log." label="Reason for the change" confirm-label="Save changes" :busy="saving" :server-error="saveError" @confirm="(reason) => save(reason)" />
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, ChevronRight, FileSpreadsheet } from 'lucide-vue-next'
import type { WeekSheet, WeekSheetRow } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { formatDocDate, todayISO } from '~/utils/inventory'
import { mondayOf, plusDays } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const route = useRoute()
const canCreate = computed(() => can('winlose.create'))
const canEdit = computed(() => can('winlose.edit'))
const { branches } = useBranchOptions()
const hqOptions = computed(() => branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })))

// Results are entered once a week: the sheet is a Monday-to-Sunday week.
const thisMonday = mondayOf(todayISO())
const weekStart = ref(mondayOf(String(route.query.week || thisMonday)))
const weekEnd = computed(() => plusDays(weekStart.value, 6))
const branchId = ref(String(route.query.branch || ''))
watch(branches, (list) => { if (!branchId.value && list.length) branchId.value = list[0]!.id }, { immediate: true })

function shiftWeek(days: number) {
  weekStart.value = plusDays(weekStart.value, days)
}

const sheet = ref<WeekSheet | null>(null)
const loading = ref(false)
const error = ref('')
const edits = reactive<Record<string, { win_lose: string; commission: string }>>({})

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
}

let loadRequest = 0
async function load() {
  if (!branchId.value) return
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<WeekSheet>('/winlose/entry', { query: { week_start: weekStart.value, branch_id: branchId.value } })
    if (request !== loadRequest) return
    sheet.value = result
    resetEdits()
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load the week.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}

// Changing the week or branch with unsaved figures asks first.
let reverting = false
watch([weekStart, branchId], ([w, b], [oldW, oldB]) => {
  if (reverting) { reverting = false; return }
  if (dirty.value && !window.confirm('Leave this week without saving your changes?')) {
    reverting = true
    if (w !== oldW) weekStart.value = oldW
    else branchId.value = oldB
    return
  }
  load()
})
onMounted(load)
onBeforeRouteLeave(() => !dirty.value || window.confirm('Leave without saving your changes?'))

function changed(row: WeekSheetRow) {
  const e = edits[row.platform_id]
  if (!e) return false
  return row.saved ? !same(e.win_lose, row.win_lose) || !same(e.commission, row.commission) : !!(e.win_lose || e.commission)
}
const dirty = computed(() => (sheet.value?.rows || []).some(changed))
const hasInvalid = computed(() => Object.values(edits).some(e => invalid(e.win_lose) || invalid(e.commission)))

// Saved rows need Correct Win/Lose; new ones Enter Win/Lose.
function editable(row: WeekSheetRow) {
  if (!sheet.value || sheet.value.locked) return false
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

function trySave() {
  saveError.value = ''
  if (hasInvalid.value) return
  // Changing a saved week needs a reason (audit log).
  if (sheet.value?.rows.some(row => row.saved && changed(row))) {
    reasonOpen.value = true
    return
  }
  save('')
}

async function save(reason: string) {
  if (!sheet.value) return
  saving.value = true
  saveError.value = ''
  try {
    sheet.value = await api<WeekSheet>('/winlose/entry', { method: 'PUT', body: { week_start: weekStart.value, branch_id: branchId.value, rows: filledRows(), reason } })
    reasonOpen.value = false
    resetEdits()
    toast.success(`${sheet.value.branch_code} · week of ${formatDocDate(sheet.value.week_start)}`, { title: 'Week saved' })
  } catch (e: any) {
    saveError.value = e?.data?.error || 'Could not save the week.'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
/* Phones: each platform becomes a card: its name, then the two amounts side by side. */
@media (max-width: 639px) {
  .wl-table thead { display: none; }
  .wl-table, .wl-table tbody, .wl-table tr, .wl-table td { display: block; width: 100%; }
  .wl-table tbody tr:not(.empty-row) { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 10px; padding: 12px; }
  .wl-table tbody tr:not(.empty-row) td { padding: 0; text-align: left; }
  .wl-table tbody tr:not(.empty-row) td:first-child { grid-column: 1 / -1; }
  .wl-table td[data-label]::before { content: attr(data-label); display: block; margin-bottom: 3px; color: #64748b; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; }
}
</style>
