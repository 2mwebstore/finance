<template>
  <div class="overflow-x-auto rounded-xl border border-border bg-white">
    <table class="w-full min-w-[620px] text-left text-xs">
      <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
        <tr>
          <th class="px-3 py-2 font-bold">Agent</th>
          <th class="px-3 py-2 font-bold">Rate</th>
          <th class="px-3 py-2 text-right font-bold">Win/Lose</th>
          <th class="px-3 py-2 text-right font-bold">Commission</th>
          <th class="px-3 py-2 text-right font-bold">Total</th>
          <th v-if="showWeeks" class="px-3 py-2 text-right font-bold">Weeks</th>
        </tr>
      </thead>
      <tbody v-if="report.lines.length" class="divide-y divide-border">
        <tr v-for="line in report.lines" :key="`${line.platform_id}-${line.currency}`" class="text-navy">
          <td class="px-3 py-2"><span class="font-mono font-semibold">{{ line.platform_code }}</span> <span class="text-muted">· {{ line.currency }}</span></td>
          <td class="px-3 py-2 text-muted">{{ rateRange(line.rate_min, line.rate_max) }}</td>
          <td class="px-3 py-2 text-right tabular-nums" :class="tone(line.win_lose)">{{ formatAmount(line.win_lose, line.currency) }}</td>
          <td class="px-3 py-2 text-right tabular-nums">{{ formatAmount(line.commission, line.currency) }}</td>
          <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(line.jame_ps)">{{ formatAmount(line.jame_ps, line.currency) }}</td>
          <td v-if="showWeeks" class="px-3 py-2 text-right tabular-nums text-muted">{{ line.entries }}</td>
        </tr>
        <!-- The total in each currency -->
        <tr v-for="t in report.totals" :key="`total-${t.currency}`" class="bg-canvas/60 font-semibold text-navy">
          <td class="px-3 py-2" colspan="2">Total · {{ t.currency }}</td>
          <td class="px-3 py-2 text-right tabular-nums" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
          <td class="px-3 py-2 text-right tabular-nums">{{ formatAmount(t.commission, t.currency) }}</td>
          <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
          <td v-if="showWeeks" class="px-3 py-2 text-right tabular-nums text-muted">{{ t.entries }}</td>
        </tr>
      </tbody>
      <tbody v-else><tr><td :colspan="showWeeks ? 6 : 5" class="px-3 py-8 text-center text-muted">No results entered for these weeks.</td></tr></tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import type { WinLoseReport } from '~/types/api'
import { formatAmount, rateRange } from '~/utils/winlose'

/** A win/lose report's lines by agent, with the rate used and the total per currency (`jame_ps` is the Total). */
withDefaults(defineProps<{ report: WinLoseReport; showWeeks?: boolean }>(), { showWeeks: false })

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
