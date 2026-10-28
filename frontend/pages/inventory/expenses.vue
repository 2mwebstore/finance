<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Inventory expenses</h1>
        <p class="mt-1 text-xs text-muted">Costs of the inventory side, like shipping, packaging and storage. They come off the sales profit in reports.</p>
      </div>
      <button v-if="canCreate" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> New expense</button>
    </header>

    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by number, payee or note…" /></div>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-4 lg:w-[680px]">
        <SearchSelect v-if="showBranchFilter" v-model="branchFilter" :options="branchOptions" aria-label="Branch" />
        <SearchSelect v-model="categoryFilter" :options="categoryFilterOptions" aria-label="Category" />
        <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Status" />
        <DateRangePicker v-model="dateRange" />
      </div>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading expenses…</div>
    <AppAlert v-else-if="error" title="Couldn’t load expenses" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="expense in items" :key="expense.id">
        <button type="button" class="flex w-full flex-col gap-2 p-4 text-left hover:bg-canvas sm:flex-row sm:items-center" @click="detail = expense">
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-rose-50 text-expense"><ReceiptText :size="18" /></span>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2"><strong class="text-sm text-navy">{{ expense.category_name || 'Expense' }}</strong><InventoryStatusBadge :status="expense.status" /></div>
              <p class="mt-0.5 truncate text-xs text-muted"><span class="font-mono">{{ expense.number }}</span> · {{ formatDocDate(expense.expense_date) }} · {{ expense.branch_code }}<template v-if="expense.payee"> · {{ expense.payee }}</template></p>
              <p class="mt-0.5 text-[11px] text-slate-400">by {{ expense.created_by }}<template v-if="expense.payment_method"> · {{ expense.payment_method }}</template></p>
            </div>
          </div>
          <p class="text-right text-base font-bold" :class="expense.status === 'voided' ? 'text-slate-400 line-through' : 'text-expense'">{{ formatMoney(expense.amount, expense.currency) }}</p>
        </button>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">No inventory expenses found.</li>
    </ul>
    <InventoryPager v-model:page="page" :page-size="pageSize" :total="total" />

    <AppModal :model-value="formOpen" form title="New inventory expense" eyebrow="INVENTORY EXPENSE" subtitle="Saved straight away; void it to cancel." :busy="saving" width="640px" @update:model-value="(open) => { if (!open && !saving) formOpen = false }" @submit="save">
      <template #icon><ReceiptText :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="space-y-1 text-xs font-medium text-muted">Branch<SearchSelect v-model="form.branch_id" :options="branchChoices" placeholder="Choose a branch" aria-label="Branch" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">Date<DatePicker v-model="form.expense_date" aria-label="Expense date" /></div>
        <div class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Category<SearchSelect v-model="form.category_id" :options="categoryChoices" placeholder="Choose a category" :aria-invalid="formCheck.invalid('category_id')" aria-label="Category" /></div>
        <p v-if="!categoryChoices.length" class="rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-800 sm:col-span-2">No inventory expense categories yet. Add them under Inventory → Expense categories.</p>
        <div class="space-y-1 text-xs font-medium text-muted">Currency<SearchSelect v-model="form.currency" :options="currencyOptions" :searchable="false" aria-label="Currency" /><small v-if="rateLabel(form.currency)" class="block font-normal">{{ rateLabel(form.currency) }}<template v-if="Number(form.amount)"> · ≈ {{ formatMoney(convert(form.amount, form.currency, 'USD'), 'USD') }}</template></small></div>
        <label class="space-y-1 text-xs font-medium text-muted">Amount<MoneyInput v-model="form.amount" :currency="form.currency" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" :aria-invalid="formCheck.invalid('amount')" aria-label="Amount" /></label>
        <div class="space-y-1 text-xs font-medium text-muted">Payment<SearchSelect v-model="form.payment_method_id" :options="paymentOptions" placeholder="Choose" aria-label="Payment method" /></div>
        <label class="space-y-1 text-xs font-medium text-muted">Paid to <span class="font-normal">(optional)</span><input v-model.trim="form.payee" maxlength="191" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. Delivery company" /></label>
      </div>
      <div class="mt-3 space-y-1 text-xs font-medium text-muted">Note <span class="font-normal">(optional)</span><RichText v-model="form.note" aria-label="Note" /></div>
      <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save this expense' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="formOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save expense' }}</button>
      </template>
    </AppModal>

    <AppModal :model-value="!!detail" :title="detail ? `${detail.category_name} · ${formatMoney(detail.amount, detail.currency)}` : ''" eyebrow="INVENTORY EXPENSE" :subtitle="detail ? `${detail.number} · ${formatDocDate(detail.expense_date)} · ${detail.branch_code} — ${detail.branch_name}` : ''" width="560px" @update:model-value="(open) => { if (!open) detail = null }">
      <template #icon><ReceiptText :size="20" /></template>
      <template v-if="detail">
        <div class="flex flex-wrap items-center gap-2 text-xs text-muted">
          <InventoryStatusBadge :status="detail.status" />
          <span v-if="detail.payee">Paid to <strong class="text-navy">{{ detail.payee }}</strong></span>
          <span v-if="detail.payment_method">· {{ detail.payment_method }}</span>
          <span>· by {{ detail.created_by }}</span>
        </div>
        <p v-if="detail.currency !== 'USD'" class="mt-2 text-xs text-muted">≈ {{ formatMoney(detail.amount_base, 'USD') }} at 1 USD = {{ formatMoney(detail.exchange_rate, detail.currency) }}</p>
        <div v-if="detail.note" class="rich-text-content mt-3 rounded-lg bg-canvas px-3 py-2 text-xs text-muted" v-html="sanitizeRichText(detail.note)"></div>
        <p v-if="detail.status === 'voided'" class="mt-3 rounded-lg bg-slate-100 px-3 py-2 text-xs text-slate-600">Voided by {{ detail.voided_by }} · {{ formatDocDateTime(detail.voided_at) }}<br>Reason: {{ detail.void_reason }}</p>
      </template>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" @click="detail = null">Close</button>
        <button v-if="detail?.can_void" type="button" class="bg-expense text-white" @click="voidOpen = true">Void expense</button>
      </template>
    </AppModal>

    <InventoryReasonDialog v-model="voidOpen" :title="`Void ${detail?.number || ''}?`" eyebrow="VOID EXPENSE" subtitle="It no longer counts in reports." confirm-label="Void expense" danger :busy="voiding" :server-error="voidError" @confirm="voidExpense" />
  </section>
</template>

<script setup lang="ts">
import { Plus, ReceiptText } from 'lucide-vue-next'
import type { Category, Currency, InventoryExpense, Paged } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { usePaymentMethods } from '~/composables/usePaymentMethods'
import { useCurrency } from '~/composables/useCurrency'
import { rule } from '~/utils/validation'
import { formatMoney } from '~/utils/money'
import { sanitizeRichText } from '~/utils/richText'
import { formatDocDate, formatDocDateTime, todayISO } from '~/utils/inventory'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canCreate = computed(() => can('inventory-expenses.create'))
const { branches, options: branchOptions, showFilter: showBranchFilter } = useBranchOptions('All branches')
const branchChoices = computed(() => branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })))
const { options: paymentOptions, defaultId: defaultPayment, load: loadPayments } = usePaymentMethods()
const { currencyOptions, rateLabel, defaultCurrency, load: loadCurrency, convert } = useCurrency()
const statusOptions = [{ value: '', label: 'All statuses' }, { value: 'completed', label: 'Done' }, { value: 'voided', label: 'Voided' }]

const categories = ref<Category[]>([])
const categoryFilterOptions = computed(() => [{ value: '', label: 'All categories' }, ...categories.value.map(c => ({ value: c.id, label: c.name }))])
// Active categories that are shared or belong to the chosen branch.
const categoryChoices = computed(() => categories.value
  .filter(c => c.is_active && (!c.branch_id || c.branch_id === form.branch_id))
  .map(c => ({ value: c.id, label: c.branch_id ? `${c.name} (${c.branch_name || 'branch'})` : c.name })))

const items = ref<InventoryExpense[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(true)
const error = ref('')
const search = ref('')
const branchFilter = ref('')
const categoryFilter = ref('')
const statusFilter = ref('')
const dateRange = ref({ from: '', to: '' })

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<Paged<InventoryExpense>>('/inventory/expenses', {
      query: { page: page.value, page_size: pageSize, search: search.value || undefined, branch_id: branchFilter.value || undefined, category_id: categoryFilter.value || undefined, status: statusFilter.value || undefined, date_from: dateRange.value.from || undefined, date_to: dateRange.value.to || undefined },
    })
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load expenses.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([search, branchFilter, categoryFilter, statusFilter, dateRange], () => { page.value = 1; load() }, { deep: true })
watch(page, load)
onMounted(async () => {
  load()
  try { categories.value = (await api<{ items: Category[] }>('/categories', { query: { type: 'inv_expense' } })).items || [] } catch { categories.value = [] }
})

// --- new expense ---
const formOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const form = reactive({ branch_id: '', category_id: '', expense_date: todayISO(), currency: 'USD' as Currency, amount: '', payment_method_id: '', payee: '', note: '' })
// Same rules as the API (services.InventoryExpenseInput / CreateExpense).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    branch_id: [rule.required('branch')],
    category_id: [rule.required('category')],
    currency: [rule.required('currency')],
    amount: [rule.required('amount'), rule.positiveAmount('amount')],
  }),
))
// A branch-only category doesn't fit another branch.
watch(() => form.branch_id, () => { if (form.category_id && !categoryChoices.value.some(c => c.value === form.category_id)) form.category_id = '' })

async function openCreate() {
  await loadCurrency()
  Object.assign(form, { branch_id: branchFilter.value || (branches.value.length === 1 ? branches.value[0]!.id : ''), category_id: '', expense_date: todayISO(), currency: defaultCurrency.value, amount: '', payment_method_id: defaultPayment.value, payee: '', note: '' })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
  await loadPayments()
  if (!form.payment_method_id) form.payment_method_id = defaultPayment.value
}

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const expense = await api<InventoryExpense>('/inventory/expenses', { method: 'POST', body: { ...form } })
    formOpen.value = false
    toast.success(`${expense.category_name} · ${formatMoney(expense.amount, expense.currency)}`, { title: 'Expense saved' })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this expense.'
  } finally {
    saving.value = false
  }
}

// --- details & void ---
const detail = ref<InventoryExpense | null>(null)
const voidOpen = ref(false)
const voiding = ref(false)
const voidError = ref('')

async function voidExpense(reason: string) {
  if (!detail.value) return
  voiding.value = true
  voidError.value = ''
  try {
    detail.value = await api<InventoryExpense>(`/inventory/expenses/${detail.value.id}/void`, { method: 'POST', body: { reason } })
    voidOpen.value = false
    toast.success('It no longer counts in reports.', { title: 'Expense voided' })
    await load()
  } catch (e: any) {
    voidError.value = e?.data?.error || 'Could not void this expense.'
  } finally {
    voiding.value = false
  }
}
</script>
