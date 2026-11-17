<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Quotations</h1>
        <p class="mt-1 text-xs text-muted">Price offers to customers. No stock moves until a quotation is turned into a sale.</p>
      </div>
      <button v-if="canCreate" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> New quotation</button>
    </header>

    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by number, customer, contact or note…" /></div>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:w-[560px]">
        <SearchSelect v-if="showBranchFilter" v-model="branchFilter" :options="branchOptions" aria-label="Branch" />
        <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Status" />
        <DateRangePicker v-model="dateRange" />
      </div>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading quotations…</div>
    <AppAlert v-else-if="error" title="Couldn’t load quotations" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="quote in items" :key="quote.id">
        <button type="button" class="flex w-full flex-col gap-2 p-4 text-left hover:bg-canvas sm:flex-row sm:items-center" @click="detail = quote">
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-blue-50 text-primary"><FileText :size="18" /></span>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2"><strong class="font-mono text-sm text-navy">{{ quote.number }}</strong><InventoryStatusBadge :status="badgeOf(quote)" /></div>
              <p class="mt-0.5 truncate text-xs text-muted">{{ formatDocDate(quote.quote_date) }} · {{ quote.branch_name }} · <strong class="font-semibold text-navy">{{ quote.customer }}</strong> · {{ quote.lines.length }} {{ quote.lines.length === 1 ? 'item' : 'items' }}</p>
              <p class="mt-0.5 text-[11px] text-slate-400">
                by {{ quote.created_by }}
                <template v-if="quote.status === 'converted'"> · sale <span class="font-mono">{{ quote.sale_number }}</span></template>
                <template v-else-if="quote.valid_until"> · <span :class="quote.expired ? 'font-semibold text-amber-600' : ''">{{ quote.expired ? 'expired' : 'valid until' }} {{ formatDocDate(quote.valid_until) }}</span></template>
              </p>
            </div>
          </div>
          <p class="text-right text-base font-bold" :class="quote.status === 'declined' ? 'text-slate-400 line-through' : 'text-navy'">{{ formatMoney(quote.total, quote.currency) }}</p>
        </button>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">No quotations found.</li>
    </ul>
    <InventoryPager v-model:page="page" :page-size="pageSize" :total="total" />

    <!-- Details -->
    <AppModal :model-value="!!detail && !convertOpen && !deleteOpen" :title="detail?.number || ''" eyebrow="QUOTATION" :subtitle="detail ? `${formatDocDate(detail.quote_date)} · ${detail.branch_name}` : ''" width="640px" :busy="acting" @update:model-value="(open) => { if (!open) detail = null }">
      <template #icon><FileText :size="20" /></template>
      <template v-if="detail">
        <div class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted">
          <InventoryStatusBadge :status="badgeOf(detail)" />
          <span>Customer: <strong class="text-navy">{{ detail.customer }}</strong></span>
          <span v-if="detail.contact">· {{ detail.contact }}</span>
          <span v-if="detail.valid_until">· Valid until <strong :class="detail.expired ? 'text-amber-600' : 'text-navy'">{{ formatDocDate(detail.valid_until) }}</strong></span>
          <span>· by {{ detail.created_by }}</span>
        </div>
        <p v-if="detail.expired" class="mt-2 rounded-lg bg-amber-50 px-3 py-2 text-xs text-amber-700">Its valid-until day has passed. It can still be turned into a sale, or edited with a new date.</p>
        <InventoryDocLines class="mt-3" :lines="detail.lines" :currency="detail.currency" price-field="unit_price" />
        <div class="mt-3 space-y-1 text-right text-sm">
          <p class="text-xs text-muted">Subtotal {{ formatMoney(detail.subtotal, detail.currency) }}<template v-if="Number(detail.discount)"> · Discount −{{ formatMoney(detail.discount, detail.currency) }}</template><template v-if="Number(detail.shipping)"> · Shipping +{{ formatMoney(detail.shipping, detail.currency) }}</template></p>
          <p>Total <strong class="ml-2 text-navy">{{ formatMoney(detail.total, detail.currency) }}</strong></p>
          <p v-if="detail.currency !== 'USD'" class="text-xs text-muted">≈ {{ formatMoney(detail.total_base, 'USD') }} at 1 USD = {{ formatMoney(detail.exchange_rate, detail.currency) }}</p>
        </div>
        <div v-if="detail.note" class="rich-text-content mt-3 rounded-lg bg-canvas px-3 py-2 text-xs text-muted" v-html="sanitizeRichText(detail.note)"></div>
        <p v-if="detail.status === 'accepted' || detail.status === 'declined'" class="mt-3 rounded-lg px-3 py-2 text-xs" :class="detail.status === 'accepted' ? 'bg-emerald-50 text-emerald-700' : 'bg-rose-50 text-rose-700'">
          Marked {{ detail.status }} by {{ detail.decided_by || '—' }} · {{ formatDocDateTime(detail.decided_at) }}
        </p>
        <p v-if="detail.status === 'converted'" class="mt-3 rounded-lg bg-violet-50 px-3 py-2 text-xs text-violet-700">
          Turned into sale
          <NuxtLink v-if="canViewSales && detail.sale_id" :to="{ path: '/inventory/sales', query: { open: detail.sale_id } }" class="font-mono font-semibold underline">{{ detail.sale_number }}</NuxtLink>
          <span v-else class="font-mono font-semibold">{{ detail.sale_number }}</span>
          by {{ detail.converted_by || '—' }} · {{ formatDocDateTime(detail.converted_at) }}
        </p>
        <!-- The customer's answer, and the other actions -->
        <div class="mt-4 flex flex-wrap items-center gap-2 border-t border-border pt-3">
          <template v-if="detail.can_edit">
            <span class="text-[10px] font-bold uppercase tracking-wide text-slate-400">Customer’s answer</span>
            <button v-if="detail.status !== 'accepted'" type="button" class="action-chip border-emerald-200 text-emerald-700 hover:bg-emerald-50" :disabled="acting" @click="setStatus('accepted')"><Check :size="14" /> Accepted</button>
            <button v-if="detail.status !== 'declined'" type="button" class="action-chip border-rose-200 text-rose-700 hover:bg-rose-50" :disabled="acting" @click="setStatus('declined')"><X :size="14" /> Declined</button>
            <button v-if="detail.status !== 'open'" type="button" class="action-chip border-border text-navy hover:bg-canvas" :disabled="acting" @click="setStatus('open')"><RotateCcw :size="14" /> Open again</button>
          </template>
          <span class="flex-1"></span>
          <button type="button" class="action-chip border-border text-navy hover:bg-canvas" :disabled="acting" @click="print(detail)"><Printer :size="14" /> Print</button>
          <button v-if="detail.can_edit" type="button" class="action-chip border-border text-navy hover:bg-canvas" :disabled="acting" @click="navigateTo(`/inventory/quotations/${detail.id}/edit`)"><Pencil :size="14" /> Edit</button>
          <button v-if="detail.can_delete" type="button" class="action-chip border-rose-200 text-expense hover:bg-rose-50" :disabled="acting" @click="deleteError = ''; deleteOpen = true"><Trash2 :size="14" /> Delete</button>
        </div>
        <AppAlert v-if="actionError" class="mt-3" compact :message="actionError" />
      </template>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="acting" @click="detail = null">Close</button>
        <button v-if="detail?.can_convert" type="button" class="inline-flex items-center justify-center gap-1.5 bg-primary text-white" :disabled="acting" @click="openConvert"><ShoppingCart :size="15" /> Turn into sale</button>
      </template>
    </AppModal>

    <!-- Turn into a sale -->
    <AppModal :model-value="convertOpen" form :title="detail ? `Turn ${detail.number} into a sale?` : ''" eyebrow="NEW SALE" subtitle="A sale is made with the quotation's customer, products and prices; its goods go out of the branch's stock now." :busy="converting" width="480px" @update:model-value="(open) => { if (!open && !converting) convertOpen = false }" @submit="convert">
      <template #icon><ShoppingCart :size="20" /></template>
      <div v-if="detail" class="space-y-3">
        <p class="rounded-lg bg-canvas px-3 py-2 text-xs text-muted"><strong class="text-navy">{{ detail.customer }}</strong> · {{ detail.lines.length }} {{ detail.lines.length === 1 ? 'item' : 'items' }} · <strong class="text-navy">{{ formatMoney(detail.total, detail.currency) }}</strong></p>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div class="space-y-1 text-xs font-medium text-muted">Sale date<DatePicker v-model="convertForm.sale_date" aria-label="Sale date" /></div>
          <div class="space-y-1 text-xs font-medium text-muted">Payment<SearchSelect v-model="convertForm.payment_method_id" :options="paymentOptions" placeholder="Choose" aria-label="Payment method" /></div>
          <div class="space-y-1 text-xs font-medium text-muted sm:col-span-2">
            Due date <span class="font-normal">(optional)</span>
            <DatePicker v-model="convertForm.due_date" aria-label="Due date" placeholder="No due date" />
            <div class="flex flex-wrap gap-1 pt-0.5">
              <button v-for="days in [7, 15, 30]" :key="days" type="button" class="rounded-md border px-2 py-0.5 text-[11px] font-semibold" :class="convertForm.due_date === addDaysISO(convertForm.sale_date, days) ? 'border-primary bg-blue-50 text-primary' : 'border-border text-muted hover:text-navy'" @click="convertForm.due_date = addDaysISO(convertForm.sale_date, days)">{{ days }} days</button>
              <button v-if="convertForm.due_date" type="button" class="px-2 py-0.5 text-[11px] font-semibold text-muted hover:text-navy" @click="convertForm.due_date = ''">No due date</button>
            </div>
          </div>
        </div>
        <AppAlert v-if="convertError" compact :message="convertError" />
      </div>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="converting" @click="convertOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="converting">{{ converting ? 'Making the sale…' : 'Make the sale' }}</button>
      </template>
    </AppModal>

    <!-- Delete -->
    <AppModal :model-value="deleteOpen" form :title="detail ? `Delete ${detail.number}?` : ''" eyebrow="DELETE QUOTATION" subtitle="It leaves the list (the audit log keeps a record). No stock is affected." :busy="deleting" width="460px" @update:model-value="(open) => { if (!open && !deleting) deleteOpen = false }" @submit="remove">
      <AppAlert v-if="deleteError" compact :message="deleteError" />
      <p v-else-if="detail" class="text-sm text-muted">{{ detail.customer }} · {{ formatMoney(detail.total, detail.currency) }}</p>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="deleting" @click="deleteOpen = false">Cancel</button>
        <button type="submit" class="bg-expense text-white disabled:opacity-60" :disabled="deleting">{{ deleting ? 'Deleting…' : 'Delete quotation' }}</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { Check, FileText, Pencil, Plus, Printer, RotateCcw, ShoppingCart, Trash2, X } from 'lucide-vue-next'
import type { Paged, Quotation, Sale } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { usePaymentMethods } from '~/composables/usePaymentMethods'
import { formatMoney } from '~/utils/money'
import { sanitizeRichText } from '~/utils/richText'
import { addDaysISO, formatDocDate, formatDocDateTime, todayISO } from '~/utils/inventory'

const { api } = useApi()
const { can, canViewSales } = usePermissions()
const toast = useToast()
const canCreate = computed(() => can('quotations.create'))
const { options: branchOptions, showFilter: showBranchFilter } = useBranchOptions('All branches')
// Payment methods shared by every branch or the quotation's branch's own.
const { options: paymentOptions, load: loadPayments, keep: keepPayment } = usePaymentMethods(() => detail.value?.branch_id || '')
const statusOptions = [
  { value: '', label: 'All statuses' },
  { value: 'open', label: 'Open' },
  { value: 'expired', label: 'Expired' },
  { value: 'accepted', label: 'Accepted' },
  { value: 'declined', label: 'Declined' },
  { value: 'converted', label: 'Turned into sale' },
]

const items = ref<Quotation[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(true)
const error = ref('')
const search = ref('')
const branchFilter = ref('')
const statusFilter = ref('')
const dateRange = ref({ from: '', to: '' })

// An open quotation past its valid-until day shows as expired.
const badgeOf = (q: Quotation) => q.expired ? 'expired' : q.status

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<Paged<Quotation>>('/inventory/quotations', {
      query: { page: page.value, page_size: pageSize, search: search.value || undefined, branch_id: branchFilter.value || undefined, status: statusFilter.value || undefined, date_from: dateRange.value.from || undefined, date_to: dateRange.value.to || undefined },
    })
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load quotations.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([search, branchFilter, statusFilter, dateRange], () => { page.value = 1; load() }, { deep: true })
watch(page, load)
onMounted(load)

// --- new quotation: its own page (the list's branch goes along) ---
function openCreate() {
  navigateTo({ path: '/inventory/quotations/new', query: branchFilter.value ? { branch: branchFilter.value } : {} })
}

// Back from a new or changed quotation: show it.
const route = useRoute()
async function openFromQuery() {
  const id = String(route.query.open || '')
  if (!id) return
  try {
    detail.value = await api<Quotation>(`/inventory/quotations/${id}`)
  } catch {
    // It may be in another branch filter; the list still shows it.
  }
  navigateTo({ path: route.path, query: {} }, { replace: true })
}
onMounted(openFromQuery)

// --- details: status, print ---
const detail = ref<Quotation | null>(null)
const acting = ref(false)
const actionError = ref('')
watch(detail, () => { actionError.value = '' })

function replace(updated: Quotation) {
  detail.value = updated
  items.value = items.value.map(q => q.id === updated.id ? updated : q)
}

const statusToast: Record<string, string> = { accepted: 'Marked accepted', declined: 'Marked declined', open: 'Open again' }
async function setStatus(status: 'open' | 'accepted' | 'declined') {
  if (!detail.value) return
  acting.value = true
  actionError.value = ''
  try {
    const updated = await api<Quotation>(`/inventory/quotations/${detail.value.id}/status`, { method: 'POST', body: { status } })
    replace(updated)
    toast.success(`${updated.number} · ${updated.customer}`, { title: statusToast[status] })
    if (statusFilter.value) await load()
  } catch (e: any) {
    actionError.value = e?.data?.error || 'Could not change this quotation.'
  } finally {
    acting.value = false
  }
}

function print(quote: Quotation) {
  window.open(`/inventory/quotations/${quote.id}/print`, '_blank', 'noopener')
}

// --- turn into a sale ---
const convertOpen = ref(false)
const converting = ref(false)
const convertError = ref('')
const convertForm = reactive({ sale_date: todayISO(), due_date: '', payment_method_id: '' })

async function openConvert() {
  convertError.value = ''
  Object.assign(convertForm, { sale_date: todayISO(), due_date: '', payment_method_id: '' })
  convertOpen.value = true
  await loadPayments()
  convertForm.payment_method_id = keepPayment(convertForm.payment_method_id)
}

async function convert() {
  if (!detail.value) return
  if (convertForm.due_date && convertForm.due_date < convertForm.sale_date) {
    convertError.value = 'The due date can’t be before the sale date.'
    return
  }
  converting.value = true
  convertError.value = ''
  try {
    const sale = await api<Sale>(`/inventory/quotations/${detail.value.id}/convert`, { method: 'POST', body: { ...convertForm } })
    convertOpen.value = false
    detail.value = null
    toast.success(`${sale.number} · ${formatMoney(sale.total, sale.currency)}. Its goods went out of stock.`, { title: 'Sale made from the quotation' })
    if (canViewSales.value) await navigateTo({ path: '/inventory/sales', query: { open: sale.id } })
    else await load()
  } catch (e: any) {
    convertError.value = e?.data?.error || 'Could not make the sale.'
  } finally {
    converting.value = false
  }
}

// --- delete ---
const deleteOpen = ref(false)
const deleting = ref(false)
const deleteError = ref('')

async function remove() {
  if (!detail.value) return
  deleting.value = true
  deleteError.value = ''
  try {
    const gone = detail.value
    await api(`/inventory/quotations/${gone.id}`, { method: 'DELETE' })
    deleteOpen.value = false
    detail.value = null
    toast.success(`${gone.number} · ${gone.customer}`, { title: 'Quotation deleted' })
    await load()
  } catch (e: any) {
    deleteError.value = e?.data?.error || 'Could not delete this quotation.'
  } finally {
    deleting.value = false
  }
}
</script>

<style scoped>
.action-chip { display: inline-flex; align-items: center; gap: 5px; border-width: 1px; border-radius: 8px; background: #fff; padding: 5px 10px; font-size: 12px; font-weight: 600; }
.action-chip:disabled { opacity: 0.5; }
</style>
