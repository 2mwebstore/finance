<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Inventory reports</h1>
        <p class="mt-1 text-xs text-muted">Stock value, sales and profit, purchases and expenses for your branches, in USD.</p>
      </div>
      <button v-if="canExport" type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy disabled:opacity-50" :disabled="!rows.length" @click="exportCsv"><Download :size="16" /> Export CSV</button>
    </header>

    <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
      <SearchSelect v-if="showBranchFilter" v-model="branchFilter" :options="branchOptions" aria-label="Branch" />
      <SearchSelect v-model="categoryFilter" :options="categoryOptions" aria-label="Category" />
      <DateRangePicker v-model="dateRange" />
    </div>

    <AppAlert v-if="summaryError" compact :message="summaryError" />
    <div v-else-if="summary" class="space-y-3">
      <article class="rounded-xl border border-border bg-white p-4">
        <p class="text-xs font-bold text-slate-500">In USD <span class="font-normal text-slate-400">· riel documents converted at their own rate</span></p>
        <div class="mt-2 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          <div><p class="text-[11px] text-muted">Stock value</p><p class="text-base font-bold text-navy">{{ formatMoney(summary.stock_value, 'USD') }}</p></div>
          <div><p class="text-[11px] text-muted">Sales ({{ summary.sales_count }})</p><p class="text-base font-bold text-income">{{ formatMoney(summary.sales_total, 'USD') }}</p></div>
          <div><p class="text-[11px] text-muted">Gross profit</p><p class="text-base font-bold" :class="Number(summary.profit) < 0 ? 'text-expense' : 'text-income'">{{ formatMoney(summary.profit, 'USD') }}</p></div>
          <div><p class="text-[11px] text-muted">Purchases ({{ summary.purchases_count }})</p><p class="text-base font-bold text-navy">{{ formatMoney(summary.purchases_total, 'USD') }}</p></div>
          <div><p class="text-[11px] text-muted">Expenses ({{ summary.expenses_count }})</p><p class="text-base font-bold text-expense">{{ formatMoney(summary.expenses_total, 'USD') }}</p></div>
          <div><p class="text-[11px] text-muted">Net profit</p><p class="text-base font-bold" :class="Number(summary.net_profit) < 0 ? 'text-expense' : 'text-income'">{{ formatMoney(summary.net_profit, 'USD') }}</p></div>
        </div>
        <p class="mt-2 text-[11px] text-slate-400">Gross profit = sales (with shipping charged) − cost of the goods sold. Net profit = gross profit − inventory expenses.</p>
        <p v-if="summary.pending_purchases" class="mt-1 text-[11px] font-semibold text-amber-700">{{ summary.pending_purchases }} purchase{{ summary.pending_purchases === 1 ? '' : 's' }} waiting for approval</p>
      </article>
      <article class="flex items-center gap-6 rounded-xl border border-border bg-white p-4">
        <div><p class="text-[11px] text-muted">Active products</p><p class="text-base font-bold text-navy">{{ summary.products }}</p></div>
        <button type="button" class="text-left" @click="tab = 'low'"><p class="text-[11px] text-muted">Low on stock</p><p class="text-base font-bold" :class="summary.low_stock ? 'text-amber-700' : 'text-navy'">{{ summary.low_stock }}</p></button>
      </article>
    </div>

    <div class="flex flex-wrap gap-1 rounded-xl bg-canvas p-1" role="tablist">
      <button v-for="t in tabs" :key="t.value" type="button" role="tab" :aria-selected="tab === t.value" class="flex-1 rounded-lg px-3 py-2 text-xs font-semibold" :class="tab === t.value ? 'bg-white text-primary shadow-sm' : 'text-muted'" @click="tab = t.value">{{ t.label }}</button>
    </div>
    <div v-if="tab === 'purchases'" class="flex gap-2 text-xs">
      <button v-for="g in [{ value: 'product', label: 'By product' }, { value: 'supplier', label: 'By supplier' }]" :key="g.value" type="button" class="rounded-full border px-3 py-1 font-semibold" :class="purchaseGroup === g.value ? 'border-primary bg-blue-50 text-primary' : 'border-border text-muted'" @click="purchaseGroup = g.value">{{ g.label }}</button>
    </div>
    <p class="text-[11px] text-muted">{{ tab === 'stock' || tab === 'low' ? 'Stock as it is now (the dates don’t apply).' : `For ${dateLabel}.` }}</p>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" :message="error" />
    <div v-else class="overflow-x-auto rounded-xl border border-border bg-white">
      <table class="w-full min-w-[640px] text-left text-xs">
        <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
          <tr><th v-for="col in columns" :key="col.key" class="px-3 py-2 font-bold" :class="col.num ? 'text-right' : ''">{{ col.label }}</th></tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="(row, i) in rows" :key="i" class="text-navy">
            <td v-for="col in columns" :key="col.key" class="px-3 py-2" :class="[col.num ? 'text-right tabular-nums' : '', col.cls?.(row) || '']">{{ col.format(row) }}</td>
          </tr>
          <tr v-if="!rows.length"><td :colspan="columns.length" class="px-3 py-6 text-center text-muted">Nothing to show.</td></tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Download } from 'lucide-vue-next'
import type { InventorySummary, ProductCategory } from '~/types/api'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { currentMonthRange } from '~/utils/dateRanges'
import { formatMoney, type SupportedCurrency } from '~/utils/money'
import { formatDocDate, formatQty } from '~/utils/inventory'

type Row = Record<string, any>
type Column = { key: string; label: string; num?: boolean; format: (row: Row) => string; csv?: (row: Row) => string; cls?: (row: Row) => string }

const { api } = useApi()
const { can } = usePermissions()
const canExport = computed(() => can('reports.export'))
const { options: branchOptions, showFilter: showBranchFilter } = useBranchOptions('All branches')

const branchFilter = ref('')
const categoryFilter = ref('')
const dateRange = ref(currentMonthRange())
const categories = ref<ProductCategory[]>([])
const categoryOptions = computed(() => [{ value: '', label: 'All categories' }, ...categories.value.map(c => ({ value: c.id, label: c.name }))])
const dateLabel = computed(() => dateRange.value.from && dateRange.value.to ? `${formatDocDate(dateRange.value.from)} – ${formatDocDate(dateRange.value.to)}` : 'all dates')

const tabs = [
  { value: 'stock', label: 'Stock value' },
  { value: 'sales', label: 'Sales & profit' },
  { value: 'purchases', label: 'Purchases' },
  { value: 'expenses', label: 'Expenses' },
  { value: 'low', label: 'Low stock' },
] as const
const tab = ref<(typeof tabs)[number]['value']>('stock')
const purchaseGroup = ref('product')

const money = (field: string) => (row: Row) => formatMoney(row[field], row.currency as SupportedCurrency)
const plain = (field: string) => (row: Row) => String(row[field] ?? '')
const qty = (field: string) => (row: Row) => `${formatQty(row[field])}${row.unit ? ` ${row.unit}` : ''}`

const columns = computed<Column[]>(() => {
  switch (tab.value) {
    case 'sales':
      return [
        { key: 'name', label: 'Product', format: row => `${row.name} (${row.sku})` },
        { key: 'quantity', label: 'Sold', num: true, format: qty('quantity'), csv: plain('quantity') },
        { key: 'revenue', label: 'Sales (after discounts)', num: true, format: money('revenue'), csv: plain('revenue') },
        { key: 'cost', label: 'Cost', num: true, format: money('cost'), csv: plain('cost') },
        { key: 'profit', label: 'Profit', num: true, format: money('profit'), csv: plain('profit'), cls: row => Number(row.profit) < 0 ? 'text-expense font-semibold' : 'text-income font-semibold' },
        { key: 'margin', label: 'Margin', num: true, format: row => `${Number(row.margin)}%`, csv: plain('margin') },
        { key: 'currency', label: 'Cur.', format: plain('currency') },
      ]
    case 'purchases':
      return purchaseGroup.value === 'supplier'
        ? [
            { key: 'name', label: 'Supplier', format: plain('name') },
            { key: 'count', label: 'Purchases', num: true, format: plain('count') },
            { key: 'total', label: 'Total', num: true, format: money('total'), csv: plain('total') },
            { key: 'currency', label: 'Cur.', format: plain('currency') },
          ]
        : [
            { key: 'name', label: 'Product', format: row => `${row.name} (${row.sku})` },
            { key: 'quantity', label: 'Bought', num: true, format: qty('quantity'), csv: plain('quantity') },
            { key: 'avg_cost', label: 'Landed cost', num: true, format: money('avg_cost'), csv: plain('avg_cost') },
            { key: 'total', label: 'Total', num: true, format: money('total'), csv: plain('total') },
            { key: 'count', label: 'Purchases', num: true, format: plain('count') },
            { key: 'currency', label: 'Cur.', format: plain('currency') },
          ]
    case 'expenses':
      return [
        { key: 'name', label: 'Category', format: row => row.name || '—' },
        { key: 'count', label: 'Expenses', num: true, format: plain('count') },
        { key: 'total', label: 'Total', num: true, format: money('total'), csv: plain('total'), cls: () => 'text-expense font-semibold' },
        { key: 'currency', label: 'Cur.', format: plain('currency') },
      ]
    case 'low':
      return [
        { key: 'name', label: 'Product', format: row => `${row.name} (${row.sku})` },
        { key: 'branch_code', label: 'Branch', format: plain('branch_code') },
        { key: 'quantity', label: 'In stock', num: true, format: qty('quantity'), csv: plain('quantity'), cls: () => 'text-amber-700 font-semibold' },
        { key: 'reorder_level', label: 'Low-stock level', num: true, format: qty('reorder_level'), csv: plain('reorder_level') },
      ]
    default:
      return [
        { key: 'name', label: 'Product', format: row => `${row.name} (${row.sku})` },
        { key: 'category_name', label: 'Category', format: row => row.category_name || '—' },
        { key: 'quantity', label: 'In stock', num: true, format: qty('quantity'), csv: plain('quantity') },
        { key: 'avg_cost', label: 'Avg cost', num: true, format: money('avg_cost'), csv: plain('avg_cost') },
        { key: 'value', label: 'Value', num: true, format: money('value'), csv: plain('value') },
        { key: 'currency', label: 'Cur.', format: plain('currency') },
      ]
  }
})

const summary = ref<InventorySummary | null>(null)
const summaryError = ref('')
const rows = ref<Row[]>([])
const loading = ref(true)
const error = ref('')

function filterQuery(withDates: boolean) {
  return {
    branch_id: branchFilter.value || undefined,
    category_id: categoryFilter.value || undefined,
    date_from: withDates ? dateRange.value.from || undefined : undefined,
    date_to: withDates ? dateRange.value.to || undefined : undefined,
  }
}

async function loadSummary() {
  summaryError.value = ''
  try {
    summary.value = await api<InventorySummary>('/inventory/reports/summary', { query: filterQuery(true) })
  } catch (e: any) {
    summaryError.value = e?.data?.error || 'Could not load the summary.'
  }
}

let loadRequest = 0
async function loadRows() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  const path = { stock: 'stock', sales: 'sales', purchases: 'purchases', expenses: 'expenses', low: 'low-stock' }[tab.value]
  try {
    const query: Record<string, unknown> = filterQuery(tab.value === 'sales' || tab.value === 'purchases' || tab.value === 'expenses')
    if (tab.value === 'purchases') query.group = purchaseGroup.value
    const result = await api<{ items: Row[] }>(`/inventory/reports/${path}`, { query })
    if (request === loadRequest) rows.value = result.items || []
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load the report.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}

watch([branchFilter, categoryFilter, dateRange], () => { loadSummary(); loadRows() }, { deep: true })
watch([tab, purchaseGroup], loadRows)
onMounted(async () => {
  loadSummary()
  loadRows()
  try { categories.value = (await api<{ items: ProductCategory[] }>('/inventory/categories')).items || [] } catch { categories.value = [] }
})

// CSV of the table as shown (amounts as plain numbers).
function exportCsv() {
  const cols = columns.value
  const escape = (value: string) => /[",\n]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value
  const lines = [cols.map(c => escape(c.label)).join(','), ...rows.value.map(row => cols.map(c => escape((c.csv || c.format)(row))).join(','))]
  const blob = new Blob(['﻿' + lines.join('\n')], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `inventory-${tab.value}${tab.value === 'purchases' ? `-${purchaseGroup.value}` : ''}-${dateRange.value.from || 'all'}.csv`
  link.click()
  URL.revokeObjectURL(url)
}
</script>
