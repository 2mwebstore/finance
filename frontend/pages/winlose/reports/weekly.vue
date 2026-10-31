<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <!-- One group's detail: back to the whole report -->
        <button v-if="groupId" type="button" class="inline-flex items-center gap-1 text-[11px] font-bold tracking-[0.14em] text-slate-400 hover:text-primary" @click="closeGroup"><ChevronLeft :size="13" /> WEEKLY REPORT</button>
        <p v-else class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS · REPORTS</p>
        <h1 class="text-xl font-bold text-navy">{{ groupId ? groupTitle : 'Weekly report' }}</h1>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy disabled:opacity-50" :disabled="!report?.lines.length" @click="exportCsv"><Download :size="16" /> Export CSV</button>
    </header>

    <div class="grid grid-cols-1 gap-2 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <WinloseDateFilter v-model="range" />
      <SearchSelect v-model="groupId" :options="groupOptions" aria-label="Group" />
    </div>

    <div v-if="loading && !report" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the report" :message="error" />
    <template v-else-if="report">
      <WinloseTotals v-if="report.totals.length" :totals="report.totals" title="Totals" />
      <WinloseGroupTotals v-if="!groupId" :groups="report.groups" selectable @select="openGroup" />
      <!-- One group: its agents, each at its rate in the group -->
      <div v-if="groupId" class="space-y-1">
        <div class="flex items-center justify-between gap-3">
          <h2 class="flex items-center gap-2 text-sm font-semibold text-navy"><Network :size="15" /> {{ groupTitle }}</h2>
          <button type="button" class="inline-flex items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="closeGroup"><ChevronLeft :size="14" /> Back</button>
        </div>
        <p class="text-[11px] text-muted">Total = (Win/Lose − Comm) × the agent's rate</p>
      </div>
      <WinloseReportTable :report="report" :by-group="!!groupId" />
    </template>

    <!-- Settlement: only for a whole Monday-to-Sunday week -->
    <WinloseSettlements v-if="isWholeWeek" :week-start="range.from" @changed="load" />
    <p v-else-if="!isWholeWeek" class="text-[11px] text-muted">Choose a whole week (Monday to Sunday) to see or finalize its settlement.</p>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, Download, Network } from 'lucide-vue-next'
import { useWinLoseReport } from '~/composables/useWinLoseReport'
import { todayISO } from '~/utils/inventory'
import { downloadCsv, mondayOf, plusDays } from '~/utils/winlose'

const route = useRoute()
const thisMonday = mondayOf(todayISO())
const range = ref({ from: String(route.query.from || thisMonday), to: String(route.query.to || plusDays(thisMonday, 6)) }) // this week
const { report, loading, error, load, groups, groupId, groupOptions } = useWinLoseReport(() => range.value)

// A group's title (in By group) opens its agents; it goes in the address, so the phone's back
// button (or Back) closes it again.
const router = useRouter()
const groupTitle = computed(() => groups.value.find(g => g.id === groupId.value)?.title || 'Group')
let openedHere = false
function openGroup(id: string) {
  openedHere = true
  router.push({ query: { ...route.query, group: id } })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
function closeGroup() {
  groupId.value = ''
  if (openedHere) {
    openedHere = false
    router.back()
  } else if (route.query.group) {
    const { group: _, ...rest } = route.query
    router.replace({ query: rest })
  }
}
watch(() => route.query.group, (group) => {
  groupId.value = String(group || '')
  if (!group) openedHere = false
})

const isWholeWeek = computed(() => !!range.value.from && mondayOf(range.value.from) === range.value.from && plusDays(range.value.from, 6) === range.value.to)

function exportCsv() {
  if (!report.value) return
  downloadCsv(`winlose-weekly-${range.value.from}-to-${range.value.to}.csv`,
    ['From', 'To', 'Agent', 'Currency', 'Win/Lose', 'Comm', 'Total'],
    report.value.lines.map(l => [report.value!.date_from, range.value.to, l.platform_code,
      l.currency,
      Number(l.win_lose), Number(l.commission), Number(l.jame_ps)]))
}
</script>
