<template>
  <form class="max-w-4xl space-y-4" novalidate @submit.prevent="save">
    <header class="flex min-w-0 items-center gap-3">
      <NuxtLink to="/inventory/sales" class="grid h-9 w-9 shrink-0 place-items-center rounded-lg border border-border bg-white text-muted hover:text-navy" aria-label="Back to sales"><ArrowLeft :size="17" /></NuxtLink>
      <div class="min-w-0">
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY · SALES</p>
        <h1 class="text-xl font-bold text-navy">{{ original ? `Edit ${original.number}` : 'New sale' }}</h1>
        <p class="mt-0.5 text-xs text-muted">{{ original ? 'Saving puts the old goods back and takes the new lines out of stock.' : 'Stock goes out of the branch when you save.' }}</p>
      </div>
    </header>

    <div v-if="loadingSale" class="text-sm text-muted">Loading the sale…</div>
    <AppAlert v-else-if="loadError" title="Couldn’t open this sale" :message="loadError" />
    <!-- One card, top to bottom: details, products, discount and shipping, note, totals. -->
    <div v-else class="space-y-5 rounded-xl border border-border bg-white p-4 sm:p-5">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="space-y-1 text-xs font-medium text-muted">Branch<SearchSelect v-model="form.branch_id" :options="branchChoices" placeholder="Choose a branch" aria-label="Branch" :aria-invalid="formCheck.invalid('branch_id')" /></div>
        <label class="block space-y-1 text-xs font-medium text-muted">Customer <span class="font-normal">(optional)</span><input v-model.trim="form.customer" maxlength="191" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Walk-in" /></label>
        <div class="space-y-1 text-xs font-medium text-muted">Sale date<DatePicker v-model="form.sale_date" aria-label="Sale date" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">
          Due date <span class="font-normal">(optional)</span>
          <DatePicker v-model="form.due_date" aria-label="Due date" placeholder="No due date" />
          <div class="flex flex-wrap gap-1 pt-0.5">
            <button v-for="days in [7, 15, 30]" :key="days" type="button" class="rounded-md border px-2 py-0.5 text-[11px] font-semibold" :class="form.due_date === addDaysISO(form.sale_date, days) ? 'border-primary bg-blue-50 text-primary' : 'border-border text-muted hover:text-navy'" @click="form.due_date = addDaysISO(form.sale_date, days)">{{ days }} days</button>
            <button v-if="form.due_date" type="button" class="px-2 py-0.5 text-[11px] font-semibold text-muted hover:text-navy" @click="form.due_date = ''">No due date</button>
          </div>
        </div>
        <div class="space-y-1 text-xs font-medium text-muted">
          Currency<SearchSelect v-model="form.currency" :options="currencyOptions" :searchable="false" aria-label="Currency" />
          <p v-if="rateLabel(form.currency)" class="text-[11px] font-normal">{{ rateLabel(form.currency) }}</p>
        </div>
        <div class="space-y-1 text-xs font-medium text-muted">Payment<SearchSelect v-model="form.payment_method_id" :options="paymentOptions" placeholder="Choose" aria-label="Payment method" /></div>
      </div>

      <section class="space-y-2">
        <h2 class="flex items-center gap-2 text-sm font-semibold text-navy"><ShoppingCart :size="16" /> Products</h2>
        <p v-if="!form.branch_id" class="rounded-lg bg-canvas px-3 py-3 text-xs text-muted">Choose a branch to see its stock.</p>
        <InventorySaleLines v-else v-model="form.lines" :products="branchProducts" :currency="form.currency" :held="held" />
      </section>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="space-y-1 text-xs font-medium text-muted">
          Sale discount
          <span class="flex overflow-hidden rounded-lg border border-border bg-white focus-within:border-primary">
            <input v-model.trim="form.discount" type="text" inputmode="decimal" autocomplete="off" placeholder="0" class="min-w-0 flex-1 border-0 px-3 py-2 text-sm text-navy outline-none" aria-label="Discount on the sale" :aria-invalid="formCheck.invalid('discount')" />
            <button type="button" class="shrink-0 border-l border-border bg-canvas px-3 text-xs font-bold text-navy" :title="form.discount_mode === 'percent' ? 'Percent of the subtotal (tap for an amount)' : 'Amount (tap for a percent)'" @click="form.discount_mode = form.discount_mode === 'percent' ? 'amount' : 'percent'">{{ form.discount_mode === 'percent' ? '%' : symbol }}</button>
          </span>
        </div>
        <div class="space-y-1 text-xs font-medium text-muted">
          Shipping <span class="font-normal">(charged to the customer)</span>
          <MoneyInput v-model="form.shipping" :currency="form.currency" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" aria-label="Shipping" />
        </div>
      </div>

      <div class="space-y-1 text-xs font-medium text-muted">
        Note <span class="font-normal">(optional)</span>
        <RichText v-model="form.note" aria-label="Note" placeholder="Anything to remember about this sale" />
      </div>

      <div class="space-y-1 rounded-xl bg-canvas p-3 text-sm">
        <div class="flex justify-between"><span class="text-muted">Subtotal</span><span class="text-navy">{{ formatMoney(gross, form.currency) }}</span></div>
        <div v-if="itemDiscounts" class="flex justify-between"><span class="text-muted">Item discounts</span><span class="text-expense">−{{ formatMoney(itemDiscounts, form.currency) }}</span></div>
        <div v-if="saleDiscount" class="flex justify-between"><span class="text-muted">Sale discount</span><span class="text-expense">−{{ formatMoney(saleDiscount, form.currency) }}</span></div>
        <div v-if="shipping" class="flex justify-between"><span class="text-muted">Shipping</span><span class="text-navy">+{{ formatMoney(shipping, form.currency) }}</span></div>
        <div class="flex items-baseline justify-between border-t border-border pt-1.5">
          <span class="font-semibold text-navy">Total</span>
          <span class="text-right"><strong class="text-lg text-navy">{{ formatMoney(total, form.currency) }}</strong><span v-if="otherCurrency" class="ml-1.5 text-xs text-muted">≈ {{ formatMoney(convert(total, form.currency, otherCurrency), otherCurrency) }}</span></span>
        </div>
      </div>

      <AppAlert v-if="formError || formCheck.messages.length" :title="formError ? 'Couldn’t save this sale' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <div class="flex justify-end gap-2">
        <NuxtLink to="/inventory/sales" class="rounded-lg border border-border bg-white px-4 py-2.5 text-center text-sm font-semibold text-muted">Cancel</NuxtLink>
        <button type="submit" class="rounded-lg bg-primary px-5 py-2.5 text-sm font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : original ? 'Save changes' : 'Save sale' }}</button>
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ArrowLeft, ShoppingCart } from 'lucide-vue-next'
import type { Currency, DiscountMode, DocLineForm, Paged, Product, Sale } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { usePaymentMethods } from '~/composables/usePaymentMethods'
import { useCurrency } from '~/composables/useCurrency'
import { rule } from '~/utils/validation'
import { currencyMeta, formatMoney } from '~/utils/money'
import { addDaysISO, discountAmount, docDiscountCheck, lineDiscount, linesCheck, linesSubtotal, roundMoney, todayISO } from '~/utils/inventory'

/** The sale form: a new sale, or (with saleId) a change to a completed one. */
const props = defineProps<{ saleId?: string }>()

const { api } = useApi()
const route = useRoute()
const toast = useToast()
const { branches } = useBranchOptions('All branches')
const branchChoices = computed(() => branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })))
const { options: paymentOptions, defaultId: defaultPayment, load: loadPayments } = usePaymentMethods()
const { active: activeCurrencies, currencyOptions, rateLabel, defaultCurrency, load: loadCurrency, convert } = useCurrency()

const saving = ref(false)
const saved = ref(false)
const original = ref<Sale | null>(null)
const loadingSale = ref(!!props.saleId)
const loadError = ref('')
// Editing: the units this sale already took out of its branch are available to it again.
const held = computed<Record<string, number>>(() => {
  if (!original.value || original.value.branch_id !== form.branch_id) return {}
  const out: Record<string, number> = {}
  for (const l of original.value.lines) out[l.product_id] = (out[l.product_id] || 0) + Number(l.quantity)
  return out
})
const formError = ref('')
const branchProducts = ref<Product[]>([])
const form = reactive({ branch_id: '', sale_date: todayISO(), due_date: '', currency: 'USD' as Currency, customer: '', payment_method_id: '', discount: '', discount_mode: 'amount' as DiscountMode, shipping: '', note: '', lines: [] as DocLineForm[] })

// Subtotal (before item discounts) − item discounts − sale discount + shipping = total.
const symbol = computed(() => currencyMeta(form.currency).symbol.trim())
const gross = computed(() => roundMoney(form.lines.reduce((sum, l) => sum + (Number(l.quantity) || 0) * (Number(l.price) || 0), 0), form.currency))
const subtotal = computed(() => linesSubtotal(form.lines, form.currency))
const itemDiscounts = computed(() => roundMoney(gross.value - subtotal.value, form.currency))
const saleDiscount = computed(() => discountAmount(subtotal.value, form.discount, form.discount_mode, form.currency))
const shipping = computed(() => Number(form.shipping) || 0)
const total = computed(() => roundMoney(subtotal.value - saleDiscount.value + shipping.value, form.currency))
// The total in the other currency: riel for a USD sale, USD for any other.
const otherCurrency = computed(() => form.currency === 'USD' ? activeCurrencies.value.find(c => !c.base)?.code : 'USD')

// Same rules as the API (services.SaleInput / CreateSale).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    branch_id: [rule.required('branch')],
    currency: [rule.required('currency')],
    due_date: [(value: unknown) => value && String(value) < form.sale_date ? 'the due date can’t be before the sale date' : ''],
    lines: [linesCheck],
    discount: [docDiscountCheck(() => form.discount_mode)],
  }),
))

async function loadBranchProducts() {
  if (!form.branch_id) { branchProducts.value = []; return }
  try {
    // Editing also loads turned-off products, so lines already on the sale keep their names.
    const active = props.saleId ? {} : { active: 'true' }
    branchProducts.value = (await api<Paged<Product>>('/inventory/products', { query: { branch_id: form.branch_id, ...active, page_size: 500 } })).items || []
  } catch {
    branchProducts.value = []
  }
}
// Another branch has other stock: the cart starts again.
watch(() => form.branch_id, (to, from) => {
  if (from && to !== from) form.lines = []
  loadBranchProducts()
})
// Switching currency converts the prices already typed (at the exchange rate).
watch(() => form.currency, (to, from) => {
  if (!from || to === from || loadingSale.value) return
  const money = (value: string) => value ? String(convert(value, from, to)) : value
  form.lines = form.lines.map(l => ({ ...l, price: money(l.price), discount: l.discount_mode === 'amount' ? money(l.discount) : l.discount }))
  form.shipping = money(form.shipping)
  if (form.discount_mode === 'amount') form.discount = money(form.discount)
})

onMounted(async () => {
  await Promise.all([loadCurrency(), loadPayments()])
  if (props.saleId) {
    try {
      const sale = await api<Sale>(`/inventory/sales/${props.saleId}`)
      if (!sale.can_edit) throw new Error(sale.status === 'voided' ? 'A voided sale can’t be changed.' : 'You can’t change this sale.')
      original.value = sale
      Object.assign(form, {
        branch_id: sale.branch_id, sale_date: sale.sale_date.slice(0, 10), due_date: sale.due_date ? sale.due_date.slice(0, 10) : '', currency: sale.currency, customer: sale.customer,
        payment_method_id: sale.payment_method_id || defaultPayment.value, discount: Number(sale.discount) ? String(Number(sale.discount)) : '',
        discount_mode: 'amount', shipping: Number(sale.shipping) ? String(Number(sale.shipping)) : '', note: sale.note || '',
        lines: sale.lines.map((l, i) => ({ key: i + 1, product_id: l.product_id, quantity: String(Number(l.quantity)), price: String(Number(l.unit_price)), discount: Number(l.discount) ? String(Number(l.discount)) : '', discount_mode: 'amount' as DiscountMode, note: l.note || '' })),
      })
      await nextTick() // let the watchers see the loaded sale before prices may be converted
    } catch (e: any) {
      loadError.value = e?.data?.error || e?.message || 'Could not open this sale.'
    } finally {
      loadingSale.value = false
    }
    return
  }
  form.currency = defaultCurrency.value
  form.payment_method_id = defaultPayment.value
  // The branch chosen on the list, or the only one the user works in.
  const fromList = String(route.query.branch || '')
  form.branch_id = fromList || (branches.value.length === 1 ? branches.value[0]!.id : '')
})
watch(branches, (list) => { if (!props.saleId && !form.branch_id && list.length === 1) form.branch_id = list[0]!.id })

// Leaving with products in the cart asks first.
onBeforeRouteLeave(() => {
  if (saved.value || !form.lines.length) return true
  return window.confirm(original.value ? 'Leave without saving your changes?' : 'Leave without saving this sale?')
})

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const sale = await api<Sale>(original.value ? `/inventory/sales/${original.value.id}` : '/inventory/sales', {
      method: original.value ? 'PATCH' : 'POST',
      body: {
        branch_id: form.branch_id, sale_date: form.sale_date, due_date: form.due_date, currency: form.currency, customer: form.customer,
        payment_method_id: form.payment_method_id, note: form.note,
        discount: String(saleDiscount.value), shipping: form.shipping || '0',
        items: form.lines.map(l => ({ product_id: l.product_id, quantity: l.quantity, unit_price: l.price || '0', discount: String(lineDiscount(l, form.currency)), note: (l.note || '').trim() })),
      },
    })
    saved.value = true
    toast.success(`${sale.number} · ${formatMoney(sale.total, sale.currency)}`, { title: original.value ? 'Sale changed' : 'Sale saved' })
    await navigateTo({ path: '/inventory/sales', query: { open: sale.id } })
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this sale.'
  } finally {
    saving.value = false
  }
}
</script>
