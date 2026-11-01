<template>
  <section class="rp-compact mx-auto max-w-4xl space-y-4">
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
      <label class="flex items-center gap-2 text-xs font-medium text-navy"><input v-model="showInactive" type="checkbox" class="accent-primary" /> Show turned-off agents</label>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading agents…</div>
    <AppAlert v-else-if="error" title="Couldn’t load agents" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <!-- Like Data entry: on phones the buttons are icons only -->
    <div v-else class="overflow-hidden rounded-xl border border-border bg-white">
      <table class="w-full text-left text-sm">
        <thead class="bg-canvas text-[10px] uppercase tracking-wide text-slate-500">
          <tr>
            <th class="px-2 py-2 font-bold sm:px-3">User</th>
            <!-- On phones the currency goes under the user -->
            <th class="hidden px-2 py-2 font-bold sm:table-cell sm:px-3">Currency</th>
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
            <td class="px-2 py-2.5 text-right tabular-nums sm:px-3">{{ platform.active_hqs }}</td>
            <td v-if="canEdit || canDelete" class="px-2 py-2 sm:px-3">
              <div class="flex justify-end gap-1 sm:gap-1.5">
                <button v-if="canEdit" type="button" class="action-chip action-edit" title="Edit" @click="openEdit(platform)"><Pencil :size="13" /> <span class="max-sm:sr-only">Edit</span></button>
                <button v-if="canEdit" type="button" class="action-chip" :class="platform.is_active ? 'action-danger' : 'action-success'" :title="platform.is_active ? 'Turn off' : 'Turn on'" :disabled="busyId === platform.id" @click="platform.is_active ? (turningOff = platform) : setActive(platform, true)"><Power :size="13" /> <span class="max-sm:sr-only">{{ platform.is_active ? 'Turn off' : 'Turn on' }}</span></button>
                <button v-if="canDelete" type="button" class="action-chip action-danger" title="Delete" @click="openDelete(platform)"><Trash2 :size="13" /> <span class="max-sm:sr-only">Delete</span></button>
              </div>
            </td>
          </tr>
          <tr v-if="!shown.length"><td :colspan="canEdit || canDelete ? 4 : 3" class="px-3 py-8 text-center text-sm text-muted">{{ search ? `No agent matches “${search}”.` : 'No agents yet.' }}</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Delete: only an agent never in a group or entered; a used one can only be turned off -->
    <AppModal :model-value="!!deleting" form :title="deleting ? `Delete ${deleting.code}?` : ''" eyebrow="DELETE AGENT" subtitle="Only an agent that was never in a group or entered can be deleted; a used one can only be turned off." :busy="deleteBusy" width="460px" @update:model-value="(open) => { if (!open && !deleteBusy) deleting = null }" @submit="deleteAgent">
      <AppAlert v-if="deleteError" compact :message="deleteError" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="deleteBusy" @click="deleting = null">Cancel</button>
        <button type="submit" class="bg-expense text-white disabled:opacity-60" :disabled="deleteBusy">{{ deleteBusy ? 'Deleting…' : 'Delete agent' }}</button>
      </template>
    </AppModal>

    <!-- Turning an agent off needs a reason (the audit log keeps it); turning on doesn't -->
    <InventoryReasonDialog :model-value="!!turningOff" :title="turningOff ? `Turn off ${turningOff.code}?` : ''" eyebrow="TURN OFF AGENT" subtitle="It leaves Data entry; weeks already saved stay. The audit log keeps your reason." label="Reason for turning off" confirm-label="Turn off" danger :busy="!!turningOff && busyId === turningOff.id" :server-error="turnOffError" @update:model-value="(open) => { if (!open) { turningOff = null; turnOffError = '' } }" @confirm="(reason) => turningOff && setActive(turningOff, false, reason)" />

    <AppModal :model-value="formOpen" form :title="editing ? `Edit ${editing.code}` : 'Add agent'" eyebrow="AGENT" subtitle="The user is required and must be unique; the name is optional." :busy="saving" width="480px" @update:model-value="(open) => { if (!open && !saving) formOpen = false }" @submit="save">
      <template #icon><Layers :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="block space-y-1 text-xs font-medium text-muted">User <span class="text-expense">*</span><input v-model.trim="form.code" maxlength="64" required :aria-invalid="formCheck.invalid('code')" class="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm text-navy" placeholder="e.g. 12K_YMLLJB_V2" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Name <span class="font-normal">(optional)</span><input v-model.trim="form.name" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <div class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Default currency<SearchSelect v-model="form.default_currency" :options="currencyOptions" :searchable="false" aria-label="Default currency" :aria-invalid="formCheck.invalid('default_currency')" /></div>
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
import { Layers, Network, Pencil, Plus, Power, Trash2 } from 'lucide-vue-next'
import type { Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useCurrency } from '~/composables/useCurrency'
import { rule } from '~/utils/validation'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
// Each action has its own permission (Add, Update, Delete Agents); turning off or on is an update.
const canCreate = computed(() => can('platforms.create'))
const canEdit = computed(() => can('platforms.edit'))
const canDelete = computed(() => can('platforms.delete'))
const { currencyOptions, load: loadCurrency } = useCurrency()

const items = ref<Platform[]>([])
const loading = ref(true)
const error = ref('')
const search = ref('')
const showInactive = ref(false)
const shown = computed(() => {
  const q = search.value.trim().toLowerCase()
  return items.value.filter(p => (showInactive.value || p.is_active) && (!q || p.code.toLowerCase().includes(q) || p.name.toLowerCase().includes(q)))
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = (await api<{ items: Platform[] }>('/platforms', { query: { include_inactive: 'true' } })).items || []
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load agents.'
  } finally {
    loading.value = false
  }
}
onMounted(() => { load(); loadCurrency() })

const formOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const editing = ref<Platform | null>(null)
const busyId = ref('')
const form = reactive({ code: '', name: '', default_currency: 'USD' })

// Same rules as the API (services.PlatformInput / SavePlatform).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    code: [rule.required('user'), (value: unknown) => /^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$/.test(String(value || '')) ? '' : 'use letters, digits, _ . or - in the user'],
    name: [rule.maxLength('name', 128)],
    default_currency: [rule.required('currency')],
  }),
))

function openCreate() {
  editing.value = null
  Object.assign(form, { code: '', name: '', default_currency: 'USD' })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

function openEdit(platform: Platform) {
  editing.value = platform
  Object.assign(form, { code: platform.code, name: platform.name, default_currency: platform.default_currency })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const body = { ...form, is_active: editing.value ? editing.value.is_active : true }
    if (editing.value) await api(`/platforms/${editing.value.id}`, { method: 'PATCH', body })
    else await api('/platforms', { method: 'POST', body })
    formOpen.value = false
    toast.success(`${form.code} is saved.`, { title: editing.value ? 'Agent updated' : 'Agent added' })
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
