<template>
  <section class="mx-auto max-w-4xl space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">PLATFORMS</p>
        <h1 class="text-xl font-bold text-navy">Platforms</h1>
        <p class="mt-1 text-xs text-muted">The platforms HQs report weekly win/lose for. The default rate and currency fill in new HQ assignments.</p>
      </div>
      <div class="flex gap-2">
        <NuxtLink v-if="can('platforms.view', 'platform-assignments.manage')" to="/admin/platform-assignments" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy hover:bg-canvas"><Network :size="16" /> HQ assignments</NuxtLink>
        <button v-if="canManage" type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> Add platform</button>
      </div>
    </header>

    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by code or name…" /></div>
      <label class="flex items-center gap-2 text-xs font-medium text-navy"><input v-model="showInactive" type="checkbox" class="accent-primary" /> Show turned-off platforms</label>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading platforms…</div>
    <AppAlert v-else-if="error" title="Couldn’t load platforms" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="platform in shown" :key="platform.id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl" :class="platform.is_active ? 'bg-blue-50 text-primary' : 'bg-slate-100 text-slate-400'"><Layers :size="18" /></span>
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="truncate font-mono text-sm font-semibold" :class="platform.is_active ? 'text-navy' : 'text-slate-400'">{{ platform.code }}</h2>
              <span v-if="!platform.is_active" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span>
            </div>
            <p class="truncate text-xs text-muted">{{ platform.name }}</p>
          </div>
        </div>
        <div class="flex items-center gap-4 text-xs sm:justify-end">
          <div class="text-right"><p class="text-[10px] uppercase tracking-wide text-slate-400">Default rate</p><p class="font-semibold text-navy">{{ formatRate(platform.default_rate) }} · {{ platform.default_currency }}</p></div>
          <div class="text-right"><p class="text-[10px] uppercase tracking-wide text-slate-400">HQs</p><p class="font-semibold text-navy">{{ platform.active_hqs }}</p></div>
          <div v-if="canManage" class="flex shrink-0 gap-1.5">
            <button type="button" class="action-chip action-edit" @click="openEdit(platform)"><Pencil :size="13" /> Edit</button>
            <button type="button" class="action-chip" :class="platform.is_active ? 'action-danger' : 'action-success'" :disabled="busyId === platform.id" @click="toggle(platform)">{{ platform.is_active ? 'Turn off' : 'Turn on' }}</button>
          </div>
        </div>
      </li>
      <li v-if="!shown.length" class="p-6 text-center text-sm text-muted">{{ search ? `No platform matches “${search}”.` : 'No platforms yet.' }}</li>
    </ul>

    <AppModal :model-value="formOpen" form :title="editing ? `Edit ${editing.code}` : 'Add platform'" eyebrow="PLATFORM" subtitle="Rates are percents: 35 means 35%." :busy="saving" width="480px" @update:model-value="(open) => { if (!open && !saving) formOpen = false }" @submit="save">
      <template #icon><Layers :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="block space-y-1 text-xs font-medium text-muted">Code<input v-model.trim="form.code" maxlength="64" :aria-invalid="formCheck.invalid('code')" class="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm text-navy" placeholder="e.g. 12K_YMLLJB_V2" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Name<input v-model.trim="form.name" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Default rate (%)<input v-model.trim="form.default_rate" type="text" inputmode="decimal" :aria-invalid="formCheck.invalid('default_rate')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. 35" /></label>
        <div class="space-y-1 text-xs font-medium text-muted">Default currency<SearchSelect v-model="form.default_currency" :options="currencyOptions" :searchable="false" aria-label="Default currency" :aria-invalid="formCheck.invalid('default_currency')" /></div>
      </div>
      <p v-if="editing" class="mt-3 rounded-lg bg-canvas px-3 py-2 text-[11px] text-muted">Changing the default only affects HQs assigned from now on. Each HQ’s rate is set on its assignment, and saved weeks keep the rate they were worked out with.</p>
      <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="formOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { Layers, Network, Pencil, Plus } from 'lucide-vue-next'
import type { Platform } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useCurrency } from '~/composables/useCurrency'
import { rule } from '~/utils/validation'
import { formatRate } from '~/utils/winlose'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canManage = computed(() => can('platforms.manage'))
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
    error.value = e?.data?.error || 'Could not load platforms.'
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
const form = reactive({ code: '', name: '', default_rate: '', default_currency: 'USD' })

// Same rules as the API (services.PlatformInput / SavePlatform).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    code: [rule.required('code'), (value: unknown) => /^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$/.test(String(value || '')) ? '' : 'use letters, digits, _ . or - in the code'],
    name: [rule.required('name'), rule.maxLength('name', 128)],
    default_rate: [rule.required('default rate'), (value: unknown) => /^\d+(\.\d{1,4})?$/.test(String(value)) && Number(value) <= 100 ? '' : 'enter a rate from 0 to 100'],
    default_currency: [rule.required('currency')],
  }),
))

function openCreate() {
  editing.value = null
  Object.assign(form, { code: '', name: '', default_rate: '', default_currency: 'USD' })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

function openEdit(platform: Platform) {
  editing.value = platform
  Object.assign(form, { code: platform.code, name: platform.name, default_rate: String(Number(platform.default_rate)), default_currency: platform.default_currency })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

// The name starts as the code (codes like 12K_YMLLJB_V1 are what people call them).
watch(() => form.code, (code, old) => { if (!editing.value && (!form.name || form.name === old)) form.name = code })

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const body = { ...form, is_active: editing.value ? editing.value.is_active : true }
    if (editing.value) await api(`/platforms/${editing.value.id}`, { method: 'PATCH', body })
    else await api('/platforms', { method: 'POST', body })
    formOpen.value = false
    toast.success(`${form.code} is saved.`, { title: editing.value ? 'Platform updated' : 'Platform added' })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this platform.'
  } finally {
    saving.value = false
  }
}

async function toggle(platform: Platform) {
  busyId.value = platform.id
  try {
    await api(`/platforms/${platform.id}`, {
      method: 'PATCH',
      body: { code: platform.code, name: platform.name, default_rate: platform.default_rate, default_currency: platform.default_currency, is_active: !platform.is_active },
    })
    toast.success(`${platform.code} is turned ${platform.is_active ? 'off' : 'on'}.`)
    await load()
  } catch (e: any) {
    toast.error(e?.data?.error || 'Could not change this platform.')
  } finally {
    busyId.value = ''
  }
}
</script>
