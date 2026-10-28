<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS · REPORTS</p>
        <h1 class="text-xl font-bold text-navy">Branch summary</h1>
        <p class="mt-1 text-xs text-muted">Each branch’s totals for the weeks starting in the dates you choose<template v-if="consolidated">, and all your branches together</template>. USD and KHR are kept apart.</p>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy disabled:opacity-50" :disabled="!report?.hqs.length" @click="exportCsv"><Download :size="16" /> Export CSV</button>
    </header>

    <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
      <DateRangePicker v-model="range" />
      <SearchSelect v-model="branchId" :options="hqOptions" placeholder="Choose a branch" aria-label="Branch" />
      <SearchSelect v-model="partnerId" :options="[{ value: '', label: 'All partners' }, ...partnerOptions]" aria-label="Partner" />
    </div>

    <div v-if="loading && !report" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the summary" :message="error" />
    <template v-else-if="report">
      <WinloseTotals v-if="report.totals.length" :totals="report.totals" :title="report.consolidated ? 'All branches (consolidated)' : 'This branch'" />

      <div class="overflow-x-auto rounded-xl border border-border bg-white">
        <table class="w-full min-w-[560px] text-left text-xs">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-3 py-2 font-bold">Branch</th>
              <th class="px-3 py-2 font-bold">Currency</th>
              <th class="px-3 py-2 text-right font-bold">Win/Lose</th>
              <th class="px-3 py-2 text-right font-bold">Commission</th>
              <th class="px-3 py-2 text-right font-bold">Total</th>
              <th class="px-3 py-2 text-right font-bold">Entries</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <template v-for="hq in report.hqs" :key="hq.branch_id">
              <tr v-for="(t, i) in hq.totals" :key="t.currency" class="text-navy">
                <td class="px-3 py-2 font-semibold">{{ i === 0 ? `${hq.branch_code} — ${hq.branch_name}` : '' }}</td>
                <td class="px-3 py-2">{{ t.currency }}</td>
                <td class="px-3 py-2 text-right tabular-nums" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
                <td class="px-3 py-2 text-right tabular-nums">{{ formatAmount(t.commission, t.currency) }}</td>
                <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
                <td class="px-3 py-2 text-right tabular-nums text-muted">{{ t.entries }}</td>
              </tr>
            </template>
            <tr v-if="!report.hqs.length"><td colspan="6" class="px-3 py-8 text-center text-muted">No results entered for these weeks.</td></tr>
          </tbody>
        </table>
      </div>

      <WinlosePartnerTotals v-if="!partnerId" :partners="report.partners" :from="report.date_from" :to="range.to" :branch-id="branchId" />

      <!-- Consolidated: each platform across all branches -->
      <article v-if="report.consolidated && report.platforms.length" class="overflow-x-auto rounded-xl border border-border bg-white">
        <h2 class="border-b border-border px-4 py-3 text-sm font-semibold text-navy">By platform, all branches</h2>
        <table class="w-full min-w-[560px] text-left text-xs">
          <tbody class="divide-y divide-border">
            <template v-for="p in report.platforms" :key="p.platform_id">
              <tr v-for="(t, i) in p.totals" :key="t.currency" class="text-navy">
                <td class="px-3 py-2 font-mono font-semibold">{{ i === 0 ? p.platform_code : '' }}</td>
                <td class="px-3 py-2">{{ t.currency }}</td>
                <td class="px-3 py-2 text-right tabular-nums" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
                <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(t.jame_ps)">Total {{ formatAmount(t.jame_ps, t.currency) }}</td>
                <td class="px-3 py-2 text-right tabular-nums text-muted">{{ t.entries }} entries</td>
              </tr>
            </template>
          </tbody>
        </table>
      </article>
    </template>
  </section>
</template>

<script setup lang="ts">
import { Download } from 'lucide-vue-next'
import { useWinLoseReport } from '~/composables/useWinLoseReport'
import { todayISO } from '~/utils/inventory'
import { downloadCsv, formatAmount } from '~/utils/winlose'

const today = todayISO()
const range = ref({ from: `${today.slice(0, 8)}01`, to: today }) // this month so far
const { report, loading, error, branchId, hqOptions, consolidated, partnerId, partnerOptions } = useWinLoseReport(() => range.value)

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}

function exportCsv() {
  if (!report.value) return
  downloadCsv(`winlose-branch-summary-${range.value.from}-to-${range.value.to}.csv`,
    ['From', 'To', 'Branch code', 'Branch', 'Currency', 'Win/Lose', 'Commission', 'Total', 'Entries'],
    report.value.hqs.flatMap(hq => hq.totals.map(t => [report.value!.date_from, range.value.to, hq.branch_code, hq.branch_name, t.currency,
      Number(t.win_lose), Number(t.commission), Number(t.jame_ps), t.entries])))
}
</script>
