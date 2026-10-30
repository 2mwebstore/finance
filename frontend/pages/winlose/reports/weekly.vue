<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS · REPORTS</p>
        <h1 class="text-xl font-bold text-navy">Weekly report</h1>
        <p class="mt-1 text-xs text-muted">The weekly win/lose added up by agent. Weeks run Monday to Sunday; a range counts every week that starts in it.</p>
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
      <WinloseReportTable :report="report" show-weeks />
      <WinloseGroupTotals v-if="!groupId" :groups="report.groups" />
    </template>

    <!-- Settlement: only for a whole Monday-to-Sunday week -->
    <WinloseSettlements v-if="isWholeWeek" :week-start="range.from" @changed="load" />
    <p v-else-if="!isWholeWeek" class="text-[11px] text-muted">Choose a whole week (Monday to Sunday) to see or finalize its settlement.</p>
  </section>
</template>

<script setup lang="ts">
import { Download } from 'lucide-vue-next'
import { useWinLoseReport } from '~/composables/useWinLoseReport'
import { todayISO } from '~/utils/inventory'
import { downloadCsv, mondayOf, plusDays } from '~/utils/winlose'

const route = useRoute()
const thisMonday = mondayOf(todayISO())
const range = ref({ from: String(route.query.from || thisMonday), to: String(route.query.to || plusDays(thisMonday, 6)) }) // this week
const { report, loading, error, load, groupId, groupOptions } = useWinLoseReport(() => range.value)

const isWholeWeek = computed(() => !!range.value.from && mondayOf(range.value.from) === range.value.from && plusDays(range.value.from, 6) === range.value.to)

const rateText = (min: string, max: string) => min === max ? String(Number(min)) : `${Number(min)}-${Number(max)}`

function exportCsv() {
  if (!report.value) return
  downloadCsv(`winlose-weekly-${range.value.from}-to-${range.value.to}.csv`,
    ['From', 'To', 'Agent', 'Currency', 'Rate %', 'Win/Lose', 'Commission', 'Total', 'Weeks'],
    report.value.lines.map(l => [report.value!.date_from, range.value.to, l.platform_code,
      l.currency,
      rateText(l.rate_min, l.rate_max),
      Number(l.win_lose), Number(l.commission), Number(l.jame_ps), l.entries]))
}
</script>
