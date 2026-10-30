<template>
  <article v-if="groups.length" class="overflow-x-auto rounded-xl border border-border bg-white">
    <h2 class="flex items-center gap-2 border-b border-border px-4 py-3 text-sm font-semibold text-navy"><Network :size="15" /> By group</h2>
    <table class="w-full min-w-[560px] text-left text-xs">
      <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
        <tr>
          <th class="px-3 py-2 font-bold">Group</th>
          <th class="px-3 py-2 font-bold">Currency</th>
          <th class="px-3 py-2 text-right font-bold">Win/Lose</th>
          <th class="px-3 py-2 text-right font-bold">Commission</th>
          <th class="px-3 py-2 text-right font-bold">Total</th>
          <th class="px-3 py-2 text-right font-bold">Entries</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        <template v-for="g in groups" :key="g.group_id || 'none'">
          <tr v-for="(t, i) in g.totals" :key="t.currency" class="text-navy">
            <td class="px-3 py-2 font-semibold" :class="g.group_id ? '' : 'font-normal italic text-muted'">{{ i === 0 ? (g.title || 'No group') : '' }}</td>
            <td class="px-3 py-2">{{ t.currency }}</td>
            <td class="px-3 py-2 text-right tabular-nums" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
            <td class="px-3 py-2 text-right tabular-nums">{{ formatAmount(t.commission, t.currency) }}</td>
            <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
            <td class="px-3 py-2 text-right tabular-nums text-muted">{{ t.entries }}</td>
          </tr>
        </template>
      </tbody>
    </table>
  </article>
</template>

<script setup lang="ts">
import { Network } from 'lucide-vue-next'
import type { WinLoseReport } from '~/types/api'
import { formatAmount } from '~/utils/winlose'

/** A report's totals for each titled group (Group Agent Assignments), per currency. */
defineProps<{ groups: WinLoseReport['groups'] }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
