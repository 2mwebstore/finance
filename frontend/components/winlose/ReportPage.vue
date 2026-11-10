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

    <!-- In a detail only the dates can change, in one entry (or week) of it nothing -->
    <div v-if="!weekStart" class="grid grid-cols-1 gap-2" :class="inDetail ? '' : showBranch ? 'md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)_minmax(0,1fr)]' : 'md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]'">
      <WinloseDateFilter v-model="range" />
      <!-- Only for people who work in more than one branch (chosen once for all the RP pages) -->
      <SearchSelect v-if="!inDetail && showBranch" v-model="branchModel" :options="branchOptions" :searchable="false" aria-label="Branch" />
      <SearchSelect v-if="!inDetail" :model-value="groupId" :options="groupOptions" aria-label="Group" @update:model-value="(id: string) => id && openGroup(id)" />
    </div>

    <div v-if="loading && !report" class="text-sm text-muted">Loading…</div>
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
        <WinloseGroupTotals v-if="!groupId" :groups="report.groups" selectable @select="openGroup" />
        <!-- One group: its agents, each at its rate in the group -->
        <div v-if="groupId" class="space-y-1">
          <h2 class="flex items-center gap-2 text-sm font-semibold text-navy"><Network :size="15" /> {{ groupTitle }}</h2>
        </div>
        <p v-else-if="report.lines.length && !summary" class="text-[11px] text-muted">Tap an agent for its groups</p>
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
            <WinloseReportTable :report="{ lines: e.lines || [], totals: e.totals }" by-group selectable @select="openAgent" />
          </article>
        </template>
        <!-- One group over the dates: every agent in one card (on Group summary with an Image button:
             one picture of all of them) -->
        <article v-else-if="groupId" id="group-card" class="space-y-1.5">
          <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-1">
            <h3 class="text-sm font-semibold text-navy">{{ formatDocDate(range.from) }} – {{ formatDocDate(range.to) }}</h3>
            <WinloseEntryImage v-if="summary" target-id="group-card" :title="`${groupTitle} ${range.from} to ${range.to}`" :group="groupTitle" />
          </div>
          <WinloseReportTable :report="report" by-group selectable @select="openAgent" />
        </article>
        <!-- Group summary: each agent's totals in each currency (an agent opens its groups) -->
        <article v-else-if="summary" class="overflow-x-auto rounded-xl border border-border bg-white">
          <h2 class="flex items-center gap-2 border-b border-border px-4 py-3 text-sm font-semibold text-navy"><Layers :size="15" /> By agent<span v-if="report.platforms.length" class="ml-auto text-[11px] font-normal text-muted">Tap an agent for its groups</span></h2>
          <table class="report-table">
            <thead class="bg-canvas uppercase text-slate-500">
              <tr>
                <th class="font-bold">Agent</th>
                <!-- On phones the currency goes under the agent -->
                <th class="hidden font-bold sm:table-cell">Currency</th>
                <th class="amount font-bold">W/L</th>
                <th class="amount font-bold">COMM</th>
                <th class="amount font-bold">Total</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-border">
              <template v-for="p in report.platforms" :key="p.platform_id">
                <tr v-for="(t, i) in p.totals" :key="t.currency" class="text-navy">
                  <td class="name">
                    <button v-if="i === 0" type="button" class="code font-mono font-semibold text-primary hover:underline" @click="openAgent(p.platform_id)">{{ p.platform_code }}</button>
                    <span class="block text-muted sm:hidden">{{ t.currency }}</span>
                  </td>
                  <td class="hidden sm:table-cell">{{ t.currency }}</td>
                  <td class="amount" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
                  <td class="amount">{{ formatAmount(t.commission, t.currency) }}</td>
                  <td class="amount font-bold" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
                </tr>
              </template>
              <tr v-if="!report.platforms.length"><td colspan="5" class="empty text-muted">No results entered for these weeks.</td></tr>
            </tbody>
          </table>
        </article>
        <WinloseReportTable v-else :report="report" selectable @select="openAgent" />
        <!-- By week: the group in each data entry (week-1, week-2, …); one opens its agents in it -->
        <WinloseEntryTotals v-if="groupId && !weekStart" title="By week" :entries="report.entries || []" selectable :reactions="summary" @select="openEntry" />
      </template>
    </template>

    <!-- Report: the settlement, only for a whole Monday-to-Sunday week -->
    <template v-if="!summary">
      <WinloseSettlements v-if="isWholeWeek" :week-start="range.from" :branch-id="branch" @changed="load" />
      <p v-else class="text-[11px] text-muted">Choose a whole week (Monday to Sunday) to see or finalize its settlement.</p>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, Download, FileText, Layers, Network } from 'lucide-vue-next'
import type { WinLoseReportEntry } from '~/types/api'
import { useWinLoseReport } from '~/composables/useWinLoseReport'
import { currentMonthRange } from '~/utils/dateRanges'
import { formatDocDate } from '~/utils/inventory'
import { downloadCsv, formatAmount, mondayOf, plusDays, weekLabel } from '~/utils/winlose'

/**
 * The win/lose Report and the Group summary: totals by group and by agent over the dates, where a
 * group opens its agents, an agent its groups and a week of a detail (By week) that week of it.
 * Report (default) has the week's settlement; Group summary (summary) has each entry's ✅ / ❌ and
 * the Image button (a picture of an entry or of a group over the dates).
 */
const props = withDefaults(defineProps<{ summary?: boolean }>(), { summary: false })

const route = useRoute()
const thisMonth = currentMonthRange()
const range = ref({ from: String(route.query.from || thisMonth.from), to: String(route.query.to || thisMonth.to) }) // this month
// A detail has its data entries too (By week); one entry of it, only that entry's results.
const { report, loading, error, load, groups, groupId, agentId, groupOptions, showBranch, branchModel, branchOptions, branch } = useWinLoseReport(() => range.value, { byEntry: () => inDetail.value, entryId: () => entryId.value })

// A group's title (in By group) opens its agents, an agent's code its groups, and a week in a
// detail's By week that week of it. Each goes in the address, so the phone's back button (or Back)
// closes it again.
const router = useRouter()
const inDetail = computed(() => !!groupId.value || !!agentId.value)
const groupTitle = computed(() => groups.value.find(g => g.id === groupId.value)?.title || 'Group')
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
// Back: to the step before when it was on this page, else out of the entry, else out of the detail.
function goBack() {
  if (String(window.history.state?.back || '').startsWith(route.path)) return router.back()
  const { week, entry: _e, ...withoutEntry } = route.query
  router.replace({ query: week ? withoutEntry : withoutDetail() })
}
watch(() => [route.query.group, route.query.agent], ([group, agent]) => {
  groupId.value = String(group || '')
  agentId.value = String(agent || '')
})

// One entry (or week) of a detail: the dates are its week, and the dates before come back when it
// closes.
const weekStart = computed(() => {
  const week = String(route.query.week || '')
  return inDetail.value && /^\d{4}-\d{2}-\d{2}$/.test(week) ? mondayOf(week) : ''
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

const isWholeWeek = computed(() => !!range.value.from && mondayOf(range.value.from) === range.value.from && plusDays(range.value.from, 6) === range.value.to)

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}

// The same columns on both; Group summary is named for it.
function exportCsv() {
  if (!report.value) return
  const r = report.value
  if (props.summary) {
    downloadCsv(`winlose-summary-${range.value.from}-to-${range.value.to}.csv`,
      ['From', 'To', 'Agent', 'Currency', 'W/L', 'COMM', 'Total'],
      r.platforms.flatMap(p => p.totals.map(t => [r.date_from, range.value.to, p.platform_code, t.currency,
        Number(t.win_lose), Number(t.commission), Number(t.jame_ps)])))
    return
  }
  downloadCsv(`winlose-weekly-${range.value.from}-to-${range.value.to}.csv`,
    ['From', 'To', 'Agent', 'Currency', 'W/L', 'COMM', 'Total'],
    r.lines.map(l => [r.date_from, range.value.to, l.platform_code, l.currency, Number(l.win_lose), Number(l.commission), Number(l.jame_ps)]))
}
</script>
