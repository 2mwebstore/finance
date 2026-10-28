<template>
  <AppModal :model-value="!!sale" :title="paid ? 'Payment complete' : sale?.number || ''" eyebrow="RECEIPT" width="420px" @update:model-value="(open) => { if (!open) emit('close') }">
    <div v-if="sale" id="pos-receipt" class="space-y-3 text-sm text-navy">
      <div class="space-y-0.5 text-center">
        <p class="text-base font-bold">{{ sale.branch_name }}</p>
        <p class="text-xs text-muted"><span class="font-mono">{{ sale.number }}</span> · {{ formatDocDateTime(sale.created_at) }}</p>
        <p class="text-xs text-muted">Cashier: {{ sale.created_by }}</p>
        <p v-if="sale.status === 'voided'" class="pt-1 text-xs font-bold text-expense">VOIDED</p>
      </div>

      <div class="space-y-1.5 border-t border-dashed border-border pt-2">
        <div v-for="(line, i) in sale.lines" :key="i">
          <div class="flex justify-between gap-3">
            <span class="min-w-0">{{ formatQty(line.quantity) }} × {{ line.name }}</span>
            <span class="shrink-0 font-mono">{{ formatMoney(line.line_total, sale.currency) }}</span>
          </div>
          <p class="pl-3 text-[11px] text-muted">{{ formatMoney(line.unit_price, sale.currency) }} each<template v-if="Number(line.discount)"> · −{{ formatMoney(line.discount, sale.currency) }}</template></p>
          <p v-if="line.note" class="pl-3 text-[11px] italic text-muted">{{ line.note }}</p>
        </div>
      </div>

      <div class="space-y-1 border-t border-dashed border-border pt-2">
        <div class="flex justify-between"><span class="text-muted">Subtotal</span><span class="font-mono">{{ formatMoney(sale.subtotal, sale.currency) }}</span></div>
        <div v-if="Number(sale.discount)" class="flex justify-between"><span class="text-muted">Discount</span><span class="font-mono">−{{ formatMoney(sale.discount, sale.currency) }}</span></div>
        <div v-if="Number(sale.shipping)" class="flex justify-between"><span class="text-muted">Shipping</span><span class="font-mono">+{{ formatMoney(sale.shipping, sale.currency) }}</span></div>
        <div class="flex justify-between text-base font-bold"><span>Total</span><span class="font-mono">{{ formatMoney(sale.total, sale.currency) }}</span></div>
        <div v-if="otherCurrency" class="flex justify-end text-xs text-muted"><span class="font-mono">≈ {{ formatMoney(convert(sale.total, sale.currency, otherCurrency), otherCurrency) }}</span></div>
      </div>

      <div class="space-y-1 border-t border-dashed border-border pt-2">
        <div class="flex justify-between"><span class="text-muted">Paid by</span><span>{{ sale.payment_method || 'Cash' }}</span></div>
        <template v-if="paid?.cash">
          <div class="flex justify-between"><span class="text-muted">Received</span><span class="font-mono">{{ receivedLabel }}</span></div>
          <div class="flex justify-between"><span class="text-muted">Change</span><span class="font-mono">{{ changeLabel }}</span></div>
        </template>
        <div v-if="paid?.reference" class="flex justify-between gap-3"><span class="text-muted">Reference</span><span class="truncate">{{ paid.reference }}</span></div>
        <div v-if="sale.customer" class="flex justify-between gap-3"><span class="text-muted">Customer</span><span class="truncate">{{ sale.customer }}</span></div>
      </div>

      <p class="pt-1 text-center text-xs text-muted">Thank you!</p>
    </div>
    <template #footer>
      <button type="button" class="inline-flex items-center gap-1.5 border border-border bg-white text-navy" @click="print"><Printer :size="15" /> Print</button>
      <button type="button" class="bg-primary text-white" @click="emit('close')">{{ paid ? 'New sale' : 'Close' }}</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import { Printer } from 'lucide-vue-next'
import type { Sale } from '~/types/api'
import type { PosPayment } from '~/composables/usePosCart'
import { useCurrency } from '~/composables/useCurrency'
import { formatMoney } from '~/utils/money'
import { formatDocDateTime, formatQty } from '~/utils/inventory'

/**
 * A POS sale's receipt. Right after paying it also shows the cash received and the change
 * (`paid`); reopened from today's sales it shows what the sale itself keeps.
 */
const props = defineProps<{ sale: Sale | null; paid?: PosPayment | null }>()
const emit = defineEmits<{ close: [] }>()

const { active, convert } = useCurrency()
const otherCurrency = computed(() => {
  if (!props.sale) return ''
  if (props.sale.currency !== 'USD') return 'USD'
  return active.value.some(c => c.code === 'KHR') ? 'KHR' : ''
})

const receivedLabel = computed(() => {
  const cash = props.paid?.cash
  if (!cash) return ''
  return [cash.received_usd ? formatMoney(cash.received_usd, 'USD') : '', cash.received_khr ? formatMoney(cash.received_khr, 'KHR') : ''].filter(Boolean).join(' + ') || formatMoney(0, props.sale?.currency || 'USD')
})
const changeLabel = computed(() => {
  const cash = props.paid?.cash
  if (!cash) return ''
  const currency = props.sale?.currency || 'USD'
  if (currency !== 'USD' && currency !== 'KHR') return formatMoney(cash.change, currency)
  return [cash.change_usd || !cash.change_khr ? formatMoney(cash.change_usd, 'USD') : '', cash.change_khr ? formatMoney(cash.change_khr, 'KHR') : ''].filter(Boolean).join(' + ')
})

// Only the receipt goes to the printer (see the print styles below).
function print() {
  const root = document.documentElement
  root.classList.add('pos-printing')
  const done = () => { root.classList.remove('pos-printing'); window.removeEventListener('afterprint', done) }
  window.addEventListener('afterprint', done)
  window.print()
}
</script>

<style>
@media print {
  html.pos-printing body * { visibility: hidden !important; }
  html.pos-printing #pos-receipt, html.pos-printing #pos-receipt * { visibility: visible !important; }
  html.pos-printing .app-modal-backdrop, html.pos-printing .app-modal, html.pos-printing .app-modal-content, html.pos-printing .app-modal-body { position: static !important; overflow: visible !important; max-height: none !important; animation: none !important; background: none !important; box-shadow: none !important; backdrop-filter: none !important; }
  html.pos-printing #pos-receipt { position: absolute; top: 0; left: 0; width: 72mm; padding: 2mm; color: #000; }
}
</style>
