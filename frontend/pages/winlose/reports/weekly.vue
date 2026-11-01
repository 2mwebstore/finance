<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <!-- A group's or an agent's detail (or one entry of it): back to where it was opened from -->
        <button v-if="inDetail" type="button" class="mb-2 flex w-fit items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="goBack"><ChevronLeft :size="14" /> Back</button>
        <p v-else class="text-[10px] font-bold tracking-[0.14em] text-slate-400">RP · REPORTS</p>
        <h1 class="text-xl font-bold text-navy" :class="agentId ? 'font-mono' : ''">{{ agentId ? agentCode || 'Agent' : groupId ? groupTitle : 'Report' }}</h1>
        <p v-if="weekStart" class="text-xs text-muted"><template v-if="entryId && report?.entries?.[0]">{{ report.entries[0].title }} · </template>{{ weekLabel(weekStart, plusDays(weekStart, 6)) }}</p>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy disabled:opacity-50" :disabled="!report?.lines.length" @click="exportCsv"><Download :size="16" /> Export CSV</button>
    </header>

    <!-- In a detail only the dates can change, in one entry (or week) of it nothing -->
    <div v-if="!weekStart" class="grid grid-cols-1 gap-2" :class="inDetail ? '' : 'md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]'">
      <WinloseDateFilter v-model="range" />
      <SearchSelect v-if="!inDetail" :model-value="groupId" :options="groupOptions" aria-label="Group" @update:model-value="(id: string) => id && openGroup(id)" />
    </div>

    <div v-if="loading && !report" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the report" :message="error" />
    <template v-else-if="report">
      <WinloseTotals v-if="report.totals.length" :totals="report.totals" title="Totals" />
      <!-- One agent: its part in each of its groups, at its rate there -->
      <template v-if="agentId">
        <p class="text-[11px] text-muted">Total = (W/L − COMM) × Rate, the Rate being 100% − the agent's rate in the group</p>
        <WinloseGroupTotals :groups="report.groups" detail />
        <!-- By week: the agent in each data entry (week-1, week-2, …); one opens its groups in it -->
        <WinloseEntryTotals v-if="!weekStart" title="By week" :entries="report.entries || []" selectable @select="openEntry" />
        <WinloseEntryTotals v-else :entries="report.entries || []" />
        <p v-if="!report.groups.length" class="rounded-xl border border-border bg-white px-4 py-8 text-center text-sm text-muted">No results entered for these weeks.</p>
      </template>
      <template v-else>
        <WinloseGroupTotals v-if="!groupId" :groups="report.groups" selectable @select="openGroup" />
        <!-- One group: its agents, each at its rate in the group -->
        <div v-if="groupId" class="space-y-1">
          <h2 class="flex items-center gap-2 text-sm font-semibold text-navy"><Network :size="15" /> {{ groupTitle }}</h2>
          <p class="text-[11px] text-muted">Total = (W/L − COMM) × Rate, the Rate being 100% − the agent's rate</p>
        </div>
        <p v-else-if="report.lines.length" class="text-[11px] text-muted">Tap an agent for its groups</p>
        <!-- One entry (or week) of a group: its agents in each data entry (a title opens the entry), with ✅ / ❌ -->
        <template v-if="weekStart && report.entries?.length">
          <article v-for="e in report.entries" :key="e.id || 'none'" class="space-y-1.5">
            <div class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 px-1">
              <h3 class="flex items-center gap-1.5 text-sm font-semibold text-navy">
                <FileText :size="14" class="shrink-0 text-muted" />
                <NuxtLink v-if="e.id" :to="`/winlose/entries/${e.id}`" class="text-primary hover:underline">{{ e.title }}</NuxtLink>
                <template v-else>{{ e.title }}</template>
              </h3>
              <div class="flex flex-wrap items-center gap-x-2 gap-y-1">
                <p v-if="e.created_at" class="text-[11px] text-muted">Created {{ formatDocDate(e.created_at) }}<template v-if="e.created_by"> by {{ e.created_by }}</template></p>
                <WinloseEntryReactions v-if="e.id" :url="`/winlose/entries/${e.id}/reactions`" :reactions="e.reactions || []" />
              </div>
            </div>
            <WinloseReportTable :report="{ lines: e.lines || [], totals: e.totals }" by-group selectable @select="openAgent" />
          </article>
        </template>
        <WinloseReportTable v-else :report="report" :by-group="!!groupId" selectable @select="openAgent" />
        <!-- By week: the group in each data entry (week-1, week-2, …); one opens its agents in it -->
        <WinloseEntryTotals v-if="groupId && !weekStart" title="By week" :entries="report.entries || []" selectable @select="openEntry" />
      </template>
    </template>

    <!-- Settlement: only for a whole Monday-to-Sunday week -->
    <WinloseSettlements v-if="isWholeWeek" :week-start="range.from" @changed="load" />
    <p v-else-if="!isWholeWeek" class="text-[11px] text-muted">Choose a whole week (Monday to Sunday) to see or finalize its settlement.</p>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, Download, FileText, Network } from 'lucide-vue-next'
import type { WinLoseReportEntry } from '~/types/api'
import { useWinLoseReport } from '~/composables/useWinLoseReport'
import { currentMonthRange } from '~/utils/dateRanges'
import { formatDocDate } from '~/utils/inventory'
import { downloadCsv, mondayOf, plusDays, weekLabel } from '~/utils/winlose'

const route = useRoute()
const thisMonth = currentMonthRange()
const range = ref({ from: String(route.query.from || thisMonth.from), to: String(route.query.to || thisMonth.to) }) // this month
// A detail has its data entries too (By week); one entry of it, only that entry's results.
const { report, loading, error, load, groups, groupId, agentId, groupOptions } = useWinLoseReport(() => range.value, { byEntry: () => inDetail.value, entryId: () => entryId.value })

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
// Back: to the step before when it was on this report, else out of the entry, else out of the detail.
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

function exportCsv() {
  if (!report.value) return
  downloadCsv(`winlose-weekly-${range.value.from}-to-${range.value.to}.csv`,
    ['From', 'To', 'Agent', 'Currency', 'W/L', 'COMM', 'Total'],
    report.value.lines.map(l => [report.value!.date_from, range.value.to, l.platform_code,
      l.currency,
      Number(l.win_lose), Number(l.commission), Number(l.jame_ps)]))
}
</script>
