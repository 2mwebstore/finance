<template>
  <div class="overflow-x-auto rounded-xl border border-border bg-white">
    <table class="report-table">
      <thead class="bg-canvas uppercase text-slate-500">
        <tr>
          <th class="font-bold">Agent</th>
          <th class="amount font-bold">W/L</th>
          <th class="amount font-bold">COMM</th>
          <!-- One group: the rate each agent's Total is at (on phones it is in the line under the agent) -->
          <th v-if="byGroup" class="amount hidden font-bold sm:table-cell">Rate</th>
          <th class="amount font-bold">Total</th>
        </tr>
      </thead>
      <tbody v-if="report.lines.length" class="divide-y divide-border">
        <template v-for="line in report.lines" :key="`${line.platform_id}-${line.currency}`">
          <tr class="text-navy">
            <td class="name">
              <button v-if="selectable" type="button" class="code font-mono font-semibold text-primary hover:underline" @click="emit('select', line.platform_id)">{{ line.platform_code }}</button>
              <span v-else class="code font-mono font-semibold">{{ line.platform_code }}</span> <span class="block text-muted sm:inline"><span class="hidden sm:inline">· </span>{{ line.currency }}<span v-if="byGroup" class="sm:hidden"> · {{ usedRate(line.rate_min, line.rate_max, line.formula) }}</span></span></td>
            <td class="amount" :class="tone(line.win_lose)">{{ formatAmount(line.win_lose, line.currency) }}</td>
            <td class="amount">{{ formatAmount(line.commission, line.currency) }}</td>
            <td v-if="byGroup" class="amount hidden sm:table-cell">{{ usedRate(line.rate_min, line.rate_max, line.formula) }}</td>
            <td class="amount font-bold" :class="tone(line.jame_ps)">{{ formatAmount(line.jame_ps, line.currency) }}</td>
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
import type { WinLoseReport } from '~/types/api'
import { formatAmount, usedRate } from '~/utils/winlose'

/**
 * A win/lose report's lines by agent, with the total per currency (`jame_ps` is the Total). For one
 * group (byGroup) each agent also shows the rate its Total is at (an agent at 60%: 40%).
 * Selectable: an agent's code opens its detail (`select` with its id).
 */
withDefaults(defineProps<{ report: Pick<WinLoseReport, 'lines' | 'totals'>; byGroup?: boolean; selectable?: boolean }>(), { byGroup: false, selectable: false })
const emit = defineEmits<{ select: [platformId: string] }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
