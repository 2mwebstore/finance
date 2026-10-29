<template>
  <section class="mx-auto max-w-5xl space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS</p>
        <h1 class="text-xl font-bold text-navy">Group agent assignments</h1>
        <p class="mt-1 text-xs text-muted">Titled groups of the agents you report every week, each with its partners (each at their own %) and currency. A group that is turned off leaves Data Entry; weeks already saved stay.</p>
      </div>
      <button v-if="canManage" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openAssign"><Plus :size="16" /> Assign agents</button>
    </header>

    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
      <SearchSelect v-model="platformFilter" :options="[{ value: '', label: 'All agents' }, ...platformOptions]" aria-label="Agent" />
      <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Status" />
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading groups…</div>
    <AppAlert v-else-if="error" title="Couldn’t load groups" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <template v-else>
      <article v-for="g in groups" :key="g.id" class="rounded-xl border border-border bg-white p-4" :class="g.is_active ? '' : 'bg-canvas/60'">
        <div class="flex flex-col gap-3 sm:flex-row sm:items-start">
          <div class="min-w-0 flex-1 space-y-2">
            <div>
              <h2 class="flex flex-wrap items-center gap-2 text-sm font-bold" :class="g.is_active ? 'text-navy' : 'text-slate-400'">
                <Network :size="15" class="shrink-0 text-muted" />{{ g.title }}
                <span v-if="!g.is_active" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span>
              </h2>
              <p class="mt-0.5 text-[11px] text-muted">From {{ formatDocDate(g.starts_on) }} · {{ g.weeks }} {{ g.weeks === 1 ? 'week' : 'weeks' }} entered<template v-if="g.created_by"> · by {{ g.created_by }}</template></p>
            </div>
            <!-- Its agents -->
            <ul class="flex flex-wrap gap-1">
              <li v-for="a in g.agents" :key="a.assignment_id" class="rounded-md border border-border bg-white px-1.5 py-0.5 text-[11px] text-navy"><span class="font-mono font-semibold">{{ a.platform_code }}</span> · {{ a.currency }}</li>
              <li v-if="!g.agents.length" class="text-[11px] italic text-slate-400">No agents</li>
            </ul>
            <!-- Its partners (the same on every agent) -->
            <p v-if="!g.partners.length" class="flex items-center gap-1 text-[11px] text-slate-400"><Handshake :size="11" />No partner<template v-if="g.agents.length"> · {{ formatRate(g.agents[0]!.commission_rate) }}</template></p>
            <ul v-else class="flex flex-wrap gap-1">
              <li v-for="p in g.partners" :key="p.partner_id" class="inline-flex items-center gap-1 rounded-md bg-blue-50 px-1.5 py-0.5 text-[11px] text-navy" :class="p.partner_active ? '' : 'opacity-60'"><Handshake :size="11" />{{ p.partner_name }}<template v-if="!p.partner_active"> (off)</template> · {{ formatRate(p.commission_rate) }}</li>
            </ul>
          </div>
          <div v-if="canManage" class="flex shrink-0 items-center gap-3">
            <label class="flex items-center gap-2 text-xs font-semibold" :class="g.is_active ? 'text-navy' : 'text-muted'">
              Active
              <input :checked="g.is_active" type="checkbox" role="switch" class="switch" :disabled="busyId === g.id" :aria-label="`${g.title} active`" @change="toggle(g)" />
            </label>
            <button type="button" class="action-chip action-edit" @click="openEdit(g)"><Pencil :size="13" /> Update</button>
            <button type="button" class="action-chip action-danger" @click="openDelete(g)"><Trash2 :size="13" /> Delete</button>
          </div>
        </div>
      </article>
      <div v-if="!groups.length" class="rounded-xl border border-border bg-white p-8 text-center text-sm text-muted">{{ platformFilter || statusFilter !== 'all' ? 'Nothing to show.' : 'No groups yet. Use Assign agents to make one.' }}</div>
    </template>

    <!-- A new group: a title and its agents (in the person's own branch; the API works it out) -->
    <AppModal :model-value="assignOpen" form title="Assign agents" eyebrow="NEW GROUP" subtitle="Give the group a title; every agent you pick gets the same partners." :busy="saving" width="560px" @update:model-value="(open) => { if (!open && !saving) assignOpen = false }" @submit="assign">
      <template #icon><Network :size="20" /></template>
      <div class="space-y-3">
        <label class="block space-y-1 text-xs font-medium text-muted">Title <span class="text-expense">*</span><input v-model="assignForm.title" maxlength="128" :aria-invalid="assignCheck.invalid('title')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. V-series Phnom Penh" /></label>
        <div class="space-y-1 text-xs font-medium text-muted">Agents<SearchSelect v-model="assignForm.platform_ids" multiple :options="activePlatformOptions" placeholder="Choose one or more agents" aria-label="Agents" :aria-invalid="assignCheck.invalid('platform_ids')" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">Partners <span class="font-normal">(optional)</span><SearchSelect v-model="assignForm.partner_ids" multiple :options="activePartnerOptions" placeholder="No partner" aria-label="Partners" :aria-invalid="assignCheck.invalid('partner_ids')" /></div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div class="space-y-1 text-xs font-medium text-muted">Currency<SearchSelect v-model="assignForm.currency" :options="[{ value: '', label: 'Each agent’s default' }, ...currencyOptions]" :searchable="false" aria-label="Currency" /></div>
          <div class="space-y-1 text-xs font-medium text-muted">From<DatePicker v-model="assignForm.starts_on" aria-label="First day" /></div>
        </div>
        <ul v-if="chosenPlatforms.length" class="flex flex-wrap gap-1.5 text-[11px]">
          <li v-for="p in chosenPlatforms" :key="p.id" class="rounded-md border border-border bg-canvas px-2 py-1 text-navy">
            <span class="font-mono font-semibold">{{ p.code }}</span> · {{ assignForm.currency || p.default_currency }} ·
            <template v-if="chosenPartners.length">{{ chosenPartners.map(c => `${c.name} ${formatRate(c.default_rate)}`).join(' + ') }}</template>
            <template v-else>no partner</template>
          </li>
        </ul>
        <p class="text-[11px] text-muted">Each partner gets their default % (change it later with Update); the Total is worked out for each of them. An agent can be in one group at a time. Results are entered once a week; the first day can be in the past to enter earlier weeks.</p>
      </div>
      <AppAlert v-if="formError || assignCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t assign' : 'Check the highlighted fields'" :message="formError" :items="assignCheck.messages" dismissible @dismiss="formError = ''; assignCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="assignOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Assign' }}</button>
      </template>
    </AppModal>

    <!-- Update a group: title, agents, partners (each with their %) and currency -->
    <AppModal :model-value="!!editing" form :title="editing ? `Update “${editing.title}”` : ''" eyebrow="GROUP" subtitle="New weeks use these agents, partners and %; weeks already saved keep theirs." :busy="saving" width="560px" @update:model-value="(open) => { if (!open && !saving) editing = null }" @submit="saveEdit">
      <div class="space-y-3">
        <label class="block space-y-1 text-xs font-medium text-muted">Title <span class="text-expense">*</span><input v-model="editForm.title" maxlength="128" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <div class="space-y-1 text-xs font-medium text-muted">Agents<SearchSelect v-model="editForm.platform_ids" multiple :options="editPlatformOptions" placeholder="Choose one or more agents" aria-label="Agents" /></div>
        <p class="text-[11px] text-muted">An agent taken out is ended if weeks were entered for it (they stay), else removed. An agent put in starts today.</p>
        <div class="space-y-1 text-xs font-medium text-muted">
          Partners
          <ul v-if="editForm.partners.length" class="divide-y divide-border rounded-lg border border-border">
            <li v-for="(p, i) in editForm.partners" :key="p.partner_id" class="flex items-center gap-2 px-3 py-2">
              <Handshake :size="14" class="shrink-0 text-muted" />
              <span class="min-w-0 flex-1 truncate text-sm font-normal text-navy">{{ p.partner_name }}<span v-if="!p.partner_active" class="text-muted"> (off)</span></span>
              <label class="flex shrink-0 items-center gap-1 text-xs text-muted"><input v-model.trim="p.commission_rate" type="text" inputmode="decimal" :aria-label="`${p.partner_name} %`" class="w-20 rounded-lg border border-border px-2 py-1.5 text-right text-sm text-navy" :aria-invalid="!!rateCheck(p.commission_rate) || !p.commission_rate" />%</label>
              <button type="button" class="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-muted hover:bg-red-50 hover:text-expense" :aria-label="`Take ${p.partner_name} off`" @click="editForm.partners.splice(i, 1)"><X :size="15" /></button>
            </li>
          </ul>
          <SearchSelect :model-value="''" :options="addPartnerOptions" :placeholder="addPartnerOptions.length ? 'Add a partner' : 'No other partners'" aria-label="Add a partner" @update:model-value="addEditPartner" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <label v-if="!editForm.partners.length" class="block space-y-1 text-xs font-medium text-muted">Partner % <span class="font-normal">(no partner)</span><input v-model.trim="editForm.commission_rate" type="text" inputmode="decimal" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
          <p v-else class="self-end pb-2 text-xs text-muted">In all: <strong class="text-navy">{{ formatRate(editShareTotal) }}</strong></p>
          <div class="space-y-1 text-xs font-medium text-muted">Currency<SearchSelect v-model="editForm.currency" :options="[{ value: '', label: 'Each agent’s default' }, ...currencyOptions]" :searchable="false" aria-label="Currency" /></div>
        </div>
      </div>
      <AppAlert v-if="formError" class="mt-3" compact :message="formError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="editing = null">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Update' }}</button>
      </template>
    </AppModal>

    <!-- Delete: only while no week has been entered -->
    <AppModal :model-value="!!deleting" form :title="deleting ? `Delete “${deleting.title}”?` : ''" eyebrow="DELETE GROUP" :subtitle="deleting && deleting.weeks ? 'Weeks are already entered for this group, so it can’t be deleted. Turn it off instead: its agents leave Data Entry and the weeks stay.' : 'The group and its agents’ assignments are removed. Its agents can then go in another group.'" :busy="saving" width="460px" @update:model-value="(open) => { if (!open && !saving) deleting = null }" @submit="saveDelete">
      <AppAlert v-if="formError" compact :message="formError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="deleting = null">Cancel</button>
        <button v-if="deleting && !deleting.weeks" type="submit" class="bg-expense text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Deleting…' : 'Delete group' }}</button>
        <button v-else-if="deleting && deleting.is_active" type="button" class="bg-primary text-white disabled:opacity-60" :disabled="saving" @click="turnOffDeleting">Turn it off</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { Handshake, Network, Pencil, Plus, Trash2, X } from 'lucide-vue-next'
import type { AssignmentGroup, AssignmentPartner, Partner, Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useCurrency } from '~/composables/useCurrency'
import { rule } from '~/utils/validation'
import { formatDocDate, todayISO } from '~/utils/inventory'
import { formatRate } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canManage = computed(() => can('platform-assignments.manage'))
const { currencyOptions, load: loadCurrency } = useCurrency()

const platforms = ref<Platform[]>([])
const platformOptions = computed(() => platforms.value.map(p => ({ value: p.id, label: p.is_active ? p.code : `${p.code} (off)` })))
const activePlatformOptions = computed(() => platforms.value.filter(p => p.is_active).map(p => ({ value: p.id, label: p.code })))
const partners = ref<Partner[]>([])
const activePartnerOptions = computed(() => partners.value.filter(p => p.is_active).map(p => ({ value: p.id, label: `${p.name} · ${formatRate(p.default_rate)}` })))
const statusOptions = [{ value: 'all', label: 'All groups' }, { value: 'active', label: 'Active' }, { value: 'off', label: 'Off' }]

// The person's own groups (the API works the branch out): switched-on first, by title.
const groups = ref<AssignmentGroup[]>([])
const loading = ref(true)
const error = ref('')
const platformFilter = ref('')
const statusFilter = ref('all')

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<{ items: AssignmentGroup[] }>('/platform-groups', { query: { platform_id: platformFilter.value || undefined, status: statusFilter.value } })
    if (request === loadRequest) groups.value = result.items || []
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load groups.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([platformFilter, statusFilter], load)
onMounted(async () => {
  load()
  loadCurrency()
  try { platforms.value = (await api<{ items: Platform[] }>('/platforms', { query: { include_inactive: 'true' } })).items || [] } catch { platforms.value = [] }
  try { partners.value = (await api<{ items: Partner[] }>('/partners', { query: { include_inactive: 'true' } })).items || [] } catch { partners.value = [] }
})

const saving = ref(false)
const formError = ref('')
const busyId = ref('')
const rateCheck = (value: unknown) => !value || (/^\d+(\.\d{1,4})?$/.test(String(value)) && Number(value) <= 100) ? '' : 'enter a partner % from 0 to 100'
const tooMuch = (sum: number) => sum > 100 ? `the partners’ % add up to ${Number(sum.toFixed(4))}; they can share at most 100%` : ''

// --- a new group ---
const assignOpen = ref(false)
const assignForm = reactive({ title: '', platform_ids: [] as string[], partner_ids: [] as string[], currency: '', starts_on: todayISO() })
const chosenPlatforms = computed(() => platforms.value.filter(p => assignForm.platform_ids.includes(p.id)))
const chosenPartners = computed(() => assignForm.partner_ids.map(id => partners.value.find(p => p.id === id)).filter((p): p is Partner => !!p))
const assignCheck = reactive(useFormErrors(
  () => ({ ...assignForm }),
  () => ({
    title: [rule.required('title'), rule.maxLength('title', 128)],
    platform_ids: [rule.chooseOne('agents')],
    partner_ids: [() => tooMuch(chosenPartners.value.reduce((sum, p) => sum + Number(p.default_rate), 0))],
  }),
))

function openAssign() {
  Object.assign(assignForm, { title: '', platform_ids: platformFilter.value ? [platformFilter.value] : [], partner_ids: [], currency: '', starts_on: todayISO() })
  formError.value = ''
  assignCheck.reset()
  assignOpen.value = true
}

async function assign() {
  formError.value = ''
  if (!assignCheck.check()) return
  saving.value = true
  try {
    const g = await api<AssignmentGroup>('/platform-groups', { method: 'POST', body: { ...assignForm, title: assignForm.title.trim() } })
    assignOpen.value = false
    toast.success(`${g.agents.length} ${g.agents.length === 1 ? 'agent' : 'agents'} assigned.`, { title: `“${g.title}” made` })
    await load()
  } catch (e: any) {
    if (!assignCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not assign the agents.'
  } finally {
    saving.value = false
  }
}

// --- update ---
const editing = ref<AssignmentGroup | null>(null)
const editForm = reactive({ title: '', platform_ids: [] as string[], partners: [] as AssignmentPartner[], commission_rate: '', currency: '' })
const editShareTotal = computed(() => editForm.partners.reduce((sum, p) => sum + Number(p.commission_rate || 0), 0))
// Active agents, plus the group's own (one turned off since stays listed).
const editPlatformOptions = computed(() => platforms.value
  .filter(p => p.is_active || editing.value?.agents.some(a => a.platform_id === p.id))
  .map(p => ({ value: p.id, label: p.code })))
// Active partners not in the group yet (one turned off since can stay but not be added again).
const addPartnerOptions = computed(() => activePartnerOptions.value.filter(o => !editForm.partners.some(p => p.partner_id === o.value)))
function openEdit(g: AssignmentGroup) {
  Object.assign(editForm, {
    title: g.title,
    platform_ids: g.agents.map(a => a.platform_id),
    partners: g.partners.map(p => ({ ...p, commission_rate: String(Number(p.commission_rate)) })),
    commission_rate: g.agents.length ? String(Number(g.agents[0]!.commission_rate)) : '',
    currency: g.currency,
  })
  formError.value = ''
  editing.value = g
}
// An added partner comes in at their default % (it can still be changed).
function addEditPartner(value: string | string[]) {
  const partner = partners.value.find(p => p.id === String(value || ''))
  if (partner) editForm.partners.push({ partner_id: partner.id, partner_name: partner.name, commission_rate: String(Number(partner.default_rate)), partner_active: true })
}
async function saveEdit() {
  if (!editing.value) return
  const bad = editForm.partners.find(p => !p.commission_rate || rateCheck(p.commission_rate))
  formError.value = !editForm.title.trim() ? 'Give the group a title.'
    : !editForm.platform_ids.length ? 'Choose at least one agent.'
      : bad ? `${bad.partner_name}: enter a partner % from 0 to 100`
        : tooMuch(editShareTotal.value) || (editForm.partners.length ? '' : rateCheck(editForm.commission_rate))
  if (formError.value) return
  const body = {
    title: editForm.title.trim(), platform_ids: editForm.platform_ids, currency: editForm.currency,
    partners: editForm.partners.map(p => ({ partner_id: p.partner_id, commission_rate: p.commission_rate })),
    commission_rate: editForm.partners.length ? '' : editForm.commission_rate,
  }
  await run(() => api(`/platform-groups/${editing.value!.id}`, { method: 'PATCH', body }), 'Group updated', `“${body.title}” is saved; new weeks use it.`, () => { editing.value = null })
}

// --- active on/off ---
async function toggle(g: AssignmentGroup) {
  busyId.value = g.id
  try {
    const updated = await api<AssignmentGroup>(`/platform-groups/${g.id}/active`, { method: 'POST', body: { is_active: !g.is_active } })
    toast.success(updated.is_active ? 'Its agents are on Data Entry again.' : 'Its agents leave Data Entry; saved weeks stay.', { title: `“${g.title}” turned ${updated.is_active ? 'on' : 'off'}` })
    await load()
  } catch (e: any) {
    toast.error(e?.data?.error || 'Could not change this group.')
    await load()
  } finally {
    busyId.value = ''
  }
}

// --- delete ---
const deleting = ref<AssignmentGroup | null>(null)
function openDelete(g: AssignmentGroup) {
  formError.value = ''
  deleting.value = g
}
async function saveDelete() {
  if (!deleting.value) return
  const g = deleting.value
  await run(() => api(`/platform-groups/${g.id}`, { method: 'DELETE' }), 'Group deleted', `“${g.title}” is removed.`, () => { deleting.value = null })
}
async function turnOffDeleting() {
  const g = deleting.value
  if (!g) return
  deleting.value = null
  await toggle(g)
}

async function run(call: () => Promise<unknown>, title: string, message: string, close: () => void) {
  saving.value = true
  formError.value = ''
  try {
    await call()
    close()
    toast.success(message, { title })
    await load()
  } catch (e: any) {
    formError.value = e?.data?.error || 'Could not save.'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.switch { position: relative; flex: 0 0 auto; width: 42px; height: 24px; margin: 0; border-radius: 999px; background: #cfd9e6; cursor: pointer; appearance: none; transition: background .15s ease; }
.switch::before { content: ''; position: absolute; top: 3px; left: 3px; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgb(16 40 80 / 25%); transition: transform .15s ease; }
.switch:checked { background: #1769e8; }
.switch:checked::before { transform: translateX(18px); }
.switch:disabled { cursor: wait; opacity: .6; }
</style>
