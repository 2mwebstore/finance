<template>
  <section class="space-y-4">
    <header class="flex items-center justify-between gap-3">
      <div><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCESS CONTROL</p><h1 class="text-xl font-bold text-navy">Roles &amp; permissions</h1><p class="mt-1 text-xs text-muted">Create roles and choose what each role can access.</p></div>
      <button v-if="canManage" class="inline-flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> Add role</button>
    </header>

    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by role name, code or description…" /></div>
      <div v-if="branchChoices.length && !activeBranch" class="sm:w-64"><SearchSelect v-model="appliesFilter" :options="appliesFilterOptions" aria-label="Roles for" /></div>
    </div>
    <p v-if="activeBranch" class="text-xs text-muted">Showing {{ activeBranch.name }}’s own roles. Choose All branches in the top bar to see the built-in roles.</p>
    <p v-else-if="isSuperAdmin && !appliesFilter" class="text-xs text-muted">Showing the built-in roles. Choose a branch in “Roles for” to see that branch’s roles.</p>
    <div v-if="loading" class="text-sm text-muted">Loading roles…</div>
    <AppAlert v-else-if="error" title="Couldn’t load roles" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <div v-else class="space-y-3">
      <article v-for="role in visibleRoles" :key="role.id" class="rounded-xl border border-border bg-white p-4">
        <div class="flex items-start justify-between gap-3">
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2"><h2 class="text-sm font-semibold text-navy">{{ role.name }}</h2><span v-if="role.is_system" class="rounded-full bg-slate-100 px-2 py-0.5 text-[9px] font-bold uppercase text-muted">System</span></div>
            <p class="mt-1 font-mono text-[10px] text-muted">{{ role.code }}</p>
            <div class="mt-2 flex flex-wrap gap-1.5">
              <span v-if="role.branch_id || !role.is_system" class="inline-flex max-w-full items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold" :class="role.branch_id ? 'bg-amber-50 text-amber-800' : 'bg-indigo-50 text-indigo-600'" :title="role.branch_id ? 'A role for this branch only' : 'Made before roles belonged to one branch: edit it to choose its branch'">
                <span class="truncate">{{ role.branch_id ? `For ${role.branch?.name || 'one branch'}` : 'No branch yet' }}</span>
              </span>
              <span class="inline-flex max-w-full items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold" :class="scopeTone(role)" :title="scopeHint(role)">
                <Building2 :size="11" class="shrink-0" /><span class="truncate">{{ scopeLabel(role) }}</span>
              </span>
            </div>
          </div>
          <div v-if="canManage && role.can_manage !== false" class="flex shrink-0 flex-wrap justify-end gap-1.5">
            <!-- Only the Super Admin: copies of this role for the branches that need it -->
            <button v-if="isSuperAdmin && role.code !== 'super_admin'" class="action-chip action-edit" @click="openClone(role)"><Copy :size="13" /> Clone</button>
            <button class="action-chip action-edit" @click="openEdit(role)"><Pencil :size="13" /> Edit</button>
            <button v-if="!role.is_system" class="action-chip action-danger" @click="deleteTarget = role"><Trash2 :size="13" /> Delete</button>
          </div>
        </div>
        <p v-if="role.description" class="mt-2 text-xs text-muted">{{ role.description }}</p>
        <!-- What the role can open, laid out like the sidebar: a row per section, a chip per page. -->
        <div v-if="role.permissions?.length" class="mt-3 space-y-2 border-t border-border pt-3">
          <div v-for="section in roleMenu(role)" :key="section.caption || 'main'" class="flex flex-col gap-1 sm:flex-row sm:items-start sm:gap-3">
            <p class="shrink-0 pt-1 text-[9px] font-bold tracking-[0.13em] text-slate-400 sm:w-32">{{ section.caption }}</p>
            <div class="flex flex-wrap gap-1.5">
              <span
                v-for="page in section.pages"
                :key="page.key"
                class="inline-flex items-center gap-1 rounded-md border px-2 py-1 text-[10px] font-semibold"
                :class="page.full ? 'border-blue-200 bg-blue-50 text-primary' : 'border-border bg-canvas text-navy'"
                :title="page.items.map(item => item.name).join(', ')"
              >
                <component :is="page.icon" :size="12" class="shrink-0" />{{ page.label }}<span v-if="!page.full" class="font-medium text-muted">{{ page.items.length }}/{{ page.codes.length }}</span>
              </span>
            </div>
          </div>
        </div>
        <p v-else class="mt-3 text-xs text-muted">No permissions assigned.</p>
      </article>
      <div v-if="!visibleRoles.length" class="rounded-xl border border-border bg-white p-8 text-center text-sm text-muted">{{ search ? `No roles match “${search}”.` : appliesFilter || activeBranch ? `${appliesFilterOptions.find(o => o.value === appliesFilter)?.label || activeBranch?.name} has no roles of its own yet.${isSuperAdmin ? ' Clone a built-in role to give it one.' : ''}` : 'No roles found.' }}</div>
    </div>

    <div v-if="modalOpen" class="fixed inset-0 z-[90] flex items-end justify-center bg-slate-950/50 p-0 backdrop-blur-sm sm:items-center sm:p-5" @click.self="close">
      <section class="max-h-[94vh] w-full max-w-3xl overflow-y-auto rounded-t-2xl border border-border bg-white shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" :aria-label="formMode === 'create' ? 'Create role' : 'Edit role'">
        <header class="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-white px-5 py-4">
          <div><p class="text-[9px] font-extrabold tracking-[0.14em] text-slate-400">{{ formMode === 'create' ? 'NEW ROLE' : 'UPDATE ROLE' }}</p><h2 class="mt-1 text-base font-bold text-navy">{{ formMode === 'create' ? 'Create role' : `Edit ${editingRole?.name}` }}</h2></div>
          <button class="grid h-8 w-8 place-items-center rounded-lg bg-canvas text-muted hover:text-navy" aria-label="Close" @click="close"><X :size="17" /></button>
        </header>
        <form class="space-y-4 p-5" novalidate @submit.prevent="save">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <label class="space-y-1.5 text-xs font-semibold text-navy">Name<input v-model.trim="form.name" required minlength="2" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm font-normal" /></label>
            <label v-if="formMode === 'create'" class="space-y-1.5 text-xs font-semibold text-navy">Code<input v-model.trim="form.code" required minlength="2" maxlength="64" :aria-invalid="formCheck.invalid('code')" class="w-full rounded-lg border border-border px-3 py-2.5 font-mono text-sm font-normal" placeholder="e.g. store-manager" /></label>
            <div class="space-y-1.5 text-xs font-semibold text-navy sm:col-span-2">
              Applies to
              <SearchSelect v-model="form.branch_id" :options="appliesOptions" :disabled="!!editingRole?.is_system" placeholder="Choose a branch" aria-label="Role applies to" :aria-invalid="formCheck.invalid('branch_id')" />
              <small class="block font-normal text-muted">{{ editingRole?.is_system ? 'Built-in roles belong to no branch, and only the owner or an admin can see and give them.' : 'Only this branch sees the role, and only its users can be given it.' }}</small>
            </div>
            <label class="space-y-1.5 text-xs font-semibold text-navy sm:col-span-2">Description<textarea v-model.trim="form.description" rows="2" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm font-normal" /></label>
          </div>
          <p class="flex items-start gap-2 rounded-lg bg-canvas px-3 py-2 text-xs text-muted">
            <Building2 :size="14" class="mt-px shrink-0" />
            <span v-if="form.permission_codes.includes('branches.manage')">This role has “Manage Branches”, so its users see every branch.</span>
            <span v-else>People with this role see only the branches they are assigned to. Data is not shared between branches.</span>
          </p>
          <div>
            <h3 class="mb-1 text-sm font-semibold text-navy">Permissions</h3>
            <p class="mb-3 text-xs text-muted">Grouped like the sidebar. Tick what this role can do; a page's checkbox gives or removes all of that page, and a section's the whole section.<template v-if="!hasFullAccess"> You can give only permissions you have yourself.</template></p>
            <PermissionPanel v-model="form.permission_codes" :permissions="grantablePermissions" :disabled="saving" />
          </div>
          <AppAlert v-if="formError || formCheck.messages.length" :title="formError ? 'Couldn’t save this role' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
          <footer class="flex justify-end gap-2 border-t border-border pt-4">
            <button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="close">Cancel</button>
            <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save role' }}</button>
          </footer>
        </form>
      </section>
    </div>

    <div v-if="cloneTarget" class="fixed inset-0 z-[100] flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-sm sm:items-center sm:p-4" @click.self="closeClone">
      <section class="w-full max-w-md rounded-t-2xl border border-border bg-white p-5 shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" aria-labelledby="clone-role-title">
        <h2 id="clone-role-title" class="text-base font-bold text-navy">Clone {{ cloneTarget.name }}</h2>
        <p class="mt-1 text-xs text-muted">Choose the branches that need this role. Each gets its own copy with the same permissions, ready to give its people.</p>
        <div class="mt-3 max-h-72 space-y-0.5 overflow-y-auto rounded-lg border border-border p-1.5">
          <label v-for="b in cloneChoices" :key="b.value" class="flex items-center gap-2.5 rounded-md px-2 py-2 text-sm text-navy" :class="b.has ? 'opacity-60' : 'cursor-pointer hover:bg-canvas'">
            <input v-model="cloneBranches" type="checkbox" :value="b.value" :disabled="b.has" class="h-4 w-4 accent-primary" />
            <span class="min-w-0 flex-1 truncate">{{ b.label }}</span>
            <small v-if="b.has" class="shrink-0 text-[11px] text-muted">has it</small>
          </label>
          <p v-if="!cloneChoices.length" class="px-2 py-3 text-xs text-muted">No other branch to copy it to.</p>
        </div>
        <AppAlert v-if="cloneError" class="mt-3" compact :message="cloneError" />
        <footer class="mt-5 flex justify-end gap-2">
          <button type="button" class="rounded-lg border border-border px-3 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="closeClone">Cancel</button>
          <button type="button" class="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving || !cloneBranches.length" @click="cloneRole">{{ saving ? 'Cloning…' : cloneBranches.length > 1 ? `Clone to ${cloneBranches.length} branches` : 'Clone' }}</button>
        </footer>
      </section>
    </div>

    <div v-if="deleteTarget" class="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm">
      <section class="w-full max-w-sm rounded-2xl border border-border bg-white p-5 shadow-2xl" role="alertdialog" aria-modal="true" aria-labelledby="delete-role-title">
        <h2 id="delete-role-title" class="text-base font-bold text-navy">Delete {{ deleteTarget.name }}?</h2>
        <p class="mt-2 text-sm text-muted">It is removed from the database for good and can’t be brought back. A role someone still has can’t be deleted: take it off them first.</p>
        <AppAlert v-if="deleteError" class="mt-3" compact :message="deleteError" />
        <footer class="mt-5 flex justify-end gap-2"><button class="rounded-lg border border-border px-3 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="closeDelete">Cancel</button><button class="rounded-lg bg-expense px-3 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving" @click="removeRole">{{ saving ? 'Deleting…' : 'Delete role' }}</button></footer>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Building2, Copy, Pencil, Plus, Trash2, X } from 'lucide-vue-next'
import type { Permission, Role } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { rule } from '~/utils/validation'
import { groupByMenu } from '~/utils/permissionMenu'
import { useFormErrors } from '~/composables/useFormErrors'
import { branchLabel, useBranchOptions } from '~/composables/useBranchOptions'

const { api } = useApi()
const { can, hasFullAccess } = usePermissions()
// Anyone with Manage Roles may add roles. The owner/admin may change every role; others only
// roles within their own permissions (the API sends can_manage per role).
const canManage = computed(() => can('roles.manage'))
const roles = ref<Role[]>([])
const permissions = ref<Permission[]>([])
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const formError = ref('')
const deleteError = ref('')
const toast = useToast()
const modalOpen = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const editingRole = ref<Role | null>(null)
const deleteTarget = ref<Role | null>(null)
const form = reactive({ code: '', name: '', description: '', permission_codes: [] as string[], branch_id: '' })
// Same rules as the API (CreateRoleInput / UpdateRoleInput); the code is set only when creating.
// Every role but the built-in ones is for one branch.
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 128)],
    ...(formMode.value === 'create' ? { code: [rule.required('code'), rule.minLength('code', 2), rule.maxLength('code', 64)] } : {}),
    ...(!editingRole.value?.is_system ? { branch_id: [rule.required('branch')] } : {}),
  }),
))

const { branches, everyBranch } = useBranchOptions()
// A branch picked in the top bar: the list is that branch's own roles (the API leaves out the built-in ones).
const { activeBranch } = useActiveBranch()
const { user } = useAuth()
const isSuperAdmin = computed(() => !!user.value?.is_super_admin)
const branchChoices = computed(() => branches.value.map(b => ({ value: b.id, label: branchLabel(b) })))
const appliesOptions = computed(() => [...(editingRole.value?.is_system ? [{ value: '', label: 'Built-in role' }] : []), ...branchChoices.value])
const appliesFilter = ref('')
// The Super Admin sees the built-in roles, or one branch's roles (the filter, or the top-bar branch);
// others see every role they may, or one branch's.
const appliesFilterOptions = computed(() => [{ value: '', label: isSuperAdmin.value ? 'Built-in roles' : 'All roles' }, ...branchChoices.value])
const visibleRoles = computed(() => roles.value.filter((role) => {
  if (appliesFilter.value) return role.branch_id === appliesFilter.value
  if (activeBranch.value) return true // the API sent only that branch's roles
  return isSuperAdmin.value ? !role.branch_id : true
}))

// Without full access a role can only get permissions the signed-in user has (the API checks too).
const grantablePermissions = computed(() => hasFullAccess.value ? permissions.value : permissions.value.filter(permission => can(permission.code)))
// Everyone sees only their own branches; Manage Branches (owner/admin) sees every branch.
function seesAll(role: Role) {
  return !!role.permissions?.some(permission => permission.code === 'branches.manage')
}

function scopeLabel(role: Role) {
  return seesAll(role) ? 'All branches' : 'Own branches'
}

function scopeHint(role: Role) {
  return seesAll(role) ? 'Has Manage Branches, so sees every branch' : 'Each user sees only the branches they are assigned to'
}

function scopeTone(role: Role) {
  return seesAll(role) ? 'bg-emerald-50 text-emerald-700' : 'bg-blue-50 text-primary'
}
// A role's permissions grouped like the sidebar; a page shows n/m when only some of it is given.
function roleMenu(role: Role) {
  return groupByMenu(role.permissions || []).map(section => ({
    ...section,
    pages: section.pages.map(page => ({ ...page, full: page.items.length >= page.codes.length })),
  }))
}

const search = ref('')
let loadRequest = 0

async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const [roleResult, permissionResult] = await Promise.all([
      api<{ items: Role[] }>('/roles', { query: { search: search.value || undefined } }),
      permissions.value.length ? Promise.resolve({ items: permissions.value }) : api<{ items: Permission[] }>('/permissions').catch(() => ({ items: [] })),
    ])
    if (request !== loadRequest) return
    roles.value = roleResult.items || []
    permissions.value = permissionResult.items || []
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load roles.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch(search, () => load())

function openCreate() {
  formMode.value = 'create'
  editingRole.value = null
  Object.assign(form, { code: '', name: '', description: '', permission_codes: [], branch_id: branches.value.length === 1 ? branches.value[0]!.id : '' })
  formError.value = ''
  formCheck.reset()
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
    branch_id: role.branch_id || '',
  })
  formError.value = ''
  formCheck.reset()
  modalOpen.value = true
}

function close() {
  if (!saving.value) modalOpen.value = false
}

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    if (formMode.value === 'create') {
      await api('/roles', { method: 'POST', body: { code: form.code, name: form.name, description: form.description, permission_codes: form.permission_codes, branch_id: form.branch_id } })
    } else if (editingRole.value) {
      await api(`/roles/${editingRole.value.id}`, { method: 'PATCH', body: { name: form.name, description: form.description, permission_codes: form.permission_codes, ...(editingRole.value.is_system ? {} : { branch_id: form.branch_id }) } })
    }
    modalOpen.value = false
    toast.success(`${form.name} is ready to assign.`, { title: formMode.value === 'create' ? 'Role created' : 'Role updated' })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this role.'
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
    toast.success(`${deleteTarget.value.name} was removed.`, { title: 'Role deleted' })
    deleteTarget.value = null
    await load()
  } catch (e: any) {
    deleteError.value = e?.data?.error || 'Could not delete this role.'
  } finally {
    saving.value = false
  }
}

// Clone (the Super Admin only): a copy of the role for each branch picked. A branch that already has
// a role of that name is shown but can't be picked (the API skips it too).
const cloneTarget = ref<Role | null>(null)
const cloneBranches = ref<string[]>([])
const cloneError = ref('')
const cloneChoices = computed(() => {
  const target = cloneTarget.value
  if (!target) return []
  const name = target.name.toLowerCase()
  return everyBranch.value
    .filter(b => b.id !== target.branch_id)
    .map(b => ({ value: b.id, label: branchLabel(b), has: roles.value.some(r => r.branch_id === b.id && r.name.toLowerCase() === name) }))
})
function openClone(role: Role) {
  cloneTarget.value = role
  cloneBranches.value = []
  cloneError.value = ''
}
function closeClone() {
  if (!saving.value) cloneTarget.value = null
}
async function cloneRole() {
  const target = cloneTarget.value
  if (!target || !cloneBranches.value.length) return
  saving.value = true
  cloneError.value = ''
  try {
    const result = await api<{ created: Role[]; skipped: { id: string; name: string }[] }>(`/roles/${target.id}/clone`, { method: 'POST', body: { branch_ids: cloneBranches.value } })
    const nameOf = (id?: string | null) => everyBranch.value.find(b => b.id === id)?.name || 'a branch'
    const made = result.created.map(r => nameOf(r.branch_id))
    const skipped = result.skipped.map(b => b.name)
    cloneTarget.value = null
    toast.success(`${target.name} is now in ${made.join(', ') || 'no new branch'}.${skipped.length ? ` ${skipped.join(', ')} already had it.` : ''}`, { title: 'Role cloned' })
    await load()
  } catch (e: any) {
    cloneError.value = e?.data?.error || 'Could not clone this role.'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
