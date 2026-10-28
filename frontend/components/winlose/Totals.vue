<template>
  <div class="grid grid-cols-1 gap-3" :class="totals.length > 1 ? 'md:grid-cols-2' : ''">
    <article v-for="t in totals" :key="t.currency" class="rounded-xl border border-border bg-white p-4">
      <p class="text-xs font-bold text-slate-500">{{ title }} · <span class="text-navy">{{ t.currency }}</span><span class="font-normal text-slate-400"> · {{ t.entries }} {{ t.entries === 1 ? 'entry' : 'entries' }}</span></p>
      <div class="mt-2 grid grid-cols-3 gap-3">
        <div><p class="text-[11px] text-muted">Win/Lose</p><p class="text-sm font-bold tabular-nums" :class="tone(t.win_lose)">{{ formatAmount(t.win_lose, t.currency) }}</p></div>
        <div><p class="text-[11px] text-muted">Commission</p><p class="text-sm font-bold tabular-nums text-navy">{{ formatAmount(t.commission, t.currency) }}</p></div>
        <div><p class="text-[11px] text-muted">Total</p><p class="text-base font-bold tabular-nums" :class="tone(t.jame_ps)">{{ formatAmount(t.jame_ps, t.currency) }}</p></div>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { formatAmount } from '~/utils/winlose'

/** Win/lose totals, one card per currency (USD and KHR are never added together). `jame_ps` is the Total. */
withDefaults(defineProps<{
  totals: { currency: string; win_lose: string | number; commission: string | number; jame_ps: string | number; entries: number }[]
  title?: string
}>(), { title: 'Summary' })

function tone(value: string | number) {
  return Number(value) < 0 ? 'text-expense' : 'text-navy'
}
</script>
