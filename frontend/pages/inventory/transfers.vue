<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Transfers</h1>
        <p class="mt-1 text-xs text-muted">Stock moved between branches, at the sending branch’s average cost.</p>
      </div>
      <button v-if="canTransfer && branches.length > 1" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> New transfer</button>
    </header>

    <div class="flex flex-col gap-2 lg:flex-row lg:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by number or note…" /></div>
      <div class="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:w-[560px]">
        <SearchSelect v-if="showBranchFilter" v-model="branchFilter" :options="branchOptions" aria-label="Branch" />
        <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Status" />
        <DateRangePicker v-model="dateRange" />
      </div>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading transfers…</div>
    <AppAlert v-else-if="error" title="Couldn’t load transfers" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="transfer in items" :key="transfer.id">
        <button type="button" class="flex w-full items-start gap-3 p-4 text-left hover:bg-canvas" @click="detail = transfer">
          <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl bg-violet-50 text-violet-600"><ArrowLeftRight :size="18" /></span>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2"><strong class="font-mono text-sm text-navy">{{ transfer.number }}</strong><InventoryStatusBadge :status="transfer.status" /></div>
            <p class="mt-0.5 text-sm font-semibold text-navy">{{ transfer.from_code }} <span class="text-muted">→</span> {{ transfer.to_code }}</p>
            <p class="mt-0.5 truncate text-xs text-muted">{{ formatDocDate(transfer.transfer_date) }} · {{ transfer.lines.length }} {{ transfer.lines.length === 1 ? 'item' : 'items' }} · by {{ transfer.created_by }}</p>
          </div>
        </button>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">No transfers found.</li>
    </ul>
    <InventoryPager v-model:page="page" :page-size="pageSize" :total="total" />

    <!-- New transfer -->
    <AppModal :model-value="formOpen" form title="New transfer" eyebrow="TRANSFER" subtitle="The goods leave one branch and arrive at the other right away." :busy="saving" width="680px" @update:model-value="(open) => { if (!open && !saving) formOpen = false }" @submit="save">
      <template #icon><ArrowLeftRight :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-3">
        <div class="space-y-1 text-xs font-medium text-muted">From<SearchSelect v-model="form.from_branch_id" :options="branchChoices" placeholder="Choose a branch" aria-label="From branch" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">To<SearchSelect v-model="form.to_branch_id" :options="toChoices" placeholder="Choose a branch" aria-label="To branch" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">Date<DatePicker v-model="form.transfer_date" aria-label="Transfer date" /></div>
      </div>
      <div class="mt-4">
        <p class="mb-2 text-sm font-semibold text-navy">Products</p>
        <p v-if="!form.from_branch_id" class="rounded-lg bg-canvas px-3 py-2 text-xs text-muted">Choose the sending branch to see its stock.</p>
        <InventoryPos v-else v-model="form.lines" :products="branchProducts" mode="transfer" />
      </div>
      <div class="mt-3 space-y-1 text-xs font-medium text-muted">Note <span class="font-normal">(optional)</span><RichText v-model="form.note" aria-label="Note" /></div>
      <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save this transfer' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="formOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Move stock' }}</button>
      </template>
    </AppModal>

    <!-- Details -->
    <AppModal :model-value="!!detail" :title="detail?.number || ''" eyebrow="TRANSFER" :subtitle="detail ? `${formatDocDate(detail.transfer_date)} · ${detail.from_code} → ${detail.to_code}` : ''" width="600px" @update:model-value="(open) => { if (!open) detail = null }">
      <template #icon><ArrowLeftRight :size="20" /></template>
      <template v-if="detail">
        <div class="flex flex-wrap items-center gap-2 text-xs text-muted">
          <InventoryStatusBadge :status="detail.status" />
          <span>{{ detail.from_code }} — {{ detail.from_name }} → {{ detail.to_code }} — {{ detail.to_name }}</span>
          <span>· by {{ detail.created_by }}</span>
        </div>
        <InventoryDocLines class="mt-3" :lines="detail.lines" />
        <div v-if="detail.note" class="rich-text-content mt-3 rounded-lg bg-canvas px-3 py-2 text-xs text-muted" v-html="sanitizeRichText(detail.note)"></div>
        <p v-if="detail.status === 'voided'" class="mt-3 rounded-lg bg-slate-100 px-3 py-2 text-xs text-slate-600">Voided by {{ detail.voided_by }} · {{ formatDocDateTime(detail.voided_at) }}<br>Reason: {{ detail.void_reason }}</p>
      </template>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" @click="detail = null">Close</button>
        <button v-if="detail?.can_void" type="button" class="bg-expense text-white" @click="voidOpen = true">Void transfer</button>
      </template>
    </AppModal>

    <InventoryReasonDialog v-model="voidOpen" :title="`Void ${detail?.number || ''}?`" eyebrow="VOID TRANSFER" subtitle="The goods go back to the sending branch (they must still be at the other one)." confirm-label="Void transfer" danger :busy="voiding" :server-error="voidError" @confirm="voidTransfer" />
  </section>
</template>

<script setup lang="ts">
import { ArrowLeftRight, Plus } from 'lucide-vue-next'
import type { DocLineForm, Paged, Product, StockTransfer } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { rule } from '~/utils/validation'
import { sanitizeRichText } from '~/utils/richText'
import { formatDocDate, formatDocDateTime, linesCheck, todayISO } from '~/utils/inventory'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canTransfer = computed(() => can('inventory.transfer'))
const { branches, options: branchOptions, showFilter: showBranchFilter } = useBranchOptions('All branches')
const branchChoices = computed(() => branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })))
const toChoices = computed(() => branchChoices.value.filter(b => b.value !== form.from_branch_id))
const statusOptions = [{ value: '', label: 'All statuses' }, { value: 'completed', label: 'Done' }, { value: 'voided', label: 'Voided' }]

const items = ref<StockTransfer[]>([])
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
    const result = await api<Paged<StockTransfer>>('/inventory/transfers', {
      query: { page: page.value, page_size: pageSize, search: search.value || undefined, branch_id: branchFilter.value || undefined, status: statusFilter.value || undefined, date_from: dateRange.value.from || undefined, date_to: dateRange.value.to || undefined },
    })
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load transfers.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([search, branchFilter, statusFilter, dateRange], () => { page.value = 1; load() }, { deep: true })
watch(page, load)
onMounted(load)

// --- new transfer ---
const formOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const branchProducts = ref<Product[]>([])
const form = reactive({ from_branch_id: '', to_branch_id: '', transfer_date: todayISO(), note: '', lines: [] as DocLineForm[] })

// Same rules as the API (services.TransferInput / CreateTransfer).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    from_branch_id: [rule.required('sending branch')],
    to_branch_id: [rule.required('receiving branch'), (value: unknown) => value && value === form.from_branch_id ? 'choose a different branch to send to' : ''],
    lines: [linesCheck],
  }),
))

async function loadBranchProducts() {
  if (!form.from_branch_id) { branchProducts.value = []; return }
  try {
    branchProducts.value = (await api<Paged<Product>>('/inventory/products', { query: { branch_id: form.from_branch_id, active: 'true', page_size: 500 } })).items || []
  } catch {
    branchProducts.value = []
  }
}
watch(() => form.from_branch_id, () => {
  if (form.to_branch_id === form.from_branch_id) form.to_branch_id = ''
  form.lines = []
  loadBranchProducts()
})

function openCreate() {
  Object.assign(form, { from_branch_id: '', to_branch_id: '', transfer_date: todayISO(), note: '', lines: [] })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const transfer = await api<StockTransfer>('/inventory/transfers', {
      method: 'POST',
      body: {
        from_branch_id: form.from_branch_id, to_branch_id: form.to_branch_id, transfer_date: form.transfer_date, note: form.note,
        items: form.lines.map(l => ({ product_id: l.product_id, quantity: l.quantity })),
      },
    })
    formOpen.value = false
    toast.success(`${transfer.from_code} → ${transfer.to_code}`, { title: `${transfer.number} done` })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this transfer.'
  } finally {
    saving.value = false
  }
}

// --- details & void ---
const detail = ref<StockTransfer | null>(null)
const voidOpen = ref(false)
const voiding = ref(false)
const voidError = ref('')

async function voidTransfer(reason: string) {
  if (!detail.value) return
  voiding.value = true
  voidError.value = ''
  try {
    detail.value = await api<StockTransfer>(`/inventory/transfers/${detail.value.id}/void`, { method: 'POST', body: { reason } })
    voidOpen.value = false
    toast.success('The goods are back at the sending branch.', { title: 'Transfer voided' })
    await load()
  } catch (e: any) {
    voidError.value = e?.data?.error || 'Could not void this transfer.'
  } finally {
    voiding.value = false
  }
}
</script>
