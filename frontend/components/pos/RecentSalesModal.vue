<template>
  <AppModal :model-value="open" title="Today’s sales" eyebrow="POS" :subtitle="branchName" width="680px" @update:model-value="(value) => { if (!value) emit('close') }">
    <p v-if="loading && !items.length" class="py-8 text-center text-sm text-muted">Loading sales…</p>
    <AppAlert v-else-if="error" compact :message="error" />
    <p v-else-if="!items.length" class="py-8 text-center text-sm text-muted">No sales yet today.</p>
    <ul v-else class="divide-y divide-border rounded-xl border border-border">
      <li v-for="sale in items" :key="sale.id" class="flex items-center gap-3 px-3 py-2.5">
        <button type="button" class="min-w-0 flex-1 text-left" @click="emit('receipt', sale)">
          <span class="flex flex-wrap items-center gap-2"><strong class="font-mono text-sm text-navy">{{ sale.number }}</strong><InventoryStatusBadge :status="sale.status" /></span>
          <span class="block truncate text-xs text-muted">{{ timeOf(sale.created_at) }} · {{ itemCount(sale) }} · {{ sale.payment_method || 'Cash' }}<template v-if="sale.customer"> · {{ sale.customer }}</template></span>
        </button>
        <span class="shrink-0 text-sm font-semibold" :class="sale.status === 'voided' ? 'text-muted line-through' : 'text-navy'">{{ formatMoney(sale.total, sale.currency) }}</span>
        <button v-if="sale.can_void" type="button" class="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-muted hover:bg-rose-50 hover:text-expense" :aria-label="`Void ${sale.number}`" :title="`Void ${sale.number}`" @click="voiding = sale"><Ban :size="15" /></button>
      </li>
    </ul>
    <InventoryPager v-model:page="page" class="mt-3" :page-size="pageSize" :total="total" />

    <InventoryReasonDialog :model-value="!!voiding" :title="`Void ${voiding?.number || ''}?`" eyebrow="VOID SALE" subtitle="The goods go back into the branch's stock." confirm-label="Void sale" danger :busy="busy" :server-error="voidError" @update:model-value="(value) => { if (!value) voiding = null }" @confirm="voidSale" />
  </AppModal>
</template>

<script setup lang="ts">
import { Ban } from 'lucide-vue-next'
import type { Paged, Sale } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { formatMoney } from '~/utils/money'
import { formatQty, todayISO } from '~/utils/inventory'

/** Today's sales in the till's branch, newest first: reopen a receipt, or void a sale. */
const props = defineProps<{ open: boolean; branchId: string; branchName?: string }>()
const emit = defineEmits<{ close: []; receipt: [sale: Sale]; voided: [] }>()

const { api } = useApi()
const toast = useToast()
const items = ref<Sale[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 10
const loading = ref(false)
const error = ref('')
const voiding = ref<Sale | null>(null)
const busy = ref(false)
const voidError = ref('')

async function load() {
  if (!props.branchId) return
  loading.value = true
  error.value = ''
  try {
    const today = todayISO()
    const result = await api<Paged<Sale>>('/inventory/sales', { query: { branch_id: props.branchId, date_from: today, date_to: today, page: page.value, page_size: pageSize } })
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load today’s sales.'
  } finally {
    loading.value = false
  }
}
watch(() => props.open, (open) => {
  if (!open) return
  page.value = 1
  load()
})
watch(page, load)

function itemCount(sale: Sale) {
  const count = sale.lines.reduce((sum, l) => sum + Number(l.quantity || 0), 0)
  return `${formatQty(count)} ${count === 1 ? 'item' : 'items'}`
}

function timeOf(iso: string) {
  return new Date(iso).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
}

async function voidSale(reason: string) {
  if (!voiding.value) return
  busy.value = true
  voidError.value = ''
  try {
    const sale = await api<Sale>(`/inventory/sales/${voiding.value.id}/void`, { method: 'POST', body: { reason } })
    toast.success('Its goods are back in stock.', { title: `${sale.number} voided` })
    voiding.value = null
    emit('voided')
    await load()
  } catch (e: any) {
    voidError.value = e?.data?.error || 'Could not void this sale.'
  } finally {
    busy.value = false
  }
}
</script>
