<template>
  <div class="overflow-x-auto rounded-xl border border-border bg-white">
    <table class="report-table">
      <thead class="bg-canvas uppercase text-slate-500">
        <tr>
          <th class="font-bold">Agent</th>
          <th class="amount font-bold">Win/Lose</th>
          <th class="amount font-bold">Comm</th>
          <!-- One group: each agent's rate in it (on phones it is in the line under the agent) -->
          <th v-if="byGroup" class="amount hidden font-bold sm:table-cell">Rate</th>
          <th class="amount font-bold">Total</th>
        </tr>
      </thead>
      <tbody v-if="report.lines.length" class="divide-y divide-border">
        <template v-for="line in report.lines" :key="`${line.platform_id}-${line.currency}`">
          <tr class="text-navy">
            <td class="name"><span class="code font-mono font-semibold">{{ line.platform_code }}</span> <span class="block text-muted sm:inline"><span class="hidden sm:inline">· </span>{{ line.currency }}</span></td>
            <td class="amount" :class="tone(line.win_lose)">{{ formatAmount(line.win_lose, line.currency) }}</td>
            <td class="amount">{{ formatAmount(line.commission, line.currency) }}</td>
            <td v-if="byGroup" class="amount hidden sm:table-cell">{{ rateText(line) }}</td>
            <td class="amount font-bold" :class="tone(line.jame_ps)">{{ formatAmount(line.jame_ps, line.currency) }}</td>
          </tr>
          <!-- How the Total is worked out, e.g. (1,000 − 100) × 10% = 90 -->
          <tr v-if="byGroup" class="calc text-muted">
            <td :colspan="5">{{ workedOut(line) }}</td>
          </tr>
        </template>
        <!-- The total in each currency -->
        <tr v-for="t in report.totals" :key="`total-${t.currency}`" class="bg-canvas/60 font-semibold text-navy">
          <td class="name">Total <span class="block text-muted sm:inline"><span class="hidden sm:inline">· </span>{{ t.currency }}</span></td>
          <td class="amount" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
          <td class="amount">{{ formatAmount(t.commission, t.currency) }}</td>
          <td v-if="byGroup" class="hidden sm:table-cell" />
          <td class="amount font-bold" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
        </tr>
      </tbody>
      <tbody v-else><tr><td :colspan="byGroup ? 5 : 4" class="empty text-muted">No results entered for these weeks.</td></tr></tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import type { WinLoseReport, WinLoseReportLine } from '~/types/api'
import { formatAmount, formatRate } from '~/utils/winlose'

/**
 * A win/lose report's lines by agent, with the total per currency (`jame_ps` is the Total). For one
 * group (byGroup) each agent also shows its rate in the group and how its Total is worked out.
 */
withDefaults(defineProps<{ report: WinLoseReport; byGroup?: boolean }>(), { byGroup: false })

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}

function rateText(line: WinLoseReportLine) {
  return Number(line.rate_min) === Number(line.rate_max) ? formatRate(line.rate_min) : `${formatRate(line.rate_min)}–${formatRate(line.rate_max)}`
}

// v2 takes the commission away, v1 added it. A negative amount after the sign is in brackets.
function workedOut(line: WinLoseReportLine) {
  const sign = line.formula === 'v2' ? '−' : line.formula === 'v1' ? '+' : ''
  if (!sign || Number(line.rate_min) !== Number(line.rate_max)) return `Rate ${rateText(line)}`
  const commission = formatAmount(line.commission, line.currency)
  const after = Number(line.commission) < 0 ? `(${commission})` : commission
  return `(${formatAmount(line.win_lose, line.currency)} ${sign} ${after}) × ${rateText(line)} = ${formatAmount(line.jame_ps, line.currency)}`
}
</script>
