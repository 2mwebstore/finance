<template>
  <section class="rp-compact space-y-4">
    <header>
      <button type="button" class="mb-2 inline-flex items-center gap-1 rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-semibold text-navy hover:bg-canvas" @click="goBack"><ChevronLeft :size="14" /> Back</button>
      <h1 class="text-xl font-bold text-navy">{{ isNew ? 'Add agent' : 'Update agent' }} <span v-if="editing" class="ml-1 font-mono text-base font-semibold text-muted">{{ editing.code }}</span></h1>
    </header>

    <AppAlert v-if="!canSave" type="warning" :title="isNew ? 'You can’t add agents' : 'You can’t change agents'" :message="isNew ? 'It needs Add Agents.' : 'It needs Update Agents.'" />
    <div v-else-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="loadError" title="Couldn’t load the agent" :message="loadError"><template #action><NuxtLink to="/admin/platforms">Back to the agents</NuxtLink></template></AppAlert>
    <form v-else class="space-y-4 rounded-xl border border-border bg-white p-4 sm:p-5" novalidate @submit.prevent="save">
      <p class="text-[11px] text-muted">The user is required and must be unique; the name and groups are optional.</p>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="block space-y-1 text-xs font-medium text-muted">User <span class="text-expense">*</span><input v-model.trim="form.code" maxlength="64" :aria-invalid="formCheck.invalid('code')" class="h-10 w-full rounded-lg border border-border px-3 font-mono text-sm text-navy" placeholder="e.g. 12K_YMLLJB_V2" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Name <span class="font-normal">(optional)</span><input v-model.trim="form.name" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="h-10 w-full rounded-lg border border-border px-3 text-sm text-navy" /></label>
        <!-- With one branch it is chosen for you; an owner/admin can move an agent to another branch -->
        <div v-if="showBranch && (!editing || hasFullAccess)" class="space-y-1 text-xs font-medium text-muted">Branch<SearchSelect v-model="form.branch_id" :options="formBranchOptions" :searchable="false" aria-label="Agent branch" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">Default currency<SearchSelect v-model="form.default_currency" :options="currencyOptions" :searchable="false" aria-label="Default currency" :aria-invalid="formCheck.invalid('default_currency')" /></div>
      </div>

      <!-- Groups (optional, needs Update Group Agent Assignments): the agent's branch's groups; a
           turned-off agent can't be in any -->
      <div v-if="showGroups" class="space-y-1 text-xs font-medium text-muted">
        <span>Groups <span class="font-normal">(optional)</span></span>
        <p v-if="groupsLoading" class="h-10 rounded-lg border border-border bg-canvas px-3 py-2.5 text-sm font-normal">Loading groups…</p>
        <p v-else-if="groupsError" class="rounded-lg bg-red-50 px-3 py-2 text-xs font-normal text-expense">{{ groupsError }} Groups are left as they are.</p>
        <SearchSelect v-else v-model="groupIds" multiple :options="groupOptions" placeholder="No group" empty-text="No groups for this branch" aria-label="Groups" />
      </div>
      <p v-else-if="editing && !editing.is_active" class="rounded-lg bg-canvas px-3 py-2 text-[11px] text-muted">This agent is turned off, so it can’t be in a group: turn it on in the list first.</p>
      <!-- Each group picked, with the agent's type and rate in it -->
      <ul v-if="showGroups && !groupsLoading && !groupsError && chosenGroups.length" class="space-y-1.5">
        <li v-for="g in chosenGroups" :key="g.id" class="flex flex-wrap items-center gap-x-2 gap-y-1.5 rounded-lg border border-border px-2.5 py-2">
          <span class="min-w-0 flex-1 basis-full truncate text-sm font-semibold text-navy sm:basis-0">{{ g.title }}<span v-if="!g.is_active" class="ml-1 text-[11px] font-normal text-muted">(off)</span><span v-if="!agentBranch && g.branch_code" class="ml-1 text-[11px] font-normal text-muted">· {{ g.branch_code }}</span></span>
          <select v-model="groupKinds[g.id]" :aria-label="`Type in ${g.title}`" class="h-9 rounded-lg border border-border bg-white px-2 text-sm font-semibold text-navy">
            <option v-for="t in CALC_TYPES" :key="t.value" :value="t.value">{{ t.label }}</option>
          </select>
          <span class="inline-flex items-center gap-1 text-xs text-muted"><input v-model.trim="groupRates[g.id]" type="text" inputmode="decimal" :aria-label="`Rate in ${g.title}`" :aria-invalid="groupsTried && !!rateProblem(groupRates[g.id]) ? 'true' : undefined" class="h-9 w-20 rounded-lg border border-border px-2 text-right text-sm text-navy" placeholder="0" />%</span>
          <span v-if="!rateProblem(groupRates[g.id])" class="whitespace-nowrap text-[11px] tabular-nums text-muted">→ {{ totalRate(groupKinds[g.id] || 'PO', Number(groupRates[g.id] || 0)) }}</span>
          <button type="button" class="ml-auto grid h-8 w-8 place-items-center rounded-lg text-muted hover:bg-red-50 hover:text-expense" :aria-label="`Take out of ${g.title}`" @click="groupIds = groupIds.filter(id => id !== g.id)"><X :size="15" /></button>
        </li>
      </ul>
      <p v-if="showGroups && editing && removedGroups.length" class="rounded-lg bg-amber-50 px-3 py-2 text-[11px] text-amber-800">Taken out of {{ removedGroups.join(', ') }}: ended today if weeks were entered (they stay), else removed.</p>

      <p v-if="editing" class="rounded-lg bg-canvas px-3 py-2 text-[11px] text-muted">Changing the currency only affects groups assigned from now on; each assignment keeps its own, and saved weeks keep the currency they were entered in.</p>
      <AppAlert v-if="formError || formCheck.messages.length" :title="formError ? 'Couldn’t save' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <div class="flex flex-wrap items-center justify-end gap-2 border-t border-border pt-4">
        <NuxtLink to="/admin/platforms" class="rounded-lg border border-border bg-white px-4 py-2 text-sm font-semibold text-muted">Cancel</NuxtLink>
        <button type="submit" class="rounded-lg bg-primary px-5 py-2 text-sm font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : isNew ? 'Add agent' : 'Update' }}</button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { ChevronLeft, X } from 'lucide-vue-next'
import type { AssignmentGroup, Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useCurrency } from '~/composables/useCurrency'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { rule } from '~/utils/validation'
import { CALC_TYPES, totalRate } from '~/utils/winlose'

// An agent's own page: /admin/platforms/new adds one (needs Add Agents), /admin/platforms/<id>
// changes it (needs Update Agents). Saving goes back to the agents.
const route = useRoute()
const { api } = useApi()
const { can, hasFullAccess } = usePermissions()
const toast = useToast()
const isNew = computed(() => route.params.id === 'new')
const canSave = computed(() => can(isNew.value ? 'platforms.create' : 'platforms.edit'))
const { currencyOptions, load: loadCurrency } = useCurrency()

// People who work in one branch just get theirs (nothing to choose); with several, a Branch choice.
const { branches, everyBranch } = useBranchOptions('All branches')
const { activeBranchId } = useActiveBranch()
const showBranch = computed(() => branches.value.length > 1)

const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const saved = ref(false)
const formError = ref('')
// The agent being changed (null: a new one).
const editing = ref<Platform | null>(null)
const form = reactive({ code: '', name: '', default_currency: 'USD', branch_id: '' })
const formBranchOptions = computed(() => [
  ...(editing.value ? [] : [{ value: '', label: 'My own branch' }]),
  ...(editing.value && !editing.value.branch_id ? [{ value: '', label: 'Shared — choose a branch to move it' }] : []),
  ...branches.value.map(branch => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })),
])

// Same rules as the API (services.PlatformInput / SavePlatform).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    code: [rule.required('user'), (value: unknown) => /^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$/.test(String(value || '')) ? '' : 'use letters, digits, _ . or - in the user'],
    name: [rule.maxLength('name', 128)],
    default_currency: [rule.required('currency')],
  }),
))

// --- the agent's groups (optional) ---
// Putting an agent in groups changes them, so it needs Update Group Agent Assignments; a turned-off
// agent can't be in a group (turn it on first).
const { user } = useAuth()
const canGroups = computed(() => can('platform-assignments.edit'))
const showGroups = computed(() => canGroups.value && !(editing.value && !editing.value.is_active))
const groups = ref<AssignmentGroup[]>([])
const groupsLoading = ref(false)
const groupsError = ref('')
const groupsTried = ref(false)
const groupIds = ref<string[]>([])
// The groups the agent was in when the page opened.
const startGroupIds = ref<string[]>([])
const groupRates = reactive<Record<string, string>>({})
const groupKinds = reactive<Record<string, 'PO' | 'GI'>>({})

// The agent's branch: the one chosen, else its own (null: a shared agent, any branch's groups). A
// new agent with none goes to the person's own branch, as the API does.
const agentBranch = computed<string | null>(() => {
  if (form.branch_id) return form.branch_id
  if (editing.value) return editing.value.branch_id || null
  const own = user.value?.default_branch_id
  if (own && everyBranch.value.some(b => b.id === own)) return own
  return everyBranch.value.length === 1 ? everyBranch.value[0]!.id : null
})
const groupFits = (g: AssignmentGroup) => !agentBranch.value || g.branch_id === agentBranch.value
// Switched-on groups of the agent's branch, plus any it is in now (one turned off since stays listed).
const groupOptions = computed(() => groups.value
  .filter(g => groupFits(g) && (g.is_active || startGroupIds.value.includes(g.id)))
  .map(g => ({ value: g.id, label: `${g.title}${!agentBranch.value && g.branch_code ? ` · ${g.branch_code}` : ''}${g.is_active ? '' : ' (off)'}` })))
const chosenGroups = computed(() => groupIds.value
  .map(id => groups.value.find(g => g.id === id))
  .filter((g): g is AssignmentGroup => !!g && groupFits(g)))
const removedGroups = computed(() => startGroupIds.value
  .filter(id => !chosenGroups.value.some(g => g.id === id))
  .map(id => groups.value.find(g => g.id === id)?.title || '?'))
// An empty rate is 0%.
const rateProblem = (value: string | undefined) => !value || (/^\d+(\.\d{1,4})?$/.test(value) && Number(value) <= 100) ? '' : 'a rate is from 0 to 100 (at most 4 decimals)'

// A group picked gets the agent's default rate (empty when none) and PO; one taken out and picked
// again gets back what it had.
watch(groupIds, (ids) => {
  for (const id of ids) {
    if (groupRates[id] === undefined) {
      const rate = Number(editing.value?.default_rate || 0)
      groupRates[id] = rate ? String(rate) : ''
    }
    if (!groupKinds[id]) groupKinds[id] = 'PO'
  }
})

async function loadGroups(agent: Platform | null) {
  if (!canGroups.value || (agent && !agent.is_active)) return
  groupsLoading.value = true
  try {
    // Every group the person works with; the page shows the agent's branch's.
    const result = await api<{ items: AssignmentGroup[] }>('/platform-groups', { query: { branch_id: 'all' } })
    groups.value = result.items || []
    if (agent) {
      for (const g of groups.value) {
        const mine = g.agents.find(a => a.platform_id === agent.id)
        if (!mine) continue
        groupRates[g.id] = String(Number(mine.commission_rate))
        groupKinds[g.id] = mine.calc_type === 'GI' ? 'GI' : 'PO'
        startGroupIds.value.push(g.id)
      }
      groupIds.value = [...startGroupIds.value]
    }
  } catch (e: any) {
    groupsError.value = e?.data?.error || 'Could not load the groups.'
  } finally {
    groupsLoading.value = false
  }
}

// What was there when the page opened, to ask before leaving with changes.
let initial = ''
const snapshot = () => JSON.stringify([form, groupIds.value, groupIds.value.map(id => [groupRates[id] || '', groupKinds[id] || 'PO'])])

onMounted(async () => {
  loadCurrency()
  try {
    if (isNew.value) {
      form.branch_id = activeBranchId.value
      await loadGroups(null)
    } else {
      const agent = await api<Platform>(`/platforms/${route.params.id}`)
      editing.value = agent
      Object.assign(form, { code: agent.code, name: agent.name, default_currency: agent.default_currency, branch_id: agent.branch_id || '' })
      await loadGroups(agent)
    }
  } catch (e: any) {
    loadError.value = e?.data?.error || 'Could not load the agent.'
  } finally {
    loading.value = false
    await nextTick()
    initial = snapshot()
  }
})

onBeforeRouteLeave(() => saved.value || loading.value || snapshot() === initial || window.confirm('Leave without saving your changes?'))

// Back where the person came from (the agents), else to the agents.
const router = useRouter()
function goBack() {
  if (window.history.state?.back) router.back()
  else navigateTo('/admin/platforms')
}

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  // The groups are sent only once they have loaded (else they are left as they are).
  const sendGroups = showGroups.value && !groupsLoading.value && !groupsError.value
  if (sendGroups) {
    groupsTried.value = true
    const bad = chosenGroups.value.find(g => rateProblem(groupRates[g.id]))
    if (bad) {
      formError.value = `${bad.title}: ${rateProblem(groupRates[bad.id])}.`
      return
    }
  }
  saving.value = true
  try {
    const body = {
      ...form,
      is_active: editing.value ? editing.value.is_active : true,
      ...(sendGroups ? { groups: chosenGroups.value.map(g => ({ group_id: g.id, commission_rate: groupRates[g.id] || '0', calc_type: groupKinds[g.id] || 'PO' })) } : {}),
    }
    const result = editing.value
      ? await api<{ counted_weeks?: number }>(`/platforms/${editing.value.id}`, { method: 'PATCH', body })
      : await api<{ counted_weeks?: number }>('/platforms', { method: 'POST', body })
    saved.value = true
    const n = chosenGroups.value.length
    const weeks = result?.counted_weeks || 0
    const counted = weeks ? ` ${weeks} ${weeks === 1 ? 'week' : 'weeks'} saved before now count in its groups.` : ''
    toast.success(`${form.code} is saved${sendGroups && n ? `, in ${n} ${n === 1 ? 'group' : 'groups'}` : ''}.${counted}`, { title: editing.value ? 'Agent updated' : 'Agent added' })
    await navigateTo('/admin/platforms')
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this agent.'
  } finally {
    saving.value = false
  }
}
</script>
