<template>
  <article class="rounded-xl border border-border bg-white">
    <header class="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-3">
      <div>
        <h2 class="flex items-center gap-2 text-sm font-semibold text-navy"><CalendarCheck :size="16" /> Weekly settlement</h2>
        <p class="text-[11px] text-muted">{{ formatDocDate(weekStart) }} – {{ formatDocDate(weekEnd) }}. Finalizing keeps the week’s totals and locks its entry.</p>
      </div>
      <span v-if="!weekOver" class="rounded-full bg-amber-50 px-2 py-0.5 text-[11px] font-semibold text-amber-700">Can be finalized from {{ formatDocDate(weekEnd) }}</span>
    </header>
    <p v-if="loading" class="px-4 py-3 text-xs text-muted">Loading…</p>
    <AppAlert v-else-if="error" class="m-3" compact :message="error" />
    <div v-else class="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center">
      <div class="min-w-0 flex-1">
        <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-navy">
          This week
          <span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-bold uppercase" :class="tone(settlement)"><component :is="settlement?.status === 'finalized' ? Lock : LockOpen" :size="11" />{{ label(settlement) }}</span>
        </p>
        <p v-if="settlement?.status === 'finalized'" class="text-[11px] text-muted">
          by {{ settlement.finalized_by }} · {{ formatDocDateTime(settlement.finalized_at) }}
          <template v-for="t in settlement.totals" :key="t.currency"> · Total {{ formatAmount(t.jame_ps, t.currency) }}</template>
        </p>
        <p v-else-if="settlement?.status === 'reopened'" class="text-[11px] text-amber-700">Reopened by {{ settlement.reopened_by }}: {{ settlement.reopen_reason }}</p>
      </div>
      <div v-if="canFinalize" class="flex shrink-0 gap-1.5">
        <button v-if="settlement?.status === 'finalized'" type="button" class="action-chip action-danger" :disabled="busy" @click="reopening = settlement">Reopen</button>
        <button v-else type="button" class="action-chip action-success" :disabled="busy || !weekOver" @click="finalize">Finalize week</button>
      </div>
    </div>
    <InventoryReasonDialog :model-value="!!reopening" title="Reopen this week?" eyebrow="REOPEN WEEK" subtitle="Its entry can be changed again; finalize it again when done." confirm-label="Reopen week" danger :busy="busy" :server-error="reopenError" @update:model-value="(open) => { if (!open) reopening = null }" @confirm="reopen" />
  </article>
</template>

<script setup lang="ts">
import { CalendarCheck, Lock, LockOpen } from 'lucide-vue-next'
import type { WeeklySettlement } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { formatDocDate, formatDocDateTime, todayISO } from '~/utils/inventory'
import { formatAmount, plusDays } from '~/utils/winlose'

/** The week's settlement for the person's own group: finalize (keeps its totals, locks its entry) or reopen. */
const props = defineProps<{ weekStart: string }>()
const emit = defineEmits<{ changed: [] }>()

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canFinalize = computed(() => can('winlose-settlements.finalize'))
const weekEnd = computed(() => plusDays(props.weekStart, 6))
const weekOver = computed(() => weekEnd.value <= todayISO())

// Without a group the API answers for the person's own.
const settlement = ref<WeeklySettlement | null>(null)
const loading = ref(false)
const error = ref('')
const busy = ref(false)
const reopening = ref<WeeklySettlement | null>(null)
const reopenError = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    settlement.value = (await api<{ items: WeeklySettlement[] }>('/winlose/settlements', { query: { week_start: props.weekStart } })).items?.[0] || null
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load the settlement.'
  } finally {
    loading.value = false
  }
}
watch(() => props.weekStart, load)
onMounted(load)

function label(s?: WeeklySettlement | null) {
  return s?.status === 'finalized' ? 'Finalized' : s?.status === 'reopened' ? 'Reopened' : 'Open'
}
function tone(s?: WeeklySettlement | null) {
  return s?.status === 'finalized' ? 'bg-emerald-50 text-emerald-700' : s?.status === 'reopened' ? 'bg-amber-50 text-amber-700' : 'bg-slate-100 text-muted'
}

async function finalize() {
  busy.value = true
  try {
    await api<WeeklySettlement>('/winlose/settlements', { method: 'POST', body: { week_start: props.weekStart } })
    toast.success('Its entry is locked.', { title: 'Week finalized' })
    await load()
    emit('changed')
  } catch (e: any) {
    toast.error(e?.data?.error || 'Could not finalize the week.')
  } finally {
    busy.value = false
  }
}

async function reopen(reason: string) {
  if (!reopening.value) return
  busy.value = true
  reopenError.value = ''
  try {
    await api<WeeklySettlement>(`/winlose/settlements/${reopening.value.id}/reopen`, { method: 'POST', body: { reason } })
    reopening.value = null
    toast.success('Its entry can be changed again.', { title: 'Week reopened' })
    await load()
    emit('changed')
  } catch (e: any) {
    reopenError.value = e?.data?.error || 'Could not reopen the week.'
  } finally {
    busy.value = false
  }
}
</script>
