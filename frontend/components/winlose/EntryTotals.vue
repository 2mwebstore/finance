<template>
  <div v-if="entries.length" class="space-y-3">
    <!-- One table per currency (USD first): each entry's totals in it -->
    <article v-for="part in tables" :key="part.currency" class="overflow-x-auto rounded-xl border border-border bg-white">
      <h2 class="flex items-center gap-2 border-b border-border px-4 py-3 text-sm font-semibold text-navy"><FileText :size="15" /> {{ title }} · {{ part.currency }}<span v-if="selectable && part === tables[0]" class="ml-auto text-[11px] font-normal text-muted">Tap one for its detail</span></h2>
      <table class="report-table">
        <thead class="bg-canvas uppercase text-slate-500">
          <tr>
            <th class="font-bold">Entry</th>
            <th class="amount font-bold">W/L</th>
            <th class="amount font-bold">COMM</th>
            <th class="amount font-bold">Total</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="{ item: e, t } in part.rows" :key="e.id || 'none'" class="text-navy">
            <td class="name">
              <button v-if="e.id && selectable" type="button" class="text-left font-semibold text-primary hover:underline" @click="emit('select', e)">{{ e.title }}</button>
              <NuxtLink v-else-if="e.id" :to="`/winlose/entries/${e.id}`" class="font-semibold text-primary hover:underline">{{ e.title }}</NuxtLink>
              <span v-else class="font-semibold">{{ e.title }}</span>
              <span v-if="e.created_at" class="block text-muted">{{ formatDocDate(e.created_at) }}<template v-if="e.created_by"> · {{ e.created_by }}</template></span>
              <!-- ✅ / ❌ once per entry: in its first currency's table -->
              <WinloseEntryReactions v-if="e.id && reactions && t.currency === e.totals[0]?.currency" class="mt-1" :url="`/winlose/entries/${e.id}/reactions`" :reactions="e.reactions || []" />
            </td>
            <td class="amount" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
            <td class="amount">{{ formatAmount(t.commission, t.currency) }}</td>
            <td class="amount font-bold" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
          </tr>
        </tbody>
      </table>
    </article>
  </div>
</template>

<script setup lang="ts">
import { FileText } from 'lucide-vue-next'
import type { WinLoseReportEntry } from '~/types/api'
import { formatDocDate } from '~/utils/inventory'
import { currencyTables, formatAmount, totalsRows } from '~/utils/winlose'

/**
 * A report's totals for each data entry (by_entry), one table per currency (USD first), with its
 * ✅ / ❌ when reactions is set (Group summary; in the entry's first currency's table). An entry's
 * title opens its page or, selectable, its detail in the report (`select` with the entry).
 */
const props = withDefaults(defineProps<{ entries: WinLoseReportEntry[]; title?: string; selectable?: boolean; reactions?: boolean }>(), { title: 'By data entry', selectable: false, reactions: false })
const tables = computed(() => currencyTables(totalsRows(props.entries), r => r.t.currency))
const emit = defineEmits<{ select: [entry: WinLoseReportEntry] }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
