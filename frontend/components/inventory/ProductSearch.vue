<template>
  <!-- Search or scan: pick from the dropdown, or Enter adds the exact barcode/SKU (or the highlighted match). -->
  <div ref="boxEl" class="relative">
    <label class="flex items-center gap-2 rounded-xl border border-border bg-white px-3 py-2 focus-within:border-primary focus-within:ring-2 focus-within:ring-blue-100">
      <ScanBarcode :size="18" class="shrink-0 text-muted" />
      <input
        ref="searchEl"
        v-model.trim="query"
        type="search"
        autocomplete="off"
        enterkeyhint="done"
        role="combobox"
        :aria-expanded="open"
        aria-controls="pos-results"
        class="min-w-0 flex-1 border-0 bg-transparent text-sm text-navy outline-none"
        placeholder="Search a product, or scan its barcode / SKU…"
        aria-label="Search or scan a product"
        @focus="open = true"
        @keydown.down.prevent="move(1)"
        @keydown.up.prevent="move(-1)"
        @keydown.enter.prevent="enterPick"
        @keydown.esc="open = false"
      />
      <span v-if="scanMessage" class="shrink-0 text-[11px] font-semibold" :class="scanMessage.ok ? 'text-income' : 'text-expense'">{{ scanMessage.text }}</span>
    </label>
    <ul v-if="open" id="pos-results" role="listbox" class="absolute inset-x-0 top-full z-30 mt-1 max-h-72 overflow-y-auto rounded-xl border border-border bg-white py-1 shadow-xl">
      <li
        v-for="(product, i) in shown"
        :key="product.id"
        role="option"
        :aria-selected="i === highlight"
        :aria-disabled="soldOut(product)"
        class="flex cursor-pointer items-center gap-2.5 px-3 py-2"
        :class="[i === highlight ? 'bg-blue-50' : 'hover:bg-canvas', soldOut(product) ? 'cursor-not-allowed opacity-50' : '']"
        @mousedown.prevent="pick(product)"
        @mouseenter="highlight = i"
      >
        <InventoryProductImage :product="product" size="sm" />
        <span class="min-w-0 flex-1">
          <span class="block truncate text-sm font-semibold text-navy">{{ product.name }}</span>
          <span class="block text-[11px] text-muted"><span class="font-mono">{{ product.sku }}</span><template v-if="product.category_name"> · {{ product.category_name }}</template><template v-if="showStock"> · <span :class="soldOut(product) ? 'text-expense' : ''">{{ isService(product) ? 'service' : soldOut(product) ? 'out of stock' : `${formatQty(availableQty(product, held))} ${product.unit} in stock` }}</span></template></span>
        </span>
        <span v-if="currency" class="shrink-0 text-sm font-semibold text-navy">{{ formatMoney(defaultPrice(product), currency) }}</span>
        <span v-if="cartQty(lines, product.id)" class="shrink-0 rounded-full bg-primary px-1.5 py-0.5 text-[10px] font-bold text-white">×{{ formatQty(cartQty(lines, product.id)) }}</span>
      </li>
      <li v-if="!shown.length" class="px-3 py-3 text-center text-xs text-muted">{{ query ? `No product matches “${query}”.` : 'No products yet.' }}</li>
      <li v-if="matches.length > shown.length" class="px-3 py-1.5 text-center text-[11px] text-muted">{{ matches.length - shown.length }} more — keep typing to narrow down</li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import { ScanBarcode } from 'lucide-vue-next'
import type { Currency, DocLineForm, Product } from '~/types/api'
import { useCurrency } from '~/composables/useCurrency'
import { formatMoney } from '~/utils/money'
import { availableQty, cartQty, formatQty, isService } from '~/utils/inventory'

/**
 * The search/scan box of a sale, purchase or transfer: a dropdown of matches (type, or scan a
 * barcode/SKU) that emits `pick` with the chosen product and its usual price. Sales and transfers
 * can't pick what the branch doesn't have. Catalog prices are USD and shown in the document's
 * currency at its rate.
 */
const props = defineProps<{
  products: Product[]
  /** The document's lines so far (for the "×2" already-added badge). */
  lines: DocLineForm[]
  mode: 'sale' | 'purchase' | 'transfer'
  currency?: Currency
  /** Editing a sale: units it already took out of this branch, available to it again. */
  held?: Record<string, number>
}>()
const emit = defineEmits<{ pick: [product: Product, price: number] }>()

const { fromUSD } = useCurrency()
const query = ref('')
const open = ref(false)
const highlight = ref(0)
const boxEl = ref<HTMLElement | null>(null)
const searchEl = ref<HTMLInputElement | null>(null)
const scanMessage = ref<{ ok: boolean; text: string } | null>(null)
let messageTimer: ReturnType<typeof setTimeout> | undefined

onClickOutside(boxEl, () => { open.value = false })

const showStock = computed(() => props.mode !== 'purchase')
// Services are only sold: purchases and transfers don't offer them.
const active = computed(() => props.products.filter(p => p.is_active && (props.mode === 'sale' || !isService(p))))
const matches = computed(() => {
  const q = query.value.toLowerCase()
  if (!q) return active.value
  return active.value.filter(p => p.name.toLowerCase().includes(q) || p.sku.toLowerCase().includes(q) || (p.barcode || '').toLowerCase().includes(q))
})
const shown = computed(() => matches.value.slice(0, 12))
watch(query, () => { highlight.value = 0; open.value = true })

// Sales and transfers can only take what the branch has.
function soldOut(product: Product) {
  return showStock.value && !isService(product) && availableQty(product, props.held) <= 0
}

// The usual price in the document's currency: sale price, or cost (else average cost) for purchases.
function defaultPrice(product: Product) {
  if (!props.currency) return 0
  const usd = props.mode === 'sale' ? product.sale_price : (Number(product.cost_price) > 0 ? product.cost_price : product.avg_cost)
  return fromUSD(usd, props.currency)
}

function flash(ok: boolean, text: string) {
  scanMessage.value = { ok, text }
  if (messageTimer) clearTimeout(messageTimer)
  messageTimer = setTimeout(() => { scanMessage.value = null }, 2000)
}

// Adds the product and clears the box, ready for the next search or scan.
function pick(product: Product) {
  if (soldOut(product)) {
    flash(false, 'Out of stock')
    return
  }
  emit('pick', product, defaultPrice(product))
  flash(true, `+1 ${product.name}`)
  query.value = ''
  open.value = false
  searchEl.value?.focus()
}

function move(by: number) {
  open.value = true
  const count = shown.value.length
  if (count) highlight.value = (highlight.value + by + count) % count
}

// Enter (or a scanner's Enter): an exact barcode/SKU first, else the highlighted match.
function enterPick() {
  const q = query.value.toLowerCase()
  const exact = q ? active.value.find(p => (p.barcode && p.barcode.toLowerCase() === q) || p.sku.toLowerCase() === q) : undefined
  const choice = exact || shown.value[highlight.value]
  if (!choice) {
    flash(false, 'Not found')
    return
  }
  pick(choice)
}
</script>
