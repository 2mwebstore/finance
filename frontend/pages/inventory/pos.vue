<template>
  <div class="flex h-screen h-dvh flex-col bg-canvas">
    <header class="flex h-14 shrink-0 items-center justify-between gap-2 border-b border-border bg-white px-3 sm:px-4">
      <div class="flex min-w-0 items-center gap-2 sm:gap-3">
        <NuxtLink to="/inventory/sales" class="grid h-9 w-9 shrink-0 place-items-center rounded-lg border border-border text-muted hover:text-navy" aria-label="Back office" title="Back office"><ArrowLeft :size="17" /></NuxtLink>
        <div class="hidden min-w-0 sm:block">
          <p class="text-sm font-bold leading-tight text-navy">Point of sale</p>
          <p class="truncate text-[11px] text-muted">{{ user?.full_name }}</p>
        </div>
        <div v-if="branchChoices.length > 1" class="w-44 sm:w-60"><SearchSelect :model-value="branchId" :options="branchChoices" aria-label="Branch" @update:model-value="changeBranch" /></div>
        <span v-else-if="branchLabel" class="truncate text-sm font-semibold text-navy sm:font-normal sm:text-muted">{{ branchLabel }}</span>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <button type="button" class="relative inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-2.5 py-2 text-sm font-medium text-navy hover:bg-canvas sm:px-3" aria-label="Held sales" @click="heldOpen = true">
          <Clock :size="16" /><span class="hidden sm:inline">Held</span>
          <span v-if="held.list.value.length" class="absolute -right-1.5 -top-1.5 grid h-4 min-w-4 place-items-center rounded-full bg-primary px-1 text-[10px] font-bold text-white sm:static">{{ held.list.value.length }}</span>
        </button>
        <button v-if="canViewSales" type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-2.5 py-2 text-sm font-medium text-navy hover:bg-canvas sm:px-3" aria-label="Today’s sales" @click="recentOpen = true">
          <ReceiptText :size="16" /><span class="hidden sm:inline">Today’s sales</span>
        </button>
      </div>
    </header>

    <div v-if="!branchId" class="grid flex-1 place-items-center p-6 text-center text-sm text-muted">You don’t work in any branch yet, so there is nothing to sell.</div>

    <div v-else class="relative flex min-h-0 flex-1">
      <!-- Cart: a side panel from md up; on phones a full-screen sheet opened from the bottom bar. -->
      <aside class="min-h-0 flex-col bg-white md:static md:z-auto md:flex md:w-80 md:shrink-0 md:border-r md:border-border lg:w-96" :class="cartOpen ? 'fixed inset-0 z-30 flex' : 'hidden'" aria-label="Cart">
        <div class="flex h-14 shrink-0 items-center justify-between gap-3 border-b border-border px-4 md:hidden">
          <p class="text-base font-bold text-navy">Cart ({{ formatQty(itemCount) }})</p>
          <button type="button" class="-mr-2 p-2 text-muted hover:text-navy" aria-label="Back to products" @click="cartOpen = false"><X :size="20" /></button>
        </div>
        <div class="border-b border-border p-3">
          <input v-model.trim="cart.customer" maxlength="191" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Customer (walk-in)" aria-label="Customer" />
        </div>

        <div ref="cartListEl" class="min-h-0 flex-1 divide-y divide-border overflow-y-auto">
          <p v-if="!cart.lines.length" class="px-4 py-10 text-center text-sm text-muted">Tap a product to add it, or scan its barcode.</p>
          <div v-for="line in cart.lines" :key="line.key" class="space-y-1.5 p-3">
            <!-- Name and line total on top, controls underneath, so a narrow cart doesn't cut the name short. -->
            <div class="flex items-start gap-2">
              <div class="min-w-0 flex-1">
                <p class="line-clamp-2 text-sm font-semibold leading-snug text-navy">{{ productOf(line.product_id)?.name || 'Product' }}</p>
                <p class="text-[11px]" :class="short(line) ? 'font-semibold text-expense' : 'text-muted'">{{ formatMoney(Number(line.price) || 0, currency) }} each<template v-if="short(line)"> · only {{ formatQty(availableQty(productOf(line.product_id))) }} in stock</template></p>
              </div>
              <p class="shrink-0 text-right font-mono text-sm font-semibold text-navy">{{ formatMoney(lineTotal(line), currency) }}</p>
            </div>
            <div class="flex items-center justify-between gap-2">
              <span class="inline-flex items-center overflow-hidden rounded-lg border border-border">
                <button type="button" class="grid h-9 w-9 place-items-center bg-canvas text-navy md:h-8 md:w-8" aria-label="One less" @click="step(line.key, -1)"><Minus :size="13" /></button>
                <input :value="line.quantity" type="text" inputmode="decimal" autocomplete="off" class="h-9 w-12 border-0 text-center text-sm font-semibold text-navy outline-none md:h-8" :aria-label="`Quantity of ${productOf(line.product_id)?.name || 'product'}`" @input="update(line.key, { quantity: ($event.target as HTMLInputElement).value })" />
                <button type="button" class="grid h-9 w-9 place-items-center bg-canvas text-navy md:h-8 md:w-8" aria-label="One more" @click="step(line.key, 1)"><Plus :size="13" /></button>
              </span>
              <span class="flex items-center gap-1">
                <button type="button" class="grid h-9 w-9 place-items-center rounded-lg text-muted hover:bg-canvas hover:text-navy md:h-8 md:w-8" aria-label="Discount or note" title="Discount or note" @click="editingKey = line.key"><Pencil :size="15" /></button>
                <button type="button" class="grid h-9 w-9 place-items-center rounded-lg text-muted hover:bg-rose-50 hover:text-expense md:h-8 md:w-8" :aria-label="`Remove ${productOf(line.product_id)?.name || 'item'}`" @click="remove(line.key)"><X :size="16" /></button>
              </span>
            </div>
            <div v-if="Number(line.discount) || line.note" class="flex items-center justify-between gap-2 text-[11px] text-muted">
              <span class="truncate italic">{{ line.note }}</span>
              <span v-if="Number(line.discount)" class="shrink-0">{{ line.discount_mode === 'percent' ? `${line.discount}% off` : 'Discount' }} (−{{ formatMoney(lineDiscount(line, currency), currency) }})</span>
            </div>
          </div>
        </div>

        <div class="space-y-3 border-t border-border p-3">
          <div class="flex items-center gap-2">
            <span class="shrink-0 text-xs font-semibold text-muted">Discount</span>
            <span class="flex min-w-0 flex-1 overflow-hidden rounded-lg border border-border focus-within:border-primary">
              <input v-model.trim="cart.discount" type="text" inputmode="decimal" autocomplete="off" placeholder="0" class="min-w-0 flex-1 border-0 px-3 py-1.5 text-sm text-navy outline-none" aria-label="Discount on the sale" />
              <button type="button" class="shrink-0 border-l border-border bg-canvas px-3 text-xs font-bold text-navy" :title="cart.discount_mode === 'percent' ? 'Percent (tap for an amount)' : 'Amount (tap for a percent)'" @click="cart.discount_mode = cart.discount_mode === 'percent' ? 'amount' : 'percent'">{{ cart.discount_mode === 'percent' ? '%' : symbol }}</button>
            </span>
          </div>
          <div class="space-y-1 text-sm">
            <div class="flex justify-between"><span class="text-muted">Subtotal</span><span class="font-mono text-navy">{{ formatMoney(gross, currency) }}</span></div>
            <div v-if="lineDiscounts" class="flex justify-between"><span class="text-muted">Item discounts</span><span class="font-mono text-expense">−{{ formatMoney(lineDiscounts, currency) }}</span></div>
            <div v-if="saleDiscount" class="flex justify-between"><span class="text-muted">Sale discount</span><span class="font-mono text-expense">−{{ formatMoney(saleDiscount, currency) }}</span></div>
            <div class="flex items-baseline justify-between border-t border-border pt-2">
              <span class="text-base font-bold text-navy">Total</span>
              <span class="text-right"><strong class="font-mono text-lg text-navy">{{ formatMoney(total, currency) }}</strong><span v-if="otherCurrency" class="block text-[11px] text-muted">≈ {{ formatMoney(convert(total, currency, otherCurrency), otherCurrency) }}</span></span>
            </div>
          </div>
          <p v-if="cart.lines.length && problem" class="rounded-lg bg-rose-50 px-3 py-2 text-xs font-medium text-expense" role="alert">{{ problem }}</p>
          <div class="grid grid-cols-2 gap-2">
            <button type="button" class="inline-flex items-center justify-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-semibold text-navy disabled:opacity-50" :disabled="!cart.lines.length" @click="holdSale"><Clock :size="15" /> Hold</button>
            <button type="button" class="inline-flex items-center justify-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-semibold text-navy disabled:opacity-50" :disabled="!cart.lines.length" @click="clearCart"><Trash2 :size="15" /> Clear</button>
          </div>
          <button type="button" class="w-full rounded-xl bg-primary py-3 text-lg font-bold text-white hover:bg-primary-hover disabled:opacity-50" :disabled="!cart.lines.length || !!problem" @click="openPayment">Pay {{ formatMoney(total, currency) }}</button>
        </div>
      </aside>

      <!-- Products -->
      <main class="flex min-w-0 flex-1 flex-col gap-3 p-3 sm:p-4">
        <label class="flex items-center gap-2 rounded-xl border border-border bg-white px-3 py-2.5 focus-within:border-primary focus-within:ring-2 focus-within:ring-blue-100">
          <ScanBarcode :size="18" class="shrink-0 text-muted" />
          <input ref="searchEl" v-model="search" type="search" autocomplete="off" enterkeyhint="done" class="min-w-0 flex-1 border-0 bg-transparent text-sm text-navy outline-none" placeholder="Search a product, or scan its barcode / SKU…" aria-label="Search or scan a product" @keydown.enter.prevent="enterPick" />
          <span v-if="scanMessage" class="shrink-0 text-[11px] font-semibold" :class="scanMessage.ok ? 'text-income' : 'text-expense'">{{ scanMessage.text }}</span>
        </label>

        <div v-if="categories.length > 1" class="flex shrink-0 gap-2 overflow-x-auto pb-1">
          <button v-for="c in [{ id: '', name: 'All' }, ...categories]" :key="c.id" type="button" class="shrink-0 whitespace-nowrap rounded-lg px-3 py-1.5 text-sm font-medium" :class="category === c.id ? 'bg-primary text-white' : 'border border-border bg-white text-muted hover:text-navy'" @click="category = c.id">{{ c.name }}</button>
        </div>

        <p v-if="loadingProducts && !products.length" class="text-sm text-muted">Loading products…</p>
        <AppAlert v-else-if="productsError" title="Couldn’t load the products" :message="productsError"><template #action><button type="button" @click="loadProducts">Try again</button></template></AppAlert>
        <!-- auto-rows-max: rows keep their tiles' height and the grid scrolls (instead of squeezing them to fit). -->
        <div v-else class="grid min-h-0 flex-1 auto-rows-max grid-cols-2 content-start gap-2 overflow-y-auto pb-24 sm:grid-cols-3 sm:gap-3 md:pb-0 lg:grid-cols-4 xl:grid-cols-5 2xl:grid-cols-6">
          <button v-for="product in shown" :key="product.id" type="button" class="flex flex-col overflow-hidden rounded-xl border border-border bg-white text-left transition hover:border-primary hover:shadow-sm disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:border-border disabled:hover:shadow-none" :disabled="soldOut(product)" @click="pick(product)">
            <span class="relative block shrink-0">
              <InventoryProductImage :product="product" size="card" />
              <span v-if="cartQty(cart.lines, product.id)" class="absolute right-2 top-2 rounded-full bg-primary px-2 py-0.5 text-xs font-bold text-white">×{{ formatQty(cartQty(cart.lines, product.id)) }}</span>
            </span>
            <span class="flex grow flex-col gap-0.5 p-2.5">
              <span class="line-clamp-2 text-sm font-semibold leading-tight text-navy">{{ product.name }}</span>
              <span class="font-mono text-[11px] text-muted">{{ product.sku }}</span>
              <span class="mt-auto flex items-end justify-between gap-1 pt-1">
                <strong class="text-sm text-navy">{{ formatMoney(priceOf(product), currency) }}</strong>
                <span class="text-[10px] font-semibold" :class="soldOut(product) ? 'text-expense' : 'text-muted'">{{ isService(product) ? 'Service' : soldOut(product) ? 'Out of stock' : `${formatQty(availableQty(product))} ${product.unit}` }}</span>
              </span>
            </span>
          </button>
          <p v-if="!shown.length" class="col-span-full py-8 text-center text-sm text-muted">{{ search ? `No product matches “${search}”.` : 'No products to sell in this branch yet.' }}</p>
        </div>
      </main>

      <!-- Phones: the cart's summary; tap to open it. -->
      <div v-if="!cartOpen" class="absolute inset-x-0 bottom-0 z-20 border-t border-border bg-white p-3 md:hidden">
        <button type="button" class="flex w-full items-center justify-between gap-3 rounded-xl bg-primary px-4 py-3 text-white" aria-label="View cart" @click="cartOpen = true">
          <span class="flex items-center gap-2 text-sm font-semibold"><ShoppingCart :size="18" /> {{ formatQty(itemCount) }} {{ itemCount === 1 ? 'item' : 'items' }}</span>
          <span class="font-mono text-lg font-bold">{{ formatMoney(total, currency) }}</span>
        </button>
      </div>
    </div>

    <PosLineEditModal :line="editingLine" :title="editingLine ? productOf(editingLine.product_id)?.name || 'Item' : ''" :currency="currency" @close="editingKey = null" @save="saveLineEdit" />
    <PosPaymentModal :open="paymentOpen" :total="total" :currency="currency" :methods="paymentMethods" :default-method-id="defaultPayment" :submitting="submitting" :error="payError" @close="paymentOpen = false" @paid="pay" />
    <PosReceiptModal :sale="receipt" :paid="receiptPaid" @close="closeReceipt" />
    <PosHeldSalesModal :open="heldOpen" :list="held.list.value" @close="heldOpen = false" @resume="resumeHeld" @discard="(id) => held.take(id)" />
    <PosRecentSalesModal :open="recentOpen" :branch-id="branchId" :branch-name="branchLabel" @close="recentOpen = false" @receipt="showReceipt" @voided="loadProducts" />
  </div>
</template>

<script setup lang="ts">
import { ArrowLeft, Clock, Minus, Pencil, Plus, ReceiptText, ScanBarcode, ShoppingCart, Trash2, X } from 'lucide-vue-next'
import type { DocLineForm, Paged, Product, Sale } from '~/types/api'
import { useToast, sentence } from '~/composables/useToast'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { usePaymentMethods } from '~/composables/usePaymentMethods'
import { useCurrency } from '~/composables/useCurrency'
import { useHeldSales, usePosCart, type PosPayment } from '~/composables/usePosCart'
import { currencyMeta, formatMoney } from '~/utils/money'
import { availableQty as stockOf, cartQty, docDiscountCheck, formatQty, isService, lineDiscount, linesCheck, todayISO } from '~/utils/inventory'

/**
 * The till: tap or scan products into the cart, then take payment. A full screen of its own
 * (no sidebar). Saving makes an ordinary sale (stock goes out of the branch at once); the cart,
 * held sales and the receipt are the till's.
 */
definePageMeta({ layout: false })
useHead({ title: 'Point of sale' })

const { api } = useApi()
const { user } = useAuth()
const { can } = usePermissions()
const toast = useToast()
const canViewSales = computed(() => can('sales.view'))
const { branches } = useBranchOptions()
const branchChoices = computed(() => branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })))
const { methods: paymentMethods, defaultId: defaultPayment, load: loadPayments } = usePaymentMethods()
const { active: activeCurrencies, defaultCurrency, load: loadCurrency, fromUSD, convert } = useCurrency()

const currency = computed(() => defaultCurrency.value)
const symbol = computed(() => currencyMeta(currency.value).symbol.trim())
const otherCurrency = computed(() => currency.value === 'USD' ? activeCurrencies.value.find(c => !c.base)?.code : 'USD')
const { cart, itemCount, gross, lineDiscounts, saleDiscount, total, lineTotal, add, update, step, remove, clear, restore } = usePosCart(() => currency.value)

// --- branch: the cart's, else the one used last on this browser, else the user's own, else the first ---
const branchKey = computed(() => `pos-branch:${user.value?.id || ''}`)
const branchId = computed(() => cart.value.branch_id)
const branchLabel = computed(() => {
  const b = branches.value.find(item => item.id === branchId.value)
  return b ? `${b.code} — ${b.name}` : ''
})
function pickBranch() {
  const ids = branches.value.map(b => b.id)
  if (ids.includes(cart.value.branch_id)) return
  let remembered = ''
  try { remembered = localStorage.getItem(branchKey.value) || '' } catch {}
  const own = (user.value?.branches || []).map(b => b.id).find(id => ids.includes(id))
  clear(ids.includes(remembered) ? remembered : own || ids[0] || '')
}
watch(branches, pickBranch, { immediate: true })

function changeBranch(value: string | string[]) {
  const id = String(value || '')
  if (!id || id === branchId.value) return
  // Another branch has other stock, so the cart starts again.
  if (cart.value.lines.length && !window.confirm('Switch branch? The cart will be emptied.')) return
  clear(id)
  try { localStorage.setItem(branchKey.value, id) } catch {}
}

// --- products of the branch, with their stock there ---
const products = ref<Product[]>([])
const loadingProducts = ref(false)
const productsError = ref('')
let productsRequest = 0
async function loadProducts() {
  if (!branchId.value) return
  const request = ++productsRequest
  loadingProducts.value = true
  productsError.value = ''
  try {
    const result = await api<Paged<Product>>('/inventory/products', { query: { branch_id: branchId.value, active: 'true', page_size: 500 } })
    if (request === productsRequest) products.value = result.items || []
  } catch (e: any) {
    if (request === productsRequest) productsError.value = e?.data?.error || 'Could not load the products.'
  } finally {
    if (request === productsRequest) loadingProducts.value = false
  }
}
watch(branchId, () => { products.value = []; category.value = ''; loadProducts() })

function productOf(id: string) {
  return products.value.find(p => p.id === id)
}
function availableQty(product: Product | undefined) {
  return stockOf(product)
}
function soldOut(product: Product) {
  return !isService(product) && availableQty(product) <= 0
}
function short(line: DocLineForm) {
  const product = productOf(line.product_id)
  return !!product && !isService(product) && cartQty(cart.value.lines, line.product_id) > availableQty(product)
}
// Catalog prices are USD; the till sells in the default currency.
function priceOf(product: Product) {
  return fromUSD(product.sale_price, currency.value)
}

// --- browsing: category chips and search ---
const search = ref('')
const category = ref('')
const categories = computed(() => {
  const seen = new Map<string, string>()
  for (const p of products.value) if (p.category_id && p.category_name) seen.set(p.category_id, p.category_name)
  return [...seen].map(([id, name]) => ({ id, name })).sort((a, b) => a.name.localeCompare(b.name))
})
const shown = computed(() => {
  const q = search.value.trim().toLowerCase()
  return products.value.filter(p => p.is_active
    && (!category.value || p.category_id === category.value)
    && (!q || p.name.toLowerCase().includes(q) || p.sku.toLowerCase().includes(q) || (p.barcode || '').toLowerCase().includes(q)))
})

const cartOpen = ref(false)
const cartListEl = ref<HTMLElement | null>(null)
const searchEl = ref<HTMLInputElement | null>(null)
const scanMessage = ref<{ ok: boolean; text: string } | null>(null)
let messageTimer: ReturnType<typeof setTimeout> | undefined
const finePointer = import.meta.client && window.matchMedia('(pointer: fine)').matches

function flash(ok: boolean, text: string) {
  scanMessage.value = { ok, text }
  if (messageTimer) clearTimeout(messageTimer)
  messageTimer = setTimeout(() => { scanMessage.value = null }, 2000)
}

function pick(product: Product) {
  if (soldOut(product)) return
  add(product.id, priceOf(product))
  // The item lands on top of the cart: scroll back up so it is in view.
  nextTick(() => cartListEl.value?.scrollTo({ top: 0 }))
  // With a mouse (a desk till), keep the search box ready for the scanner.
  if (finePointer) searchEl.value?.focus()
}

// Enter (or a scanner's Enter): the exact barcode/SKU, else the only match.
function enterPick() {
  const q = search.value.trim().toLowerCase()
  if (!q) return
  const exact = products.value.find(p => p.is_active && ((p.barcode && p.barcode.toLowerCase() === q) || p.sku.toLowerCase() === q))
  const choice = exact || (shown.value.length === 1 ? shown.value[0] : undefined)
  if (!choice) {
    flash(false, shown.value.length ? 'Pick one below' : 'Not found')
    return
  }
  if (soldOut(choice)) {
    flash(false, 'Out of stock')
    return
  }
  pick(choice)
  flash(true, `+1 ${choice.name}`)
  search.value = ''
}

// Paid, held or cleared: back to the products.
watch(() => cart.value.lines.length, (n) => { if (n === 0) cartOpen.value = false })

// --- what stops payment (same rules as the API) ---
const problem = computed(() => {
  const lines = linesCheck(cart.value.lines) || docDiscountCheck(() => cart.value.discount_mode)(cart.value.discount)
  if (lines) return sentence(lines)
  const shortNames = cart.value.lines.filter(short).map(l => productOf(l.product_id)?.name || 'a product')
  return shortNames.length ? `Not enough stock for ${shortNames.join(', ')}.` : ''
})

// --- line discount and note ---
const editingKey = ref<number | null>(null)
const editingLine = computed(() => cart.value.lines.find(l => l.key === editingKey.value) || null)
function saveLineEdit(patch: Pick<DocLineForm, 'discount' | 'discount_mode' | 'note'>) {
  if (editingKey.value !== null) update(editingKey.value, patch)
  editingKey.value = null
}

// --- hold, resume, clear ---
const held = useHeldSales(() => `${user.value?.id || ''}:${branchId.value}`)
const heldOpen = ref(false)
const recentOpen = ref(false)

function holdSale() {
  if (!cart.value.lines.length) return
  held.hold({ currency: currency.value, total: total.value, customer: cart.value.customer, discount: cart.value.discount, discount_mode: cart.value.discount_mode, lines: cart.value.lines })
  clear()
  toast.success('Resume it from Held when the customer is ready.', { title: 'Sale held' })
}

function resumeHeld(id: number) {
  if (cart.value.lines.length && !window.confirm('Replace the items in the cart with the held sale?')) return
  const sale = held.take(id)
  if (!sale) return
  const left = restore(sale, productId => !!productOf(productId))
  heldOpen.value = false
  if (left) toast.error(`${left} ${left === 1 ? 'item is' : 'items are'} no longer sold here and ${left === 1 ? 'was' : 'were'} left out.`, { title: 'Sale resumed' })
  else toast.success('Its items are back in the cart.', { title: 'Sale resumed' })
}

function clearCart() {
  if (cart.value.lines.length && window.confirm('Remove everything from the cart?')) clear()
}

// --- payment and receipt ---
const paymentOpen = ref(false)
const submitting = ref(false)
const payError = ref('')
const receipt = ref<Sale | null>(null)
const receiptPaid = ref<PosPayment | null>(null)
// A receipt opened from today's sales: the list steps aside and comes back after.
const receiptFromList = ref(false)

function openPayment() {
  if (!cart.value.lines.length || problem.value) return
  payError.value = ''
  paymentOpen.value = true
}

function escapeHtml(text: string) {
  return text.replace(/[&<>"']/g, ch => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[ch]!))
}

async function pay(payment: PosPayment) {
  if (submitting.value) return
  submitting.value = true
  payError.value = ''
  try {
    const sale = await api<Sale>('/inventory/sales', {
      method: 'POST',
      body: {
        branch_id: branchId.value, sale_date: todayISO(), currency: currency.value, customer: cart.value.customer,
        payment_method_id: payment.payment_method_id, discount: String(saleDiscount.value), shipping: '0',
        note: payment.reference ? `<p>Payment reference: ${escapeHtml(payment.reference)}</p>` : '',
        items: cart.value.lines.map(l => ({ product_id: l.product_id, quantity: l.quantity, unit_price: l.price || '0', discount: String(lineDiscount(l, currency.value)), note: (l.note || '').trim() })),
      },
    })
    paymentOpen.value = false
    clear()
    receiptPaid.value = payment
    receipt.value = sale
    loadProducts() // stock went out
  } catch (e: any) {
    payError.value = e?.data?.error || 'Could not save this sale.'
  } finally {
    submitting.value = false
  }
}

function showReceipt(sale: Sale) {
  recentOpen.value = false
  receiptFromList.value = true
  receiptPaid.value = null
  receipt.value = sale
}

function closeReceipt() {
  receipt.value = null
  receiptPaid.value = null
  if (receiptFromList.value) {
    receiptFromList.value = false
    recentOpen.value = true
  } else if (finePointer) {
    nextTick(() => searchEl.value?.focus())
  }
}

onMounted(async () => {
  await Promise.all([loadCurrency(), loadPayments(), loadProducts()])
  if (finePointer) searchEl.value?.focus()
})
</script>
