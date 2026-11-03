<template>
  <section class="rp-compact space-y-4">
    <header>
      <button type="button" class="mb-2 inline-flex items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="goBack"><ChevronLeft :size="14" /> Back</button>
      <h1 class="text-xl font-bold text-navy">{{ isNew ? 'Assign agents' : 'Update group' }}</h1>
    </header>

    <AppAlert v-if="!canSave" type="warning" :title="isNew ? 'You can’t add groups' : 'You can’t change groups'" :message="isNew ? 'It needs Add Group Agent Assignments.' : 'It needs Update Group Agent Assignments.'" />
    <div v-else-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="loadError" title="Couldn’t load the group" :message="loadError"><template #action><NuxtLink to="/admin/platform-assignments">Back to the groups</NuxtLink></template></AppAlert>
    <form v-else class="space-y-4 rounded-xl border border-border bg-white p-4 sm:p-5" novalidate @submit.prevent="save">
      <!-- Only with several branches: a new group picks its branch (its agents change with it); a saved one keeps its own -->
      <div v-if="rp.show.value" class="space-y-1 text-xs font-medium text-muted">Branch
        <SearchSelect v-if="isNew" v-model="branchId" :options="rp.optionsFor(false)" :searchable="false" aria-label="Group branch" />
        <p v-else class="h-10 rounded-lg border border-border bg-canvas px-3 py-2.5 text-sm font-normal text-muted">{{ group?.branch_code }}</p>
      </div>

      <label class="block space-y-1 text-xs font-medium text-muted">Title <span class="text-expense">*</span>
        <input v-model="title" maxlength="128" :aria-invalid="tried && !title.trim() ? 'true' : undefined" class="h-10 w-full rounded-lg border border-border px-3 text-sm text-navy" placeholder="e.g. V-series Phnom Penh" />
      </label>

      <div class="space-y-1 text-xs font-medium text-muted">Agents <span class="text-expense">*</span>
        <SearchSelect v-model="agentIds" multiple :options="agentOptions" placeholder="Choose one or more agents" aria-label="Agents" :aria-invalid="tried && !agentIds.length" />
      </div>

      <!-- Each agent picked, with its rate -->
      <!-- Phones: one card per agent (code and remove on top, then Type and Rate) -->
      <ul v-if="rows.length && phone" class="space-y-2">
        <li v-for="r in rows" :key="r.id" class="rounded-xl border border-border bg-white p-3">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <p class="break-all font-mono text-sm font-bold text-navy">{{ r.code }}</p>
              <p v-if="r.weeks" class="text-[11px] text-muted">{{ r.weeks }} {{ r.weeks === 1 ? 'week' : 'weeks' }} entered</p>
            </div>
            <button type="button" class="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-muted hover:bg-red-50 hover:text-expense" :aria-label="`Take ${r.code} out`" @click="agentIds = agentIds.filter(id => id !== r.id)"><X :size="16" /></button>
          </div>
          <div class="keep-cols mt-3 grid grid-cols-2 gap-3">
            <div class="space-y-1">
              <p class="text-[10px] font-bold uppercase tracking-wide text-slate-500">Type</p>
              <div class="keep-cols grid grid-cols-2 gap-1 rounded-lg bg-canvas p-1" role="radiogroup" :aria-label="`Type for ${r.code}`">
                <button v-for="t in CALC_TYPES" :key="t.value" type="button" role="radio" :aria-checked="kinds[r.id] === t.value" class="h-8 rounded-md text-sm font-bold" :class="kinds[r.id] === t.value ? 'bg-white text-primary shadow-sm' : 'text-muted'" @click="kinds[r.id] = t.value">{{ t.label }}</button>
              </div>
            </div>
            <div class="space-y-1">
              <p class="text-[10px] font-bold uppercase tracking-wide text-slate-500">Rate</p>
              <div class="flex items-center gap-1.5 text-xs text-muted">
                <input v-model.trim="rates[r.id]" type="text" inputmode="decimal" :aria-label="`Rate for ${r.code}`" :aria-invalid="tried && !!rateProblem(rates[r.id]) ? 'true' : undefined" class="h-10 min-w-0 flex-1 rounded-lg border border-border px-2.5 text-right text-sm text-navy" placeholder="0" />%
              </div>
            </div>
          </div>
          <p class="mt-2 flex flex-wrap items-center justify-between gap-x-3 gap-y-0.5 rounded-lg bg-canvas px-2.5 py-1.5 text-[11px] text-muted">
            <span>{{ CALC_TYPES.find(t => t.value === kinds[r.id])?.formula }}</span>
            <span v-if="!rateProblem(rates[r.id])" class="font-semibold tabular-nums text-navy">Total at {{ totalRate(kinds[r.id] || 'PO', Number(rates[r.id] || 0)) }}</span>
          </p>
        </li>
      </ul>
      <div v-else-if="rows.length" class="overflow-hidden rounded-lg border border-border">
        <table class="w-full text-left text-sm">
          <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
            <tr>
              <th class="px-2 py-2 font-bold sm:px-3">Agent</th>
              <th class="px-2 py-2 font-bold sm:w-80 sm:px-3">Type</th>
              <th class="px-2 py-2 font-bold sm:w-56 sm:px-3">Rate → Total at</th>
              <th class="px-1 py-2 sm:w-12 sm:px-2"><span class="sr-only">Take out</span></th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border">
            <tr v-for="r in rows" :key="r.id">
              <td class="px-2 py-2 sm:px-3">
                <span class="font-mono text-xs font-semibold text-navy sm:text-sm">{{ r.code }}</span>
                <span v-if="r.weeks" class="block text-[11px] text-muted">{{ r.weeks }} {{ r.weeks === 1 ? 'week' : 'weeks' }} entered</span>
              </td>
              <td class="px-2 py-2 sm:px-3">
                <div class="flex flex-wrap items-center gap-x-3 gap-y-1">
                  <select v-model="kinds[r.id]" :aria-label="`Type for ${r.code}`" class="h-9 rounded-lg border border-border bg-white px-2 text-sm font-semibold text-navy">
                    <option v-for="t in CALC_TYPES" :key="t.value" :value="t.value">{{ t.label }}</option>
                  </select>
                  <span class="hidden whitespace-nowrap text-[11px] text-muted sm:inline">{{ CALC_TYPES.find(t => t.value === kinds[r.id])?.formula }}</span>
                </div>
              </td>
              <td class="px-2 py-2 sm:px-3">
                <label class="flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-xs text-muted sm:flex-nowrap">
                  <span class="inline-flex items-center gap-1.5"><input v-model.trim="rates[r.id]" type="text" inputmode="decimal" :aria-label="`Rate for ${r.code}`" :aria-invalid="tried && !!rateProblem(rates[r.id]) ? 'true' : undefined" class="h-9 w-16 rounded-lg border border-border px-2.5 text-right text-sm text-navy sm:w-24" placeholder="0" />%</span>
                  <!-- PO: the Total is at what is left, 60% → 40%; GI: at the rate with a minus, 60% → −60% (under the rate on phones) -->
                  <span v-if="!rateProblem(rates[r.id])" class="whitespace-nowrap tabular-nums max-sm:w-full">→ {{ totalRate(kinds[r.id] || 'PO', Number(rates[r.id] || 0)) }}</span>
                </label>
              </td>
              <td class="px-1 py-2 text-right sm:px-2">
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
      <p v-else-if="isNew" class="text-[11px] text-muted">An agent can be in more than one group, at its own rate in each. Results are entered once a week; the first day can be in the past to enter earlier weeks.</p>

      <AppAlert v-if="formError" title="Couldn’t save the group" :message="formError" dismissible @dismiss="formError = ''" />
      <div class="flex flex-wrap items-center justify-end gap-2 border-t border-border pt-4">
        <NuxtLink to="/admin/platform-assignments" class="rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-muted">Cancel</NuxtLink>
        <button type="submit" class="rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : isNew ? 'Assign' : 'Update' }}</button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { useMediaQuery } from '@vueuse/core'
import { ChevronLeft, X } from 'lucide-vue-next'
import type { AssignmentGroup, Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { todayISO } from '~/utils/inventory'
import { useRpBranch } from '~/composables/useRpBranch'
import { byAgentCode, CALC_TYPES, formatRate, totalRate } from '~/utils/winlose'

const route = useRoute()
const phone = useMediaQuery('(max-width: 639px)')
const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
// Assign agents needs Add Group Agent Assignments, a group's page Update.
const canSave = computed(() => can(isNew.value ? 'platform-assignments.create' : 'platform-assignments.edit'))
const isNew = computed(() => route.params.id === 'new')

const platforms = ref<Platform[]>([])
const group = ref<AssignmentGroup | null>(null)
const loading = ref(true)
const loadError = ref('')

const title = ref('')
const kinds = reactive<Record<string, 'PO' | 'GI'>>({})
const agentIds = ref<string[]>([])
const rates = reactive<Record<string, string>>({})
const startsOn = ref(todayISO())
let initial = ''
const snapshot = () => JSON.stringify([title.value.trim(), agentIds.value, agentIds.value.map(id => rates[id] || ''), agentIds.value.map(id => kinds[id] || 'PO'), startsOn.value])

// Active agents, plus the group's own (one turned off since stays listed).
const agentOptions = computed(() => platforms.value
  .filter(p => p.is_active || group.value?.agents.some(a => a.platform_id === p.id))
  .map(p => ({ value: p.id, label: p.is_active ? p.code : `${p.code} (off)` })))
// The agent picked last is on top, so a pick is seen at once (the list can be 100 long); the
// group's own agents, as loaded, follow A to Z.
const loadedIds = ref<string[]>([])
const rows = computed(() => {
  const row = (id: string) => {
    const p = platforms.value.find(x => x.id === id)
    const saved = group.value?.agents.find(a => a.platform_id === id)
    return { id, code: p?.code || saved?.platform_code || '?', weeks: saved?.weeks || 0 }
  }
  const added = agentIds.value.filter(id => !loadedIds.value.includes(id)).reverse().map(row)
  const loaded = agentIds.value.filter(id => loadedIds.value.includes(id)).map(row).sort((a, b) => byAgentCode(a.code, b.code))
  return [...added, ...loaded]
})

// An agent picked gets an empty rate (its own default, when it has one); one taken out loses it.
watch(agentIds, (ids) => {
  for (const id of ids) {
    if (rates[id] === undefined) {
      const rate = Number(platforms.value.find(p => p.id === id)?.default_rate || 0)
      rates[id] = rate ? String(rate) : ''
    }
  }
  for (const id of ids) if (!kinds[id]) kinds[id] = 'PO'
  for (const id of Object.keys(rates)) if (!ids.includes(id)) { delete rates[id]; delete kinds[id] }
})

// The branch the group is in: a new one takes the branch chosen for the RP pages (the person's own
// when none); one being updated keeps its own. The agents offered are that branch's and the shared ones.
const rp = useRpBranch()
const branchId = ref('')
async function loadPlatforms() {
  platforms.value = (await api<{ items: Platform[] }>('/platforms', { query: { include_inactive: 'true', assignable: 'true', branch_id: branchId.value || undefined } })).items || []
}
watch(branchId, async () => {
  if (!isNew.value || loading.value) return
  try { await loadPlatforms() } catch { platforms.value = [] }
  // An agent of the branch left is taken out of the picks.
  agentIds.value = agentIds.value.filter(id => platforms.value.some(p => p.id === id))
})

// The branches can arrive after the page opens (an owner's are loaded from the API): a new group then
// starts in the branch chosen for the RP pages.
watch(() => rp.effective.value, (value) => { if (isNew.value && !branchId.value && value) branchId.value = value })

onMounted(async () => {
  try {
    if (isNew.value) {
      branchId.value = rp.effective.value
      await loadPlatforms()
    } else {
      const g = await api<AssignmentGroup>(`/platform-groups/${route.params.id}`)
      branchId.value = g.branch_id
      await loadPlatforms()
      group.value = g
      title.value = g.title
      for (const a of g.agents) {
        rates[a.platform_id] = String(Number(a.commission_rate))
        kinds[a.platform_id] = a.calc_type === 'GI' ? 'GI' : 'PO'
      }
      agentIds.value = g.agents.map(a => a.platform_id)
      loadedIds.value = [...agentIds.value]
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
// An empty rate is 0%.
const rateProblem = (value: string | undefined) => !value || (/^\d+(\.\d{1,4})?$/.test(value) && Number(value) <= 100) ? '' : 'a rate is from 0 to 100 (at most 4 decimals)'

onBeforeRouteLeave(() => saved.value || loading.value || snapshot() === initial || window.confirm('Leave without saving your changes?'))

// Back where the person came from (the groups), else to the groups.
const router = useRouter()
function goBack() {
  if (window.history.state?.back) router.back()
  else navigateTo('/admin/platform-assignments')
}

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
    agents: agentIds.value.map(id => ({ platform_id: id, commission_rate: rates[id] || '0', calc_type: kinds[id] || 'PO' })),
    ...(isNew.value ? { starts_on: startsOn.value, branch_id: branchId.value || undefined } : {}),
  }
  saving.value = true
  try {
    const g = isNew.value
      ? await api<AssignmentGroup>('/platform-groups', { method: 'POST', body })
      : await api<AssignmentGroup>(`/platform-groups/${route.params.id}`, { method: 'PATCH', body })
    saved.value = true
    toast.success(`${g.agents.length} ${g.agents.length === 1 ? 'agent' : 'agents'}; new weeks use these types and rates.`, { title: `“${g.title}” ${isNew.value ? 'made' : 'updated'}` })
    await navigateTo('/admin/platform-assignments')
  } catch (e: any) {
    formError.value = e?.data?.error || 'Could not save the group.'
  } finally {
    saving.value = false
  }
}
</script>
