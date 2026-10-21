<template>
  <section class="space-y-4">
    <header class="flex items-center justify-between gap-3">
      <div><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCESS CONTROL</p><h1 class="text-xl font-bold text-navy">Roles &amp; permissions</h1><p class="mt-1 text-xs text-muted">Create roles and choose what each role can access.</p></div>
      <button v-if="canManage" class="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate">+ Role</button>
    </header>

    <p v-if="notice" class="rounded-lg bg-green-50 px-3 py-2 text-sm text-income">{{ notice }}</p>
    <div v-if="loading" class="text-sm text-muted">Loading roles…</div>
    <div v-else-if="error" class="rounded-lg bg-red-50 px-3 py-3 text-sm text-expense">{{ error }}</div>
    <div v-else class="space-y-3">
      <article v-for="role in roles" :key="role.id" class="rounded-xl border border-border bg-white p-4">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2"><h2 class="text-sm font-semibold text-navy">{{ role.name }}</h2><span v-if="role.is_system" class="rounded-full bg-slate-100 px-2 py-0.5 text-[9px] font-bold uppercase text-muted">System</span></div>
            <p class="mt-1 font-mono text-[10px] text-muted">{{ role.code }}</p>
            <p v-if="role.description" class="mt-1 text-xs text-muted">{{ role.description }}</p>
          </div>
          <div v-if="canManage" class="flex shrink-0 gap-1.5">
            <button class="action-chip action-edit" @click="openEdit(role)"><Pencil :size="13" /> Edit</button>
            <button v-if="!role.is_system" class="action-chip action-danger" @click="deleteTarget = role"><Trash2 :size="13" /> Delete</button>
          </div>
        </div>
        <div v-if="role.permissions?.length" class="mt-3 flex flex-wrap gap-1.5">
          <span v-for="permission in role.permissions" :key="permission.id || permission.code" class="rounded-md border border-border bg-canvas px-2 py-1 font-mono text-[9px] text-navy">{{ permission.code || permission.name }}</span>
        </div>
        <p v-else class="mt-3 text-xs text-muted">No permissions assigned.</p>
      </article>
      <div v-if="!roles.length" class="rounded-xl border border-border bg-white p-8 text-center text-sm text-muted">No roles found.</div>
    </div>

    <div v-if="modalOpen" class="fixed inset-0 z-[90] flex items-end justify-center bg-slate-950/50 p-0 backdrop-blur-sm sm:items-center sm:p-5" @click.self="close">
      <section class="max-h-[94vh] w-full max-w-3xl overflow-y-auto rounded-t-2xl border border-border bg-white shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" :aria-label="formMode === 'create' ? 'Create role' : 'Edit role'">
        <header class="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-white px-5 py-4">
          <div><p class="text-[9px] font-extrabold tracking-[0.14em] text-slate-400">{{ formMode === 'create' ? 'NEW ROLE' : 'UPDATE ROLE' }}</p><h2 class="mt-1 text-base font-bold text-navy">{{ formMode === 'create' ? 'Create role' : `Edit ${editingRole?.name}` }}</h2></div>
          <button class="grid h-8 w-8 place-items-center rounded-lg bg-canvas text-muted hover:text-navy" aria-label="Close" @click="close"><X :size="17" /></button>
        </header>
        <form class="space-y-4 p-5" @submit.prevent="save">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label class="space-y-1.5 text-xs font-semibold text-navy">Name<input v-model.trim="form.name" required minlength="2" maxlength="128" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm font-normal" /></label>
            <label v-if="formMode === 'create'" class="space-y-1.5 text-xs font-semibold text-navy">Code<input v-model.trim="form.code" required minlength="2" maxlength="64" class="w-full rounded-lg border border-border px-3 py-2.5 font-mono text-sm font-normal" placeholder="e.g. store-manager" /></label>
            <label class="space-y-1.5 text-xs font-semibold text-navy sm:col-span-2">Description<textarea v-model.trim="form.description" rows="2" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm font-normal" /></label>
          </div>
          <div>
            <h3 class="mb-1 text-sm font-semibold text-navy">Permissions</h3>
            <p class="mb-3 text-xs text-muted">Tick what this role can do. Use a module's top checkbox to give or remove the whole module.</p>
            <PermissionPanel v-model="form.permission_codes" :permissions="permissions" :disabled="saving" />
          </div>
          <p v-if="formError" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-expense" role="alert">{{ formError }}</p>
          <footer class="flex justify-end gap-2 border-t border-border pt-4">
            <button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="close">Cancel</button>
            <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save role' }}</button>
          </footer>
        </form>
      </section>
    </div>

    <div v-if="deleteTarget" class="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
      <section class="w-full max-w-sm rounded-2xl border border-border bg-white p-5 shadow-2xl" role="alertdialog" aria-modal="true" aria-labelledby="delete-role-title">
        <h2 id="delete-role-title" class="text-base font-bold text-navy">Delete {{ deleteTarget.name }}?</h2>
        <p class="mt-2 text-sm text-muted">This role can be deleted only after it is removed from every user.</p>
        <p v-if="deleteError" class="mt-3 rounded-lg bg-red-50 px-3 py-2 text-xs text-expense">{{ deleteError }}</p>
        <footer class="mt-5 flex justify-end gap-2"><button class="rounded-lg border border-border px-3 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="closeDelete">Cancel</button><button class="rounded-lg bg-expense px-3 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving" @click="removeRole">{{ saving ? 'Deleting…' : 'Delete role' }}</button></footer>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Pencil, Trash2, X } from 'lucide-vue-next'
import type { Permission, Role } from '~/types/api'

const { api } = useApi()
const { can, hasFullAccess } = usePermissions()
// Roles and permissions are system-wide, so only the owner/admin may change them.
const canManage = computed(() => can('roles.manage') && hasFullAccess.value)
const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const formError = ref('')
const deleteError = ref('')
const notice = ref('')
const modalOpen = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const editingRole = ref<Role | null>(null)
const deleteTarget = ref<Role | null>(null)
const form = reactive({ code: '', name: '', description: '', permission_codes: [] as string[] })

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [roleResult, permissionResult] = await Promise.all([
      api<{ items: Role[] }>('/roles'),
      api<{ items: Permission[] }>('/permissions').catch(() => ({ items: [] })),
    ])
    roles.value = roleResult.items || []
    permissions.value = permissionResult.items || []
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load roles.'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  formMode.value = 'create'
  editingRole.value = null
  Object.assign(form, { code: '', name: '', description: '', permission_codes: [] })
  formError.value = ''
  modalOpen.value = true
}

function openEdit(role: Role) {
  formMode.value = 'edit'
  editingRole.value = role
  Object.assign(form, {
    code: role.code,
    name: role.name,
    description: role.description || '',
    permission_codes: (role.permissions || []).map((permission) => permission.code),
  })
  formError.value = ''
  modalOpen.value = true
}

function close() {
  if (!saving.value) modalOpen.value = false
}

async function save() {
  saving.value = true
  formError.value = ''
  try {
    if (formMode.value === 'create') {
      await api('/roles', { method: 'POST', body: { code: form.code, name: form.name, description: form.description, permission_codes: form.permission_codes } })
    } else if (editingRole.value) {
      await api(`/roles/${editingRole.value.id}`, { method: 'PATCH', body: { name: form.name, description: form.description, permission_codes: form.permission_codes } })
    }
    modalOpen.value = false
    notice.value = 'Role saved.'
    await load()
  } catch (e: any) {
    formError.value = e?.data?.error || 'Could not save this role.'
  } finally {
    saving.value = false
  }
}

function closeDelete() {
  if (!saving.value) {
    deleteTarget.value = null
    deleteError.value = ''
  }
}

async function removeRole() {
  if (!deleteTarget.value) return
  saving.value = true
  deleteError.value = ''
  try {
    await api(`/roles/${deleteTarget.value.id}`, { method: 'DELETE' })
    deleteTarget.value = null
    notice.value = 'Role deleted.'
    await load()
  } catch (e: any) {
    deleteError.value = e?.data?.error || 'Could not delete this role.'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
