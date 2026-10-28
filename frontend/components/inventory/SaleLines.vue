<template>
  <div class="space-y-2">
    <InventoryProductSearch :products="products" :lines="modelValue" mode="sale" :currency="currency" :held="held" @pick="add" />
    <p v-if="!modelValue.length" class="rounded-xl border border-dashed border-border py-5 text-center text-xs text-muted">Search or scan a product above to add it.</p>

    <div v-for="(line, index) in modelValue" :key="line.key" class="space-y-2 rounded-xl border border-border bg-white p-3">
      <div class="flex items-start gap-2.5">
        <InventoryProductImage v-if="productOf(line.product_id)" :product="productOf(line.product_id)!" size="sm" />
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-semibold text-navy">{{ productOf(line.product_id)?.name || 'Product' }}</p>
          <p class="text-[11px]" :class="short(line) ? 'font-semibold text-expense' : 'text-muted'">
            <span class="font-mono">{{ productOf(line.product_id)?.sku }}</span>
            <template v-if="isService(productOf(line.product_id))"> · service</template>
            <template v-else> · {{ formatQty(availableQty(productOf(line.product_id), held)) }} in stock<template v-if="short(line)"> · not enough</template></template>
          </p>
        </div>
        <button type="button" class="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-muted hover:bg-rose-50 hover:text-expense" :aria-label="`Remove ${productOf(line.product_id)?.name || 'line'}`" @click="remove(index)"><X :size="15" /></button>
      </div>

      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3">
        <div class="space-y-1 text-[11px] font-medium text-muted">
          Qty
          <span class="flex h-9 overflow-hidden rounded-lg border border-border">
            <button type="button" class="grid w-8 shrink-0 place-items-center bg-canvas text-navy" aria-label="Less" @click="step(index, -1)"><Minus :size="12" /></button>
            <input :value="line.quantity" type="text" inputmode="decimal" autocomplete="off" class="min-w-0 flex-1 border-0 text-center text-sm font-semibold text-navy outline-none" :aria-label="`Quantity of ${productOf(line.product_id)?.name || 'product'}`" @input="update(index, { quantity: ($event.target as HTMLInputElement).value })" />
            <button type="button" class="grid w-8 shrink-0 place-items-center bg-canvas text-navy" aria-label="More" @click="step(index, 1)"><Plus :size="12" /></button>
          </span>
        </div>
        <div class="space-y-1 text-[11px] font-medium text-muted">
          Unit price
          <MoneyInput :model-value="line.price" :currency="currency" class="h-9 w-full rounded-lg border border-border px-2.5 text-sm text-navy" aria-label="Unit price" @update:model-value="(value) => update(index, { price: value })" />
        </div>
        <div class="col-span-2 space-y-1 text-[11px] font-medium text-muted sm:col-span-1">
          Discount
          <span class="flex h-9 overflow-hidden rounded-lg border border-border focus-within:border-primary">
            <input :value="line.discount" type="text" inputmode="decimal" autocomplete="off" placeholder="0" class="min-w-0 flex-1 border-0 px-2.5 text-sm text-navy outline-none" aria-label="Line discount" @input="update(index, { discount: ($event.target as HTMLInputElement).value })" />
            <button type="button" class="shrink-0 border-l border-border bg-canvas px-2.5 text-xs font-bold text-navy" :title="line.discount_mode === 'percent' ? 'Percent (tap for an amount)' : 'Amount (tap for a percent)'" @click="update(index, { discount_mode: line.discount_mode === 'percent' ? 'amount' : 'percent' })">{{ line.discount_mode === 'percent' ? '%' : symbol }}</button>
          </span>
        </div>
      </div>

      <input :value="line.note" type="text" maxlength="255" autocomplete="off" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Note for this item (optional)" :aria-label="`Note for ${productOf(line.product_id)?.name || 'this item'}`" @input="update(index, { note: ($event.target as HTMLInputElement).value })" />

      <p class="text-right text-sm font-semibold text-navy">
        <span v-if="lineDiscount(line, currency)" class="mr-2 font-normal text-muted line-through">{{ formatMoney(gross(line), currency) }}</span>
        {{ formatMoney(gross(line) - lineDiscount(line, currency), currency) }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Minus, Plus, X } from 'lucide-vue-next'
import type { Currency, DocLineForm, Product } from '~/types/api'
import { currencyMeta, formatMoney } from '~/utils/money'
import { availableQty, cartQty, formatQty, isService, lineDiscount } from '~/utils/inventory'

/**
 * A sale's lines as cards: quantity, price, % or amount discount and a note on each. The product
 * picked lands on top; picking one already on the sale adds one to it and brings it to the top.
 */
const props = defineProps<{
  modelValue: DocLineForm[]
  products: Product[]
  currency: Currency
  /** Editing a sale: units it already took out of this branch, available to it again. */
  held?: Record<string, number>
}>()
const emit = defineEmits<{ 'update:modelValue': [lines: DocLineForm[]] }>()

let nextKey = Date.now()

const symbol = computed(() => currencyMeta(props.currency).symbol.trim())

function productOf(id: string) {
  return props.products.find(p => p.id === id)
}

function short(line: DocLineForm) {
  const product = productOf(line.product_id)
  return !isService(product) && cartQty(props.modelValue, line.product_id) > availableQty(product, props.held)
}

function gross(line: DocLineForm) {
  return (Number(line.quantity) || 0) * (Number(line.price) || 0)
}

function update(index: number, patch: Partial<DocLineForm>) {
  emit('update:modelValue', props.modelValue.map((line, i) => i === index ? { ...line, ...patch } : line))
}

function step(index: number, by: number) {
  const next = Math.max(1, (Number(props.modelValue[index]!.quantity) || 0) + by)
  update(index, { quantity: String(next) })
}

function remove(index: number) {
  emit('update:modelValue', props.modelValue.filter((_, i) => i !== index))
}

function add(product: Product, price: number) {
  const index = props.modelValue.findIndex(l => l.product_id === product.id)
  if (index >= 0) {
    const line = props.modelValue[index]!
    const rest = props.modelValue.filter((_, i) => i !== index)
    emit('update:modelValue', [{ ...line, quantity: String((Number(line.quantity) || 0) + 1) }, ...rest])
    return
  }
  emit('update:modelValue', [{ key: nextKey++, product_id: product.id, quantity: '1', price: price > 0 ? String(price) : '', discount: '', discount_mode: 'amount', note: '' }, ...props.modelValue])
}
</script>
