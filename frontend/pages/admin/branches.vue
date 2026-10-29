<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">BUSINESS LOCATIONS</p>
        <h1 class="text-xl font-bold text-navy">Branches</h1>
        <p class="mt-1 text-xs text-muted">Create, view, update, and archive business locations.</p>
      </div>
      <button v-if="canCreate" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate">
        <Plus :size="16" /> Add branch
      </button>
    </header>

    <AppModal
      :model-value="showCreate || !!editing"
      form
      :title="editing ? `Edit ${editing.name}` : 'Create branch'"
      :eyebrow="editing ? 'EDIT BRANCH' : 'NEW BRANCH'"
      subtitle="Location details shown on transactions and reports."
      :busy="saving"
      @update:model-value="(open) => { if (!open) closeForm() }"
      @submit="save"
    >
      <template #icon><Building2 :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <!-- Codes are made up by the API (BR-001, BR-002…) and can't be changed here -->
        <label class="space-y-1 text-xs font-medium text-muted">Branch code<input :value="form.code" readonly tabindex="-1" class="w-full cursor-default rounded-lg border border-border bg-canvas px-3 py-2 font-mono text-sm text-muted" :placeholder="editing ? '' : 'Made automatically'" /><small v-if="!editing" class="block font-normal">Given automatically when the branch is created.</small></label>
        <label class="space-y-1 text-xs font-medium text-muted">Branch name<input v-model.trim="form.name" required minlength="2" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Phnom Penh HQ" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Address<input v-model.trim="form.address" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Phone<input v-model.trim="form.phone" maxlength="32" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Email<input v-model.trim="form.email" type="email" maxlength="255" :aria-invalid="formCheck.invalid('email')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
      </div>
      <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save this branch' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="closeForm">Cancel</button>
        <button type="submit" :disabled="saving" class="bg-primary text-white disabled:opacity-60">{{ saving ? 'Saving…' : editing ? 'Save changes' : 'Create branch' }}</button>
      </template>
    </AppModal>

    <BranchTelegramModal :model-value="!!telegramBranch" :branch="telegramBranch" @update:model-value="(open) => { if (!open) telegramBranch = null }" @saved="onTelegramSaved" />

    <ListSearch v-model="search" placeholder="Search by code, name, address, phone or email…" />
    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="err" title="Couldn’t load branches" :message="err"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="branch in items" :key="branch.id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="text-sm font-semibold text-navy">{{ branch.name }}</h2>
            <span class="rounded-full px-2 py-1 text-[10px] font-semibold uppercase" :class="branch.is_active ? 'bg-green-50 text-income' : 'bg-slate-100 text-muted'">{{ branch.is_active ? 'Active' : 'Inactive' }}</span>
            <span v-if="branch.telegram_enabled" class="inline-flex items-center gap-1 rounded-full bg-sky-50 px-2 py-1 text-[10px] font-semibold text-sky-700" title="Sends transaction updates to Telegram"><Send :size="10" /> Telegram</span>
          </div>
          <p class="mt-1 font-mono text-xs text-muted">{{ branch.code }}</p>
          <p v-if="branch.address || branch.phone || branch.email" class="mt-1 break-words text-xs text-muted">
            <span v-if="branch.address">{{ branch.address }}</span><span v-if="branch.phone"> · {{ branch.phone }}</span><span v-if="branch.email"> · {{ branch.email }}</span>
          </p>
        </div>
        <div v-if="canEdit || canManage" class="flex flex-wrap gap-2 sm:shrink-0 sm:justify-end">
          <button v-if="canEdit" type="button" class="action-chip action-edit" @click="openEdit(branch)"><Pencil :size="13" /> Edit</button>
          <button v-if="canEdit" type="button" class="action-chip action-edit" @click="telegramBranch = branch"><Send :size="13" /> Telegram</button>
          <button v-if="canManage && branch.is_active" type="button" class="action-chip action-danger" @click="confirmDelete(branch)"><Trash2 :size="13" /> Delete</button>
          <button v-else-if="canManage" type="button" class="action-chip action-success" :disabled="saving" @click="setActive(branch, true)"><RotateCcw :size="13" /> Restore</button>
        </div>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">{{ search ? `No branches match “${search}”.` : 'No branches' }}</li>
    </ul>

    <div v-if="archiveTarget" class="fixed inset-0 z-[100] flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-sm sm:items-center sm:p-4" @click.self="archiveTarget = null">
      <section class="w-full max-w-md rounded-t-2xl border border-border bg-white p-5 shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" aria-labelledby="archive-branch-title">
        <h2 id="archive-branch-title" class="text-base font-bold text-navy">Delete {{ archiveTarget.name }}?</h2>
        <p class="mt-2 text-sm text-muted">This archives the branch and keeps its transaction history. You can restore it later.</p>
        <AppAlert v-if="archiveError" class="mt-3" compact :message="archiveError" />
        <footer class="mt-5 flex flex-col-reverse gap-2 border-t border-border pt-3 sm:flex-row sm:justify-end">
          <button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="archiveTarget = null">Cancel</button>
          <button type="button" class="rounded-lg bg-expense px-4 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving" @click="deleteBranch">{{ saving ? 'Deleting…' : 'Delete branch' }}</button>
        </footer>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Building2, Pencil, Plus, RotateCcw, Send, Trash2 } from 'lucide-vue-next'
import type { Branch, BranchTelegramSettings } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { rule } from '~/utils/validation'
import { useFormErrors } from '~/composables/useFormErrors'

const { api } = useApi()
const { can, canCreateBranch } = usePermissions()
const canCreate = canCreateBranch
const canEdit = computed(() => can('branches.edit', 'branches.manage'))
const canManage = computed(() => can('branches.manage'))
const items = ref<Branch[]>([])
const loading = ref(true)
const err = ref('')
const toast = useToast()
const showCreate = ref(false)
const saving = ref(false)
const formError = ref('')
const archiveError = ref('')
const editing = ref<Branch | null>(null)
const archiveTarget = ref<Branch | null>(null)
// The branch whose Telegram settings are open.
const telegramBranch = ref<Branch | null>(null)

function onTelegramSaved(settings: BranchTelegramSettings) {
  const branch = items.value.find(item => item.id === telegramBranch.value?.id)
  if (branch) branch.telegram_enabled = settings.enabled && !!settings.chat_id
}
const form = reactive({ code: '', name: '', address: '', phone: '', email: '' })

const search = ref('')
let loadRequest = 0

async function load() {
  const request = ++loadRequest
  loading.value = true
  err.value = ''
  try {
    const result = await api<{ items: Branch[] }>('/branches', { query: { page_size: 100, search: search.value || undefined } })
    if (request === loadRequest) items.value = result.items || []
  } catch (error: any) {
    if (request === loadRequest) err.value = error?.data?.error || 'Could not load branches.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch(search, () => load())

// Same rules as the API (CreateBranchInput / UpdateBranchInput).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 128)],
    email: [rule.email('email'), rule.maxLength('email', 255)],
  }),
))

function resetForm() {
  Object.assign(form, { code: '', name: '', address: '', phone: '', email: '' })
  formError.value = ''
  formCheck.reset()
}

function openCreate() {
  editing.value = null
  resetForm()
  showCreate.value = true
  // Show the code it will get (the API gives it for real when saving).
  api<{ code: string }>('/branches/next-code').then((r) => { if (showCreate.value && !editing.value) form.code = r.code }).catch(() => {})
}

function openEdit(branch: Branch) {
  showCreate.value = false
  editing.value = branch
  Object.assign(form, {
    code: branch.code,
    name: branch.name,
    address: branch.address || '',
    phone: branch.phone || '',
    email: branch.email || '',
  })
  formError.value = ''
  formCheck.reset()
}

function closeForm() {
  if (saving.value) return
  showCreate.value = false
  editing.value = null
  resetForm()
}

function confirmDelete(branch: Branch) {
  archiveTarget.value = branch
  archiveError.value = ''
}

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  // The code is never sent: a new branch gets the next one, and an existing one keeps its own.
  const body = { name: form.name.trim(), address: form.address.trim(), phone: form.phone.trim(), email: form.email.trim() }
  try {
    let code = editing.value?.code || ''
    if (editing.value) await api(`/branches/${editing.value.id}`, { method: 'PATCH', body })
    else code = (await api<Branch>('/branches', { method: 'POST', body })).code
    toast.success(`${code} · ${body.name} is saved.`, { title: editing.value ? 'Branch updated' : 'Branch created' })
    showCreate.value = false
    editing.value = null
    resetForm()
    await load()
  } catch (error: any) {
    if (!formCheck.fromApi(error)) formError.value = error?.data?.error || 'Could not save this branch.'
  } finally {
    saving.value = false
  }
}

async function setActive(branch: Branch, isActive: boolean) {
  saving.value = true
  try {
    await api(`/branches/${branch.id}`, { method: 'PATCH', body: { is_active: isActive } })
    toast.success(isActive ? `${branch.name} is active again.` : `${branch.name} was deleted.`, { title: isActive ? 'Branch restored' : 'Branch deleted' })
    await load()
  } catch (error: any) {
    toast.error(error?.data?.error || 'Please try again.', { title: 'Couldn’t update the branch' })
  } finally {
    saving.value = false
  }
}

async function deleteBranch() {
  if (!archiveTarget.value) return
  saving.value = true
  archiveError.value = ''
  try {
    await api(`/branches/${archiveTarget.value.id}`, { method: 'DELETE' })
    toast.success(`${archiveTarget.value.name} was deleted. You can restore it later.`, { title: 'Branch deleted' })
    archiveTarget.value = null
    await load()
  } catch (error: any) {
    archiveError.value = error?.data?.error || 'Could not delete this branch.'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
