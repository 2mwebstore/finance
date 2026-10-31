<template>
  <article v-if="groups.length" class="overflow-x-auto rounded-xl border border-border bg-white">
    <h2 class="flex items-center gap-2 border-b border-border px-4 py-3 text-sm font-semibold text-navy"><Network :size="15" /> By group<span v-if="selectable" class="ml-auto text-[11px] font-normal text-muted">Tap a group for its agents</span></h2>
    <table class="report-table">
      <thead class="bg-canvas uppercase text-slate-500">
        <tr>
          <th class="font-bold">Group</th>
          <!-- On phones the currency goes under the group -->
          <th class="hidden font-bold sm:table-cell">Currency</th>
          <th class="amount font-bold">Win/Lose</th>
          <th class="amount font-bold">Comm</th>
          <th class="amount font-bold">Total</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        <template v-for="g in groups" :key="g.group_id || 'none'">
          <tr v-for="(t, i) in g.totals" :key="t.currency" class="text-navy">
            <td class="name font-semibold" :class="g.group_id ? '' : 'font-normal italic text-muted'">
              <button v-if="selectable && g.group_id && i === 0" type="button" class="text-left font-semibold text-primary hover:underline" @click="emit('select', g.group_id)">{{ g.title }}</button>
              <template v-else>{{ i === 0 ? (g.title || 'No group') : '' }}</template>
              <span class="block font-normal not-italic text-muted sm:hidden">{{ t.currency }}</span>
            </td>
            <td class="hidden sm:table-cell">{{ t.currency }}</td>
            <td class="amount" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
            <td class="amount">{{ formatAmount(t.commission, t.currency) }}</td>
            <td class="amount font-bold" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
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

/** A report's totals for each titled group (Group Agent Assignments), per currency. Selectable: a
 * group's title opens its agents (`select` with its id). */
withDefaults(defineProps<{ groups: WinLoseReport['groups']; selectable?: boolean }>(), { selectable: false })
const emit = defineEmits<{ select: [groupId: string] }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
