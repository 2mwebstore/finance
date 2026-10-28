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
    <ul v-else class="divide-y divide-border">
      <li v-for="hq in hqs" :key="hq.id" class="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center">
        <div class="min-w-0 flex-1">
          <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-navy">
            {{ hq.code }} — {{ hq.name }}
            <span class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-bold uppercase" :class="tone(byHQ[hq.id])"><component :is="byHQ[hq.id]?.status === 'finalized' ? Lock : LockOpen" :size="11" />{{ label(byHQ[hq.id]) }}</span>
          </p>
          <p v-if="byHQ[hq.id]?.status === 'finalized'" class="text-[11px] text-muted">
            by {{ byHQ[hq.id]!.finalized_by }} · {{ formatDocDateTime(byHQ[hq.id]!.finalized_at) }}
            <template v-for="t in byHQ[hq.id]!.totals" :key="t.currency"> · Total {{ formatAmount(t.jame_ps, t.currency) }}</template>
          </p>
          <p v-else-if="byHQ[hq.id]?.status === 'reopened'" class="text-[11px] text-amber-700">Reopened by {{ byHQ[hq.id]!.reopened_by }}: {{ byHQ[hq.id]!.reopen_reason }}</p>
        </div>
        <div v-if="canFinalize" class="flex shrink-0 gap-1.5">
          <button v-if="byHQ[hq.id]?.status === 'finalized'" type="button" class="action-chip action-danger" :disabled="busy" @click="reopening = byHQ[hq.id]!">Reopen</button>
          <button v-else type="button" class="action-chip action-success" :disabled="busy || !weekOver" @click="finalize(hq.id)">Finalize week</button>
        </div>
      </li>
      <li v-if="!hqs.length" class="px-4 py-3 text-xs text-muted">No HQ to show.</li>
    </ul>
    <InventoryReasonDialog :model-value="!!reopening" :title="`Reopen ${reopening?.branch_code || ''}'s week?`" eyebrow="REOPEN WEEK" subtitle="Its days can be changed again; finalize it again when done." confirm-label="Reopen week" danger :busy="busy" :server-error="reopenError" @update:model-value="(open) => { if (!open) reopening = null }" @confirm="reopen" />
  </article>
</template>

<script setup lang="ts">
import { CalendarCheck, Lock, LockOpen } from 'lucide-vue-next'
import type { WeeklySettlement } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { formatDocDate, formatDocDateTime, todayISO } from '~/utils/inventory'
import { formatAmount, plusDays } from '~/utils/winlose'

/** The week's settlement for each HQ: finalize (keeps its totals, locks its entry) or reopen. */
const props = defineProps<{ weekStart: string; hqs: { id: string; code: string; name: string }[] }>()
const emit = defineEmits<{ changed: [] }>()

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canFinalize = computed(() => can('winlose-settlements.finalize'))
const weekEnd = computed(() => plusDays(props.weekStart, 6))
const weekOver = computed(() => weekEnd.value <= todayISO())

const list = ref<WeeklySettlement[]>([])
const byHQ = computed(() => Object.fromEntries(list.value.map(s => [s.branch_id, s])) as Record<string, WeeklySettlement | undefined>)
const loading = ref(false)
const error = ref('')
const busy = ref(false)
const reopening = ref<WeeklySettlement | null>(null)
const reopenError = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    list.value = (await api<{ items: WeeklySettlement[] }>('/winlose/settlements', { query: { week_start: props.weekStart } })).items || []
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load the settlements.'
  } finally {
    loading.value = false
  }
}
watch(() => props.weekStart, load)
onMounted(load)

function label(s?: WeeklySettlement) {
  return s?.status === 'finalized' ? 'Finalized' : s?.status === 'reopened' ? 'Reopened' : 'Open'
}
function tone(s?: WeeklySettlement) {
  return s?.status === 'finalized' ? 'bg-emerald-50 text-emerald-700' : s?.status === 'reopened' ? 'bg-amber-50 text-amber-700' : 'bg-slate-100 text-muted'
}

async function finalize(branchId: string) {
  busy.value = true
  try {
    const st = await api<WeeklySettlement>('/winlose/settlements', { method: 'POST', body: { branch_id: branchId, week_start: props.weekStart } })
    toast.success('Its entry is locked.', { title: `${st.branch_code}'s week finalized` })
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
    const st = await api<WeeklySettlement>(`/winlose/settlements/${reopening.value.id}/reopen`, { method: 'POST', body: { reason } })
    reopening.value = null
    toast.success('Its entry can be changed again.', { title: `${st.branch_code}'s week reopened` })
    await load()
    emit('changed')
  } catch (e: any) {
    reopenError.value = e?.data?.error || 'Could not reopen the week.'
  } finally {
    busy.value = false
  }
}
</script>
