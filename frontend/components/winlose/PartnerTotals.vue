<template>
  <article v-if="partners.length" class="overflow-x-auto rounded-xl border border-border bg-white">
    <header class="flex flex-wrap items-baseline justify-between gap-2 border-b border-border px-4 py-3">
      <h2 class="flex items-center gap-2 text-sm font-semibold text-navy"><Handshake :size="15" /> By partner</h2>
      <p class="text-[11px] text-muted">Each partner’s share of the Total; an agent shared by several partners counts for each.</p>
    </header>
    <table class="w-full min-w-[560px] text-left text-xs">
      <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
        <tr>
          <th class="px-3 py-2 font-bold">Partner</th>
          <th class="px-3 py-2 font-bold">Currency</th>
          <th class="px-3 py-2 text-right font-bold">Win/Lose</th>
          <th class="px-3 py-2 text-right font-bold">Commission</th>
          <th class="px-3 py-2 text-right font-bold">Share</th>
          <th class="px-3 py-2 text-right font-bold">Entries</th>
          <th class="px-3 py-2"><span class="sr-only">Statement</span></th>
        </tr>
      </thead>
      <tbody class="divide-y divide-border">
        <template v-for="p in partners" :key="p.partner_id || 'none'">
          <tr v-for="(t, i) in p.totals" :key="t.currency" class="text-navy">
            <td class="px-3 py-2 font-semibold" :class="p.partner_id ? '' : 'font-normal italic text-muted'">{{ i === 0 ? (p.partner_name || 'No partner') : '' }}</td>
            <td class="px-3 py-2">{{ t.currency }}</td>
            <td class="px-3 py-2 text-right tabular-nums" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</td>
            <td class="px-3 py-2 text-right tabular-nums">{{ formatAmount(t.commission, t.currency) }}</td>
            <td class="px-3 py-2 text-right font-bold tabular-nums" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</td>
            <td class="px-3 py-2 text-right tabular-nums text-muted">{{ t.entries }}</td>
            <td class="px-3 py-2 text-right">
              <NuxtLink v-if="i === 0 && p.partner_id" :to="{ path: '/winlose/reports/partner-statement', query: { partner: p.partner_id, from, to } }" class="font-semibold text-primary">Statement</NuxtLink>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
  </article>
</template>

<script setup lang="ts">
import { Handshake } from 'lucide-vue-next'
import type { WinLoseReport } from '~/types/api'
import { formatAmount } from '~/utils/winlose'

/** A report's totals for each partner (their share of the Total), per currency, with a link to the partner's statement. */
defineProps<{ partners: WinLoseReport['partners']; from: string; to: string }>()

function tone(value: string) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
