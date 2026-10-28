<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Stock</h1>
        <p class="mt-1 text-xs text-muted">What each of your branches has now, valued at average cost.</p>
      </div>
      <button v-if="canAdjust" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openAdjust()"><SlidersHorizontal :size="16" /> Adjust stock</button>
    </header>

    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by SKU, name or barcode…" /></div>
      <div class="grid grid-cols-2 gap-2 lg:w-[420px]">
        <SearchSelect v-if="showBranchFilter" v-model="branchFilter" :options="branchOptions" aria-label="Branch" />
        <SearchSelect v-model="categoryFilter" :options="categoryOptions" aria-label="Category" />
      </div>
      <label class="inline-flex items-center gap-2 whitespace-nowrap text-xs font-semibold text-navy"><input v-model="lowOnly" type="checkbox" class="accent-primary" /> Low stock only</label>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading stock…</div>
    <AppAlert v-else-if="error" title="Couldn’t load stock" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="row in items" :key="row.branch_id + row.product_id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
        <div class="min-w-0 flex-1">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="text-sm font-semibold text-navy">{{ row.name }}</h2>
            <span v-if="row.low" class="rounded-full bg-amber-50 px-2 py-0.5 text-[10px] font-bold uppercase text-amber-700">Low stock</span>
          </div>
          <p class="mt-0.5 text-xs text-muted"><span class="font-mono">{{ row.sku }}</span> · {{ row.branch_code }} — {{ row.branch_name }}<template v-if="row.category_name"> · {{ row.category_name }}</template></p>
        </div>
        <div class="flex flex-wrap items-center justify-between gap-4 sm:justify-end">
          <div class="text-left sm:text-right">
            <p class="text-base font-bold" :class="row.low ? 'text-amber-700' : 'text-navy'">{{ formatQty(row.quantity) }} <span class="text-xs font-semibold text-muted">{{ row.unit }}</span></p>
            <p class="text-[11px] text-muted">Avg cost {{ formatMoney(row.avg_cost, row.currency) }} · Value {{ formatMoney(row.value, row.currency) }}</p>
          </div>
          <div class="flex gap-1.5">
            <button type="button" class="action-chip action-edit" @click="openHistory(row)"><History :size="13" /> History</button>
            <button v-if="canAdjust" type="button" class="action-chip action-edit" @click="openAdjust(row)"><SlidersHorizontal :size="13" /> Adjust</button>
          </div>
        </div>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">{{ lowOnly ? 'Nothing is low on stock.' : 'No stock yet. It appears here once purchases are approved or stock is added.' }}</li>
    </ul>
    <InventoryPager v-model:page="page" :page-size="pageSize" :total="total" />

    <AppModal :model-value="adjustOpen" form title="Adjust stock" eyebrow="STOCK" subtitle="Add or remove stock, or set the counted quantity. Every change keeps its reason." :busy="saving" width="560px" @update:model-value="(open) => { if (!open && !saving) adjustOpen = false }" @submit="saveAdjust">
      <template #icon><SlidersHorizontal :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="space-y-1 text-xs font-medium text-muted">Branch<SearchSelect v-model="adjust.branch_id" :options="branchChoices" placeholder="Choose a branch" aria-label="Branch" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">Product<SearchSelect v-model="adjust.product_id" :options="productChoices" placeholder="Choose a product" search-placeholder="Search by SKU or name…" aria-label="Product" /></div>
      </div>
      <p v-if="adjustProduct" class="mt-2 rounded-lg bg-canvas px-3 py-2 text-xs text-muted">In stock now: <strong class="text-navy">{{ formatQty(adjustProduct.quantity) }} {{ adjustProduct.unit }}</strong> · average cost {{ formatMoney(adjustProduct.avg_cost, adjustProduct.currency) }}</p>
      <div class="mt-3 grid grid-cols-3 gap-2" role="radiogroup" aria-label="Change">
        <label v-for="option in modeOptions" :key="option.value" class="flex cursor-pointer flex-col rounded-xl border px-3 py-2 text-xs" :class="adjust.mode === option.value ? 'border-primary bg-blue-50 ring-2 ring-blue-100' : 'border-border bg-white'">
          <input v-model="adjust.mode" type="radio" :value="option.value" class="sr-only" />
          <strong class="text-[13px] text-navy">{{ option.label }}</strong><small class="text-muted">{{ option.hint }}</small>
        </label>
      </div>
      <div class="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="space-y-1 text-xs font-medium text-muted">{{ adjust.mode === 'set' ? 'Counted quantity' : 'Quantity' }}<input v-model.trim="adjust.quantity" type="text" inputmode="decimal" :aria-invalid="adjustCheck.invalid('quantity')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="0" /></label>
        <label v-if="adjust.mode !== 'remove' && adjustProduct" class="space-y-1 text-xs font-medium text-muted">Unit cost of stock added <span class="font-normal">(optional)</span><MoneyInput v-model="adjust.unit_cost" :currency="adjustProduct.currency" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" aria-label="Unit cost" /></label>
        <label class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Reason<input v-model.trim="adjust.reason" maxlength="512" :aria-invalid="adjustCheck.invalid('reason')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. Opening stock, damaged, monthly count" /></label>
      </div>
      <AppAlert v-if="adjustError || adjustCheck.messages.length" class="mt-3" :title="adjustError ? 'Couldn’t adjust the stock' : 'Check the highlighted fields'" :message="adjustError" :items="adjustCheck.messages" dismissible @dismiss="adjustError = ''; adjustCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="adjustOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
      </template>
    </AppModal>

    <AppModal v-model="historyOpen" :title="historyRow ? `${historyRow.name} · ${historyRow.branch_code}` : 'History'" eyebrow="STOCK HISTORY" subtitle="Every change to this stock, newest first." width="640px">
      <template #icon><History :size="20" /></template>
      <div v-if="historyLoading" class="py-6 text-center text-sm text-muted">Loading…</div>
      <AppAlert v-else-if="historyError" compact :message="historyError" />
      <ul v-else class="divide-y divide-border rounded-xl border border-border">
        <li v-for="move in moves" :key="move.id" class="flex items-start justify-between gap-3 px-3 py-2.5">
          <div class="min-w-0">
            <p class="text-sm font-semibold text-navy">{{ MOVEMENT_LABELS[move.type] || move.type }} <span v-if="move.ref_number" class="font-mono text-[11px] font-normal text-muted">{{ move.ref_number }}</span></p>
            <p class="text-[11px] text-muted">{{ formatDocDateTime(move.created_at) }}<template v-if="move.created_by"> · {{ move.created_by }}</template></p>
            <p v-if="move.note" class="mt-0.5 break-words text-[11px] text-slate-500">{{ move.note }}</p>
          </div>
          <div class="shrink-0 text-right">
            <p class="text-sm font-bold" :class="Number(move.quantity) < 0 ? 'text-expense' : 'text-income'">{{ Number(move.quantity) > 0 ? '+' : '' }}{{ formatQty(move.quantity) }}</p>
            <p class="text-[11px] text-muted">Balance {{ formatQty(move.balance_after) }}</p>
          </div>
        </li>
        <li v-if="!moves.length" class="p-4 text-center text-xs text-muted">No changes yet.</li>
      </ul>
      <InventoryPager v-model:page="historyPage" class="mt-3" :page-size="20" :total="historyTotal" />
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { History, SlidersHorizontal } from 'lucide-vue-next'
import type { Paged, Product, ProductCategory, StockMovement, StockRow } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { rule } from '~/utils/validation'
import { formatMoney } from '~/utils/money'
import { MOVEMENT_LABELS, formatDocDateTime, formatQty } from '~/utils/inventory'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canAdjust = computed(() => can('inventory.adjust'))
const { branches, options: branchOptions, showFilter: showBranchFilter } = useBranchOptions('All branches')
const branchChoices = computed(() => branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })))

const items = ref<StockRow[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 30
const loading = ref(true)
const error = ref('')
const search = ref('')
const branchFilter = ref('')
const categoryFilter = ref('')
const lowOnly = ref(false)
const categories = ref<ProductCategory[]>([])
const categoryOptions = computed(() => [{ value: '', label: 'All categories' }, ...categories.value.map(c => ({ value: c.id, label: c.name }))])

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<Paged<StockRow>>('/inventory/stock', {
      query: { page: page.value, page_size: pageSize, search: search.value || undefined, branch_id: branchFilter.value || undefined, category_id: categoryFilter.value || undefined, low: lowOnly.value ? 'true' : undefined },
    })
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load stock.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([search, branchFilter, categoryFilter, lowOnly], () => { page.value = 1; load() })
watch(page, load)
onMounted(async () => {
  load()
  try { categories.value = (await api<{ items: ProductCategory[] }>('/inventory/categories')).items || [] } catch { categories.value = [] }
})

// --- adjust ---
const modeOptions = [
  { value: 'add', label: 'Add', hint: 'Opening stock, found' },
  { value: 'remove', label: 'Remove', hint: 'Damaged, lost, used' },
  { value: 'set', label: 'Count', hint: 'Set what you counted' },
]
const adjustOpen = ref(false)
const saving = ref(false)
const adjustError = ref('')
const adjust = reactive({ branch_id: '', product_id: '', mode: 'add', quantity: '', unit_cost: '', reason: '' })
const branchProducts = ref<Product[]>([])
const productChoices = computed(() => branchProducts.value.map(p => ({ value: p.id, label: `${p.sku} — ${p.name} (${formatQty(p.quantity)} ${p.unit})` })))
const adjustProduct = computed(() => branchProducts.value.find(p => p.id === adjust.product_id))
// Same rules as the API (services.AdjustInput / Adjust).
const adjustCheck = reactive(useFormErrors(
  () => ({ ...adjust }),
  () => ({
    branch_id: [rule.required('branch')],
    product_id: [rule.required('product')],
    quantity: [rule.required('quantity'), (value: unknown) => {
      const ok = /^\d+(\.\d{1,4})?$/.test(String(value))
      return ok && (adjust.mode === 'set' || Number(value) > 0) ? '' : adjust.mode === 'set' ? 'the counted quantity must be 0 or more' : 'quantity must be a number above 0 (at most 4 decimals)'
    }],
    reason: [rule.required('reason'), rule.minLength('reason', 3), rule.maxLength('reason', 512)],
  }),
))

// Products with this branch's stock, for the picker.
async function loadBranchProducts() {
  if (!adjust.branch_id) { branchProducts.value = []; return }
  try {
    branchProducts.value = (await api<Paged<Product>>('/inventory/products', { query: { branch_id: adjust.branch_id, active: 'true', type: 'standard', page_size: 500 } })).items || []
  } catch {
    branchProducts.value = []
  }
}
watch(() => adjust.branch_id, loadBranchProducts)

function openAdjust(row?: StockRow) {
  Object.assign(adjust, { branch_id: row?.branch_id || branchFilter.value || (branches.value.length === 1 ? branches.value[0]!.id : ''), product_id: row?.product_id || '', mode: 'add', quantity: '', unit_cost: '', reason: '' })
  adjustError.value = ''
  adjustCheck.reset()
  adjustOpen.value = true
  loadBranchProducts()
}

async function saveAdjust() {
  adjustError.value = ''
  if (!adjustCheck.check()) return
  saving.value = true
  try {
    await api('/inventory/adjustments', { method: 'POST', body: { ...adjust, unit_cost: adjust.mode === 'remove' ? '' : adjust.unit_cost } })
    adjustOpen.value = false
    toast.success('The stock is updated.', { title: 'Stock adjusted' })
    await load()
  } catch (e: any) {
    if (!adjustCheck.fromApi(e)) adjustError.value = e?.data?.error || 'Could not adjust the stock.'
  } finally {
    saving.value = false
  }
}

// --- history ---
const historyOpen = ref(false)
const historyRow = ref<StockRow | null>(null)
const moves = ref<StockMovement[]>([])
const historyTotal = ref(0)
const historyPage = ref(1)
const historyLoading = ref(false)
const historyError = ref('')

async function loadHistory() {
  if (!historyRow.value) return
  historyLoading.value = true
  historyError.value = ''
  try {
    const result = await api<Paged<StockMovement>>('/inventory/movements', { query: { branch_id: historyRow.value.branch_id, product_id: historyRow.value.product_id, page: historyPage.value, page_size: 20 } })
    moves.value = result.items || []
    historyTotal.value = result.total || 0
  } catch (e: any) {
    historyError.value = e?.data?.error || 'Could not load the history.'
  } finally {
    historyLoading.value = false
  }
}
watch(historyPage, loadHistory)

function openHistory(row: StockRow) {
  historyRow.value = row
  historyOpen.value = true
  if (historyPage.value === 1) loadHistory()
  else historyPage.value = 1
}
</script>
