<template>
  <AppModal :model-value="open" title="Held sales" eyebrow="POS" subtitle="Sales parked on this till to finish later." width="560px" @update:model-value="(value) => { if (!value) emit('close') }">
    <p v-if="!list.length" class="py-8 text-center text-sm text-muted">No held sales.</p>
    <ul v-else class="space-y-2">
      <li v-for="held in list" :key="held.id" class="flex items-center gap-3 rounded-xl border border-border p-3">
        <div class="min-w-0 flex-1">
          <p class="text-sm font-semibold text-navy">{{ itemsLabel(held) }} · {{ formatMoney(held.total, held.currency) }}</p>
          <p class="truncate text-xs text-muted">Held at {{ timeOf(held.held_at) }}<template v-if="held.customer"> · {{ held.customer }}</template></p>
        </div>
        <button type="button" class="rounded-lg bg-primary px-3.5 py-1.5 text-sm font-semibold text-white" @click="emit('resume', held.id)">Resume</button>
        <button type="button" class="grid h-8 w-8 place-items-center rounded-lg text-muted hover:bg-rose-50 hover:text-expense" :aria-label="`Throw away the sale held at ${timeOf(held.held_at)}`" @click="discard(held)"><Trash2 :size="15" /></button>
      </li>
    </ul>
  </AppModal>
</template>

<script setup lang="ts">
import { Trash2 } from 'lucide-vue-next'
import type { HeldSale } from '~/composables/usePosCart'
import { formatMoney } from '~/utils/money'
import { formatQty } from '~/utils/inventory'

/** The till's held sales: resume one into the cart, or throw it away. */
defineProps<{ open: boolean; list: HeldSale[] }>()
const emit = defineEmits<{ close: []; resume: [id: number]; discard: [id: number] }>()

function itemsLabel(held: HeldSale) {
  const count = held.lines.reduce((sum, l) => sum + (Number(l.quantity) || 0), 0)
  return `${formatQty(count)} ${count === 1 ? 'item' : 'items'}`
}

function timeOf(iso: string) {
  return new Date(iso).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
}

function discard(held: HeldSale) {
  if (window.confirm(`Throw away the held sale (${itemsLabel(held)}, ${formatMoney(held.total, held.currency)})?`)) emit('discard', held.id)
}
</script>
