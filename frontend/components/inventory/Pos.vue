<template>
  <div class="space-y-3">
    <InventoryProductSearch :products="products" :lines="modelValue" :mode="mode" :currency="currency" :held="held" @pick="add" />

    <!-- Cart table (rows stack into cards on phones). -->
    <div class="overflow-hidden rounded-xl border border-border bg-white">
      <table class="pos-table w-full text-left text-sm">
        <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
          <tr>
            <th class="px-3 py-2 font-bold">Product</th>
            <th class="w-32 px-2 py-2 font-bold">Qty</th>
            <th v-if="currency" class="w-36 px-2 py-2 font-bold">{{ mode === 'sale' ? 'Unit price' : 'Unit cost' }}</th>
            <th v-if="currency" class="w-36 px-2 py-2 font-bold">Discount</th>
            <th v-if="currency" class="w-28 px-2 py-2 text-right font-bold">Total</th>
            <th class="w-10 px-2 py-2"><span class="sr-only">Remove</span></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="(line, index) in modelValue" :key="line.key">
            <td class="px-3 py-2">
              <div class="flex items-center gap-2">
                <InventoryProductImage v-if="productOf(line.product_id)" :product="productOf(line.product_id)!" size="sm" />
                <div class="min-w-0">
                  <p class="truncate font-semibold text-navy">{{ productOf(line.product_id)?.name || 'Product' }}</p>
                  <p class="text-[11px]" :class="short(line) ? 'font-semibold text-expense' : 'text-muted'">
                    <span class="font-mono">{{ productOf(line.product_id)?.sku }}</span>
                    <template v-if="showStock && isService(productOf(line.product_id))"> · service</template>
                    <template v-else-if="showStock"> · {{ formatQty(available(productOf(line.product_id))) }} in stock<template v-if="short(line)"> · not enough</template></template>
                  </p>
                </div>
              </div>
            </td>
            <td class="px-2 py-2" data-label="Qty">
              <span class="inline-flex items-center overflow-hidden rounded-lg border border-border">
                <button type="button" class="grid h-8 w-7 place-items-center bg-canvas text-navy" aria-label="Less" @click="step(index, -1)"><Minus :size="12" /></button>
                <input :value="line.quantity" type="text" inputmode="decimal" autocomplete="off" class="h-8 w-12 border-0 text-center text-sm font-semibold text-navy outline-none" :aria-label="`Quantity of ${productOf(line.product_id)?.name || 'product'}`" @input="update(index, { quantity: ($event.target as HTMLInputElement).value })" />
                <button type="button" class="grid h-8 w-7 place-items-center bg-canvas text-navy" aria-label="More" @click="step(index, 1)"><Plus :size="12" /></button>
              </span>
            </td>
            <td v-if="currency" class="px-2 py-2" :data-label="mode === 'sale' ? 'Unit price' : 'Unit cost'">
              <MoneyInput :model-value="line.price" :currency="currency" class="h-8 w-full rounded-lg border border-border px-2 text-sm text-navy" :aria-label="mode === 'sale' ? 'Unit price' : 'Unit cost'" @update:model-value="(value) => update(index, { price: value })" />
            </td>
            <td v-if="currency" class="px-2 py-2" data-label="Discount">
              <span class="flex h-8 overflow-hidden rounded-lg border border-border focus-within:border-primary">
                <input :value="line.discount" type="text" inputmode="decimal" autocomplete="off" placeholder="0" class="min-w-0 flex-1 border-0 px-2 text-sm text-navy outline-none" aria-label="Line discount" @input="update(index, { discount: ($event.target as HTMLInputElement).value })" />
                <button type="button" class="shrink-0 border-l border-border bg-canvas px-2 text-xs font-bold text-navy" :title="line.discount_mode === 'percent' ? 'Percent (tap for an amount)' : 'Amount (tap for a percent)'" @click="update(index, { discount_mode: line.discount_mode === 'percent' ? 'amount' : 'percent' })">{{ line.discount_mode === 'percent' ? '%' : symbol }}</button>
              </span>
            </td>
            <td v-if="currency" class="whitespace-nowrap px-2 py-2 text-right font-semibold text-navy" data-label="Total">{{ formatMoney(lineTotal(line), currency) }}</td>
            <td class="px-2 py-2 text-right">
              <button type="button" class="grid h-8 w-8 place-items-center rounded-lg bg-rose-50 text-expense" :aria-label="`Remove ${productOf(line.product_id)?.name || 'line'}`" @click="remove(index)"><Trash2 :size="13" /></button>
            </td>
          </tr>
          <tr v-if="!modelValue.length" class="empty-row"><td :colspan="currency ? 6 : 3" class="px-3 py-5 text-center text-xs text-muted">Search or scan a product above to add it.</td></tr>
        </tbody>
        <tfoot v-if="modelValue.length" class="bg-canvas">
          <tr>
            <td class="px-3 py-2 text-xs font-semibold text-navy">{{ modelValue.length }} {{ modelValue.length === 1 ? 'product' : 'products' }} · {{ formatQty(totalQty) }} units</td>
            <td :colspan="currency ? 3 : 1" class="px-2 py-2 text-right text-xs text-muted">{{ currency ? 'Subtotal' : '' }}</td>
            <td v-if="currency" class="whitespace-nowrap px-2 py-2 text-right text-sm font-bold text-navy">{{ formatMoney(subtotal, currency) }}</td>
            <td class="px-2 py-2 text-right"><button type="button" class="text-[11px] font-semibold text-expense" @click="emit('update:modelValue', [])">Clear</button></td>
          </tr>
        </tfoot>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Minus, Plus, Trash2 } from 'lucide-vue-next'
import type { Currency, DocLineForm, Product } from '~/types/api'
import { currencyMeta, formatMoney } from '~/utils/money'
import { availableQty, cartQty, formatQty, isService, lineDiscount, linesSubtotal } from '~/utils/inventory'

/**
 * Product picker for sales, purchases and transfers: the search/scan box adds the product to the
 * cart table below; adding it again adds one more. Catalog prices are USD and shown in the
 * document's currency at its rate.
 */
const props = defineProps<{
  modelValue: DocLineForm[]
  products: Product[]
  mode: 'sale' | 'purchase' | 'transfer'
  currency?: Currency
  /** Editing a sale: units it already took out of this branch, available to it again. */
  held?: Record<string, number>
}>()
const emit = defineEmits<{ 'update:modelValue': [lines: DocLineForm[]] }>()

let nextKey = Date.now()

const showStock = computed(() => props.mode !== 'purchase')
const symbol = computed(() => currencyMeta(props.currency || 'USD').symbol.trim())
const subtotal = computed(() => props.currency ? linesSubtotal(props.modelValue, props.currency) : 0)
const totalQty = computed(() => props.modelValue.reduce((sum, l) => sum + (Number(l.quantity) || 0), 0))

function productOf(id: string) {
  return props.products.find(p => p.id === id)
}

// What the branch has for this document (plus what an edited sale already holds).
function available(product: Product | undefined) {
  return availableQty(product, props.held)
}

function short(line: DocLineForm) {
  return showStock.value && !isService(productOf(line.product_id)) && cartQty(props.modelValue, line.product_id) > available(productOf(line.product_id))
}

function lineTotal(line: DocLineForm) {
  const gross = (Number(line.quantity) || 0) * (Number(line.price) || 0)
  return gross - (props.currency ? lineDiscount(line, props.currency) : 0)
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
    step(index, 1)
    return
  }
  emit('update:modelValue', [...props.modelValue, { key: nextKey++, product_id: product.id, quantity: '1', price: price > 0 ? String(price) : '', discount: '', discount_mode: 'amount' }])
}
</script>

<style scoped>
/* Phones: each cart row becomes a card with labelled fields. */
@media (max-width: 639px) {
  .pos-table thead { display: none; }
  .pos-table, .pos-table tbody, .pos-table tfoot, .pos-table tr, .pos-table td { display: block; width: 100%; }
  .pos-table tbody tr:not(.empty-row) { display: grid; grid-template-columns: 1fr 1fr; gap: 6px 8px; padding: 10px 12px; }
  .pos-table tbody tr:not(.empty-row) td { padding: 0; }
  .pos-table tbody tr:not(.empty-row) td:first-child { grid-column: 1 / -1; }
  .pos-table tbody tr:not(.empty-row) td:last-child { position: absolute; right: 12px; width: auto; }
  .pos-table tbody tr:not(.empty-row) { position: relative; }
  .pos-table td[data-label]::before { content: attr(data-label); display: block; margin-bottom: 3px; color: #64748b; font-size: 10px; font-weight: 700; text-transform: uppercase; letter-spacing: .04em; }
  .pos-table td[data-label='Total'] { text-align: left; }
  .pos-table tfoot tr { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 6px; padding: 8px 12px; }
  .pos-table tfoot td { width: auto; padding: 0; }
}
</style>
