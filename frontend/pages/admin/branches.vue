<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">BUSINESS LOCATIONS</p>
        <h1 class="text-xl font-bold text-navy">Branches</h1>
        <p class="mt-1 text-xs text-muted">Create, view, update, and archive business locations.</p>
      </div>
      <button v-if="canCreate" class="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate">
        {{ showCreate ? 'Cancel' : '+ Branch' }}
      </button>
    </header>

    <form v-if="showCreate || editing" class="space-y-3 rounded-xl border border-border bg-white p-4" @submit.prevent="save">
      <div class="flex items-start justify-between gap-3">
        <div><h2 class="text-sm font-semibold text-navy">{{ editing ? `Edit ${editing.name}` : 'Create branch' }}</h2><p class="mt-1 text-xs text-muted">Update the location details used on transactions.</p></div>
        <button type="button" class="text-xs font-medium text-muted" :disabled="saving" @click="closeForm">Cancel</button>
      </div>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="space-y-1 text-xs font-medium text-muted">Branch code<input v-model.trim="form.code" required minlength="2" maxlength="32" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="PP-01" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Branch name<input v-model.trim="form.name" required minlength="2" maxlength="128" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Phnom Penh HQ" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Address<input v-model.trim="form.address" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Phone<input v-model.trim="form.phone" maxlength="32" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Email<input v-model.trim="form.email" type="email" maxlength="255" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
      </div>
      <p v-if="formError" class="text-sm text-expense" role="alert">{{ formError }}</p>
      <button type="submit" :disabled="saving" class="rounded-lg bg-primary px-4 py-2 text-sm text-white disabled:opacity-60">{{ saving ? 'Saving…' : editing ? 'Save changes' : 'Create branch' }}</button>
    </form>

    <p v-if="notice" class="rounded-lg bg-green-50 px-3 py-2 text-sm text-income" role="status">{{ notice }}</p>
    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <div v-else-if="err" class="text-sm text-expense">{{ err }}</div>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="branch in items" :key="branch.id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
        <div class="min-w-0">
          <div class="flex flex-wrap items-center gap-2">
            <h2 class="text-sm font-semibold text-navy">{{ branch.name }}</h2>
            <span class="rounded-full px-2 py-1 text-[10px] font-semibold uppercase" :class="branch.is_active ? 'bg-green-50 text-income' : 'bg-slate-100 text-muted'">{{ branch.is_active ? 'Active' : 'Inactive' }}</span>
          </div>
          <p class="mt-1 font-mono text-xs text-muted">{{ branch.code }}</p>
          <p v-if="branch.address || branch.phone || branch.email" class="mt-1 break-words text-xs text-muted">
            <span v-if="branch.address">{{ branch.address }}</span><span v-if="branch.phone"> · {{ branch.phone }}</span><span v-if="branch.email"> · {{ branch.email }}</span>
          </p>
        </div>
        <div v-if="canEdit || canManage" class="flex flex-wrap gap-2 sm:shrink-0 sm:justify-end">
          <button v-if="canEdit" type="button" class="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-navy hover:bg-canvas" @click="openEdit(branch)">Edit</button>
          <button v-if="canManage && branch.is_active" type="button" class="rounded-md border border-red-100 px-3 py-1.5 text-xs font-medium text-expense hover:bg-red-50" @click="confirmDelete(branch)">Delete</button>
          <button v-else-if="canManage" type="button" class="rounded-md border border-green-100 px-3 py-1.5 text-xs font-medium text-income hover:bg-green-50" :disabled="saving" @click="setActive(branch, true)">Restore</button>
        </div>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">No branches</li>
    </ul>

    <div v-if="archiveTarget" class="fixed inset-0 z-[100] flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-sm sm:items-center sm:p-4" @click.self="archiveTarget = null">
      <section class="w-full max-w-md rounded-t-2xl border border-border bg-white p-5 shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" aria-labelledby="archive-branch-title">
        <h2 id="archive-branch-title" class="text-base font-bold text-navy">Delete {{ archiveTarget.name }}?</h2>
        <p class="mt-2 text-sm text-muted">This archives the branch and keeps its transaction history. You can restore it later.</p>
        <p v-if="archiveError" class="mt-3 text-xs text-expense" role="alert">{{ archiveError }}</p>
        <footer class="mt-5 flex flex-col-reverse gap-2 border-t border-border pt-3 sm:flex-row sm:justify-end">
          <button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="archiveTarget = null">Cancel</button>
          <button type="button" class="rounded-lg bg-expense px-4 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving" @click="deleteBranch">{{ saving ? 'Deleting…' : 'Delete branch' }}</button>
        </footer>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import type { Branch } from '~/types/api'

const { api } = useApi()
const { can, canCreateBranch } = usePermissions()
const canCreate = canCreateBranch
const canEdit = computed(() => can('branches.edit', 'branches.manage'))
const canManage = computed(() => can('branches.manage'))
const items = ref<Branch[]>([])
const loading = ref(true)
const err = ref('')
const notice = ref('')
const showCreate = ref(false)
const saving = ref(false)
const formError = ref('')
const archiveError = ref('')
const editing = ref<Branch | null>(null)
const archiveTarget = ref<Branch | null>(null)
const form = reactive({ code: '', name: '', address: '', phone: '', email: '' })

async function load() {
  loading.value = true
  err.value = ''
  try {
    const result = await api<{ items: Branch[] }>('/branches?page_size=100')
    items.value = result.items || []
  } catch (error: any) {
    err.value = error?.data?.error || 'Could not load branches.'
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, { code: '', name: '', address: '', phone: '', email: '' })
  formError.value = ''
}

function openCreate() {
  if (showCreate.value) {
    closeForm()
    return
  }
  editing.value = null
  resetForm()
  showCreate.value = true
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
  saving.value = true
  formError.value = ''
  notice.value = ''
  const body = { code: form.code.trim(), name: form.name.trim(), address: form.address.trim(), phone: form.phone.trim(), email: form.email.trim() }
  try {
    if (editing.value) await api(`/branches/${editing.value.id}`, { method: 'PATCH', body })
    else await api('/branches', { method: 'POST', body })
    notice.value = editing.value ? 'Branch updated.' : 'Branch created.'
    showCreate.value = false
    editing.value = null
    resetForm()
    await load()
  } catch (error: any) {
    formError.value = error?.data?.error || 'Could not save this branch.'
  } finally {
    saving.value = false
  }
}

async function setActive(branch: Branch, isActive: boolean) {
  saving.value = true
  err.value = ''
  notice.value = ''
  try {
    await api(`/branches/${branch.id}`, { method: 'PATCH', body: { is_active: isActive } })
    notice.value = isActive ? 'Branch restored.' : 'Branch deleted.'
    await load()
  } catch (error: any) {
    err.value = error?.data?.error || 'Could not update branch status.'
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
    archiveTarget.value = null
    notice.value = 'Branch deleted.'
    await load()
  } catch (error: any) {
    archiveError.value = error?.data?.error || 'Could not delete this branch.'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
