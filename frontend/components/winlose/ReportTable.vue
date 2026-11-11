<template>
  <div class="space-y-3">
    <!-- One table per currency (USD first): its agents A to Z and its total; merged (Group summary):
         one table, the USD agents, then the KHR ones, and a Total for each currency -->
    <div v-for="part in tables" :key="part.currency || 'all'" class="overflow-x-auto rounded-xl border border-border bg-white">
      <component :is="title ? 'h2' : 'h4'" v-if="title || part.currency" class="flex items-center gap-2 border-b border-border px-4 font-semibold text-navy" :class="title ? 'py-3 text-sm' : 'py-2 text-xs'">
        <Layers v-if="title" :size="15" />{{ [title, part.currency].filter(Boolean).join(' · ') }}
        <span v-if="hint && part === tables[0]" class="ml-auto text-[11px] font-normal text-muted" data-capture-skip>{{ hint }}</span>
      </component>
      <table class="report-table">
        <thead class="bg-canvas uppercase text-slate-500">
          <tr>
            <!-- "AG" in the picture of the card (EntryImage) -->
            <th class="font-bold"><span data-capture-skip>Agent</span><span class="hidden" data-capture-show>AG</span></th>
            <th class="amount font-bold">W/L</th>
            <th class="amount font-bold">COMM</th>
            <!-- One group: the rate each agent's Total is at (on phones it is next to the agent) -->
            <th v-if="showRates" class="amount hidden font-bold sm:table-cell">Rate</th>
            <th class="amount font-bold">Total</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="line in part.rows" :key="`${line.platform_id}-${line.currency}`" class="text-navy">
            <td class="name">
              <!-- On phones the rate is next to the agent, on the same line -->
              <span class="whitespace-nowrap">
                <button v-if="selectable" type="button" class="code font-mono font-semibold text-primary hover:underline" @click="emit('select', line.platform_id)">{{ line.platform_code }}</button>
                <span v-else class="code font-mono font-semibold">{{ line.platform_code }}</span>
                <span v-if="showRates" class="ml-1 text-muted sm:hidden">{{ usedRate(line.rate_min, line.rate_max, line.formula) }}</span>
              </span>
              <!-- The agent's result in an entry: its ✅ / ❌ (left out of the picture) -->
              <div v-if="reactions && line.result_id" class="mt-1" data-capture-skip><WinloseEntryReactions :url="`/winlose/results/${line.result_id}/reactions`" :reactions="line.reactions || []" /></div>
            </td>
            <td class="amount" :class="tone(line.win_lose)">{{ formatAmount(line.win_lose, line.currency) }}</td>
            <td class="amount">{{ formatAmount(line.commission, line.currency) }}</td>
            <td v-if="showRates" class="amount hidden sm:table-cell">{{ usedRate(line.rate_min, line.rate_max, line.formula) }}</td>
            <td class="amount font-bold" :class="tone(line.jame_ps)">{{ formatAmount(line.jame_ps, line.currency) }}</td>
          </tr>
          <!-- The currency's total (merged: one for each currency, "Total USD", "Total KHR") -->
          <tr v-for="t in part.currencies.map(c => totalsIn(report.totals, c))" :key="`total-${t.currency}`" class="bg-canvas/60 font-semibold text-navy">
            <td class="name">{{ part.currencies.length > 1 ? `Total ${t.currency}` : 'Total' }}</td>
            <td class="amount" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
            <td class="amount">{{ formatAmount(t.commission, t.currency) }}</td>
            <td v-if="showRates" class="hidden sm:table-cell" />
            <td class="amount font-bold" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p v-if="!tables.length" class="rounded-xl border border-border bg-white px-4 py-8 text-center text-sm text-muted">No results entered for these weeks.</p>
  </div>
</template>

<script setup lang="ts">
import { Layers } from 'lucide-vue-next'
import type { WinLoseReport } from '~/types/api'
import { currencyTables, formatAmount, mergedTables, totalsIn, usedRate } from '~/utils/winlose'

/**
 * A win/lose report's lines by agent, one table per currency (USD first, each A to Z) with its
 * total (`jame_ps` is the Total); merged (Group summary), one table: the USD agents, then the KHR
 * ones, ending with each currency's total. For one group (byGroup) each agent also shows the rate its Total
 * is at (an agent at 60%: 40%). Selectable: an agent's code opens its detail (`select` with its id).
 * With reactions, an agent's line that is its result in an entry (result_id) has its ✅ / ❌ under
 * it. Title: each table is headed "<title> · USD" (else just the currency; a merged table of both,
 * just the title); hint goes beside the first one.
 */
const props = withDefaults(defineProps<{ report: Pick<WinLoseReport, 'lines' | 'totals'>; byGroup?: boolean; selectable?: boolean; reactions?: boolean; title?: string; hint?: string; merged?: boolean }>(), { byGroup: false, selectable: false, reactions: false, title: '', hint: '', merged: false })
// Rates need View Rates in Reports (the API leaves them out without it).
const { can } = usePermissions()
const showRates = computed(() => props.byGroup && can('winlose-reports.rates'))
const tables = computed(() => mergedTables(currencyTables(props.report.lines, l => l.currency), props.merged))
const emit = defineEmits<{ select: [platformId: string] }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
