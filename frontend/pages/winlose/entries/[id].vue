<template>
  <section class="rp-compact space-y-4">
    <header class="space-y-2">
      <button type="button" class="inline-flex items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="goBack"><ChevronLeft :size="14" /> Back</button>
      <h1 class="text-xl font-bold text-navy">{{ sheet?.entry?.title || 'Entry' }}</h1>
      <p v-if="sheet" class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted">
        <span v-if="sheet.entry">Created {{ formatDocDate(sheet.entry.created_at) }}<template v-if="sheet.entry.created_by"> by <span class="font-medium text-navy">{{ sheet.entry.created_by }}</span></template></span>
        <span v-if="sheet.rows.length">· {{ savedCount }} of {{ sheet.rows.length }} {{ sheet.rows.length === 1 ? 'agent' : 'agents' }} entered</span>
      </p>
    </header>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the entry" :message="error"><template #action><NuxtLink to="/winlose/entry">Back to Data entry</NuxtLink></template></AppAlert>
    <template v-else-if="sheet">
      <!-- As on New entry: search, sort, and one currency or code start -->
      <WinloseAgentFilters v-if="sheet.rows.length" v-model:search="search" v-model:sort="sort" v-model:currency="currency" v-model:prefix="prefix" :currencies="currencies" :prefixes="prefixes" :total="sheet.rows.length" />

      <!-- Every agent, entered or not: one table per currency, USD first then KHR, each numbered from 1
           (one table with the message when no agent is shown) -->
      <div v-for="part in tables.length ? tables : [{ currency: '', rows: [] }]" :key="part.currency || 'none'" class="overflow-hidden rounded-xl border border-border bg-white">
        <h2 v-if="part.currency" class="border-b border-border px-4 py-2.5 text-sm font-semibold text-navy">{{ part.currency }}</h2>
        <table class="wl-table w-full text-left text-sm">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="w-12 px-3 py-2 text-center font-bold">No</th>
              <th class="px-3 py-2 font-bold">Agent</th>
              <th class="w-44 px-3 py-2 text-right font-bold">W/L</th>
              <th class="w-44 px-3 py-2 text-right font-bold">COMM</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="{ row, no } in part.rows" :key="row.platform_id">
              <td class="row-no px-3 py-2.5 text-center text-sm tabular-nums text-muted">{{ no }}</td>
              <td class="px-3 py-2.5">
                <!-- The agent, its currency, who saved it and its ✅ / ❌ / 📤 in the entry, on one line
                     (on phones the row's number comes first) -->
                <div class="flex items-center gap-2">
                  <p class="shrink-0 whitespace-nowrap font-mono text-sm font-semibold text-navy"><span class="row-no-inline font-sans text-muted">{{ no }}. </span>{{ row.platform_code }} <span class="font-sans text-xs font-normal text-muted">· {{ row.currency }} ·</span></p>
                  <span class="min-w-0 shrink-[100] truncate text-[11px] text-muted" :title="statusOf(row)">{{ statusOf(row) }}</span>
                  <WinloseEntryReactions v-if="row.result_id" class="shrink-0" :url="`/winlose/results/${row.result_id}/reactions`" :reactions="row.reactions || []" />
                </div>
                <p v-if="row.platform_name && row.platform_name !== row.platform_code" class="text-[11px] text-muted">{{ row.platform_name }}</p>
              </td>
              <td class="px-3 py-2.5 text-right text-sm tabular-nums" :class="tone(row.win_lose, row)" data-label="W/L">{{ amount(row.win_lose, row) }}</td>
              <td class="px-3 py-2.5 text-right text-sm tabular-nums" :class="tone(row.commission, row)" data-label="COMM">{{ amount(row.commission, row) }}</td>
            </tr>
            <tr v-if="!sheet.rows.length" class="empty-row"><td colspan="4" class="px-3 py-8 text-center text-sm text-muted">No agents.</td></tr>
            <tr v-else-if="!part.currency" class="empty-row"><td colspan="4" class="px-3 py-8 text-center text-sm text-muted">{{ noMatch }}</td></tr>
          </tbody>
        </table>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft } from 'lucide-vue-next'
import type { WeekSheet, WeekSheetRow } from '~/types/api'
import { SHEET_ROW_FIELDS, useAgentFilter } from '~/composables/useAgentFilter'
import { formatDocDate } from '~/utils/inventory'
import { formatAmount } from '~/utils/winlose'

/** An entry, read-only: every agent with what was saved for it (empty when not entered), laid out
 * as on New entry (a table per currency, with search, sort and filters). */
const route = useRoute()
const router = useRouter()
const { api } = useApi()

const sheet = ref<WeekSheet | null>(null)
const loading = ref(true)
const error = ref('')

const { search, sort, currencies, currency, prefixes, prefix, tables, noMatch } = useAgentFilter(() => sheet.value?.rows || [], SHEET_ROW_FIELDS)
const savedCount = computed(() => (sheet.value?.rows || []).filter(row => row.saved).length)

onMounted(async () => {
  try {
    sheet.value = await api<WeekSheet>('/winlose/entry', { query: { entry_id: String(route.params.id) } })
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load the entry.'
  } finally {
    loading.value = false
  }
})

// Back where the person came from (the entries), else to Data entry.
function goBack() {
  if (window.history.state?.back) router.back()
  else navigateTo('/winlose/entry')
}

// As on New entry.
function statusOf(row: WeekSheetRow) {
  return row.saved ? `Saved${row.updated_by ? ` by ${row.updated_by}` : ''}` : 'Not entered'
}

function amount(value: string | null, row: WeekSheetRow) {
  return row.saved && value !== null ? formatAmount(value, row.currency) : '—'
}

function tone(value: string | null, row: WeekSheetRow) {
  if (!row.saved || value === null) return 'text-muted'
  return Number(value) < 0 ? 'text-expense' : 'text-navy'
}
</script>
