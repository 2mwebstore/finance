<template>
  <div class="overflow-hidden rounded-xl border border-border">
    <div v-for="line in lines" :key="line.product_id + line.quantity + line.line_total" class="flex items-start justify-between gap-3 border-b border-border px-3 py-2.5 last:border-b-0">
      <div class="min-w-0">
        <p class="truncate text-sm font-semibold text-navy">{{ line.name }}</p>
        <p class="text-[11px] text-muted"><span class="font-mono">{{ line.sku }}</span> · {{ formatQty(line.quantity) }} {{ line.unit }}<template v-if="currency && priceField"> × {{ formatMoney(line[priceField], currency) }}</template><template v-if="currency && Number(line.discount)"> − {{ formatMoney(line.discount, currency) }}</template></p>
        <p v-if="line.note" class="mt-0.5 text-[11px] italic text-slate-500">{{ line.note }}</p>
        <p v-if="currency && priceField === 'unit_cost' && line.landed_cost && (currency !== 'USD' || Number(line.landed_cost) !== Number(line.unit_cost))" class="text-[11px] text-slate-400">Into stock at {{ formatMoney(line.landed_cost, 'USD') }} each (in USD, with shipping and discount)</p>
      </div>
      <p v-if="currency" class="shrink-0 text-sm font-semibold text-navy">{{ formatMoney(line.line_total, currency) }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Currency, DocLine } from '~/types/api'
import { formatMoney } from '~/utils/money'
import { formatQty } from '~/utils/inventory'

/** The lines of a saved sale, purchase or transfer (transfers have no currency: quantities only). */
defineProps<{ lines: DocLine[]; currency?: Currency; priceField?: 'unit_price' | 'unit_cost' }>()
</script>
