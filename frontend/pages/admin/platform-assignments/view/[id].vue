<template>
  <section class="rp-compact space-y-4">
    <header class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <button type="button" class="mb-2 inline-flex items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="goBack"><ChevronLeft :size="14" /> Back</button>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">GROUP</p>
        <h1 class="flex items-center gap-2 text-xl font-bold text-navy"><Network :size="20" class="shrink-0 text-muted" />{{ group?.title || 'Group' }}</h1>
        <p v-if="group" class="mt-0.5 text-xs text-muted">From {{ formatDocDate(group.starts_on) }} · {{ group.weeks }} {{ group.weeks === 1 ? 'week' : 'weeks' }} entered</p>
      </div>
      <NuxtLink v-if="group && canEdit" :to="`/admin/platform-assignments/${group.id}`" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover"><Pencil :size="15" /> Update</NuxtLink>
    </header>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load the group" :message="error"><template #action><NuxtLink to="/admin/platform-assignments">Back to the groups</NuxtLink></template></AppAlert>
    <template v-else-if="group">
      <dl class="keep-cols grid grid-cols-2 gap-4 rounded-xl border border-border bg-white p-4 text-xs sm:grid-cols-4">
        <div><dt class="text-muted">Status</dt><dd class="font-semibold" :class="group.is_active ? 'text-emerald-700' : 'text-muted'">{{ group.is_active ? 'Active' : 'Off (new results don’t count in it)' }}</dd></div>
        <div><dt class="text-muted">Agents</dt><dd class="font-semibold text-navy">{{ group.agents.length }}</dd></div>
        <div><dt class="text-muted">Created at</dt><dd class="font-semibold text-navy">{{ formatDocDate(group.created_at) }}</dd></div>
        <div><dt class="text-muted">Created by</dt><dd class="font-semibold text-navy">{{ group.created_by || '—' }}</dd></div>
      </dl>

      <div class="overflow-x-auto rounded-xl border border-border bg-white">
        <table class="w-full text-left text-sm">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-2 py-2.5 sm:px-4 font-bold">Agent</th>
              <th v-if="!group.partners.length" class="px-2 py-2.5 sm:px-4 font-bold">Type</th>
              <th v-if="!group.partners.length" class="px-2 py-2.5 sm:px-4 text-right font-bold">Rate</th>
              <th v-if="!group.partners.length" class="px-2 py-2.5 sm:px-4 text-right font-bold">Total at</th>
              <th class="px-2 py-2.5 sm:px-4 text-right font-bold">Weeks</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="a in group.agents" :key="a.assignment_id" class="text-navy">
              <td class="px-2 py-2.5 sm:px-4"><span class="font-mono font-semibold">{{ a.platform_code }}</span> <span class="text-muted">· {{ a.currency }}</span><span class="block text-[11px] text-muted">From {{ formatDocDate(a.starts_on) }}</span></td>
              <td v-if="!group.partners.length" class="px-2 py-2.5 sm:px-4">
                <span class="font-semibold">{{ a.calc_type || 'PO' }}</span>
                <span class="hidden text-[11px] text-muted sm:block">{{ a.calc_type === 'GI' ? '−((W/L + Comm) × Rate)' : '(W/L + Comm) × (100% − Rate)' }}</span>
              </td>
              <td v-if="!group.partners.length" class="px-2 py-2.5 sm:px-4 text-right font-semibold tabular-nums">{{ formatRate(a.commission_rate) }}</td>
              <td v-if="!group.partners.length" class="px-2 py-2.5 sm:px-4 text-right font-semibold tabular-nums">{{ totalRate(a.calc_type, Number(a.commission_rate)) }}</td>
              <td class="px-2 py-2.5 sm:px-4 text-right tabular-nums">{{ a.weeks }}</td>
            </tr>
            <tr v-if="!group.agents.length"><td colspan="5" class="px-4 py-8 text-center text-muted">No agents</td></tr>
          </tbody>
        </table>
      </div>

      <!-- Partners a group still has from before: their % decide the Total until it is updated -->
      <div v-if="group.partners.length" class="space-y-1.5 rounded-xl border border-border bg-white p-4">
        <p class="text-xs font-medium text-muted">Partners (their % decide the Total until the group is updated)</p>
        <ul class="flex flex-wrap gap-1">
          <li v-for="p in group.partners" :key="p.partner_id" class="inline-flex items-center gap-1 rounded-md bg-blue-50 px-1.5 py-0.5 text-[11px] text-navy" :class="p.partner_active ? '' : 'opacity-60'"><Handshake :size="11" />{{ p.partner_name }}<template v-if="!p.partner_active"> (off)</template> · {{ formatRate(p.commission_rate) }}</li>
        </ul>
      </div>
    </template>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, Handshake, Network, Pencil } from 'lucide-vue-next'
import type { AssignmentGroup } from '~/types/api'
import { formatDocDate } from '~/utils/inventory'
import { formatRate, totalRate } from '~/utils/winlose'

/** A group's details (read-only): its agents with type, rate and what the Total is at. */
const route = useRoute()
const router = useRouter()
const { api } = useApi()
const { can } = usePermissions()
const canEdit = computed(() => can('platform-assignments.edit'))

const group = ref<AssignmentGroup | null>(null)
const loading = ref(true)
const error = ref('')

onMounted(async () => {
  try {
    group.value = await api<AssignmentGroup>(`/platform-groups/${route.params.id}`)
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load the group.'
  } finally {
    loading.value = false
  }
})

// Back where the person came from (the groups), else to the groups.
function goBack() {
  if (window.history.state?.back) router.back()
  else navigateTo('/admin/platform-assignments')
}
</script>
