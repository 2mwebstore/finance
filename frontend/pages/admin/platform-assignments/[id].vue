<template>
  <section class="mx-auto max-w-3xl space-y-4">
    <header>
      <NuxtLink to="/admin/platform-assignments" class="inline-flex items-center gap-1 text-[11px] font-bold tracking-[0.14em] text-slate-400 hover:text-primary"><ChevronLeft :size="13" /> GROUP AGENT ASSIGNMENTS</NuxtLink>
      <h1 class="text-xl font-bold text-navy">{{ isNew ? 'Assign agents' : 'Update group' }}</h1>
      <p class="mt-1 text-xs text-muted">{{ isNew ? 'Give the group a title, pick its agents and set each agent’s rate. The Total is worked out at each agent’s rate.' : 'New weeks use these agents and rates; weeks already saved keep what they were worked out with.' }}</p>
    </header>

    <AppAlert v-if="!canManage" type="warning" title="You can’t change groups" message="It needs Manage Group Agent Assignments." />
    <div v-else-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="loadError" title="Couldn’t load the group" :message="loadError"><template #action><NuxtLink to="/admin/platform-assignments">Back to the groups</NuxtLink></template></AppAlert>
    <form v-else class="space-y-4 rounded-xl border border-border bg-white p-4 sm:p-5" novalidate @submit.prevent="save">
      <label class="block space-y-1 text-xs font-medium text-muted">Title <span class="text-expense">*</span>
        <input v-model="title" maxlength="128" :aria-invalid="tried && !title.trim() ? 'true' : undefined" class="h-10 w-full rounded-lg border border-border px-3 text-sm text-navy" placeholder="e.g. V-series Phnom Penh" />
      </label>

      <div class="space-y-1 text-xs font-medium text-muted">Agents <span class="text-expense">*</span>
        <SearchSelect v-model="agentIds" multiple :options="agentOptions" placeholder="Choose one or more agents" aria-label="Agents" :aria-invalid="tried && !agentIds.length" />
      </div>

      <!-- Each agent picked, with its rate -->
      <div v-if="rows.length" class="overflow-hidden rounded-lg border border-border">
        <table class="w-full text-left text-sm">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-3 py-2 font-bold">Agent</th>
              <th class="w-40 px-3 py-2 font-bold">Rate</th>
              <th class="w-12 px-2 py-2"><span class="sr-only">Take out</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="r in rows" :key="r.id">
              <td class="px-3 py-2">
                <span class="font-mono text-sm font-semibold text-navy">{{ r.code }}</span>
                <span v-if="r.weeks" class="block text-[11px] text-muted">{{ r.weeks }} {{ r.weeks === 1 ? 'week' : 'weeks' }} entered</span>
              </td>
              <td class="px-3 py-2">
                <label class="flex items-center gap-1.5 text-xs text-muted">
                  <input v-model.trim="rates[r.id]" type="text" inputmode="decimal" :aria-label="`Rate for ${r.code}`" :aria-invalid="tried && !!rateProblem(rates[r.id]) ? 'true' : undefined" class="h-9 w-24 rounded-lg border border-border px-2.5 text-right text-sm text-navy" placeholder="0" />%
                </label>
              </td>
              <td class="px-2 py-2 text-right">
                <button type="button" class="grid h-8 w-8 place-items-center rounded-lg text-muted hover:bg-red-50 hover:text-expense" :aria-label="`Take ${r.code} out`" @click="agentIds = agentIds.filter(id => id !== r.id)"><X :size="15" /></button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p v-else class="rounded-lg bg-canvas px-3 py-2 text-xs text-muted">Pick one or more agents above; each one gets its own rate here.</p>

      <!-- Each agent's amounts are in its own currency (Agents page) -->
      <div v-if="isNew" class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="space-y-1 text-xs font-medium text-muted">From<DatePicker v-model="startsOn" aria-label="First day" /></div>
      </div>

      <p v-if="!isNew" class="text-[11px] text-muted">An agent taken out is ended if weeks were entered for it (they stay), else removed. An agent put in starts today.</p>
      <AppAlert v-if="group?.partners.length" type="warning" title="This group still has partners" :message="`${group.partners.map(p => `${p.partner_name} ${formatRate(p.commission_rate)}`).join(', ')}. Saving takes them off: from then on each agent’s rate above is used. Weeks already saved keep their partner shares.`" />
      <p v-else-if="isNew" class="text-[11px] text-muted">An agent can be in one group at a time. Results are entered once a week; the first day can be in the past to enter earlier weeks.</p>

      <AppAlert v-if="formError" title="Couldn’t save the group" :message="formError" dismissible @dismiss="formError = ''" />
      <div class="flex flex-wrap items-center justify-end gap-2 border-t border-border pt-4">
        <NuxtLink to="/admin/platform-assignments" class="rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-muted">Cancel</NuxtLink>
        <button type="submit" class="rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : isNew ? 'Assign' : 'Update' }}</button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, X } from 'lucide-vue-next'
import type { AssignmentGroup, Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { todayISO } from '~/utils/inventory'
import { formatRate } from '~/utils/winlose'

const route = useRoute()
const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canManage = computed(() => can('platform-assignments.manage'))
const isNew = computed(() => route.params.id === 'new')

const platforms = ref<Platform[]>([])
const group = ref<AssignmentGroup | null>(null)
const loading = ref(true)
const loadError = ref('')

const title = ref('')
const agentIds = ref<string[]>([])
const rates = reactive<Record<string, string>>({})
const startsOn = ref(todayISO())
let initial = ''
const snapshot = () => JSON.stringify([title.value.trim(), agentIds.value, agentIds.value.map(id => rates[id] || ''), startsOn.value])

// Active agents, plus the group's own (one turned off since stays listed).
const agentOptions = computed(() => platforms.value
  .filter(p => p.is_active || group.value?.agents.some(a => a.platform_id === p.id))
  .map(p => ({ value: p.id, label: p.is_active ? p.code : `${p.code} (off)` })))
const rows = computed(() => agentIds.value.map((id) => {
  const p = platforms.value.find(x => x.id === id)
  const saved = group.value?.agents.find(a => a.platform_id === id)
  return { id, code: p?.code || saved?.platform_code || '?', weeks: saved?.weeks || 0 }
}))

// An agent picked gets an empty rate (its own default, when it has one); one taken out loses it.
watch(agentIds, (ids) => {
  for (const id of ids) {
    if (rates[id] === undefined) {
      const rate = Number(platforms.value.find(p => p.id === id)?.default_rate || 0)
      rates[id] = rate ? String(rate) : ''
    }
  }
  for (const id of Object.keys(rates)) if (!ids.includes(id)) delete rates[id]
})

onMounted(async () => {
  try {
    platforms.value = (await api<{ items: Platform[] }>('/platforms', { query: { include_inactive: 'true' } })).items || []
    if (!isNew.value) {
      const g = await api<AssignmentGroup>(`/platform-groups/${route.params.id}`)
      group.value = g
      title.value = g.title
      for (const a of g.agents) rates[a.platform_id] = String(Number(a.commission_rate))
      agentIds.value = g.agents.map(a => a.platform_id)
    }
  } catch (e: any) {
    loadError.value = e?.data?.error || 'Could not load the group.'
  } finally {
    loading.value = false
    await nextTick()
    initial = snapshot()
  }
})

const saving = ref(false)
const saved = ref(false)
const formError = ref('')
const tried = ref(false)
const rateProblem = (value: string | undefined) => !value ? 'a rate is needed' : /^\d+(\.\d{1,4})?$/.test(value) && Number(value) <= 100 ? '' : 'a rate is from 0 to 100 (at most 4 decimals)'

onBeforeRouteLeave(() => saved.value || loading.value || snapshot() === initial || window.confirm('Leave without saving your changes?'))

async function save() {
  tried.value = true
  formError.value = ''
  const bad = rows.value.find(r => rateProblem(rates[r.id]))
  formError.value = !title.value.trim() ? 'Give the group a title.'
    : !agentIds.value.length ? 'Choose at least one agent.'
      : bad ? `${bad.code}: ${rateProblem(rates[bad.id])}.` : ''
  if (formError.value) return
  const body = {
    title: title.value.trim(),
    agents: agentIds.value.map(id => ({ platform_id: id, commission_rate: rates[id] })),
    ...(isNew.value ? { starts_on: startsOn.value } : {}),
  }
  saving.value = true
  try {
    const g = isNew.value
      ? await api<AssignmentGroup>('/platform-groups', { method: 'POST', body })
      : await api<AssignmentGroup>(`/platform-groups/${route.params.id}`, { method: 'PATCH', body })
    saved.value = true
    toast.success(`${g.agents.length} ${g.agents.length === 1 ? 'agent' : 'agents'}; new weeks use these rates.`, { title: `“${g.title}” ${isNew.value ? 'made' : 'updated'}` })
    await navigateTo('/admin/platform-assignments')
  } catch (e: any) {
    formError.value = e?.data?.error || 'Could not save the group.'
  } finally {
    saving.value = false
  }
}
</script>
