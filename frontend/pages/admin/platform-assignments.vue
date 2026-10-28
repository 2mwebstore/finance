<template>
  <section class="mx-auto max-w-5xl space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS</p>
        <h1 class="text-xl font-bold text-navy">Branch platform assignments</h1>
        <p class="mt-1 text-xs text-muted">Which platforms each branch reports every week, with its partners (each at their own %) and currency. Ending or moving an assignment keeps it as history.</p>
      </div>
      <button v-if="canManage" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openAssign"><Plus :size="16" /> Assign platforms</button>
    </header>

    <div class="grid grid-cols-1 gap-2 sm:grid-cols-3">
      <SearchSelect v-model="branchFilter" :options="[{ value: '', label: 'All branches' }, ...hqOptions]" aria-label="Branch" />
      <SearchSelect v-model="platformFilter" :options="[{ value: '', label: 'All platforms' }, ...platformOptions]" aria-label="Platform" />
      <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Status" />
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading assignments…</div>
    <AppAlert v-else-if="error" title="Couldn’t load assignments" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <template v-else>
      <article v-for="group in groups" :key="group.branch_id" class="overflow-hidden rounded-xl border border-border bg-white">
        <header class="flex items-center justify-between gap-3 border-b border-border bg-canvas px-4 py-2.5">
          <h2 class="flex min-w-0 items-center gap-2 text-sm font-semibold text-navy"><Building2 :size="15" class="shrink-0 text-muted" /><span class="truncate">{{ group.branch_code }} — {{ group.branch_name }}</span></h2>
          <span class="shrink-0 text-[11px] text-muted">{{ group.rows.filter(r => r.is_active).length }} active</span>
        </header>
        <ul class="divide-y divide-border">
          <li v-for="a in group.rows" :key="a.id" class="flex flex-col gap-2 px-4 py-3 sm:flex-row sm:items-center">
            <div class="min-w-0 flex-1">
              <div class="flex flex-wrap items-center gap-2">
                <strong class="font-mono text-sm" :class="a.is_active ? 'text-navy' : 'text-slate-400'">{{ a.platform_code }}</strong>
                <span v-if="!a.is_active" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Ended</span>
              </div>
              <p class="text-[11px] text-muted">
                From {{ formatDocDate(a.starts_on) }}<template v-if="a.ends_on"> to {{ formatDocDate(a.ends_on) }}</template> · {{ a.weeks }} {{ a.weeks === 1 ? 'week' : 'weeks' }} entered
                <template v-if="!a.is_active && (a.ended_by || a.end_reason)"> · ended by {{ a.ended_by }}<template v-if="a.end_reason">: {{ a.end_reason }}</template></template>
              </p>
            </div>
            <div class="flex items-center gap-3">
              <div class="min-w-0 text-right">
                <p class="text-sm font-semibold text-navy">{{ formatRate(rateOf(a)) }} <span class="text-xs font-normal text-muted">· {{ a.currency }}</span></p>
                <p v-if="!a.partners.length" class="flex items-center justify-end gap-1 text-[11px] text-slate-400"><Handshake :size="11" />No partner</p>
                <ul v-else class="mt-0.5 flex flex-wrap justify-end gap-1">
                  <li v-for="p in a.partners" :key="p.partner_id" class="inline-flex items-center gap-1 rounded-md bg-blue-50 px-1.5 py-0.5 text-[11px] text-navy" :class="p.partner_active ? '' : 'opacity-60'"><Handshake :size="11" />{{ p.partner_name }}<template v-if="!p.partner_active"> (off)</template> · {{ formatRate(p.commission_rate) }}</li>
                </ul>
              </div>
              <div v-if="canManage && a.is_active" class="flex shrink-0 gap-1.5">
                <button type="button" class="action-chip action-edit" @click="openEdit(a)"><Pencil :size="13" /> Partners</button>
                <button type="button" class="action-chip action-edit" @click="openMove(a)"><ArrowRightLeft :size="13" /> Move</button>
                <button type="button" class="action-chip action-danger" @click="openEnd(a)">End</button>
              </div>
            </div>
          </li>
        </ul>
      </article>
      <div v-if="!groups.length" class="rounded-xl border border-border bg-white p-8 text-center text-sm text-muted">{{ statusFilter === 'active' ? 'No platforms are assigned yet.' : 'Nothing to show.' }}</div>
    </template>

    <!-- Assign one or more platforms to one or more branches -->
    <AppModal :model-value="assignOpen" form title="Assign platforms" eyebrow="BRANCH ASSIGNMENT" subtitle="Every platform you pick goes to every branch you pick, with the same partners." :busy="saving" width="560px" @update:model-value="(open) => { if (!open && !saving) assignOpen = false }" @submit="assign">
      <template #icon><Network :size="20" /></template>
      <div class="space-y-3">
        <div class="space-y-1 text-xs font-medium text-muted">Platforms<SearchSelect v-model="assignForm.platform_ids" multiple :options="activePlatformOptions" placeholder="Choose one or more platforms" aria-label="Platforms" :aria-invalid="assignCheck.invalid('platform_ids')" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">Branches<SearchSelect v-model="assignForm.branch_ids" multiple :options="hqOptions" placeholder="Choose one or more branches" aria-label="Branches" :aria-invalid="assignCheck.invalid('branch_ids')" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">Partners <span class="font-normal">(optional)</span><SearchSelect v-model="assignForm.partner_ids" multiple :options="activePartnerOptions" placeholder="No partner" aria-label="Partners" :aria-invalid="assignCheck.invalid('partner_ids')" /></div>
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div class="space-y-1 text-xs font-medium text-muted">Currency<SearchSelect v-model="assignForm.currency" :options="[{ value: '', label: 'Each platform’s default' }, ...currencyOptions]" :searchable="false" aria-label="Currency" /></div>
          <div class="space-y-1 text-xs font-medium text-muted">From<DatePicker v-model="assignForm.starts_on" aria-label="First day" /></div>
        </div>
        <ul v-if="chosenPlatforms.length" class="flex flex-wrap gap-1.5 text-[11px]">
          <li v-for="p in chosenPlatforms" :key="p.id" class="rounded-md border border-border bg-canvas px-2 py-1 text-navy">
            <span class="font-mono font-semibold">{{ p.code }}</span> · {{ assignForm.currency || p.default_currency }} ·
            <template v-if="chosenPartners.length">{{ chosenPartners.map(c => `${c.name} ${formatRate(c.default_rate)}`).join(' + ') }}</template>
            <template v-else>{{ formatRate(p.default_rate) }}, no partner</template>
          </li>
        </ul>
        <p class="text-[11px] text-muted">Each partner gets their default % (change it later with Partners); the Total is worked out for each of them. With no partner, each platform’s default % is used. Leave the currency empty for each platform’s. Results are entered once a week; the first day can be in the past to enter earlier weeks.<template v-if="assignCount > 1"> This makes {{ assignCount }} assignments.</template></p>
      </div>
      <AppAlert v-if="formError || assignCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t assign' : 'Check the highlighted fields'" :message="formError" :items="assignCheck.messages" dismissible @dismiss="formError = ''; assignCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="assignOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Assign' }}</button>
      </template>
    </AppModal>

    <!-- Partners (each with their %) and currency -->
    <AppModal :model-value="!!editing" form :title="editing ? `${editing.platform_code} at ${editing.branch_code}` : ''" eyebrow="PARTNERS" subtitle="New weeks use these partners and %; weeks already saved keep theirs." :busy="saving" width="500px" @update:model-value="(open) => { if (!open && !saving) editing = null }" @submit="saveEdit">
      <div class="space-y-3">
        <ul v-if="editForm.partners.length" class="divide-y divide-border rounded-lg border border-border">
          <li v-for="(p, i) in editForm.partners" :key="p.partner_id" class="flex items-center gap-2 px-3 py-2">
            <Handshake :size="14" class="shrink-0 text-muted" />
            <span class="min-w-0 flex-1 truncate text-sm text-navy">{{ p.partner_name }}<span v-if="!p.partner_active" class="text-muted"> (off)</span></span>
            <label class="flex shrink-0 items-center gap-1 text-xs text-muted"><input v-model.trim="p.commission_rate" type="text" inputmode="decimal" :aria-label="`${p.partner_name} %`" class="w-20 rounded-lg border border-border px-2 py-1.5 text-right text-sm text-navy" :aria-invalid="!!rateCheck(p.commission_rate) || !p.commission_rate" />%</label>
            <button type="button" class="grid h-8 w-8 shrink-0 place-items-center rounded-lg text-muted hover:bg-red-50 hover:text-expense" :aria-label="`Take ${p.partner_name} off`" @click="editForm.partners.splice(i, 1)"><X :size="15" /></button>
          </li>
        </ul>
        <div class="space-y-1 text-xs font-medium text-muted">Add a partner<SearchSelect :model-value="''" :options="addPartnerOptions" :placeholder="addPartnerOptions.length ? 'Choose a partner' : 'No other partners'" aria-label="Add a partner" @update:model-value="addEditPartner" /></div>
        <div class="grid grid-cols-2 gap-3">
          <label v-if="!editForm.partners.length" class="block space-y-1 text-xs font-medium text-muted">Partner % <span class="font-normal">(no partner)</span><input v-model.trim="editForm.commission_rate" type="text" inputmode="decimal" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
          <p v-else class="self-end pb-2 text-xs text-muted">In all: <strong class="text-navy">{{ formatRate(editShareTotal) }}</strong></p>
          <div class="space-y-1 text-xs font-medium text-muted">Currency<SearchSelect v-model="editForm.currency" :options="currencyOptions" :searchable="false" aria-label="Currency" /></div>
        </div>
      </div>
      <AppAlert v-if="formError" class="mt-3" compact :message="formError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="editing = null">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
      </template>
    </AppModal>

    <!-- Move to another branch -->
    <AppModal :model-value="!!moving" form :title="moving ? `Move ${moving.platform_code} from ${moving.branch_code}` : ''" eyebrow="MOVE" subtitle="It ends here the day before and starts at the new branch with the same partners and %. The week it moves in is on both branches’ sheets." :busy="saving" width="460px" @update:model-value="(open) => { if (!open && !saving) moving = null }" @submit="saveMove">
      <div class="space-y-3">
        <div class="space-y-1 text-xs font-medium text-muted">New branch<SearchSelect v-model="moveForm.branch_id" :options="hqOptions.filter(o => o.value !== moving?.branch_id)" placeholder="Choose a branch" aria-label="New branch" /></div>
        <div class="space-y-1 text-xs font-medium text-muted">First day at the new branch<DatePicker v-model="moveForm.on" aria-label="First day at the new branch" /></div>
        <label class="block space-y-1 text-xs font-medium text-muted">Reason <span class="font-normal">(optional)</span><input v-model.trim="moveForm.reason" maxlength="255" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
      </div>
      <AppAlert v-if="formError" class="mt-3" compact :message="formError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="moving = null">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving || !moveForm.branch_id">{{ saving ? 'Saving…' : 'Move' }}</button>
      </template>
    </AppModal>

    <!-- End -->
    <AppModal :model-value="!!ending" form :title="ending ? `End ${ending.platform_code} at ${ending.branch_code}?` : ''" eyebrow="END ASSIGNMENT" subtitle="The branch stops reporting this platform after its last day. Weeks already entered stay." :busy="saving" width="440px" @update:model-value="(open) => { if (!open && !saving) ending = null }" @submit="saveEnd">
      <div class="space-y-3">
        <div class="space-y-1 text-xs font-medium text-muted">Last day<DatePicker v-model="endForm.ends_on" aria-label="Last day" /></div>
        <label class="block space-y-1 text-xs font-medium text-muted">Reason <span class="font-normal">(optional)</span><input v-model.trim="endForm.reason" maxlength="255" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
      </div>
      <AppAlert v-if="formError" class="mt-3" compact :message="formError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="ending = null">Cancel</button>
        <button type="submit" class="bg-expense text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'End assignment' }}</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { ArrowRightLeft, Building2, Handshake, Network, Pencil, Plus, X } from 'lucide-vue-next'
import type { AssignmentPartner, Partner, Platform, PlatformAssignment } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useBranchOptions } from '~/composables/useBranchOptions'
import { useCurrency } from '~/composables/useCurrency'
import { rule } from '~/utils/validation'
import { formatDocDate, todayISO } from '~/utils/inventory'
import { formatRate } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canManage = computed(() => can('platform-assignments.manage'))
const { branches } = useBranchOptions()
const { currencyOptions, load: loadCurrency } = useCurrency()
const hqOptions = computed(() => branches.value.map(b => ({ value: b.id, label: `${b.code} — ${b.name}` })))

const platforms = ref<Platform[]>([])
const platformOptions = computed(() => platforms.value.map(p => ({ value: p.id, label: p.is_active ? p.code : `${p.code} (off)` })))
const activePlatformOptions = computed(() => platforms.value.filter(p => p.is_active).map(p => ({ value: p.id, label: p.code })))
const partners = ref<Partner[]>([])
const activePartnerOptions = computed(() => partners.value.filter(p => p.is_active).map(p => ({ value: p.id, label: `${p.name} · ${formatRate(p.default_rate)}` })))
const statusOptions = [{ value: 'active', label: 'Active' }, { value: 'ended', label: 'Ended (history)' }, { value: 'all', label: 'Active and ended' }]

const items = ref<PlatformAssignment[]>([])
const loading = ref(true)
const error = ref('')
const route = useRoute()
const branchFilter = ref(String(route.query.branch || ''))
const platformFilter = ref('')
const statusFilter = ref('active')

// Grouped by branch: a branch's platforms together.
const groups = computed(() => {
  const map = new Map<string, { branch_id: string; branch_code: string; branch_name: string; rows: PlatformAssignment[] }>()
  for (const a of items.value) {
    if (!map.has(a.branch_id)) map.set(a.branch_id, { branch_id: a.branch_id, branch_code: a.branch_code, branch_name: a.branch_name, rows: [] })
    map.get(a.branch_id)!.rows.push(a)
  }
  return [...map.values()].sort((a, b) => a.branch_code.localeCompare(b.branch_code))
    .map(g => ({ ...g, rows: g.rows.sort((a, b) => Number(b.is_active) - Number(a.is_active) || a.platform_code.localeCompare(b.platform_code)) }))
})

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<{ items: PlatformAssignment[] }>('/platform-assignments', {
      query: { branch_id: branchFilter.value || undefined, platform_id: platformFilter.value || undefined, status: statusFilter.value },
    })
    if (request === loadRequest) items.value = result.items || []
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load assignments.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([branchFilter, platformFilter, statusFilter], load)
onMounted(async () => {
  load()
  loadCurrency()
  try { platforms.value = (await api<{ items: Platform[] }>('/platforms', { query: { include_inactive: 'true' } })).items || [] } catch { platforms.value = [] }
  try { partners.value = (await api<{ items: Partner[] }>('/partners', { query: { include_inactive: 'true' } })).items || [] } catch { partners.value = [] }
})

const saving = ref(false)
const formError = ref('')
const rateCheck = (value: unknown) => !value || (/^\d+(\.\d{1,4})?$/.test(String(value)) && Number(value) <= 100) ? '' : 'enter a partner % from 0 to 100'
const tooMuch = (sum: number) => sum > 100 ? `the partners’ % add up to ${Number(sum.toFixed(4))}; they can share at most 100%` : ''
// An assignment's Partner %: its partners' % added up, or its own when it has none.
const rateOf = (a: PlatformAssignment) => a.partners.length ? a.partners.reduce((sum, p) => sum + Number(p.commission_rate), 0) : a.commission_rate

// --- assign ---
const assignOpen = ref(false)
const assignForm = reactive({ platform_ids: [] as string[], branch_ids: [] as string[], partner_ids: [] as string[], currency: '', starts_on: todayISO() })
const chosenPlatforms = computed(() => platforms.value.filter(p => assignForm.platform_ids.includes(p.id)))
const chosenPartners = computed(() => assignForm.partner_ids.map(id => partners.value.find(p => p.id === id)).filter((p): p is Partner => !!p))
const assignCount = computed(() => assignForm.platform_ids.length * assignForm.branch_ids.length)
const assignCheck = reactive(useFormErrors(
  () => ({ ...assignForm }),
  () => ({
    platform_ids: [rule.chooseOne('platforms')], branch_ids: [rule.chooseOne('branches')],
    partner_ids: [() => tooMuch(chosenPartners.value.reduce((sum, p) => sum + Number(p.default_rate), 0))],
  }),
))

function openAssign() {
  Object.assign(assignForm, { platform_ids: platformFilter.value ? [platformFilter.value] : [], branch_ids: branchFilter.value ? [branchFilter.value] : [], partner_ids: [], currency: '', starts_on: todayISO() })
  formError.value = ''
  assignCheck.reset()
  assignOpen.value = true
}

async function assign() {
  formError.value = ''
  if (!assignCheck.check()) return
  saving.value = true
  try {
    const result = await api<{ items: unknown[] }>('/platform-assignments', { method: 'POST', body: { ...assignForm } })
    assignOpen.value = false
    toast.success(`${result.items.length} ${result.items.length === 1 ? 'assignment' : 'assignments'} made.`, { title: chosenPlatforms.value.length === 1 ? 'Platform assigned' : 'Platforms assigned' })
    await load()
  } catch (e: any) {
    if (!assignCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not assign the platforms.'
  } finally {
    saving.value = false
  }
}

// --- partners, move, end ---
const editing = ref<PlatformAssignment | null>(null)
const editForm = reactive({ partners: [] as AssignmentPartner[], commission_rate: '', currency: '' })
const editShareTotal = computed(() => editForm.partners.reduce((sum, p) => sum + Number(p.commission_rate || 0), 0))
// Active partners not on the assignment yet (one turned off since can stay but not be added again).
const addPartnerOptions = computed(() => activePartnerOptions.value.filter(o => !editForm.partners.some(p => p.partner_id === o.value)))
function openEdit(a: PlatformAssignment) {
  Object.assign(editForm, {
    partners: a.partners.map(p => ({ ...p, commission_rate: String(Number(p.commission_rate)) })),
    commission_rate: String(Number(a.commission_rate)), currency: a.currency,
  })
  formError.value = ''
  editing.value = a
}
// An added partner comes in at their default % (it can still be changed).
function addEditPartner(value: string | string[]) {
  const partner = partners.value.find(p => p.id === String(value || ''))
  if (partner) editForm.partners.push({ partner_id: partner.id, partner_name: partner.name, commission_rate: String(Number(partner.default_rate)), partner_active: true })
}
async function saveEdit() {
  if (!editing.value) return
  const bad = editForm.partners.find(p => !p.commission_rate || rateCheck(p.commission_rate))
  formError.value = bad ? `${bad.partner_name}: enter a partner % from 0 to 100`
    : tooMuch(editShareTotal.value) || (editForm.partners.length ? '' : rateCheck(editForm.commission_rate) || (editForm.commission_rate ? '' : 'enter a partner %'))
  if (formError.value) return
  const body = { partners: editForm.partners.map(p => ({ partner_id: p.partner_id, commission_rate: p.commission_rate })), commission_rate: editForm.commission_rate, currency: editForm.currency }
  await run(() => api(`/platform-assignments/${editing.value!.id}`, { method: 'PATCH', body }), 'Partners saved', 'New weeks use these partners and %; saved weeks keep theirs.', () => { editing.value = null })
}

const moving = ref<PlatformAssignment | null>(null)
const moveForm = reactive({ branch_id: '', on: todayISO(), reason: '' })
function openMove(a: PlatformAssignment) {
  Object.assign(moveForm, { branch_id: '', on: todayISO(), reason: '' })
  formError.value = ''
  moving.value = a
}
async function saveMove() {
  if (!moving.value || !moveForm.branch_id) return
  await run(() => api(`/platform-assignments/${moving.value!.id}/move`, { method: 'POST', body: { ...moveForm } }), 'Platform moved', 'The old assignment is kept under Ended.', () => { moving.value = null })
}

const ending = ref<PlatformAssignment | null>(null)
const endForm = reactive({ ends_on: todayISO(), reason: '' })
function openEnd(a: PlatformAssignment) {
  Object.assign(endForm, { ends_on: todayISO(), reason: '' })
  formError.value = ''
  ending.value = a
}
async function saveEnd() {
  if (!ending.value) return
  await run(() => api(`/platform-assignments/${ending.value!.id}/end`, { method: 'POST', body: { ...endForm } }), 'Assignment ended', 'It is kept under Ended with the weeks entered.', () => { ending.value = null })
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
