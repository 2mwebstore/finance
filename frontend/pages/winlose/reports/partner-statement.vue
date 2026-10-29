<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS · REPORTS</p>
        <h1 class="text-xl font-bold text-navy">Partner statement</h1>
        <p class="mt-1 text-xs text-muted">One partner’s weeks across their agents: the agent’s Win/Lose and Commission, and their share of the Total at their %. USD and KHR are kept apart.</p>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy disabled:opacity-50" :disabled="!report?.lines.length || !partnerId" @click="exportCsv"><Download :size="16" /> Export CSV</button>
    </header>

    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
      <SearchSelect v-model="partnerId" :options="partnerOptions" placeholder="Choose a partner" aria-label="Partner" />
      <DateRangePicker v-model="range" />
    </div>

    <p v-if="!partnerId" class="rounded-xl border border-border bg-white p-6 text-center text-sm text-muted">Choose a partner to see their statement.</p>
    <template v-else>
      <article v-if="partner" class="flex flex-wrap items-center gap-3 rounded-xl border border-border bg-white p-4">
        <span class="grid h-10 w-10 place-items-center rounded-xl bg-blue-50 text-primary"><Handshake :size="18" /></span>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-bold text-navy">{{ partner.name }}<span v-if="!partner.is_active" class="ml-2 rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span></p>
          <p class="text-xs text-muted">{{ [partner.phone, partner.note].filter(Boolean).join(' · ') || 'No phone or note' }} · default {{ formatRate(partner.default_rate) }}</p>
        </div>
        <p class="text-xs text-muted">Weeks from {{ formatDocDate(report?.date_from || range.from) }} to {{ formatDocDate(range.to) }}</p>
      </article>

      <div v-if="loading && !report" class="text-sm text-muted">Loading…</div>
      <AppAlert v-else-if="error" title="Couldn’t load the statement" :message="error" />
      <template v-else-if="report">
        <WinloseTotals v-if="report.totals.length" :totals="report.totals" title="Statement" />

        <div class="overflow-x-auto rounded-xl border border-border bg-white">
          <table class="w-full min-w-[640px] text-left text-xs">
            <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
              <tr>
                <th class="px-3 py-2 font-bold">Agent</th>
                <th class="px-3 py-2 font-bold">Partner %</th>
                <th class="px-3 py-2 text-right font-bold">Win/Lose</th>
                <th class="px-3 py-2 text-right font-bold">Commission</th>
                <th class="px-3 py-2 text-right font-bold">Their share</th>
              </tr>
            </thead>
            <tbody v-for="week in weeks" :key="week.start" class="divide-y divide-border border-t border-border first:border-t-0">
              <tr class="bg-slate-50/70"><td colspan="5" class="px-3 py-2 text-xs font-bold text-navy">Week of {{ formatDocDate(week.start) }} – {{ formatDocDate(plusDays(week.start, 6)) }}</td></tr>
              <tr v-for="(l, i) in week.lines" :key="i" class="text-navy">
                <td class="px-3 py-2"><span class="font-mono font-semibold">{{ l.platform_code }}</span> <span class="text-muted">· {{ l.currency }}</span></td>
                <td class="px-3 py-2 text-muted">{{ rateRange(l.rate_min, l.rate_max) }}</td>
                <td class="px-3 py-2 text-right tabular-nums" :class="tone(l.win_lose)">{{ formatAmount(l.win_lose, l.currency) }}</td>
                <td class="px-3 py-2 text-right tabular-nums">{{ formatAmount(l.commission, l.currency) }}</td>
                <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(l.jame_ps)">{{ formatAmount(l.jame_ps, l.currency) }}</td>
              </tr>
              <!-- The week's subtotal in each currency -->
              <tr v-for="t in week.totals" :key="`total-${t.currency}`" class="bg-canvas/60 font-semibold text-navy">
                <td class="px-3 py-2" colspan="2">Week total · {{ t.currency }}</td>
                <td class="px-3 py-2 text-right tabular-nums" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
                <td class="px-3 py-2 text-right tabular-nums">{{ formatAmount(t.commission, t.currency) }}</td>
                <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
              </tr>
            </tbody>
            <tbody v-if="!report.lines.length"><tr><td colspan="5" class="px-3 py-8 text-center text-muted">No weeks entered for this partner in these dates.</td></tr></tbody>
          </table>
        </div>
      </template>
    </template>
  </section>
</template>

<script setup lang="ts">
import { Download, Handshake } from 'lucide-vue-next'
import type { WinLoseReportLine } from '~/types/api'
import { useWinLoseReport } from '~/composables/useWinLoseReport'
import { formatDocDate, todayISO } from '~/utils/inventory'
import { downloadCsv, formatAmount, formatRate, plusDays, rateRange } from '~/utils/winlose'

const route = useRoute()
const today = todayISO()
const range = ref({ from: String(route.query.from || `${today.slice(0, 8)}01`), to: String(route.query.to || today) }) // this month so far
const { report, loading, error, partners, partnerId, partnerOptions } = useWinLoseReport(() => range.value, { byWeek: true, needPartner: true })
const partner = computed(() => partners.value.find(p => p.id === partnerId.value))

// The lines by week, newest first, each week with its subtotal per currency.
const weeks = computed(() => {
  const map = new Map<string, WinLoseReportLine[]>()
  for (const l of report.value?.lines || []) {
    const start = l.week_start || ''
    map.set(start, [...(map.get(start) || []), l])
  }
  return [...map.entries()].sort(([a], [b]) => b.localeCompare(a)).map(([start, lines]) => {
    const totals = new Map<string, { currency: string; win_lose: number; commission: number; jame_ps: number }>()
    for (const l of lines) {
      const t = totals.get(l.currency) || { currency: l.currency, win_lose: 0, commission: 0, jame_ps: 0 }
      t.win_lose += Number(l.win_lose)
      t.commission += Number(l.commission)
      t.jame_ps += Number(l.jame_ps)
      totals.set(l.currency, t)
    }
    return { start, lines, totals: [...totals.values()].sort((a, b) => a.currency.localeCompare(b.currency)) }
  })
})

function tone(value: string | number) {
  return Number(value) < 0 ? 'text-expense' : ''
}

function exportCsv() {
  if (!report.value || !partner.value) return
  downloadCsv(`partner-statement-${partner.value.name.replace(/[^\w-]+/g, '_')}-${range.value.from}-to-${range.value.to}.csv`,
    ['Partner', 'Week start', 'Week end', 'Agent', 'Currency', 'Partner %', 'Win/Lose', 'Commission', 'Share'],
    report.value.lines.map(l => [partner.value!.name, l.week_start || '', l.week_start ? plusDays(l.week_start, 6) : '', l.platform_code, l.currency,
      l.rate_min === l.rate_max ? Number(l.rate_min) : `${Number(l.rate_min)}-${Number(l.rate_max)}`, Number(l.win_lose), Number(l.commission), Number(l.jame_ps)]))
}
</script>
