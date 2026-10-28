<template>
  <form class="space-y-4" novalidate @submit.prevent="save">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-3">
        <NuxtLink to="/inventory/sales" class="grid h-9 w-9 shrink-0 place-items-center rounded-lg border border-border bg-white text-muted hover:text-navy" aria-label="Back to sales"><ArrowLeft :size="17" /></NuxtLink>
        <div class="min-w-0">
          <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY · SALES</p>
          <h1 class="text-xl font-bold text-navy">New sale</h1>
          <p class="mt-0.5 text-xs text-muted">Stock goes out of the branch when you save.</p>
        </div>
      </div>
    </header>

    <!-- Phones: details, products, note, totals. Wide screens: products and note on the left; details and totals on the right. -->
    <div class="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_340px] lg:grid-rows-[auto_auto_1fr]">
      <div class="space-y-3 rounded-xl border border-border bg-white p-4 lg:col-start-2 lg:row-start-1">
        <h2 class="text-sm font-semibold text-navy">Details</h2>
        <div class="space-y-1 text-xs font-medium text-muted">Branch<SearchSelect v-model="form.branch_id" :options="branchChoices" placeholder="Choose a branch" aria-label="Branch" :aria-invalid="formCheck.invalid('branch_id')" /></div>
        <div class="grid grid-cols-2 gap-3">
          <div class="space-y-1 text-xs font-medium text-muted">Date<DatePicker v-model="form.sale_date" aria-label="Sale date" /></div>
          <div class="space-y-1 text-xs font-medium text-muted">Currency<SearchSelect v-model="form.currency" :options="currencyOptions" :searchable="false" aria-label="Currency" /></div>
        </div>
        <p v-if="rateLabel(form.currency)" class="-mt-1 text-[11px] text-muted">{{ rateLabel(form.currency) }}</p>
        <label class="block space-y-1 text-xs font-medium text-muted">Customer <span class="font-normal">(optional)</span><input v-model.trim="form.customer" maxlength="191" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Walk-in" /></label>
        <div class="space-y-1 text-xs font-medium text-muted">Payment<SearchSelect v-model="form.payment_method_id" :options="paymentOptions" placeholder="Choose" aria-label="Payment method" /></div>
      </div>

      <div class="min-w-0 rounded-xl border border-border bg-white p-4 lg:col-start-1 lg:row-span-2 lg:row-start-1">
        <h2 class="mb-3 flex items-center gap-2 text-sm font-semibold text-navy"><ShoppingCart :size="16" /> Products</h2>
        <p v-if="!form.branch_id" class="rounded-lg bg-canvas px-3 py-3 text-xs text-muted">Choose a branch to see its stock.</p>
        <InventoryPos v-else v-model="form.lines" :products="branchProducts" mode="sale" :currency="form.currency" />
      </div>

      <div class="min-w-0 rounded-xl border border-border bg-white p-4 lg:col-start-1 lg:row-start-3">
        <h2 class="mb-2 text-sm font-semibold text-navy">Note <span class="text-xs font-normal text-muted">(optional)</span></h2>
        <RichText v-model="form.note" aria-label="Note" placeholder="Anything to remember about this sale" />
      </div>

      <div class="space-y-4 lg:col-start-2 lg:row-start-2">
        <InventoryDocSummary v-if="form.branch_id" v-model:discount="form.discount" v-model:discount-mode="form.discount_mode" v-model:shipping="form.shipping" class="!max-w-none !bg-white" :currency="form.currency" :subtotal="subtotal" shipping-label="Shipping (charged)" />
        <AppAlert v-if="formError || formCheck.messages.length" :title="formError ? 'Couldn’t save this sale' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
        <div class="flex gap-2">
          <NuxtLink to="/inventory/sales" class="flex-1 rounded-lg border border-border bg-white px-4 py-2.5 text-center text-sm font-semibold text-muted">Cancel</NuxtLink>
          <button type="submit" class="flex-1 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save sale' }}</button>
        </div>
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
import { formatMoney } from '~/utils/money'
import { discountAmount, docDiscountCheck, lineDiscount, linesCheck, linesSubtotal, todayISO } from '~/utils/inventory'

const { api } = useApi()
const route = useRoute()
const toast = useToast()
const { branches } = useBranchOptions('All branches')
const branchChoices = computed(() => branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })))
const { options: paymentOptions, defaultId: defaultPayment, load: loadPayments } = usePaymentMethods()
const { currencyOptions, rateLabel, defaultCurrency, load: loadCurrency, convert } = useCurrency()

const saving = ref(false)
const saved = ref(false)
const formError = ref('')
const branchProducts = ref<Product[]>([])
const form = reactive({ branch_id: '', sale_date: todayISO(), currency: 'USD' as Currency, customer: '', payment_method_id: '', discount: '', discount_mode: 'amount' as DiscountMode, shipping: '', note: '', lines: [] as DocLineForm[] })
const subtotal = computed(() => linesSubtotal(form.lines, form.currency))

// Same rules as the API (services.SaleInput / CreateSale).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    branch_id: [rule.required('branch')],
    currency: [rule.required('currency')],
    lines: [linesCheck],
    discount: [docDiscountCheck(() => form.discount_mode)],
  }),
))

async function loadBranchProducts() {
  if (!form.branch_id) { branchProducts.value = []; return }
  try {
    branchProducts.value = (await api<Paged<Product>>('/inventory/products', { query: { branch_id: form.branch_id, active: 'true', page_size: 500 } })).items || []
  } catch {
    branchProducts.value = []
  }
}
// Another branch has other stock: the cart starts again.
watch(() => form.branch_id, (to, from) => {
  if (from) form.lines = []
  loadBranchProducts()
})
// Switching currency converts the prices already typed (at the exchange rate).
watch(() => form.currency, (to, from) => {
  if (!from || to === from) return
  const money = (value: string) => value ? String(convert(value, from, to)) : value
  form.lines = form.lines.map(l => ({ ...l, price: money(l.price), discount: l.discount_mode === 'amount' ? money(l.discount) : l.discount }))
  form.shipping = money(form.shipping)
  if (form.discount_mode === 'amount') form.discount = money(form.discount)
})

onMounted(async () => {
  await Promise.all([loadCurrency(), loadPayments()])
  form.currency = defaultCurrency.value
  form.payment_method_id = defaultPayment.value
  // The branch chosen on the list, or the only one the user works in.
  const fromList = String(route.query.branch || '')
  form.branch_id = fromList || (branches.value.length === 1 ? branches.value[0]!.id : '')
})
watch(branches, (list) => { if (!form.branch_id && list.length === 1) form.branch_id = list[0]!.id })

// Leaving with products in the cart asks first.
onBeforeRouteLeave(() => {
  if (saved.value || !form.lines.length) return true
  return window.confirm('Leave without saving this sale?')
})

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const sale = await api<Sale>('/inventory/sales', {
      method: 'POST',
      body: {
        branch_id: form.branch_id, sale_date: form.sale_date, currency: form.currency, customer: form.customer,
        payment_method_id: form.payment_method_id, note: form.note,
        discount: String(discountAmount(subtotal.value, form.discount, form.discount_mode, form.currency)), shipping: form.shipping || '0',
        items: form.lines.map(l => ({ product_id: l.product_id, quantity: l.quantity, unit_price: l.price || '0', discount: String(lineDiscount(l, form.currency)) })),
      },
    })
    saved.value = true
    toast.success(`${sale.number} · ${formatMoney(sale.total, sale.currency)}`, { title: 'Sale saved' })
    await navigateTo({ path: '/inventory/sales', query: { open: sale.id } })
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this sale.'
  } finally {
    saving.value = false
  }
}
</script>
