<template>
  <form class="space-y-4" novalidate @submit.prevent="save">
    <header class="flex min-w-0 items-center gap-3">
      <NuxtLink to="/inventory/quotations" class="grid h-9 w-9 shrink-0 place-items-center rounded-lg border border-border bg-white text-muted hover:text-navy" aria-label="Back to quotations"><ArrowLeft :size="17" /></NuxtLink>
      <div class="min-w-0">
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY · QUOTATIONS</p>
        <h1 class="text-xl font-bold text-navy">{{ original ? `Edit ${original.number}` : 'New quotation' }}</h1>
        <p class="mt-0.5 text-xs text-muted">{{ original && original.status !== 'open' ? 'Saving makes it open again (the customer hasn’t answered the new offer).' : 'A price offer to a customer: no stock moves until it is turned into a sale.' }}</p>
      </div>
    </header>

    <div v-if="loadingQuote" class="text-sm text-muted">Loading the quotation…</div>
    <AppAlert v-else-if="loadError" title="Couldn’t open this quotation" :message="loadError" />
    <!-- One full-width card, top to bottom: details, products, discount and shipping, note, totals. -->
    <div v-else class="space-y-5 rounded-xl border border-border bg-white p-4 sm:p-5">
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <div v-if="branches.length !== 1" class="space-y-1 text-xs font-medium text-muted">Branch<SearchSelect v-model="form.branch_id" :options="branchChoices" placeholder="Choose a branch" aria-label="Branch" :aria-invalid="formCheck.invalid('branch_id')" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">Customer<InventoryPartySelect v-model="form.customer_id" kind="customers" :branch-id="form.branch_id" :name="form.customer" placeholder="Choose or add a customer" aria-label="Customer" :invalid="!!formCheck.invalid('customer_id')" @pick="pickCustomer" /></div>
        <label class="block space-y-1 text-xs font-medium text-muted">Contact <span class="font-normal">(optional)</span><input v-model.trim="form.contact" maxlength="191" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Phone, email or address" /></label>
        <div class="space-y-1 text-xs font-medium text-muted">Quotation date<DatePicker v-model="form.quote_date" aria-label="Quotation date" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">
          Valid until <span class="font-normal">(optional)</span>
          <DatePicker v-model="form.valid_until" aria-label="Valid until" placeholder="No end date" />
          <div class="flex flex-wrap gap-1 pt-0.5">
            <button v-for="days in [7, 15, 30]" :key="days" type="button" class="rounded-md border px-2 py-0.5 text-[11px] font-semibold" :class="form.valid_until === addDaysISO(form.quote_date, days) ? 'border-primary bg-blue-50 text-primary' : 'border-border text-muted hover:text-navy'" @click="form.valid_until = addDaysISO(form.quote_date, days)">{{ days }} days</button>
            <button v-if="form.valid_until" type="button" class="px-2 py-0.5 text-[11px] font-semibold text-muted hover:text-navy" @click="form.valid_until = ''">No end date</button>
          </div>
        </div>
        <div class="space-y-1 text-xs font-medium text-muted">
          Currency<SearchSelect v-model="form.currency" :options="currencyOptions" :searchable="false" aria-label="Currency" />
          <p v-if="rateLabel(form.currency)" class="text-[11px] font-normal">{{ rateLabel(form.currency) }}</p>
        </div>
      </div>

      <section class="space-y-2">
        <h2 class="flex items-center gap-2 text-sm font-semibold text-navy"><FileText :size="16" /> Products</h2>
        <p v-if="!form.branch_id" class="rounded-lg bg-canvas px-3 py-3 text-xs text-muted">Choose a branch to see its products.</p>
        <InventorySaleLines v-else v-model="form.lines" :products="branchProducts" :currency="form.currency" quote />
      </section>

      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="space-y-1 text-xs font-medium text-muted">
          Discount <span class="font-normal">(10% = percent, 10 = amount)</span>
          <InventoryDiscountInput v-model:value="form.discount" v-model:mode="form.discount_mode" :currency="form.currency" aria-label="Discount on the quotation" :invalid="!!formCheck.invalid('discount')" />
        </div>
        <div class="space-y-1 text-xs font-medium text-muted">
          Shipping <span class="font-normal">(charged to the customer)</span>
          <MoneyInput v-model="form.shipping" :currency="form.currency" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" aria-label="Shipping" />
        </div>
      </div>

      <div class="space-y-1 text-xs font-medium text-muted">
        Note <span class="font-normal">(optional, printed on the quotation)</span>
        <RichText v-model="form.note" aria-label="Note" placeholder="Terms, delivery time, what the price includes…" />
      </div>

      <!-- The totals and save, on the right -->
      <div class="ml-auto w-full space-y-3 sm:max-w-md">
        <div class="space-y-1 rounded-xl bg-canvas p-3 text-sm">
          <div class="flex justify-between"><span class="text-muted">Subtotal</span><span class="text-navy">{{ formatMoney(gross, form.currency) }}</span></div>
          <div v-if="itemDiscounts" class="flex justify-between"><span class="text-muted">Item discounts</span><span class="text-expense">−{{ formatMoney(itemDiscounts, form.currency) }}</span></div>
          <div v-if="quoteDiscount" class="flex justify-between"><span class="text-muted">Discount</span><span class="text-expense">−{{ formatMoney(quoteDiscount, form.currency) }}</span></div>
          <div v-if="shipping" class="flex justify-between"><span class="text-muted">Shipping</span><span class="text-navy">+{{ formatMoney(shipping, form.currency) }}</span></div>
          <div class="flex items-baseline justify-between border-t border-border pt-1.5">
            <span class="font-semibold text-navy">Total</span>
            <span class="text-right"><strong class="text-lg text-navy">{{ formatMoney(total, form.currency) }}</strong><span v-if="otherCurrency" class="ml-1.5 text-xs text-muted">≈ {{ formatMoney(convert(total, form.currency, otherCurrency), otherCurrency) }}</span></span>
          </div>
        </div>

        <AppAlert v-if="formError || formCheck.messages.length" :title="formError ? 'Couldn’t save this quotation' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
        <div class="flex gap-2">
          <NuxtLink to="/inventory/quotations" class="flex-1 rounded-lg border border-border bg-white px-4 py-2.5 text-center text-sm font-semibold text-muted">Cancel</NuxtLink>
          <button type="submit" class="flex-1 rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : original ? 'Save changes' : 'Save quotation' }}</button>
        </div>
      </div>
    </div>
  </form>
</template>

<script setup lang="ts">
import { ArrowLeft, FileText } from 'lucide-vue-next'
import type { Currency, DiscountMode, DocLineForm, Paged, Party, Product, Quotation } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { branchLabel, useBranchOptions } from '~/composables/useBranchOptions'
import { useCurrency } from '~/composables/useCurrency'
import { rule } from '~/utils/validation'
import { formatMoney } from '~/utils/money'
import { addDaysISO, discountAmount, docDiscountCheck, lineDiscount, linesCheck, linesSubtotal, roundMoney, todayISO } from '~/utils/inventory'

/**
 * The quotation form: a new quotation, or (with quotationId) a change to one not yet turned into a
 * sale. It moves no stock, so it can offer more than the branch has.
 */
const props = defineProps<{ quotationId?: string }>()

const { api } = useApi()
const route = useRoute()
const toast = useToast()
const { branches } = useBranchOptions('All branches')
const branchChoices = computed(() => branches.value.map(b => ({ value: b.id, label: branchLabel(b) })))
const { active: activeCurrencies, currencyOptions, rateLabel, defaultCurrency, load: loadCurrency, convert } = useCurrency()

const saving = ref(false)
const saved = ref(false)
const original = ref<Quotation | null>(null)
const loadingQuote = ref(!!props.quotationId)
const loadError = ref('')
const formError = ref('')
const branchProducts = ref<Product[]>([])
const form = reactive({ branch_id: '', customer_id: '', customer: '', contact: '', quote_date: todayISO(), valid_until: addDaysISO(todayISO(), 15), currency: 'USD' as Currency, discount: '', discount_mode: 'amount' as DiscountMode, shipping: '', note: '', lines: [] as DocLineForm[] })

// Subtotal (before item discounts) − item discounts − discount + shipping = total.
const gross = computed(() => roundMoney(form.lines.reduce((sum, l) => sum + (Number(l.quantity) || 0) * (Number(l.price) || 0), 0), form.currency))
const subtotal = computed(() => linesSubtotal(form.lines, form.currency))
const itemDiscounts = computed(() => roundMoney(gross.value - subtotal.value, form.currency))
const quoteDiscount = computed(() => discountAmount(subtotal.value, form.discount, form.discount_mode, form.currency))
const shipping = computed(() => Number(form.shipping) || 0)
const total = computed(() => roundMoney(subtotal.value - quoteDiscount.value + shipping.value, form.currency))
// The total in the other currency: riel for a USD quotation, USD for any other.
const otherCurrency = computed(() => form.currency === 'USD' ? activeCurrencies.value.find(c => !c.base)?.code : 'USD')

// Same rules as the API (services.QuotationInput / checkQuotation).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    branch_id: [rule.required('branch')],
    customer_id: [(value: unknown) => !value && !form.customer ? 'choose the customer the quotation is for' : ''],
    currency: [rule.required('currency')],
    valid_until: [(value: unknown) => value && String(value) < form.quote_date ? 'the valid-until date can’t be before the quotation date' : ''],
    lines: [linesCheck],
    discount: [docDiscountCheck(() => form.discount_mode)],
  }),
))

async function loadBranchProducts() {
  if (!form.branch_id) { branchProducts.value = []; return }
  try {
    // Editing also loads turned-off products, so lines already on the quotation keep their names.
    const active = props.quotationId ? {} : { active: 'true' }
    branchProducts.value = (await api<Paged<Product>>('/inventory/products', { query: { branch_id: form.branch_id, ...active, page_size: 500 } })).items || []
  } catch {
    branchProducts.value = []
  }
}
// Another branch has other products and stock: the lines start again.
watch(() => form.branch_id, (to, from) => {
  if (from && to !== from) form.lines = []
  loadBranchProducts()
})
// Switching currency converts the prices already typed (at the exchange rate).
watch(() => form.currency, (to, from) => {
  if (!from || to === from || loadingQuote.value) return
  const money = (value: string) => value ? String(convert(value, from, to)) : value
  form.lines = form.lines.map(l => ({ ...l, price: money(l.price), discount: l.discount_mode === 'amount' ? money(l.discount) : l.discount }))
  form.shipping = money(form.shipping)
  if (form.discount_mode === 'amount') form.discount = money(form.discount)
})
// Picking a customer fills the contact with their phone and email (when it is empty).
function pickCustomer(p: Party | null) {
  form.customer = p?.name || ''
  if (p && !form.contact.trim()) form.contact = [p.phone, p.email].filter(Boolean).join(' · ')
}
// Moving the quotation date keeps the same validity (7, 15 or 30 days stay 7, 15 or 30 days).
watch(() => form.quote_date, (to, from) => {
  if (!from || !to || to === from || loadingQuote.value || !form.valid_until) return
  const days = [7, 15, 30].find(d => form.valid_until === addDaysISO(from, d))
  if (days) form.valid_until = addDaysISO(to, days)
})

onMounted(async () => {
  await loadCurrency()
  if (props.quotationId) {
    try {
      const quote = await api<Quotation>(`/inventory/quotations/${props.quotationId}`)
      if (!quote.can_edit) throw new Error(quote.status === 'converted' ? `This quotation was turned into sale ${quote.sale_number || ''}; it can’t be changed.`.replace(' ;', ';') : 'You can’t change this quotation.')
      original.value = quote
      Object.assign(form, {
        branch_id: quote.branch_id, customer_id: quote.customer_id || '', customer: quote.customer, contact: quote.contact || '', quote_date: quote.quote_date.slice(0, 10),
        valid_until: quote.valid_until ? quote.valid_until.slice(0, 10) : '', currency: quote.currency,
        discount: Number(quote.discount) ? String(Number(quote.discount)) : '', discount_mode: 'amount',
        shipping: Number(quote.shipping) ? String(Number(quote.shipping)) : '', note: quote.note || '',
        lines: quote.lines.map((l, i) => ({ key: i + 1, product_id: l.product_id, quantity: String(Number(l.quantity)), price: String(Number(l.unit_price)), discount: Number(l.discount) ? String(Number(l.discount)) : '', discount_mode: 'amount' as DiscountMode, note: l.note || '' })),
      })
      await nextTick() // let the watchers see the loaded quotation before prices may be converted
    } catch (e: any) {
      loadError.value = e?.data?.error || e?.message || 'Could not open this quotation.'
    } finally {
      loadingQuote.value = false
    }
    return
  }
  form.currency = defaultCurrency.value
  // The branch chosen on the list, or the only one the user works in.
  const fromList = String(route.query.branch || '')
  form.branch_id = fromList || (branches.value.length === 1 ? branches.value[0]!.id : '')
})
watch(branches, (list) => { if (!props.quotationId && !form.branch_id && list.length === 1) form.branch_id = list[0]!.id })

// Leaving with products on the quotation asks first.
onBeforeRouteLeave(() => {
  if (saved.value || !form.lines.length) return true
  return window.confirm(original.value ? 'Leave without saving your changes?' : 'Leave without saving this quotation?')
})

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const quote = await api<Quotation>(original.value ? `/inventory/quotations/${original.value.id}` : '/inventory/quotations', {
      method: original.value ? 'PATCH' : 'POST',
      body: {
        branch_id: form.branch_id, customer_id: form.customer_id, customer: form.customer, contact: form.contact, quote_date: form.quote_date, valid_until: form.valid_until,
        currency: form.currency, note: form.note, discount: String(quoteDiscount.value), shipping: form.shipping || '0',
        items: form.lines.map(l => ({ product_id: l.product_id, quantity: l.quantity, unit_price: l.price || '0', discount: String(lineDiscount(l, form.currency)), note: (l.note || '').trim() })),
      },
    })
    saved.value = true
    toast.success(`${quote.number} · ${quote.customer} · ${formatMoney(quote.total, quote.currency)}`, { title: original.value ? 'Quotation changed' : 'Quotation saved' })
    await navigateTo({ path: '/inventory/quotations', query: { open: quote.id } })
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this quotation.'
  } finally {
    saving.value = false
  }
}
</script>
