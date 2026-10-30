<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS · REPORTS</p>
        <h1 class="text-xl font-bold text-navy">Group summary</h1>
        <p class="mt-1 text-xs text-muted">Totals for the weeks starting in the dates you choose, by agent and by group. USD and KHR are kept apart.</p>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy disabled:opacity-50" :disabled="!report?.platforms.length" @click="exportCsv"><Download :size="16" /> Export CSV</button>
    </header>

    <div class="grid grid-cols-1 gap-2 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
      <WinloseDateFilter v-model="range" />
      <SearchSelect v-model="groupId" :options="groupOptions" aria-label="Group" />
    </div>

    <div v-if="loading && !report" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the summary" :message="error" />
    <template v-else-if="report">
      <WinloseTotals v-if="report.totals.length" :totals="report.totals" title="Totals" />

      <!-- Each agent's totals in each currency -->
      <article class="overflow-x-auto rounded-xl border border-border bg-white">
        <h2 class="flex items-center gap-2 border-b border-border px-4 py-3 text-sm font-semibold text-navy"><Layers :size="15" /> By agent</h2>
        <table class="w-full min-w-[560px] text-left text-xs">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-3 py-2 font-bold">Agent</th>
              <th class="px-3 py-2 font-bold">Currency</th>
              <th class="px-3 py-2 text-right font-bold">Win/Lose</th>
              <th class="px-3 py-2 text-right font-bold">Commission</th>
              <th class="px-3 py-2 text-right font-bold">Total</th>
              <th class="px-3 py-2 text-right font-bold">Entries</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <template v-for="p in report.platforms" :key="p.platform_id">
              <tr v-for="(t, i) in p.totals" :key="t.currency" class="text-navy">
                <td class="px-3 py-2 font-mono font-semibold">{{ i === 0 ? p.platform_code : '' }}</td>
                <td class="px-3 py-2">{{ t.currency }}</td>
                <td class="px-3 py-2 text-right tabular-nums" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
                <td class="px-3 py-2 text-right tabular-nums">{{ formatAmount(t.commission, t.currency) }}</td>
                <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
                <td class="px-3 py-2 text-right tabular-nums text-muted">{{ t.entries }}</td>
              </tr>
            </template>
            <tr v-if="!report.platforms.length"><td colspan="6" class="px-3 py-8 text-center text-muted">No results entered for these weeks.</td></tr>
          </tbody>
        </table>
      </article>

      <WinloseGroupTotals v-if="!groupId" :groups="report.groups" />
    </template>
  </section>
</template>

<script setup lang="ts">
import { Download, Layers } from 'lucide-vue-next'
import { useWinLoseReport } from '~/composables/useWinLoseReport'
import { currentMonthRange } from '~/utils/dateRanges'
import { downloadCsv, formatAmount } from '~/utils/winlose'

const range = ref(currentMonthRange()) // this month
const { report, loading, error, groupId, groupOptions } = useWinLoseReport(() => range.value)

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}

function exportCsv() {
  if (!report.value) return
  downloadCsv(`winlose-summary-${range.value.from}-to-${range.value.to}.csv`,
    ['From', 'To', 'Agent', 'Currency', 'Win/Lose', 'Commission', 'Total', 'Entries'],
    report.value.platforms.flatMap(p => p.totals.map(t => [report.value!.date_from, range.value.to, p.platform_code, t.currency,
      Number(t.win_lose), Number(t.commission), Number(t.jame_ps), t.entries])))
}
</script>
