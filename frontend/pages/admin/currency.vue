<template>
  <section class="mx-auto max-w-4xl space-y-4">
    <header class="flex flex-wrap items-end justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">MANAGE</p>
        <h1 class="text-xl font-bold text-navy">Currency</h1>
        <p class="mt-1 text-xs text-muted">USD is the base: product prices, stock cost and inventory reports are in USD. Add any other currency with its rate; transactions, sales, purchases and expenses can use every active currency.</p>
      </div>
      <button v-if="canManage" type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-dashed border-primary/60 bg-white px-3 py-2 text-sm font-semibold text-primary disabled:opacity-50" :disabled="rows.length >= 20" @click="addRow"><Plus :size="16" /> Add currency</button>
    </header>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <form v-else class="space-y-4" novalidate @submit.prevent="save">
      <div class="overflow-hidden rounded-xl border border-border bg-white">
        <table class="cur-table w-full text-left text-sm">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="w-24 px-3 py-2 font-bold">Code</th>
              <th class="px-3 py-2 font-bold">Name</th>
              <th class="w-24 px-3 py-2 font-bold">Symbol</th>
              <th class="w-24 px-3 py-2 font-bold">Decimals</th>
              <th class="w-44 px-3 py-2 font-bold">Rate (for 1 USD)</th>
              <th class="w-24 px-3 py-2 font-bold">Active</th>
              <th class="w-12 px-3 py-2"><span class="sr-only">Remove</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="(row, index) in rows" :key="row.key">
              <td class="px-3 py-2" data-label="Code">
                <input v-model.trim="row.code" :disabled="row.base || row.saved || !canManage" maxlength="3" class="w-full rounded-lg border border-border px-2 py-1.5 font-mono text-sm uppercase disabled:bg-canvas" placeholder="THB" @input="row.code = row.code.toUpperCase()" />
              </td>
              <td class="px-3 py-2" data-label="Name"><input v-model.trim="row.name" :disabled="row.base || !canManage" maxlength="64" class="w-full rounded-lg border border-border px-2 py-1.5 text-sm disabled:bg-canvas" placeholder="Thai Baht" /></td>
              <td class="px-3 py-2" data-label="Symbol"><input v-model.trim="row.symbol" :disabled="row.base || !canManage" maxlength="6" class="w-full rounded-lg border border-border px-2 py-1.5 text-sm disabled:bg-canvas" placeholder="฿" /></td>
              <td class="px-3 py-2" data-label="Decimals">
                <select v-model.number="row.decimals" :disabled="row.base || !canManage" class="w-full rounded-lg border border-border bg-white px-2 py-1.5 text-sm disabled:bg-canvas">
                  <option v-for="d in [0, 1, 2, 3, 4]" :key="d" :value="d">{{ d }}</option>
                </select>
              </td>
              <td class="px-3 py-2" data-label="Rate (for 1 USD)">
                <div class="flex items-center gap-1.5">
                  <input v-model.trim="row.rate" :disabled="row.base || !canManage" type="text" inputmode="decimal" class="min-w-0 flex-1 rounded-lg border border-border px-2 py-1.5 text-sm disabled:bg-canvas" placeholder="36" />
                  <span class="shrink-0 text-xs text-muted">{{ row.code || '—' }}</span>
                </div>
              </td>
              <td class="px-3 py-2" data-label="Active">
                <label class="inline-flex items-center gap-1.5 text-xs text-navy"><input v-model="row.active" type="checkbox" :disabled="row.base || !canManage" class="accent-primary" /> {{ row.base ? 'Base' : row.active ? 'On' : 'Off' }}</label>
              </td>
              <td class="px-3 py-2 text-right">
                <button v-if="!row.base && canManage" type="button" class="grid h-8 w-8 place-items-center rounded-lg bg-rose-50 text-expense" :title="row.saved ? 'Remove (only when nothing uses it; otherwise turn it off)' : 'Remove'" :aria-label="`Remove ${row.code || 'currency'}`" @click="rows.splice(index, 1)"><Trash2 :size="13" /></button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <div class="grid grid-cols-1 gap-3 rounded-xl border border-border bg-white p-4 sm:grid-cols-2">
        <div class="space-y-1.5 text-xs font-semibold text-navy">
          Default currency on new forms
          <SearchSelect v-model="defaultCurrency" :options="defaultOptions" :searchable="false" :disabled="!canManage" aria-label="Default currency" />
        </div>
        <div class="text-xs text-muted">
          <p class="font-semibold text-navy">Examples</p>
          <p v-for="row in rows.filter(r => !r.base && Number(r.rate) > 0)" :key="row.key">$10.00 = {{ example(row) }}</p>
        </div>
      </div>

      <p v-if="settings?.updated_at" class="text-[11px] text-muted">Last changed {{ formatDocDateTime(settings.updated_at) }}<template v-if="settings.updated_by"> by {{ settings.updated_by }}</template>. Saved sales, purchases and expenses keep the rate they were made with. A currency that is already used can be turned off but not removed.</p>
      <AppAlert v-if="formError || problems.length" :title="formError ? 'Couldn’t save' : 'Check the currencies'" :message="formError" :items="problems" dismissible @dismiss="formError = ''; problems = []" />
      <footer v-if="canManage" class="flex justify-end">
        <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-sm font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save currencies' }}</button>
      </footer>
    </form>
  </section>
</template>

<script setup lang="ts">
import { Plus, Trash2 } from 'lucide-vue-next'
import type { CurrencySettings } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useCurrency } from '~/composables/useCurrency'
import { setCurrencyMeta } from '~/utils/money'
import { formatDocDateTime } from '~/utils/inventory'

interface Row { key: number; code: string; name: string; symbol: string; decimals: number; rate: string; active: boolean; base: boolean; saved: boolean }

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const { settings, load } = useCurrency()
const canManage = computed(() => can('currency-settings.manage'))

const loading = ref(true)
const saving = ref(false)
const formError = ref('')
const problems = ref<string[]>([])
const rows = ref<Row[]>([])
const defaultCurrency = ref('USD')
let nextKey = 1

const defaultOptions = computed(() => rows.value.filter(r => r.active && /^[A-Z]{3}$/.test(r.code)).map(r => ({ value: r.code, label: `${r.code} — ${r.name || r.code}` })))

function example(row: Row) {
  const amount = 10 * Number(row.rate)
  return `${row.symbol}${new Intl.NumberFormat('en-US', { minimumFractionDigits: row.decimals, maximumFractionDigits: row.decimals }).format(amount)}`
}

function apply(next: CurrencySettings) {
  rows.value = next.currencies.map(c => ({ key: nextKey++, code: c.code, name: c.name, symbol: c.symbol, decimals: c.decimals, rate: String(Number(c.rate)), active: c.active, base: c.base, saved: true }))
  defaultCurrency.value = next.default_currency
}

function addRow() {
  rows.value.push({ key: nextKey++, code: '', name: '', symbol: '', decimals: 2, rate: '', active: true, base: false, saved: false })
}

// Same rules as the API (services.CurrencyService.Save).
function check() {
  const out: string[] = []
  const seen = new Set<string>()
  rows.value.forEach((r, i) => {
    const label = r.code || `Row ${i + 1}`
    if (!/^[A-Z]{3}$/.test(r.code)) out.push(`${label}: the code must be 3 letters, like THB`)
    else if (seen.has(r.code)) out.push(`${r.code} is listed twice`)
    seen.add(r.code)
    if (r.base) return
    if (!r.name) out.push(`${label}: enter a name`)
    if (!r.symbol) out.push(`${label}: enter a symbol`)
    if (!/^\d+(\.\d{1,6})?$/.test(r.rate) || !(Number(r.rate) > 0)) out.push(`${label}: enter how many ${r.code || 'units'} make 1 USD`)
  })
  if (!rows.value.some(r => r.code === defaultCurrency.value && r.active)) out.push('The default currency must be one of the active currencies')
  problems.value = out
  return !out.length
}

onMounted(async () => {
  const current = await load(true)
  if (current) apply(current)
  loading.value = false
})

async function save() {
  formError.value = ''
  if (!check()) return
  saving.value = true
  try {
    const next = await api<CurrencySettings>('/currency', {
      method: 'PUT',
      body: { default_currency: defaultCurrency.value, currencies: rows.value.map(r => ({ code: r.code, name: r.name, symbol: r.symbol, decimals: r.decimals, rate: r.base ? '1' : r.rate, active: r.base || r.active })) },
    })
    settings.value = next
    setCurrencyMeta(next.currencies)
    apply(next)
    toast.success(`${next.currencies.filter(c => c.active).length} active currencies.`, { title: 'Currencies saved' })
  } catch (e: any) {
    formError.value = e?.data?.fields?.currencies || e?.data?.fields?.default_currency || e?.data?.error || 'Could not save the currencies.'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
@media (max-width: 719px) {
  .cur-table thead { display: none; }
  .cur-table, .cur-table tbody, .cur-table tr, .cur-table td { display: block; width: 100%; }
  .cur-table tr { position: relative; display: grid; grid-template-columns: 1fr 1fr; gap: 6px 10px; padding: 12px; }
  .cur-table td { padding: 0 !important; }
  .cur-table td:last-child { position: absolute; top: 10px; right: 10px; width: auto; }
  .cur-table td[data-label]::before { content: attr(data-label); display: block; margin-bottom: 3px; color: #64748b; font-size: 10px; font-weight: 700; text-transform: uppercase; }
}
</style>
