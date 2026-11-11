<template>
  <div v-if="groups.length" class="space-y-3">
    <!-- One table per currency (USD first): the groups A to Z, agents in no group last -->
    <article v-for="part in tables" :key="part.currency" class="overflow-x-auto rounded-xl border border-border bg-white">
      <h2 class="flex items-center gap-2 border-b border-border px-4 py-3 text-sm font-semibold text-navy"><Network :size="15" /> By group · {{ part.currency }}<span v-if="selectable && part === tables[0]" class="ml-auto text-[11px] font-normal text-muted">Tap a group for its agents</span></h2>
      <table class="report-table">
        <thead class="bg-canvas uppercase text-slate-500">
          <tr>
            <th class="font-bold">Group</th>
            <th class="amount font-bold">W/L</th>
            <th class="amount font-bold">COMM</th>
            <!-- One agent: the rate its Total is at in each group (on phones in the line under the group) -->
            <th v-if="showRates" class="amount hidden font-bold sm:table-cell">Rate</th>
            <th class="amount font-bold">Total</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="{ item: g, t } in part.rows" :key="g.group_id || 'none'" class="text-navy">
            <td class="name font-semibold" :class="g.group_id ? '' : 'font-normal italic text-muted'">
              <button v-if="selectable && g.group_id" type="button" class="text-left font-semibold text-primary hover:underline" @click="emit('select', g.group_id)">{{ g.title }}</button>
              <template v-else>{{ g.title || 'No group' }}</template>
              <span v-if="showRates && g.group_id" class="block font-normal not-italic text-muted sm:hidden">{{ usedRate(g.rate_min, g.rate_max, g.formula) }}</span>
            </td>
            <td class="amount" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
            <td class="amount">{{ formatAmount(t.commission, t.currency) }}</td>
            <td v-if="showRates" class="amount hidden sm:table-cell">{{ g.group_id ? usedRate(g.rate_min, g.rate_max, g.formula) : '—' }}</td>
            <td class="amount font-bold" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
          </tr>
        </tbody>
      </table>
    </article>
  </div>
</template>

<script setup lang="ts">
import { Network } from 'lucide-vue-next'
import type { WinLoseReport } from '~/types/api'
import { currencyTables, formatAmount, totalsRows, usedRate } from '~/utils/winlose'

/** A report's totals for each titled group (Group Agent Assignments), one table per currency (USD
 * first). Selectable: a group's title opens its agents (`select` with its id). Detail (one agent's
 * report): the rate its Total is at in each group (an agent at 60%: 40%). No total row: an agent in
 * two groups counts in both (the Totals above are the report's). */
const props = withDefaults(defineProps<{ groups: WinLoseReport['groups']; selectable?: boolean; detail?: boolean }>(), { selectable: false, detail: false })
// Rates need View Rates in Reports (the API leaves them out without it).
const { can } = usePermissions()
const showRates = computed(() => props.detail && can('winlose-reports.rates'))
const tables = computed(() => currencyTables(totalsRows(props.groups), r => r.t.currency))
const emit = defineEmits<{ select: [groupId: string] }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
