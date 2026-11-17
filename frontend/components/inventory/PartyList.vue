<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">{{ title }}</h1>
        <p class="mt-1 text-xs text-muted">{{ intro }}</p>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openAdd"><Plus :size="16" /> Add {{ noun }}</button>
    </header>

    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search name, phone, email or address…" /></div>
      <div class="keep-cols grid grid-cols-2 gap-2 sm:w-96">
        <SearchSelect v-model="owner" :options="ownerFilterOptions" :searchable="false" aria-label="Shared or branch" />
        <SearchSelect v-model="active" :options="activeOptions" :searchable="false" aria-label="On or off" />
      </div>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading {{ kind }}…</div>
    <AppAlert v-else-if="error" :title="`Couldn’t load ${kind}`" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="p in items" :key="p.id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
        <div class="flex min-w-0 flex-1 items-start gap-3">
          <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl text-sm font-bold" :class="p.is_active ? 'bg-blue-50 text-primary' : 'bg-slate-100 text-slate-400'">{{ initials(p.name) }}</span>
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="truncate text-sm font-semibold" :class="p.is_active ? 'text-navy' : 'text-slate-400'">{{ p.name }}</h2>
              <InventoryOwnerBadge :branch-id="p.branch_id" :branch-name="p.branch_name" />
              <span v-if="!p.is_active" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span>
            </div>
            <p v-if="p.phone || p.email" class="mt-0.5 flex flex-wrap gap-x-3 text-xs text-muted">
              <span v-if="p.phone" class="inline-flex items-center gap-1"><Phone :size="12" /> {{ p.phone }}</span>
              <span v-if="p.email" class="inline-flex min-w-0 items-center gap-1"><Mail :size="12" /> <span class="truncate">{{ p.email }}</span></span>
            </p>
            <p v-if="p.address" class="mt-0.5 line-clamp-2 whitespace-pre-line text-xs text-muted">{{ p.address }}</p>
            <p v-if="p.note" class="mt-0.5 text-[11px] italic text-slate-500">{{ p.note }}</p>
            <p class="mt-0.5 text-[11px] text-slate-400">{{ p.documents ? `${p.documents} ${docNoun(p.documents)}` : `No ${docWord} yet` }}</p>
          </div>
        </div>
        <div v-if="canChange(p)" class="flex shrink-0 gap-1.5 self-end sm:self-center">
          <button type="button" class="action-chip action-edit" @click="openEdit(p)"><Pencil :size="13" /> Edit</button>
          <button type="button" class="action-chip" :class="p.is_active ? 'action-danger' : 'action-success'" :disabled="busyId === p.id" @click="toggle(p)">{{ p.is_active ? 'Turn off' : 'Turn on' }}</button>
        </div>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">{{ search || active || owner ? `No ${kind} match.` : `No ${kind} yet. Add one here, or from a ${docWord.replace(/s$/, '')} form.` }}</li>
    </ul>
    <InventoryPager v-model:page="page" :page-size="pageSize" :total="total" />

    <InventoryPartyModal v-model="formOpen" :kind="kind" :party="editing" @saved="load" />
  </section>
</template>

<script setup lang="ts">
import { Mail, Pencil, Phone, Plus } from 'lucide-vue-next'
import type { Paged, Party, PartyKind } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useOwner } from '~/composables/useOwner'

/** The customer or supplier list: search, shared/branch and on/off filters, add, edit, turn off. */
const props = defineProps<{ kind: PartyKind }>()

const { api } = useApi()
const toast = useToast()
const { canChange, filterOptions: ownerFilterOptions, hasFullAccess } = useOwner()
const noun = computed(() => props.kind === 'customers' ? 'customer' : 'supplier')
const title = computed(() => props.kind === 'customers' ? 'Customers' : 'Suppliers')
const intro = computed(() => (props.kind === 'customers'
  ? 'Who you sell to. Sales, quotations and the POS pick a customer shared by every branch or the branch’s own; someone turned off can’t be picked for new ones.'
  : 'Who you buy from. Purchases pick a supplier shared by every branch or the branch’s own; someone turned off can’t be picked for new ones.')
  + (hasFullAccess.value ? '' : ' Shared ones are managed by the owner or an admin.'))
const docWord = computed(() => props.kind === 'customers' ? 'sales or quotations' : 'purchases')
const docNoun = (n: number) => props.kind === 'customers' ? (n === 1 ? 'sale or quotation' : 'sales and quotations') : (n === 1 ? 'purchase' : 'purchases')
const activeOptions = [{ value: '', label: 'On and off' }, { value: 'true', label: 'On' }, { value: 'false', label: 'Off' }]
const initials = (name: string) => name.split(/\s+/).filter(Boolean).slice(0, 2).map(w => w[0]!.toUpperCase()).join('')

const items = ref<Party[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 30
const loading = ref(true)
const error = ref('')
const search = ref('')
const active = ref('')
const owner = ref('')
const busyId = ref('')

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<Paged<Party>>(`/inventory/${props.kind}`, { query: { page: page.value, page_size: pageSize, search: search.value || undefined, active: active.value || undefined, owner: owner.value || undefined } })
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || `Could not load ${props.kind}.`
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch([search, active, owner], () => { page.value = 1; load() })
watch(page, load)
onMounted(load)

async function toggle(p: Party) {
  busyId.value = p.id
  try {
    await api(`/inventory/${props.kind}/${p.id}`, { method: 'PATCH', body: { name: p.name, phone: p.phone, email: p.email, address: p.address, note: p.note, is_active: !p.is_active } })
    toast.success(p.is_active ? `New ${docWord.value} can’t pick ${p.name}; theirs keep them.` : `${p.name} can be picked again.`, { title: p.is_active ? 'Turned off' : 'Turned on' })
    await load()
  } catch (e: any) {
    toast.error(e?.data?.error || 'Please try again.', { title: 'Couldn’t change it' })
  } finally {
    busyId.value = ''
  }
}

const formOpen = ref(false)
const editing = ref<Party | null>(null)
function openAdd() {
  editing.value = null
  formOpen.value = true
}
function openEdit(p: Party) {
  editing.value = p
  formOpen.value = true
}
</script>
