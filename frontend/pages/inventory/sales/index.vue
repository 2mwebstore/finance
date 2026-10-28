<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Sales</h1>
        <p class="mt-1 text-xs text-muted">Goods sold by your branches. Stock goes out as soon as a sale is saved.</p>
      </div>
      <div v-if="canCreate" class="flex gap-2">
        <NuxtLink to="/inventory/pos" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy hover:bg-canvas"><Store :size="16" /> Open POS</NuxtLink>
        <button type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> New sale</button>
      </div>
    </header>

    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by number, customer or note…" /></div>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:w-[560px]">
        <SearchSelect v-if="showBranchFilter" v-model="branchFilter" :options="branchOptions" aria-label="Branch" />
        <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Status" />
        <DateRangePicker v-model="dateRange" />
      </div>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading sales…</div>
    <AppAlert v-else-if="error" title="Couldn’t load sales" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="sale in items" :key="sale.id">
        <button type="button" class="flex w-full flex-col gap-2 p-4 text-left hover:bg-canvas sm:flex-row sm:items-center" @click="detail = sale">
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-emerald-50 text-income"><ShoppingCart :size="18" /></span>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2"><strong class="font-mono text-sm text-navy">{{ sale.number }}</strong><InventoryStatusBadge :status="sale.status" /></div>
              <p class="mt-0.5 truncate text-xs text-muted">{{ formatDocDate(sale.sale_date) }} · {{ sale.branch_code }}<template v-if="sale.customer"> · {{ sale.customer }}</template><template v-if="sale.due_date"> · due {{ formatDocDate(sale.due_date) }}</template> · {{ sale.lines.length }} {{ sale.lines.length === 1 ? 'item' : 'items' }}</p>
              <p class="mt-0.5 text-[11px] text-slate-400">by {{ sale.created_by }}</p>
            </div>
          </div>
          <div class="text-right">
            <p class="text-base font-bold" :class="sale.status === 'voided' ? 'text-slate-400 line-through' : 'text-income'">{{ formatMoney(sale.total, sale.currency) }}</p>
            <p v-if="canSeeProfit && sale.status !== 'voided'" class="text-[11px] text-muted">Profit {{ formatMoney(sale.profit, 'USD') }}</p>
          </div>
        </button>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">No sales found.</li>
    </ul>
    <InventoryPager v-model:page="page" :page-size="pageSize" :total="total" />

    <!-- Details -->
    <AppModal :model-value="!!detail" :title="detail?.number || ''" eyebrow="SALE" :subtitle="detail ? `${formatDocDate(detail.sale_date)} · ${detail.branch_code} — ${detail.branch_name}` : ''" width="640px" @update:model-value="(open) => { if (!open) detail = null }">
      <template #icon><ShoppingCart :size="20" /></template>
      <template v-if="detail">
        <div class="flex flex-wrap items-center gap-2 text-xs text-muted">
          <InventoryStatusBadge :status="detail.status" />
          <span v-if="detail.customer">Customer: <strong class="text-navy">{{ detail.customer }}</strong></span>
          <span v-if="detail.payment_method">· Paid by {{ detail.payment_method }}</span>
          <span v-if="detail.due_date">· Due <strong class="text-navy">{{ formatDocDate(detail.due_date) }}</strong></span>
          <span>· by {{ detail.created_by }}</span>
        </div>
        <InventoryDocLines class="mt-3" :lines="detail.lines" :currency="detail.currency" price-field="unit_price" />
        <div class="mt-3 space-y-1 text-right text-sm">
          <p class="text-xs text-muted">Subtotal {{ formatMoney(detail.subtotal, detail.currency) }}<template v-if="Number(detail.discount)"> · Discount −{{ formatMoney(detail.discount, detail.currency) }}</template><template v-if="Number(detail.shipping)"> · Shipping +{{ formatMoney(detail.shipping, detail.currency) }}</template></p>
          <p>Total <strong class="ml-2 text-navy">{{ formatMoney(detail.total, detail.currency) }}</strong></p>
          <p v-if="detail.currency !== 'USD'" class="text-xs text-muted">≈ {{ formatMoney(detail.total_base, 'USD') }} at 1 USD = {{ formatMoney(detail.exchange_rate, detail.currency) }}</p>
          <p v-if="canSeeProfit" class="text-xs text-muted">Cost {{ formatMoney(detail.total_cost, 'USD') }} · Profit <strong :class="Number(detail.profit) < 0 ? 'text-expense' : 'text-income'">{{ formatMoney(detail.profit, 'USD') }}</strong></p>
        </div>
        <div v-if="detail.note" class="rich-text-content mt-3 rounded-lg bg-canvas px-3 py-2 text-xs text-muted" v-html="sanitizeRichText(detail.note)"></div>
        <p v-if="detail.status === 'voided'" class="mt-3 rounded-lg bg-slate-100 px-3 py-2 text-xs text-slate-600">Voided by {{ detail.voided_by }} · {{ formatDocDateTime(detail.voided_at) }}<br>Reason: {{ detail.void_reason }}</p>
      </template>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" @click="detail = null">Close</button>
        <button v-if="detail?.can_edit" type="button" class="border border-border bg-white text-navy" @click="navigateTo(`/inventory/sales/${detail.id}/edit`)">Edit</button>
        <button v-if="detail?.can_void" type="button" class="bg-expense text-white" @click="voidOpen = true">Void sale</button>
      </template>
    </AppModal>

    <InventoryReasonDialog v-model="voidOpen" :title="`Void ${detail?.number || ''}?`" eyebrow="VOID SALE" subtitle="The goods go back into the branch's stock." confirm-label="Void sale" danger :busy="voiding" :server-error="voidError" @confirm="voidSale" />
  </section>
</template>

<script setup lang="ts">
import { Plus, ShoppingCart, Store } from 'lucide-vue-next'
import type { Paged, Sale } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { formatMoney } from '~/utils/money'
import { sanitizeRichText } from '~/utils/richText'
import { formatDocDate, formatDocDateTime } from '~/utils/inventory'

const { api } = useApi()
const { can, canViewInventoryReports } = usePermissions()
const toast = useToast()
const canCreate = computed(() => can('sales.create'))
const canSeeProfit = canViewInventoryReports
const { options: branchOptions, showFilter: showBranchFilter } = useBranchOptions('All branches')
const statusOptions = [{ value: '', label: 'All statuses' }, { value: 'completed', label: 'Done' }, { value: 'voided', label: 'Voided' }]

const items = ref<Sale[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(true)
const error = ref('')
const search = ref('')
const branchFilter = ref('')
const statusFilter = ref('')
const dateRange = ref({ from: '', to: '' })

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<Paged<Sale>>('/inventory/sales', {
      query: { page: page.value, page_size: pageSize, search: search.value || undefined, branch_id: branchFilter.value || undefined, status: statusFilter.value || undefined, date_from: dateRange.value.from || undefined, date_to: dateRange.value.to || undefined },
    })
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load sales.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([search, branchFilter, statusFilter, dateRange], () => { page.value = 1; load() }, { deep: true })
watch(page, load)
onMounted(load)

// --- new sale: its own page (the list's branch goes along) ---
function openCreate() {
  navigateTo({ path: '/inventory/sales/new', query: branchFilter.value ? { branch: branchFilter.value } : {} })
}

// Back from a new sale: show it.
const route = useRoute()
async function openFromQuery() {
  const id = String(route.query.open || '')
  if (!id) return
  try {
    detail.value = await api<Sale>(`/inventory/sales/${id}`)
  } catch {
    // It may be in another branch filter; the list still shows it.
  }
  navigateTo({ path: route.path, query: {} }, { replace: true })
}
onMounted(openFromQuery)

// --- details & void ---
const detail = ref<Sale | null>(null)
const voidOpen = ref(false)
const voiding = ref(false)
const voidError = ref('')

async function voidSale(reason: string) {
  if (!detail.value) return
  voiding.value = true
  voidError.value = ''
  try {
    detail.value = await api<Sale>(`/inventory/sales/${detail.value.id}/void`, { method: 'POST', body: { reason } })
    voidOpen.value = false
    toast.success('Its goods are back in stock.', { title: 'Sale voided' })
    await load()
  } catch (e: any) {
    voidError.value = e?.data?.error || 'Could not void this sale.'
  } finally {
    voiding.value = false
  }
}
</script>
