<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Purchases</h1>
        <p class="mt-1 text-xs text-muted">Goods bought from suppliers. Stock goes in once a purchase is approved.</p>
      </div>
      <button v-if="canCreate" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> New purchase</button>
    </header>

    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by number, supplier or note…" /></div>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:w-[560px]">
        <SearchSelect v-if="showBranchFilter" v-model="branchFilter" :options="branchOptions" aria-label="Branch" />
        <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Status" />
        <DateRangePicker v-model="dateRange" />
      </div>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading purchases…</div>
    <AppAlert v-else-if="error" title="Couldn’t load purchases" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="purchase in items" :key="purchase.id" :class="purchase.status === 'pending' ? 'bg-amber-50/30' : ''">
        <button type="button" class="flex w-full flex-col gap-2 p-4 text-left hover:bg-canvas sm:flex-row sm:items-center" @click="detail = purchase">
          <div class="flex min-w-0 flex-1 items-start gap-3">
            <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-blue-50 text-primary"><Truck :size="18" /></span>
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2"><strong class="font-mono text-sm text-navy">{{ purchase.number }}</strong><InventoryStatusBadge :status="purchase.status" /></div>
              <p class="mt-0.5 truncate text-xs text-muted">{{ formatDocDate(purchase.purchase_date) }} · {{ purchase.branch_code }}<template v-if="purchase.supplier"> · {{ purchase.supplier }}</template> · {{ purchase.lines.length }} {{ purchase.lines.length === 1 ? 'item' : 'items' }}</p>
              <p class="mt-0.5 text-[11px] text-slate-400">by {{ purchase.created_by }}<template v-if="purchase.decided_by"> · {{ purchase.status === 'rejected' ? 'rejected' : 'approved' }} by {{ purchase.decided_by }}</template></p>
            </div>
          </div>
          <div class="flex items-center justify-between gap-3 sm:justify-end">
            <p class="text-base font-bold" :class="purchase.status === 'voided' || purchase.status === 'rejected' ? 'text-slate-400 line-through' : 'text-navy'">{{ formatMoney(purchase.total, purchase.currency) }}</p>
            <span v-if="purchase.can_approve" class="rounded-lg bg-emerald-600 px-2.5 py-1 text-[11px] font-bold text-white">Review</span>
          </div>
        </button>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">No purchases found.</li>
    </ul>
    <InventoryPager v-model:page="page" :page-size="pageSize" :total="total" />

    <!-- Details -->
    <AppModal :model-value="!!detail" :title="detail?.number || ''" eyebrow="PURCHASE" :subtitle="detail ? `${formatDocDate(detail.purchase_date)} · ${detail.branch_code} — ${detail.branch_name}` : ''" width="640px" @update:model-value="(open) => { if (!open) detail = null }">
      <template #icon><Truck :size="20" /></template>
      <template v-if="detail">
        <div class="flex flex-wrap items-center gap-2 text-xs text-muted">
          <InventoryStatusBadge :status="detail.status" />
          <span v-if="detail.supplier">Supplier: <strong class="text-navy">{{ detail.supplier }}</strong></span>
          <span v-if="detail.payment_method">· Paid by {{ detail.payment_method }}</span>
          <span>· by {{ detail.created_by }}</span>
        </div>
        <InventoryDocLines class="mt-3" :lines="detail.lines" :currency="detail.currency" price-field="unit_cost" />
        <div class="mt-3 space-y-1 text-right text-sm">
          <p class="text-xs text-muted">Subtotal {{ formatMoney(detail.subtotal, detail.currency) }}<template v-if="Number(detail.discount)"> · Discount −{{ formatMoney(detail.discount, detail.currency) }}</template><template v-if="Number(detail.shipping)"> · Shipping +{{ formatMoney(detail.shipping, detail.currency) }}</template></p>
          <p>Total <strong class="ml-2 text-navy">{{ formatMoney(detail.total, detail.currency) }}</strong></p>
          <p v-if="detail.currency !== 'USD'" class="text-xs text-muted">≈ {{ formatMoney(detail.total_base, 'USD') }} at 1 USD = {{ formatMoney(detail.exchange_rate, detail.currency) }}</p>
        </div>
        <div v-if="detail.note" class="rich-text-content mt-3 rounded-lg bg-canvas px-3 py-2 text-xs text-muted" v-html="sanitizeRichText(detail.note)"></div>
        <p v-if="detail.decided_by" class="mt-3 rounded-lg px-3 py-2 text-xs" :class="detail.status === 'rejected' ? 'bg-rose-50 text-rose-700' : 'bg-emerald-50 text-emerald-700'">
          {{ detail.status === 'rejected' ? 'Rejected' : 'Approved' }} by {{ detail.decided_by }} · {{ formatDocDateTime(detail.decided_at) }}<template v-if="detail.decision_note"><br>{{ detail.status === 'rejected' ? 'Reason' : 'Note' }}: {{ detail.decision_note }}</template>
        </p>
        <p v-if="detail.status === 'voided'" class="mt-3 rounded-lg bg-slate-100 px-3 py-2 text-xs text-slate-600">Voided by {{ detail.voided_by }} · {{ formatDocDateTime(detail.voided_at) }}<br>Reason: {{ detail.void_reason }}</p>
      </template>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" @click="detail = null">Close</button>
        <button v-if="detail?.can_void" type="button" class="border border-rose-200 bg-white text-expense" @click="openAction('void')">Void</button>
        <button v-if="detail?.can_approve" type="button" class="border border-rose-200 bg-white text-expense" @click="openAction('reject')">Reject</button>
        <button v-if="detail?.can_approve" type="button" class="bg-emerald-600 text-white" @click="openAction('approve')">Approve</button>
      </template>
    </AppModal>

    <InventoryReasonDialog v-model="actionOpen" :title="actionCopy.title" :eyebrow="actionCopy.eyebrow" :subtitle="actionCopy.subtitle" :label="action === 'approve' ? 'Note' : 'Reason'" :required="action !== 'approve'" :confirm-label="actionCopy.confirm" :danger="action !== 'approve'" :busy="acting" :server-error="actionError" @confirm="runAction" />
  </section>
</template>

<script setup lang="ts">
import { Plus, Truck } from 'lucide-vue-next'
import type { Paged, Purchase } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { formatMoney } from '~/utils/money'
import { sanitizeRichText } from '~/utils/richText'
import { formatDocDate, formatDocDateTime } from '~/utils/inventory'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canCreate = computed(() => can('purchases.create'))
const { options: branchOptions, showFilter: showBranchFilter } = useBranchOptions('All branches')
const statusOptions = [{ value: '', label: 'All statuses' }, { value: 'pending', label: 'Waiting for approval' }, { value: 'approved', label: 'Approved' }, { value: 'rejected', label: 'Rejected' }, { value: 'voided', label: 'Voided' }]

const items = ref<Purchase[]>([])
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
    const result = await api<Paged<Purchase>>('/inventory/purchases', {
      query: { page: page.value, page_size: pageSize, search: search.value || undefined, branch_id: branchFilter.value || undefined, status: statusFilter.value || undefined, date_from: dateRange.value.from || undefined, date_to: dateRange.value.to || undefined },
    })
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load purchases.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([search, branchFilter, statusFilter, dateRange], () => { page.value = 1; load() }, { deep: true })
watch(page, load)
onMounted(load)

// --- new purchase: its own page (the list's branch goes along) ---
function openCreate() {
  navigateTo({ path: '/inventory/purchases/new', query: branchFilter.value ? { branch: branchFilter.value } : {} })
}

// Back from a new purchase: show it.
const route = useRoute()
async function openFromQuery() {
  const id = String(route.query.open || '')
  if (!id) return
  try {
    detail.value = await api<Purchase>(`/inventory/purchases/${id}`)
  } catch {
    // It may be in another branch filter; the list still shows it.
  }
  navigateTo({ path: route.path, query: {} }, { replace: true })
}
onMounted(openFromQuery)

// --- details: approve, reject, void ---
const detail = ref<Purchase | null>(null)
const action = ref<'approve' | 'reject' | 'void'>('approve')
const actionOpen = ref(false)
const acting = ref(false)
const actionError = ref('')
const actionCopy = computed(() => ({
  approve: { title: `Approve ${detail.value?.number || ''}?`, eyebrow: 'APPROVE PURCHASE', subtitle: 'The goods go into the branch’s stock.', confirm: 'Approve' },
  reject: { title: `Reject ${detail.value?.number || ''}?`, eyebrow: 'REJECT PURCHASE', subtitle: 'No stock changes. The person who entered it sees your reason.', confirm: 'Reject' },
  void: { title: `Void ${detail.value?.number || ''}?`, eyebrow: 'VOID PURCHASE', subtitle: detail.value?.status === 'approved' ? 'Its goods are taken back out of stock (they must still be there).' : 'It is cancelled; no stock changes.', confirm: 'Void' },
}[action.value]))

function openAction(next: 'approve' | 'reject' | 'void') {
  action.value = next
  actionError.value = ''
  actionOpen.value = true
}

async function runAction(text: string) {
  if (!detail.value) return
  acting.value = true
  actionError.value = ''
  try {
    const body = action.value === 'void' ? { reason: text } : { note: text }
    detail.value = await api<Purchase>(`/inventory/purchases/${detail.value.id}/${action.value}`, { method: 'POST', body })
    actionOpen.value = false
    toast.success({ approve: 'The goods are in stock.', reject: 'No stock was added.', void: 'The purchase is cancelled.' }[action.value], { title: { approve: 'Purchase approved', reject: 'Purchase rejected', void: 'Purchase voided' }[action.value] })
    await load()
  } catch (e: any) {
    actionError.value = e?.data?.error || 'Could not save this.'
  } finally {
    acting.value = false
  }
}
</script>
