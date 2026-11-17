<template>
  <div class="ml-auto w-full max-w-sm space-y-2 rounded-xl border border-border bg-canvas p-3 text-sm">
    <div class="flex items-center justify-between text-muted"><span>Subtotal</span><strong class="text-navy">{{ formatMoney(subtotal, currency) }}</strong></div>
    <div class="flex items-center justify-between gap-3">
      <span class="text-muted">Discount</span>
      <div class="flex items-center gap-2">
        <InventoryDiscountInput class="w-32" :value="discount" :mode="discountMode" :currency="currency" input-class="py-1.5" aria-label="Discount on the total" @update:value="(value) => emit('update:discount', value)" @update:mode="(mode) => emit('update:discountMode', mode)" />
        <span class="w-24 text-right text-expense">−{{ formatMoney(discountValue, currency) }}</span>
      </div>
    </div>
    <div class="flex items-center justify-between gap-3">
      <span class="text-muted">{{ shippingLabel }}</span>
      <div class="flex items-center gap-2">
        <MoneyInput :model-value="shipping" :currency="currency" class="w-32 rounded-lg border border-border bg-white px-2.5 py-1.5 text-sm text-navy" :aria-label="shippingLabel" @update:model-value="(value) => emit('update:shipping', value)" />
        <span class="w-24 text-right text-navy">+{{ formatMoney(Number(shipping) || 0, currency) }}</span>
      </div>
    </div>
    <div class="flex items-center justify-between border-t border-border pt-2"><span class="font-semibold text-navy">Total</span><strong class="text-lg text-navy">{{ formatMoney(total, currency) }}</strong></div>
    <p v-if="hint" class="text-[11px] text-muted">{{ hint }}</p>
  </div>
</template>

<script setup lang="ts">
import type { Currency, DiscountMode } from '~/types/api'
import { formatMoney } from '~/utils/money'
import { discountAmount, roundMoney } from '~/utils/inventory'

/** Subtotal − discount (amount or %) + shipping = total, for sales and purchases. */
const props = withDefaults(defineProps<{
  currency: Currency
  subtotal: number
  discount: string
  discountMode: DiscountMode
  shipping: string
  shippingLabel?: string
  hint?: string
}>(), { shippingLabel: 'Shipping', hint: '' })
const emit = defineEmits<{ 'update:discount': [value: string]; 'update:discountMode': [value: DiscountMode]; 'update:shipping': [value: string] }>()

const discountValue = computed(() => discountAmount(props.subtotal, props.discount, props.discountMode, props.currency))
const total = computed(() => roundMoney(props.subtotal - discountValue.value + (Number(props.shipping) || 0), props.currency))
</script>
