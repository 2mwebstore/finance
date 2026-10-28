<template>
  <section class="rounded-2xl border border-border bg-white p-4 shadow-sm">
    <header class="flex flex-wrap items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="grid h-8 w-8 place-items-center rounded-lg" :class="type === 'income' ? 'bg-emerald-50 text-income' : 'bg-rose-50 text-expense'"><ArrowDownLeft v-if="type === 'income'" :size="16" /><ArrowUpRight v-else :size="16" /></span>
        <div>
          <h2 class="text-sm font-bold text-navy">{{ type === 'income' ? 'Income' : 'Expense' }}</h2>
          <p class="text-[11px] text-muted">{{ type === 'income' ? 'Income' : 'Expense' }} transactions in this report’s dates{{ branchId ? ' and branch' : '' }}.</p>
        </div>
      </div>
      <button v-if="canExport" type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy disabled:opacity-50" :disabled="exporting || !total" @click="exportCsv"><Download :size="14" /> {{ exporting ? 'Exporting…' : 'Export table' }}</button>
    </header>

    <div class="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-3">
      <ListSearch v-model="search" placeholder="Number or description…" />
      <SearchSelect v-model="status" :options="statusOptions" :searchable="false" aria-label="Status" />
      <SearchSelect v-model="currency" :options="currencyOptions" :searchable="false" aria-label="Currency" />
    </div>

    <div v-if="loading" class="py-6 text-center text-sm text-muted">Loading transactions…</div>
    <AppAlert v-else-if="error" class="mt-3" compact :message="error" />
    <div v-else class="mt-3 overflow-x-auto rounded-xl border border-border">
      <table class="w-full min-w-[680px] text-left text-xs">
        <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
          <tr>
            <th class="px-3 py-2 font-bold">Date</th>
            <th class="px-3 py-2 font-bold">Number</th>
            <th v-if="type === 'expense'" class="px-3 py-2 font-bold">Category</th>
            <th class="px-3 py-2 font-bold">Branch</th>
            <th class="px-3 py-2 font-bold">By</th>
            <th class="px-3 py-2 font-bold">Status</th>
            <th class="px-3 py-2 text-right font-bold">Amount</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border text-navy">
          <tr v-for="t in items" :key="t.id">
            <td class="whitespace-nowrap px-3 py-2">{{ formatDocDate(t.transaction_date) }}</td>
            <td class="whitespace-nowrap px-3 py-2 font-mono text-[11px]">{{ t.transaction_number }}</td>
            <td v-if="type === 'expense'" class="px-3 py-2">{{ t.category?.name || '—' }}</td>
            <td class="whitespace-nowrap px-3 py-2">{{ t.branch?.code || '—' }}</td>
            <td class="px-3 py-2">{{ t.created_by?.name || '—' }}</td>
            <td class="px-3 py-2 capitalize">{{ t.status }}</td>
            <td class="whitespace-nowrap px-3 py-2 text-right font-semibold tabular-nums" :class="type === 'income' ? 'text-income' : 'text-expense'">{{ formatMoney(t.amount, t.currency as SupportedCurrency) }}</td>
          </tr>
          <tr v-if="!items.length"><td :colspan="columns" class="px-3 py-6 text-center text-muted">No {{ type }} transactions match.</td></tr>
        </tbody>
        <tfoot class="border-t-2 border-border bg-canvas">
          <tr v-for="(row, i) in footer" :key="row.currency">
            <td :colspan="columns - 1" class="px-3 py-2 text-right text-[11px] font-semibold text-slate-500">
              <template v-if="i === 0">Total {{ type }}{{ status && status !== 'approved' ? ' (approved only)' : '' }} · </template>{{ row.currency }}
            </td>
            <td class="whitespace-nowrap px-3 py-2 text-right text-sm font-bold tabular-nums" :class="type === 'income' ? 'text-income' : 'text-expense'">{{ formatMoney(row.amount, row.currency) }}</td>
          </tr>
          <tr v-if="!footer.length"><td :colspan="columns" class="px-3 py-2 text-right text-[11px] text-muted">Total {{ type }}: nothing approved</td></tr>
        </tfoot>
      </table>
    </div>
    <InventoryPager v-model:page="page" class="mt-3" :page-size="pageSize" :total="total" />
  </section>
</template>

<script setup lang="ts">
import { ArrowDownLeft, ArrowUpRight, Download } from 'lucide-vue-next'
import type { Paged, Transaction } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { formatMoney, type SupportedCurrency } from '~/utils/money'
import { formatDocDate } from '~/utils/inventory'
import { useCurrency } from '~/composables/useCurrency'

/**
 * One type of the report's transactions (income or expense) as a table, with its own filters,
 * paging, CSV export, and a footer with the approved total per currency.
 */
const props = defineProps<{ type: 'income' | 'expense'; dateFrom: string; dateTo: string; branchId: string }>()

const { api } = useApi()
const { canExportReports } = usePermissions()
const toast = useToast()
const canExport = canExportReports
const statusOptions = [{ value: 'approved', label: 'Approved' }, { value: '', label: 'Any status' }, { value: 'pending', label: 'Pending' }, { value: 'rejected', label: 'Rejected' }, { value: 'voided', label: 'Voided' }]
const { currencies: currencyList, options: currencyChoices } = useCurrency()
const currencyOptions = computed(() => [{ value: '', label: 'All currencies' }, ...currencyChoices(currency.value)])
const columns = computed(() => props.type === 'expense' ? 7 : 6)

const items = ref<Transaction[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(true)
const error = ref('')
const exporting = ref(false)
const search = ref('')
const status = ref('approved')
const currency = ref('')
// Approved totals per currency for the same filters (all pages), in Manage → Currency order.
const footer = ref<{ currency: string; amount: string }[]>([])

function query(pageNo: number, size: number) {
  return {
    page: pageNo, page_size: size, date_from: props.dateFrom || undefined, date_to: props.dateTo || undefined,
    branch_id: props.branchId || undefined, type: props.type, status: status.value || undefined,
    currency: currency.value || undefined, search: search.value || undefined,
  }
}

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const [result, totals] = await Promise.all([
      api<Paged<Transaction>>('/transactions', { query: query(page.value, pageSize) }),
      page.value === 1 ? api<{ by_currency?: Record<string, { income: string; expense: string }> }>('/transactions/totals', { query: totalsQuery() }).catch(() => null) : Promise.resolve(undefined),
    ])
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
    if (totals !== undefined) footer.value = footerRows(totals?.by_currency)
  } catch (e: any) {
    if (request !== loadRequest) return
    items.value = []
    total.value = 0
    error.value = e?.statusCode === 403 || e?.status === 403 ? 'You don’t have permission to see the transactions list.' : e?.data?.error || 'Could not load the transactions.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}

function totalsQuery() {
  return {
    date_from: props.dateFrom || undefined, date_to: props.dateTo || undefined, branch_id: props.branchId || undefined,
    type: props.type, currency: currency.value || undefined, search: search.value || undefined,
  }
}

function footerRows(by?: Record<string, { income: string; expense: string }>) {
  const order = currencyList.value.map(c => c.code)
  const rank = (code: string) => order.indexOf(code) === -1 ? order.length : order.indexOf(code)
  return Object.entries(by || {})
    .map(([code, flow]) => ({ currency: code, amount: props.type === 'income' ? flow.income : flow.expense }))
    .filter(row => Number(row.amount))
    .sort((a, b) => rank(a.currency) - rank(b.currency))
}

watch(() => [props.dateFrom, props.dateTo, props.branchId, search.value, status.value, currency.value], () => {
  if (page.value === 1) load()
  else page.value = 1
})
watch(page, load)
onMounted(load)

// Every matching row (not just this page), as a CSV file.
async function exportCsv() {
  exporting.value = true
  try {
    const rows: Transaction[] = []
    for (let p = 1; rows.length < total.value && p <= 50; p++) {
      const result = await api<Paged<Transaction>>('/transactions', { query: query(p, 100) })
      rows.push(...(result.items || []))
      if (!result.items?.length) break
    }
    const escape = (value: string) => /[",\n]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value
    const lines = [
      ['Date', 'Number', 'Category', 'Branch', 'By', 'Status', 'Currency', 'Amount'].join(','),
      ...rows.map(t => [t.transaction_date.slice(0, 10), t.transaction_number, t.category?.name || '', t.branch?.code || '', t.created_by?.name || '', t.status, t.currency, t.amount].map(v => escape(String(v ?? ''))).join(',')),
      ...footer.value.map(row => ['', `Total ${props.type} (approved)`, '', '', '', '', row.currency, row.amount].map(v => escape(String(v))).join(',')),
    ]
    const blob = new Blob(['﻿' + lines.join('\n')], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${props.type}-${props.dateFrom || 'all'}-to-${props.dateTo || 'all'}.csv`
    link.click()
    URL.revokeObjectURL(url)
    toast.success(`${rows.length} transactions in ${link.download}.`, { title: 'Table exported' })
  } catch (e: any) {
    toast.error(e?.data?.error || 'Please try again.', { title: 'Export failed' })
  } finally {
    exporting.value = false
  }
}
</script>
