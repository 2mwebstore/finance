<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS · REPORTS</p>
        <h1 class="text-xl font-bold text-navy">Weekly report</h1>
        <p class="mt-1 text-xs text-muted">The weekly win/lose added up by HQ and platform. Weeks run Monday to Sunday; a range counts every week that starts in it.</p>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy disabled:opacity-50" :disabled="!report?.lines.length" @click="exportCsv"><Download :size="16" /> Export CSV</button>
    </header>

    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="flex gap-1.5">
        <button v-for="w in quickWeeks" :key="w.label" type="button" class="rounded-lg border px-3 py-2 text-xs font-semibold" :class="range.from === w.from && range.to === w.to ? 'border-primary bg-blue-50 text-primary' : 'border-border bg-white text-muted hover:text-navy'" @click="range = { from: w.from, to: w.to }">{{ w.label }}</button>
      </div>
      <div class="grid min-w-0 flex-1 grid-cols-1 gap-2 sm:grid-cols-2">
        <DateRangePicker v-model="range" />
        <SearchSelect v-model="branchId" :options="hqOptions" placeholder="Choose an HQ" aria-label="HQ" />
      </div>
    </div>

    <div v-if="loading && !report" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the report" :message="error" />
    <template v-else-if="report">
      <WinloseTotals v-if="report.totals.length" :totals="report.totals" :title="report.consolidated ? 'All HQs' : 'This HQ'" />
      <WinloseReportTable :report="report" show-weeks />
    </template>

    <!-- Settlement: only for a whole Monday-to-Sunday week -->
    <WinloseSettlements v-if="isWholeWeek && settlementHQs.length" :week-start="range.from" :hqs="settlementHQs" @changed="load" />
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
const quickWeeks = [
  { label: 'This week', from: thisMonday, to: plusDays(thisMonday, 6) },
  { label: 'Last week', from: plusDays(thisMonday, -7), to: plusDays(thisMonday, -1) },
]
const range = ref({ from: String(route.query.from || quickWeeks[0]!.from), to: String(route.query.to || quickWeeks[0]!.to) })
const { report, loading, error, load, branchId, branches, hqOptions } = useWinLoseReport(() => range.value)

const isWholeWeek = computed(() => !!range.value.from && mondayOf(range.value.from) === range.value.from && plusDays(range.value.from, 6) === range.value.to)
// The chosen HQ, or every HQ the user works in.
const settlementHQs = computed(() => branches.value
  .filter(b => !branchId.value || b.id === branchId.value)
  .map(b => ({ id: b.id, code: b.code, name: b.name })))

function exportCsv() {
  if (!report.value) return
  downloadCsv(`winlose-weekly-${range.value.from}-to-${range.value.to}.csv`,
    ['From', 'To', 'HQ code', 'HQ', 'Platform', 'Currency', 'Rate %', 'Win/Lose', 'Commission', 'Total', 'Weeks'],
    report.value.lines.map(l => [report.value!.date_from, range.value.to, l.branch_code, l.branch_name, l.platform_code, l.currency,
      l.rate_min === l.rate_max ? Number(l.rate_min) : `${Number(l.rate_min)}-${Number(l.rate_max)}`,
      Number(l.win_lose), Number(l.commission), Number(l.jame_ps), l.entries]))
}
</script>
