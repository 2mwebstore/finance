<template>
  <!-- One table: a row per group (A to Z, agents in no group last) with its USD and KHR amounts in
       the same row ("$1,000.00 / -៛500,000"; on phones one under the other) -->
  <article v-if="groups.length" class="overflow-x-auto rounded-xl border border-border bg-white">
    <h2 class="flex items-center gap-2 border-b border-border px-4 py-3 text-sm font-semibold text-navy"><Network :size="15" /> By group<template v-if="currencies.length === 1"> · {{ currencies[0] }}</template><span v-if="selectable" class="ml-auto text-[11px] font-normal text-muted">Tap a group for its agents</span></h2>
    <table class="report-table">
      <thead class="bg-canvas uppercase text-slate-500">
        <tr>
          <th class="font-bold">Group</th>
          <th class="amount font-bold">W/L</th>
          <th class="amount font-bold">COMM</th>
          <!-- One agent: the rate its Total is at in each group (on phones next to the group) -->
          <th v-if="showRates" class="amount hidden font-bold sm:table-cell">Rate</th>
          <th class="amount font-bold">Total</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        <tr v-for="g in groups" :key="g.group_id || 'none'" class="text-navy">
          <td class="name font-semibold" :class="g.group_id ? '' : 'font-normal italic text-muted'">
            <button v-if="selectable && g.group_id" type="button" class="text-left font-semibold text-primary hover:underline" @click="emit('select', g.group_id)">{{ g.title }}</button>
            <template v-else>{{ g.title || 'No group' }}</template>
            <span v-if="showRates && g.group_id" class="ml-1 whitespace-nowrap font-normal not-italic text-muted sm:hidden">{{ usedRate(g.rate_min, g.rate_max, g.formula) }}</span>
          </td>
          <td class="amount">
            <template v-for="(t, i) in totalsOf(g)" :key="t.currency"><span v-if="i" class="hidden text-muted sm:inline"> / </span><span class="block sm:inline" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</span></template>
          </td>
          <td class="amount">
            <template v-for="(t, i) in totalsOf(g)" :key="t.currency"><span v-if="i" class="hidden text-muted sm:inline"> / </span><span class="block sm:inline">{{ formatAmount(t.commission, t.currency) }}</span></template>
          </td>
          <td v-if="showRates" class="amount hidden sm:table-cell">{{ g.group_id ? usedRate(g.rate_min, g.rate_max, g.formula) : '—' }}</td>
          <td class="amount font-bold">
            <template v-for="(t, i) in totalsOf(g)" :key="t.currency"><span v-if="i" class="hidden font-normal text-muted sm:inline"> / </span><span class="block sm:inline" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</span></template>
          </td>
        </tr>
      </tbody>
    </table>
  </article>
</template>

<script setup lang="ts">
import { Network } from 'lucide-vue-next'
import type { WinLoseReport } from '~/types/api'
import { byCurrency, formatAmount, usedRate } from '~/utils/winlose'

/** A report's totals for each titled group (Group Agent Assignments) in one table: a row per group
 * with its amounts in each of its agents' currencies, USD first ("$1,000.00 / -៛500,000"; USD and
 * KHR are never added together). Selectable: a group's title opens its agents (`select` with its
 * id). Detail (one agent's report): the rate its Total is at in each group (an agent at 60%: 40%).
 * No total row: an agent in two groups counts in both (the Totals above are the report's). */
const props = withDefaults(defineProps<{ groups: WinLoseReport['groups']; selectable?: boolean; detail?: boolean }>(), { selectable: false, detail: false })
// Rates need View Rates in Reports (the API leaves them out without it).
const { can } = usePermissions()
const showRates = computed(() => props.detail && can('winlose-reports.rates'))
type Group = WinLoseReport['groups'][number]
const totalsOf = (g: Group) => [...g.totals].sort((a, b) => byCurrency(a.currency, b.currency))
// Only one currency in the whole table (e.g. one KHR agent's groups): the heading says which.
const currencies = computed(() => [...new Set(props.groups.flatMap(g => g.totals.map(t => t.currency)))].sort(byCurrency))
const emit = defineEmits<{ select: [groupId: string] }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
