<template>
  <section class="mx-auto max-w-4xl space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS</p>
        <h1 class="text-xl font-bold text-navy">Partners</h1>
        <p class="mt-1 text-xs text-muted">Who shares in each branch-platform’s Total, each at their own %. A platform can have several partners; a partner’s default % is used when they’re assigned.</p>
      </div>
      <button v-if="canManage" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> Add partner</button>
    </header>

    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by name, phone or note…" /></div>
      <label class="flex items-center gap-2 text-xs font-medium text-navy"><input v-model="showInactive" type="checkbox" class="accent-primary" /> Show turned-off partners</label>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading partners…</div>
    <AppAlert v-else-if="error" title="Couldn’t load partners" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="partner in shown" :key="partner.id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl" :class="partner.is_active ? 'bg-blue-50 text-primary' : 'bg-slate-100 text-slate-400'"><Handshake :size="18" /></span>
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="truncate text-sm font-semibold" :class="partner.is_active ? 'text-navy' : 'text-slate-400'">{{ partner.name }}</h2>
              <span v-if="!partner.is_active" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span>
            </div>
            <p class="truncate text-xs text-muted">{{ [partner.phone, partner.note].filter(Boolean).join(' · ') || 'No phone or note' }}</p>
          </div>
        </div>
        <div class="flex items-center gap-4 text-xs sm:justify-end">
          <div class="text-right"><p class="text-[10px] uppercase tracking-wide text-slate-400">Default partner %</p><p class="font-semibold text-navy">{{ formatRate(partner.default_rate) }}</p></div>
          <div class="text-right"><p class="text-[10px] uppercase tracking-wide text-slate-400">Assignments</p><p class="font-semibold text-navy">{{ partner.active_assignments }}</p></div>
          <NuxtLink v-if="can('winlose-reports.view')" :to="{ path: '/winlose/reports/partner-statement', query: { partner: partner.id } }" class="action-chip action-edit"><ScrollText :size="13" /> Statement</NuxtLink>
          <div v-if="canManage" class="flex shrink-0 gap-1.5">
            <button type="button" class="action-chip action-edit" @click="openEdit(partner)"><Pencil :size="13" /> Edit</button>
            <button type="button" class="action-chip" :class="partner.is_active ? 'action-danger' : 'action-success'" :disabled="busyId === partner.id" @click="toggle(partner)">{{ partner.is_active ? 'Turn off' : 'Turn on' }}</button>
          </div>
        </div>
      </li>
      <li v-if="!shown.length" class="p-6 text-center text-sm text-muted">{{ search ? `No partner matches “${search}”.` : 'No partners yet.' }}</li>
    </ul>

    <AppModal :model-value="formOpen" form :title="editing ? `Edit ${editing.name}` : 'Add partner'" eyebrow="PARTNER" subtitle="Partner % is a percent: 35 means 35%." :busy="saving" width="480px" @update:model-value="(open) => { if (!open && !saving) formOpen = false }" @submit="save">
      <template #icon><Handshake :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="block space-y-1 text-xs font-medium text-muted sm:col-span-2">Name<input v-model.trim="form.name" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. Sok Dara" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Phone <span class="font-normal">(optional)</span><input v-model.trim="form.phone" maxlength="32" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Default partner %<input v-model.trim="form.default_rate" type="text" inputmode="decimal" :aria-invalid="formCheck.invalid('default_rate')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. 35" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted sm:col-span-2">Note <span class="font-normal">(optional)</span><input v-model.trim="form.note" maxlength="255" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
      </div>
      <p v-if="editing" class="mt-3 rounded-lg bg-canvas px-3 py-2 text-[11px] text-muted">Changing the default only affects assignments made from now on. Each assignment keeps this partner’s own %, and saved weeks keep the partners and % they were entered with.</p>
      <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="formOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { Handshake, Pencil, Plus, ScrollText } from 'lucide-vue-next'
import type { Partner } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { rule } from '~/utils/validation'
import { formatRate } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canManage = computed(() => can('platforms.manage'))

const items = ref<Partner[]>([])
const loading = ref(true)
const error = ref('')
const search = ref('')
const showInactive = ref(false)
const shown = computed(() => {
  const q = search.value.trim().toLowerCase()
  return items.value.filter(p => (showInactive.value || p.is_active) && (!q || [p.name, p.phone, p.note].some(t => t.toLowerCase().includes(q))))
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = (await api<{ items: Partner[] }>('/partners', { query: { include_inactive: 'true' } })).items || []
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load partners.'
  } finally {
    loading.value = false
  }
}
onMounted(load)

const formOpen = ref(false)
const saving = ref(false)
const formError = ref('')
const editing = ref<Partner | null>(null)
const busyId = ref('')
const form = reactive({ name: '', phone: '', note: '', default_rate: '' })

// Same rules as the API (services.PartnerInput / SavePartner).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    name: [rule.required('name'), rule.maxLength('name', 128)],
    default_rate: [rule.required('default partner %'), (value: unknown) => /^\d+(\.\d{1,4})?$/.test(String(value)) && Number(value) <= 100 ? '' : 'enter a partner % from 0 to 100'],
  }),
))

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', phone: '', note: '', default_rate: '' })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

function openEdit(partner: Partner) {
  editing.value = partner
  Object.assign(form, { name: partner.name, phone: partner.phone, note: partner.note, default_rate: String(Number(partner.default_rate)) })
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
    if (editing.value) await api(`/partners/${editing.value.id}`, { method: 'PATCH', body })
    else await api('/partners', { method: 'POST', body })
    formOpen.value = false
    toast.success(`${form.name} is saved.`, { title: editing.value ? 'Partner updated' : 'Partner added' })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this partner.'
  } finally {
    saving.value = false
  }
}

async function toggle(partner: Partner) {
  busyId.value = partner.id
  try {
    await api(`/partners/${partner.id}`, {
      method: 'PATCH',
      body: { name: partner.name, phone: partner.phone, note: partner.note, default_rate: partner.default_rate, is_active: !partner.is_active },
    })
    toast.success(`${partner.name} is turned ${partner.is_active ? 'off' : 'on'}.`)
    await load()
  } catch (e: any) {
    toast.error(e?.data?.error || 'Could not change this partner.')
  } finally {
    busyId.value = ''
  }
}
</script>
