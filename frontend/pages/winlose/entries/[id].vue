<template>
  <section class="rp-compact space-y-4">
    <header class="space-y-2">
      <button type="button" class="inline-flex items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="goBack"><ChevronLeft :size="14" /> Back</button>
      <h1 class="text-xl font-bold text-navy">{{ sheet?.entry?.title || 'Entry' }}</h1>
      <p v-if="sheet" class="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-muted">
        <span v-if="sheet.entry">Created {{ formatDocDate(sheet.entry.created_at) }}<template v-if="sheet.entry.created_by"> by <span class="font-medium text-navy">{{ sheet.entry.created_by }}</span></template></span>
      </p>
    </header>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the entry" :message="error"><template #action><NuxtLink to="/winlose/entry">Back to Data entry</NuxtLink></template></AppAlert>
    <!-- Every agent, entered or not -->
    <div v-else-if="sheet" class="overflow-x-auto rounded-xl border border-border bg-white">
      <table class="report-table">
        <thead class="bg-canvas uppercase text-slate-500">
          <tr>
            <th class="font-bold">Agent</th>
            <th class="amount font-bold">W/L</th>
            <th class="amount font-bold">COMM</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="(row, i) in sheet.rows" :key="row.platform_id" class="text-navy">
            <td class="name">
              <span class="code font-mono font-semibold"><span class="font-sans font-normal text-muted">{{ i + 1 }}. </span>{{ row.platform_code }}</span>
              <span class="block text-muted">{{ row.currency }} · {{ row.saved ? `Saved${row.updated_by ? ` by ${row.updated_by}` : ''}` : 'Not entered' }}</span>
              <!-- ✅ / ❌ on this agent in the entry -->
              <div v-if="row.result_id" class="mt-1"><WinloseEntryReactions :url="`/winlose/results/${row.result_id}/reactions`" :reactions="row.reactions || []" /></div>
            </td>
            <td class="amount" :class="tone(row.win_lose)">{{ amount(row.win_lose, row) }}</td>
            <td class="amount" :class="tone(row.commission)">{{ amount(row.commission, row) }}</td>
          </tr>
          <tr v-if="!sheet.rows.length"><td colspan="3" class="empty text-muted">No agents.</td></tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft } from 'lucide-vue-next'
import type { WeekSheet, WeekSheetRow } from '~/types/api'
import { formatDocDate } from '~/utils/inventory'
import { formatAmount } from '~/utils/winlose'

/** An entry, read-only: every agent with what was saved for it (empty when not entered). */
const route = useRoute()
const router = useRouter()
const { api } = useApi()

const sheet = ref<WeekSheet | null>(null)
const loading = ref(true)
const error = ref('')

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

function amount(value: string | null, row: WeekSheetRow) {
  return row.saved && value !== null ? formatAmount(value, row.currency) : '—'
}

function tone(value: string | null) {
  return Number(value) < 0 ? 'text-expense' : ''
}
</script>
