<template>
  <article v-if="entries.length" class="overflow-x-auto rounded-xl border border-border bg-white">
    <h2 class="flex items-center gap-2 border-b border-border px-4 py-3 text-sm font-semibold text-navy"><FileText :size="15" /> {{ title }}<span v-if="selectable" class="ml-auto text-[11px] font-normal text-muted">Tap one for its detail</span></h2>
    <table class="report-table">
      <thead class="bg-canvas uppercase text-slate-500">
        <tr>
          <th class="font-bold">Entry</th>
          <!-- On phones the currency goes under the entry -->
          <th class="hidden font-bold sm:table-cell">Currency</th>
          <th class="amount font-bold">W/L</th>
          <th class="amount font-bold">COMM</th>
          <th class="amount font-bold">Total</th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        <template v-for="e in entries" :key="e.id || 'none'">
          <tr v-for="(t, i) in e.totals" :key="t.currency" class="text-navy">
            <td class="name">
              <template v-if="i === 0">
                <button v-if="e.id && selectable" type="button" class="text-left font-semibold text-primary hover:underline" @click="emit('select', e)">{{ e.title }}</button>
                <NuxtLink v-else-if="e.id" :to="`/winlose/entries/${e.id}`" class="font-semibold text-primary hover:underline">{{ e.title }}</NuxtLink>
                <span v-else class="font-semibold">{{ e.title }}</span>
                <span v-if="e.created_at" class="block text-muted">{{ formatDocDate(e.created_at) }}<template v-if="e.created_by"> · {{ e.created_by }}</template></span>
                <WinloseEntryReactions v-if="e.id" class="mt-1" :url="`/winlose/entries/${e.id}/reactions`" :reactions="e.reactions || []" />
              </template>
              <span class="block text-muted sm:hidden">{{ t.currency }}</span>
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
import { FileText } from 'lucide-vue-next'
import type { WinLoseReportEntry } from '~/types/api'
import { formatDocDate } from '~/utils/inventory'
import { formatAmount } from '~/utils/winlose'

/**
 * A report's totals for each data entry (by_entry), per currency, with its ✅ / ❌. An entry's title
 * opens its page or, selectable, its detail in the report (`select` with the entry).
 */
withDefaults(defineProps<{ entries: WinLoseReportEntry[]; title?: string; selectable?: boolean }>(), { title: 'By data entry', selectable: false })
const emit = defineEmits<{ select: [entry: WinLoseReportEntry] }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
