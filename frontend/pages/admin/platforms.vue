<template>
  <section class="rp-compact space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">RP</p>
        <h1 class="text-xl font-bold text-navy">Agents</h1>
      </div>
      <div class="flex gap-2">
        <NuxtLink v-if="can('platform-assignments.view', 'platform-assignments.create', 'platform-assignments.edit', 'platform-assignments.delete')" to="/admin/platform-assignments" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy hover:bg-canvas"><Network :size="16" /> Group assignments</NuxtLink>
        <button v-if="canCreate" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> Add agent</button>
      </div>
    </header>

    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by user or name…" /></div>
      <!-- Only for people who work in more than one branch -->
      <div v-if="showBranch" class="sm:w-56"><SearchSelect v-model="branchFilter" :options="branchOptions" aria-label="Branch" /></div>
      <label class="flex items-center gap-2 text-xs font-medium text-navy"><input v-model="showInactive" type="checkbox" class="accent-primary" /> Show turned-off agents</label>
    </div>

    <div v-if="loading && !items.length" class="text-sm text-muted">Loading agents…</div>
    <AppAlert v-else-if="error && !items.length" title="Couldn’t load agents" :message="error"><template #action><button type="button" @click="reset">Try again</button></template></AppAlert>
    <!-- Like Data entry: on phones the buttons are icons only -->
    <div v-else class="overflow-hidden rounded-xl border border-border bg-white">
      <table class="w-full text-left text-sm">
        <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
          <tr>
            <th class="px-2 py-2 font-bold sm:px-3">User</th>
            <!-- On phones the currency goes under the user -->
            <th class="hidden px-2 py-2 font-bold sm:table-cell sm:px-3">Currency</th>
            <th v-if="showBranch" class="hidden px-2 py-2 font-bold sm:table-cell sm:px-3">Branch</th>
            <th class="px-2 py-2 text-right font-bold sm:px-3">Groups</th>
            <th v-if="canEdit || canDelete" class="px-2 py-2 sm:px-3"><span class="sr-only">Actions</span></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-border">
          <tr v-for="platform in shown" :key="platform.id" :class="platform.is_active ? 'text-navy' : 'bg-canvas/60 text-slate-400'">
            <td class="px-2 py-2.5 sm:px-3">
              <span class="whitespace-nowrap font-mono font-semibold">{{ platform.code }}</span>
              <span v-if="!platform.is_active" class="ml-1.5 rounded-full bg-slate-100 px-1.5 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span>
              <span class="block text-[11px] text-muted"><span class="sm:hidden">{{ platform.default_currency }}</span><template v-if="platform.name && platform.name !== platform.code"><span class="sm:hidden"> · </span>{{ platform.name }}</template></span>
            </td>
            <td class="hidden px-2 py-2.5 sm:table-cell sm:px-3">{{ platform.default_currency }}</td>
            <td v-if="showBranch" class="hidden px-2 py-2.5 sm:table-cell sm:px-3"><span v-if="platform.branch_code">{{ platform.branch_code }}</span><span v-else class="text-muted" title="Shared: every branch sees it">Shared</span></td>
            <td class="px-2 py-2.5 text-right tabular-nums sm:px-3">{{ platform.active_hqs }}</td>
            <td v-if="canEdit || canDelete" class="px-2 py-2 sm:px-3">
              <div class="flex justify-end gap-1 sm:gap-1.5">
                <button v-if="canEdit" type="button" class="action-chip action-edit" title="Edit" @click="openEdit(platform)"><Pencil :size="13" /> <span class="max-sm:sr-only">Edit</span></button>
                <button v-if="canEdit" type="button" class="action-chip" :class="platform.is_active ? 'action-danger' : 'action-success'" :title="platform.is_active ? 'Turn off' : 'Turn on'" :disabled="busyId === platform.id" @click="platform.is_active ? (turningOff = platform) : setActive(platform, true)"><Power :size="13" /> <span class="max-sm:sr-only">{{ platform.is_active ? 'Turn off' : 'Turn on' }}</span></button>
                <button v-if="canDelete" type="button" class="action-chip action-danger" title="Delete" @click="openDelete(platform)"><Trash2 :size="13" /> <span class="max-sm:sr-only">Delete</span></button>
              </div>
            </td>
          </tr>
          <tr v-if="!shown.length"><td :colspan="(canEdit || canDelete ? 4 : 3) + (showBranch ? 1 : 0)" class="px-3 py-8 text-center text-sm text-muted">{{ search ? `No agent matches “${search}”.` : 'No agents yet.' }}</td></tr>
        </tbody>
      </table>
    </div>
    <InfiniteLoader :has-more="hasMore" :loading="loadingMore" :count="items.length" @more="loadMore" />

    <!-- Delete: only an agent never in a group or entered; a used one can only be turned off -->
    <AppModal :model-value="!!deleting" form :title="deleting ? `Delete ${deleting.code}?` : ''" eyebrow="DELETE AGENT" subtitle="Only an agent that was never in a group or entered can be deleted; a used one can only be turned off." :busy="deleteBusy" width="460px" @update:model-value="(open) => { if (!open && !deleteBusy) deleting = null }" @submit="deleteAgent">
      <AppAlert v-if="deleteError" compact :message="deleteError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="deleteBusy" @click="deleting = null">Cancel</button>
        <button type="submit" class="bg-expense text-white disabled:opacity-60" :disabled="deleteBusy">{{ deleteBusy ? 'Deleting…' : 'Delete agent' }}</button>
      </template>
    </AppModal>

    <!-- Turning an agent off needs a reason (the audit log keeps it); turning on doesn't -->
    <InventoryReasonDialog :model-value="!!turningOff" :title="turningOff ? `Turn off ${turningOff.code}?` : ''" eyebrow="TURN OFF AGENT" subtitle="It and its weeks are hidden from Data entry and the reports until it is turned on again; nothing is deleted. The audit log keeps your reason." label="Reason for turning off" confirm-label="Turn off" danger :busy="!!turningOff && busyId === turningOff.id" :server-error="turnOffError" @update:model-value="(open) => { if (!open) { turningOff = null; turnOffError = '' } }" @confirm="(reason) => turningOff && setActive(turningOff, false, reason)" />

    <AppModal :model-value="formOpen" form :title="editing ? `Edit ${editing.code}` : 'Add agent'" eyebrow="AGENT" subtitle="The user is required and must be unique; the name and groups are optional." :busy="saving" width="540px" @update:model-value="(open) => { if (!open && !saving) formOpen = false }" @submit="save">
      <template #icon><Layers :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="block space-y-1 text-xs font-medium text-muted">User <span class="text-expense">*</span><input v-model.trim="form.code" maxlength="64" required :aria-invalid="formCheck.invalid('code')" class="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm text-navy" placeholder="e.g. 12K_YMLLJB_V2" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Name <span class="font-normal">(optional)</span><input v-model.trim="form.name" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <!-- With one branch it is chosen for you; an owner/admin can move an agent to another branch -->
        <div v-if="showBranch && (!editing || hasFullAccess)" class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Branch<SearchSelect v-model="form.branch_id" :options="formBranchOptions" :searchable="false" aria-label="Agent branch" /></div>
        <div class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Default currency<SearchSelect v-model="form.default_currency" :options="currencyOptions" :searchable="false" aria-label="Default currency" :aria-invalid="formCheck.invalid('default_currency')" /></div>
        <!-- Groups (optional, needs Update Group Agent Assignments): the agent's branch's groups; a
             turned-off agent can't be in any -->
        <div v-if="showGroups" class="space-y-1 text-xs font-medium text-muted sm:col-span-2">
          <span>Groups <span class="font-normal">(optional)</span></span>
          <p v-if="groupsLoading" class="h-10 rounded-lg border border-border bg-canvas px-3 py-2.5 text-sm font-normal">Loading groups…</p>
          <p v-else-if="groupsError" class="rounded-lg bg-red-50 px-3 py-2 text-xs font-normal text-expense">{{ groupsError }} Groups are left as they are.</p>
          <SearchSelect v-else v-model="groupIds" multiple :options="groupOptions" placeholder="No group" empty-text="No groups for this branch" aria-label="Groups" />
        </div>
        <!-- Each group picked, with the agent's type and rate in it -->
        <ul v-if="showGroups && !groupsLoading && !groupsError && chosenGroups.length" class="space-y-1.5 sm:col-span-2">
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
        <p v-if="showGroups && editing && removedGroups.length" class="rounded-lg bg-amber-50 px-3 py-2 text-[11px] text-amber-800 sm:col-span-2">Taken out of {{ removedGroups.join(', ') }}: ended today if weeks were entered (they stay), else removed.</p>
      </div>
      <p v-if="editing" class="mt-3 rounded-lg bg-canvas px-3 py-2 text-[11px] text-muted">Changing the currency only affects groups assigned from now on; each assignment keeps its own, and saved weeks keep the currency they were entered in.</p>
      <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="formOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { Layers, Network, Pencil, Plus, Power, Trash2, X } from 'lucide-vue-next'
import type { AssignmentGroup, Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useCurrency } from '~/composables/useCurrency'
import { useInfiniteList } from '~/composables/useInfiniteList'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { rule } from '~/utils/validation'
import { CALC_TYPES, totalRate } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
// Each action has its own permission (Add, Update, Delete Agents); turning off or on is an update.
const canCreate = computed(() => can('platforms.create'))
const canEdit = computed(() => can('platforms.edit'))
const canDelete = computed(() => can('platforms.delete'))
const { currencyOptions, load: loadCurrency } = useCurrency()

const search = ref('')
const showInactive = ref(false)
// Agents belong to a branch. People who work in one branch just get theirs (nothing to choose);
// with several, a Branch filter and a Branch choice on the form.
const { branches, everyBranch, options: branchOptions } = useBranchOptions('All branches')
const { hasFullAccess } = usePermissions()
const { activeBranchId } = useActiveBranch()
const showBranch = computed(() => branches.value.length > 1)
const branchFilter = ref('')
const formBranchOptions = computed(() => [
  ...(editing.value ? [] : [{ value: '', label: 'My own branch' }]),
  ...(editing.value && !editing.value.branch_id ? [{ value: '', label: 'Shared — choose a branch to move it' }] : []),
  ...branches.value.map(branch => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })),
])
// Agents load a page at a time as the list is scrolled (search and turned-off agents are worked out
// by the API); a save or delete fetches the pages already shown again.
const { items, loading, loadingMore, error, hasMore, reset, loadMore, refresh: load } = useInfiniteList<Platform>(
  async (page, size) => await api<{ items: Platform[]; total: number }>('/platforms', {
    query: { include_inactive: showInactive.value ? 'true' : undefined, search: search.value.trim() || undefined, branch_id: branchFilter.value || undefined, page, page_size: size },
  }),
  platform => platform.id, 30, 'Could not load agents.',
)
const shown = computed(() => items.value)
// Another search or filter starts from the top again.
let searchTimer: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(reset, 300)
})
watch([showInactive, branchFilter], reset)
onMounted(() => { reset(); loadCurrency() })

const formOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const editing = ref<Platform | null>(null)
const busyId = ref('')
const form = reactive({ code: '', name: '', default_currency: 'USD', branch_id: '' })

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
// The groups the agent was in when the form opened.
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

let groupsRequest = 0
async function loadGroups(agent: Platform | null) {
  const request = ++groupsRequest
  groups.value = []
  groupIds.value = []
  startGroupIds.value = []
  groupsError.value = ''
  groupsTried.value = false
  for (const key of Object.keys(groupRates)) delete groupRates[key]
  for (const key of Object.keys(groupKinds)) delete groupKinds[key]
  if (!canGroups.value || (agent && !agent.is_active)) return
  groupsLoading.value = true
  try {
    // Every group the person works with; the form shows the agent's branch's.
    const result = await api<{ items: AssignmentGroup[] }>('/platform-groups', { query: { branch_id: 'all' } })
    if (request !== groupsRequest) return
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
    if (request === groupsRequest) groupsError.value = e?.data?.error || 'Could not load the groups.'
  } finally {
    if (request === groupsRequest) groupsLoading.value = false
  }
}

function openCreate() {
  editing.value = null
  Object.assign(form, { code: '', name: '', default_currency: 'USD', branch_id: activeBranchId.value })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
  loadGroups(null)
}

function openEdit(platform: Platform) {
  editing.value = platform
  Object.assign(form, { code: platform.code, name: platform.name, default_currency: platform.default_currency, branch_id: platform.branch_id || '' })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
  loadGroups(platform)
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
    const saved = editing.value
      ? await api<{ counted_weeks?: number }>(`/platforms/${editing.value.id}`, { method: 'PATCH', body })
      : await api<{ counted_weeks?: number }>('/platforms', { method: 'POST', body })
    formOpen.value = false
    const n = chosenGroups.value.length
    const weeks = saved?.counted_weeks || 0
    const counted = weeks ? ` ${weeks} ${weeks === 1 ? 'week' : 'weeks'} saved before now count in its groups.` : ''
    toast.success(`${form.code} is saved${sendGroups && n ? `, in ${n} ${n === 1 ? 'group' : 'groups'}` : ''}.${counted}`, { title: editing.value ? 'Agent updated' : 'Agent added' })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this agent.'
  } finally {
    saving.value = false
  }
}

const deleting = ref<Platform | null>(null)
const deleteBusy = ref(false)
const deleteError = ref('')
function openDelete(platform: Platform) {
  deleteError.value = ''
  deleting.value = platform
}
async function deleteAgent() {
  const p = deleting.value
  if (!p) return
  deleteBusy.value = true
  deleteError.value = ''
  try {
    await api(`/platforms/${p.id}`, { method: 'DELETE' })
    deleting.value = null
    toast.success(`${p.code} is removed.`, { title: 'Agent deleted' })
    await load()
  } catch (e: any) {
    deleteError.value = e?.data?.error || 'Could not delete this agent.'
  } finally {
    deleteBusy.value = false
  }
}

const turningOff = ref<Platform | null>(null)
const turnOffError = ref('')
async function setActive(platform: Platform, active: boolean, reason = '') {
  busyId.value = platform.id
  turnOffError.value = ''
  try {
    await api(`/platforms/${platform.id}`, {
      method: 'PATCH',
      body: { code: platform.code, name: platform.name, default_currency: platform.default_currency, is_active: active, reason: reason || undefined },
    })
    turningOff.value = null
    toast.success(`${platform.code} is turned ${active ? 'on' : 'off'}.`)
    await load()
  } catch (e: any) {
    // Turning off: the error stays in the window (e.g. groups still have the agent)
    if (active) toast.error(e?.data?.error || 'Could not change this agent.')
    else turnOffError.value = e?.data?.error || 'Could not turn this agent off.'
  } finally {
    busyId.value = ''
  }
}
</script>
