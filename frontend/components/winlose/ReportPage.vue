<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <!-- A group's or an agent's detail (or one entry of it): back to where it was opened from -->
        <button v-if="inDetail" type="button" class="mb-2 flex w-fit items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="goBack"><ChevronLeft :size="14" /> Back</button>
        <p v-else class="text-[10px] font-bold tracking-[0.14em] text-slate-400">RP · REPORTS</p>
        <h1 class="text-xl font-bold text-navy" :class="agentId ? 'font-mono' : ''">{{ agentId ? agentCode || 'Agent' : groupId ? groupTitle : summary ? 'Group summary' : 'Report' }}</h1>
        <p v-if="weekStart" class="text-xs text-muted"><template v-if="entryId && report?.entries?.[0]">{{ report.entries[0].title }} · </template>{{ weekLabel(weekStart, plusDays(weekStart, 6)) }}</p>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy disabled:opacity-50" :disabled="!report?.lines.length" @click="exportCsv"><Download :size="16" /> Export CSV</button>
    </header>

    <!-- Report: the dates (in a detail only they can change, in one entry or week of it nothing).
         Group summary: a data entry first (nothing shows until one is chosen), then the group. -->
    <div v-if="summary ? !inDetail : !weekStart" class="grid grid-cols-1 gap-2" :class="filterColumns">
      <SearchSelect v-if="summary" :model-value="summaryEntry" :options="entryOptions" placeholder="Choose a data entry" search-placeholder="Search data entries…" :empty-text="entriesLoading ? 'Loading…' : 'No data entries'" aria-label="Data entry" @update:model-value="pickEntry" />
      <WinloseDateFilter v-else v-model="range" />
      <!-- Only for people who work in more than one branch (chosen once for all the RP pages) -->
      <SearchSelect v-if="!inDetail && showBranch" v-model="branchModel" :options="branchOptions" :searchable="false" aria-label="Branch" />
      <SearchSelect v-if="showGroupFilter" :model-value="groupId" :options="groupOptions" aria-label="Group" @update:model-value="(id: string) => id && openGroup(id)" />
    </div>

    <!-- Group summary before a data entry is chosen -->
    <div v-if="summary && !summaryEntry" class="rounded-xl border border-dashed border-border bg-white px-4 py-10 text-center">
      <FileText :size="22" class="mx-auto text-muted" />
      <p class="mt-2 text-sm font-semibold text-navy">{{ entriesLoading ? 'Loading data entries…' : entries.length ? 'Choose a data entry' : 'No data entries yet' }}</p>
      <p class="mt-1 text-xs text-muted">{{ entries.length || entriesLoading ? 'Its groups and agents show here, with ✅ / ❌ / 📤 and Image.' : 'Entries made in Data Entry show up here.' }}</p>
    </div>
    <div v-else-if="loading && !report" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" :title="summary ? 'Couldn’t load the summary' : 'Couldn’t load the report'" :message="error" />
    <template v-else-if="report">
      <WinloseTotals v-if="report.totals.length" :totals="report.totals" title="Totals" />
      <!-- One agent: its part in each of its groups, at its rate there -->
      <template v-if="agentId">
        <WinloseGroupTotals :groups="report.groups" detail />
        <!-- By week: the agent in each data entry (week-1, week-2, …); one opens its groups in it -->
        <WinloseEntryTotals v-if="!weekStart" title="By week" :entries="report.entries || []" selectable :reactions="summary" @select="openEntry" />
        <WinloseEntryTotals v-else :entries="report.entries || []" :reactions="summary" />
        <p v-if="!report.groups.length" class="rounded-xl border border-border bg-white px-4 py-8 text-center text-sm text-muted">No results entered for these weeks.</p>
      </template>
      <template v-else>
        <!-- Group summary (one data entry): each group with its ✅ / ❌ in the entry -->
        <WinloseGroupTotals v-if="!groupId" :groups="report.groups" selectable :entry-id="summary ? entryId : ''" @select="openGroup" />
        <!-- One group: its agents, each at its rate in the group (on Group summary with the group's ✅ / ❌) -->
        <div v-if="groupId" class="space-y-1">
          <h2 class="flex flex-wrap items-center gap-2 text-sm font-semibold text-navy"><Network :size="15" /> {{ groupTitle }}
            <WinloseEntryReactions v-if="summary && entryId && groupRow?.group_id" class="font-normal" :url="`/winlose/entries/${entryId}/groups/${groupRow.group_id}/reactions`" :reactions="groupRow.reactions || []" />
          </h2>
        </div>
        <!-- One entry (or week) of a group: its agents in each data entry (a title opens the entry);
             on Group summary with ✅ / ❌ and an Image button -->
        <template v-if="weekStart && report.entries?.length">
          <article v-for="e in report.entries" :id="`entry-card-${e.id || 'none'}`" :key="e.id || 'none'" class="space-y-1.5">
            <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-1">
              <h3 class="flex items-center gap-1.5 text-sm font-semibold text-navy">
                <FileText :size="14" class="shrink-0 text-muted" data-capture-skip />
                <NuxtLink v-if="e.id" :to="`/winlose/entries/${e.id}`" class="text-primary hover:underline">{{ e.title }}</NuxtLink>
                <template v-else>{{ e.title }}</template>
              </h3>
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <p v-if="e.created_at" class="text-[11px] text-muted">Created {{ formatDocDate(e.created_at) }}<template v-if="e.created_by"> by {{ e.created_by }}</template></p>
                <template v-if="summary">
                  <!-- Left out of the picture -->
                  <div v-if="e.id" class="flex items-center gap-x-2" data-capture-skip><WinloseEntryReactions :url="`/winlose/entries/${e.id}/reactions`" :reactions="e.reactions || []" /></div>
                  <WinloseEntryImage :target-id="`entry-card-${e.id || 'none'}`" :title="e.title" :group="groupTitle" />
                </template>
              </div>
            </div>
            <WinloseReportTable :report="{ lines: e.lines || [], totals: e.totals }" by-group selectable :reactions="summary" @select="openAgent" />
          </article>
        </template>
        <!-- One group over the dates: every agent in one card (on Group summary with an Image button:
             one picture of all of them, and each agent's ✅ / ❌ when the dates hold one entry) -->
        <article v-else-if="groupId" id="group-card" class="space-y-1.5">
          <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-1">
            <h3 class="text-sm font-semibold text-navy">{{ formatDocDate(range.from) }} – {{ formatDocDate(range.to) }}</h3>
            <WinloseEntryImage v-if="summary" target-id="group-card" :title="`${groupTitle} ${range.from} to ${range.to}`" :group="groupTitle" />
          </div>
          <WinloseReportTable :report="groupCard || report" by-group selectable :reactions="summary" @select="openAgent" />
          <p v-if="summary && (report.entries?.length || 0) > 1 && can('winlose.react')" class="px-1 text-[11px] text-muted">To give an agent ✅ / ❌, open its week in By week.</p>
        </article>
        <!-- Every agent's totals, a table per currency (an agent opens its groups) -->
        <WinloseReportTable v-else :report="report" selectable title="By agent" hint="Tap an agent for its groups" @select="openAgent" />
        <!-- By week: the group in each data entry (week-1, week-2, …); one opens its agents in it -->
        <WinloseEntryTotals v-if="groupId && !weekStart" title="By week" :entries="report.entries || []" selectable :reactions="summary" @select="openEntry" />
      </template>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, Download, FileText, Network } from 'lucide-vue-next'
import type { WinLoseEntry, WinLoseReportEntry } from '~/types/api'
import { useWinLoseReport } from '~/composables/useWinLoseReport'
import { currentMonthRange } from '~/utils/dateRanges'
import { formatDocDate } from '~/utils/inventory'
import { currencyTables, downloadCsv, mondayOf, plusDays, weekLabel } from '~/utils/winlose'

/**
 * The win/lose Report and the Group summary: totals by group and by agent, where a group opens its
 * agents, an agent its groups and a week of a detail (By week) that week of it. Report (default)
 * is over the dates and lists only the groups and agents with results in them. Group summary (summary) is one data entry, chosen in its Data entry filter (nothing
 * shows until one is): every group and agent of the entry's branch (at 0 when it has no results
 * there), the entry's ✅ / ❌, each group's and agent's (an agent's as in Data entry) and the Image button (a picture of the entry, or of a group in it). A to Z: By group in one table, a row per group with its USD and KHR amounts in it
 * ("$1,000.00 / -៛500,000"); the rest in a table per currency (USD first, then KHR).
 */
const props = withDefaults(defineProps<{ summary?: boolean }>(), { summary: false })

const route = useRoute()
const thisMonth = currentMonthRange()
const range = ref({ from: String(route.query.from || thisMonth.from), to: String(route.query.to || thisMonth.to) }) // this month
// A detail has its data entries too (By week); one entry of it, only that entry's results.
// Group summary: the data entry chosen (its id and week are in the address as entry and week).
const summaryEntry = computed(() => props.summary ? String(route.query.entry || '') : '')
const { report, loading, error, load, groups, groupId, agentId, groupOptions, showBranch, branchModel, branchOptions, branch } = useWinLoseReport(() => range.value, {
  byEntry: () => inDetail.value || !!summaryEntry.value,
  entryId: () => entryId.value,
  everyone: () => props.summary,
  enabled: () => !props.summary || !!summaryEntry.value,
})
const { api } = useApi()

// Group summary's Data entry filter: the entries of the branch chosen (newest first). Choosing one
// shows it; one that isn't in the list after the branch changes is let go.
const entries = ref<WinLoseEntry[]>([])
const entriesLoading = ref(false)
const entryOptions = computed(() => entries.value.map(e => ({
  value: e.id,
  label: `${e.title} · ${e.branch_name || e.branch_code}`, // only the title and the branch
})))
async function loadEntries() {
  if (!props.summary) return
  entriesLoading.value = true
  try {
    entries.value = (await api<{ items: WinLoseEntry[] }>('/winlose/entries', { query: { branch_id: branch.value || undefined } })).items || []
  } catch {
    entries.value = []
  } finally {
    entriesLoading.value = false
  }
  if (summaryEntry.value && !entries.value.some(e => e.id === summaryEntry.value)) pickEntry('')
}
onMounted(loadEntries)
watch(branch, loadEntries)
function pickEntry(id: string) {
  const e = entries.value.find(item => item.id === id)
  const { entry: _e, week: _w, ...rest } = withoutDetail()
  router.replace({ query: e ? { ...rest, entry: e.id, week: e.week_start } : rest })
}

// A group's title (in By group) opens its agents, an agent's code its groups, and a week in a
// detail's By week that week of it. Each goes in the address, so the phone's back button (or Back)
// closes it again.
const router = useRouter()
const inDetail = computed(() => !!groupId.value || !!agentId.value)
// Group summary has the group filter once a data entry is chosen.
const showGroupFilter = computed(() => !inDetail.value && (!props.summary || !!summaryEntry.value))
const filterColumns = computed(() => {
  const count = 1 + (!inDetail.value && showBranch.value ? 1 : 0) + (showGroupFilter.value ? 1 : 0)
  return count === 3 ? 'md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)]' : count === 2 ? 'md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]' : ''
})
const groupTitle = computed(() => groups.value.find(g => g.id === groupId.value)?.title || 'Group')
// The open group's row in the report (its ✅ / ❌ in Group summary's entry).
const groupRow = computed(() => report.value?.groups.find(g => g.group_id === groupId.value))
const agentCode = ref('')
watch(report, (r) => {
  const line = r?.lines.find(l => l.platform_id === agentId.value)
  if (line) agentCode.value = line.platform_code
})
function withoutDetail() {
  const { group: _g, agent: _a, ...rest } = route.query
  return rest
}
function openDetail(query: Record<string, string>) {
  router.push({ query: { ...withoutDetail(), ...query } })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
const openGroup = (id: string) => openDetail({ group: id })
function openAgent(id: string) {
  agentCode.value = report.value?.lines.find(l => l.platform_id === id)?.platform_code || 'Agent'
  openDetail({ agent: id })
}
// An entry of By week: its week, only its results.
function openEntry(e: WinLoseReportEntry) {
  if (!e.id) return
  router.push({ query: { ...route.query, week: e.week_start, entry: e.id } })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
// Back: to the step before when it was on this page, else out of the entry, else out of the detail
// (Group summary keeps its data entry).
function goBack() {
  if (String(window.history.state?.back || '').startsWith(route.path)) return router.back()
  if (props.summary) return router.replace({ query: withoutDetail() })
  const { week, entry: _e, ...withoutEntry } = route.query
  router.replace({ query: week ? withoutEntry : withoutDetail() })
}
watch(() => [route.query.group, route.query.agent], ([group, agent]) => {
  groupId.value = String(group || '')
  agentId.value = String(agent || '')
})

// One entry (or week) of a detail, or Group summary's data entry: the dates are its week, and the
// dates before come back when it closes.
const weekStart = computed(() => {
  const week = String(route.query.week || '')
  if (!/^\d{4}-\d{2}-\d{2}$/.test(week)) return ''
  if (props.summary) return summaryEntry.value ? mondayOf(week) : ''
  return inDetail.value ? mondayOf(week) : ''
})
const entryId = computed(() => weekStart.value ? String(route.query.entry || '') : '')
let rangeBefore: { from: string; to: string } | null = null
// Opening or closing an entry reloads even when the dates stay the same (a one-week range).
watch(entryId, () => load())
watch(weekStart, (week) => {
  if (week) {
    if (!rangeBefore && range.value.from !== week) rangeBefore = { ...range.value }
    range.value = { from: week, to: plusDays(week, 6) }
  } else if (rangeBefore) {
    range.value = rangeBefore
    rangeBefore = null
  }
}, { immediate: true })

// The dates hold one data entry: the group card's agents are their results in it, so they get their
// ✅ / ❌ there too (over more entries an agent's line adds them up: its week in By week has them).
const { can } = usePermissions()
const groupCard = computed(() => {
  const r = report.value
  const only = r?.entries?.length === 1 ? r.entries[0] : undefined
  if (!r || !props.summary || !only?.id) return r
  const inEntry = new Map((only.lines || []).map(l => [`${l.platform_id}|${l.currency}`, l]))
  return { ...r, lines: r.lines.map((l) => {
    const e = inEntry.get(`${l.platform_id}|${l.currency}`)
    return e?.entry_id ? { ...l, entry_id: e.entry_id, result_id: e.result_id, reactions: e.reactions } : l
  }) }
})

// The same columns on both, in the order shown (USD agents, then KHR); Group summary is named for it.
function exportCsv() {
  if (!report.value) return
  const r = report.value
  const name = props.summary ? 'winlose-summary' : 'winlose-weekly'
  downloadCsv(`${name}-${range.value.from}-to-${range.value.to}.csv`,
    ['From', 'To', 'Agent', 'Currency', 'W/L', 'COMM', 'Total'],
    currencyTables(r.lines, l => l.currency).flatMap(t => t.rows)
      .map(l => [r.date_from, range.value.to, l.platform_code, l.currency, Number(l.win_lose), Number(l.commission), Number(l.jame_ps)]))
}
</script>
