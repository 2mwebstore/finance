<template>
  <section class="space-y-4">
    <header class="flex items-center justify-between gap-3">
      <div><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCESS CONTROL</p><h1 class="text-xl font-bold text-navy">Users</h1></div>
      <button v-if="canCreate" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="toggleCreate"><UserPlus :size="16" /> Add user</button>
    </header>

    <AppModal v-model="showCreate" form title="Create user" eyebrow="NEW USER" subtitle="Give them a role and a branch. Managers are optional." :busy="saving" width="660px" @submit="create">
      <template #icon><UserPlus :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="space-y-1 text-xs font-medium text-muted">Name<input v-model.trim="createForm.full_name" required minlength="2" maxlength="128" :aria-invalid="createCheck.invalid('full_name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. dara" /></label>
        <div class="space-y-1 text-xs font-medium text-muted"><span class="block">Branch</span><SearchSelect v-model="createForm.branch_id" :options="branchOptions" placeholder="Select a branch" search-placeholder="Search branches…" aria-label="User branch" :aria-invalid="createCheck.invalid('branch_ids')" /></div>
        <!-- name@branch, filled in from the Name and the Branch: what they sign in with -->
        <label class="space-y-1 text-xs font-medium text-muted">Username<input :value="createUsername" readonly tabindex="-1" :aria-invalid="createCheck.invalid('username')" class="w-full cursor-default rounded-lg border border-border bg-canvas px-3 py-2 text-sm text-muted" placeholder="Name and branch fill it in" /><small class="block font-normal">They sign in with it.</small></label>
        <label class="space-y-1 text-xs font-medium text-muted">Info <small>(optional)</small><input v-model.trim="createForm.info" maxlength="255" :aria-invalid="createCheck.invalid('info')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. cashier, morning shift" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Temporary password ({{ MIN_PASSWORD_LENGTH }}–{{ MAX_PASSWORD_LENGTH }} characters)<input v-model="createForm.password" type="password" :maxlength="MAX_PASSWORD_LENGTH" :minlength="MIN_PASSWORD_LENGTH" required autocomplete="new-password" autocapitalize="off" autocorrect="off" spellcheck="false" :aria-invalid="createCheck.invalid('password')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Letters, digits or signs" /></label>
        <div class="space-y-1 text-xs font-medium text-muted"><span class="block">Role</span><SearchSelect v-model="createForm.role_code" :options="createRoleOptions" placeholder="Select a role" search-placeholder="Search roles…" :empty-text="createRoleOptions.length ? undefined : createForm.branch_id ? 'No roles for this branch yet: add one in Roles & Permissions' : 'Choose a branch first'" aria-label="User role" :aria-invalid="createCheck.invalid('role_codes')" /></div>
        <div class="space-y-1 text-xs font-medium text-muted sm:col-span-2">
          <span class="block">Managers <small>(approve or reject this user's transactions · optional)</small></span>
          <SearchSelect v-model="createForm.manager_ids" :options="managerOptions(null, createForm.branch_id ? [createForm.branch_id] : [])" multiple :disabled="!createForm.branch_id" :placeholder="createForm.branch_id ? 'Anyone with approval rights' : 'Choose a branch first'" search-placeholder="Search people…" aria-label="User managers" />
        </div>
      </div>
      <AppAlert v-if="createError || createCheck.messages.length" class="mt-3" :title="createError ? 'Couldn’t create this user' : 'Check the highlighted fields'" :message="createError" :items="createCheck.messages" dismissible @dismiss="createError = ''; createCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="showCreate = false">Cancel</button>
        <button type="submit" :disabled="saving" class="bg-primary text-white disabled:opacity-60">{{ saving ? 'Creating…' : 'Create user' }}</button>
      </template>
    </AppModal>

    <AppModal :model-value="!!editingUser" form :title="`Edit ${editingUser?.full_name || ''}`" eyebrow="EDIT USER" :subtitle="editingUser ? editingUser.username : ''" :busy="saving" width="660px" @update:model-value="(open) => { if (!open) cancelEdit() }" @submit="saveEdit">
      <template #icon><UserPen :size="20" /></template>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="space-y-1 text-xs font-medium text-muted">Name<input v-model.trim="editForm.full_name" required minlength="2" maxlength="128" :aria-invalid="editCheck.invalid('full_name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <!-- Changes with the Name or the branch (name@branch); they then sign in with the new one -->
        <label class="space-y-1 text-xs font-medium text-muted">Username<input :value="editForm.username" readonly tabindex="-1" :aria-invalid="editCheck.invalid('username')" class="w-full cursor-default rounded-lg border border-border bg-canvas px-3 py-2 text-sm text-muted" /><small v-if="editingUser && editForm.username !== editingUser.username" class="block font-normal text-amber-700">Was {{ editingUser.username }}: they’ll sign in with the new one.</small></label>
        <label class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Info <small>(optional)</small><input v-model.trim="editForm.info" maxlength="255" :aria-invalid="editCheck.invalid('info')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <div v-if="canManageUsers && !isEditingSelf" class="space-y-1 text-xs font-medium text-muted"><span class="block">Roles <small>(select one or more)</small></span><SearchSelect v-model="editForm.role_codes" :options="editRoleOptions" multiple placeholder="Select roles" search-placeholder="Search roles…" aria-label="User roles" :aria-invalid="editCheck.invalid('role_codes')" /></div>
        <div v-if="canManageUsers && !isEditingSelf && branches.length" class="space-y-1 text-xs font-medium text-muted"><span class="block">Branches <small>(select one or more)</small></span><SearchSelect v-model="editForm.branch_ids" :options="branchOptions" multiple placeholder="Select branches" search-placeholder="Search branches…" aria-label="User branches" :aria-invalid="editCheck.invalid('branch_ids')" /></div>
        <div v-if="canManageUsers && !isEditingSelf" class="space-y-1 text-xs font-medium text-muted sm:col-span-2">
          <span class="block">Managers <small>(only they can approve or reject this user's transactions; empty = anyone with approval rights)</small></span>
          <SearchSelect v-model="editForm.manager_ids" :options="managerOptions(editingUser, editForm.branch_ids)" multiple placeholder="Anyone with approval rights" search-placeholder="Search people…" aria-label="User managers" />
        </div>
      </div>
      <AppAlert v-if="editError || editCheck.messages.length" class="mt-3" :title="editError ? 'Couldn’t save the changes' : 'Check the highlighted fields'" :message="editError" :items="editCheck.messages" dismissible @dismiss="editError = ''; editCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="cancelEdit">Cancel</button>
        <button type="submit" :disabled="saving" class="bg-primary text-white disabled:opacity-60">{{ saving ? 'Saving…' : 'Save changes' }}</button>
      </template>
    </AppModal>

    <ListSearch v-model="search" placeholder="Search by name, username or info…" />
    <FilterCard :summary="userFilterSummary" :active="userFiltersActive">
      <div v-if="branches.length > 1 && !activeBranchId" class="filter-field">
        <span>Branch</span>
        <SearchSelect v-model="branchFilter" :options="filterBranchOptions" placeholder="All branches" search-placeholder="Search branches…" aria-label="Filter users by branch" />
      </div>
      <div class="filter-field">
        <span>Role</span>
        <SearchSelect v-model="roleFilter" :options="filterRoleOptions" :searchable="filterRoleOptions.length > 7" placeholder="All roles" aria-label="Filter users by role" />
      </div>
      <button type="button" class="filter-reset" :disabled="!userFiltersActive" @click="branchFilter = ''; roleFilter = ''"><RotateCcw :size="15" /> Reset</button>
    </FilterCard>
    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load users" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="item in items" :key="item.id" class="p-4 sm:flex sm:items-start sm:justify-between sm:gap-4">
        <div class="flex min-w-0 gap-3">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl text-xs font-bold" :class="item.is_active ? 'bg-blue-50 text-primary' : 'bg-slate-100 text-muted'" aria-hidden="true">{{ initials(item.full_name || item.username) }}</span>
          <div class="min-w-0 flex-1">
            <div class="flex min-w-0 items-center gap-2">
              <span class="truncate text-sm font-semibold text-navy">{{ item.full_name }}</span>
              <span class="shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase" :class="item.is_active ? 'bg-green-50 text-income' : 'bg-slate-100 text-muted'">{{ item.is_active ? 'Active' : 'Inactive' }}</span>
            </div>
            <div class="truncate text-xs text-muted">{{ item.username }}<template v-if="item.info"> · {{ item.info }}</template></div>
            <div class="mt-2 flex flex-wrap gap-1.5">
              <span v-if="item.is_super_admin" class="rounded-full bg-primary/10 px-2 py-1 text-[10px] font-semibold text-primary">Super Admin</span>
              <span v-for="role in item.roles || []" :key="role.id || role.code" class="rounded-full bg-blue-50 px-2 py-1 text-[10px] font-medium text-primary">{{ role.name }}</span>
              <span v-if="!(item.roles || []).length && !item.is_super_admin" class="text-xs text-muted">No role assigned</span>
            </div>
            <p class="mt-2 text-xs text-muted">Branches: <span class="text-navy">{{ (item.branches || []).map(branch => branch.name).join(', ') || '—' }}</span></p>
            <!-- Data scope from the user's roles: whose data they actually see. -->
            <p class="mt-1 text-xs text-muted" :title="scopeTitle(item)">Sees data of: <span class="font-medium" :class="item.data_scope?.all ? 'text-income' : 'text-navy'">{{ scopeText(item) }}</span></p>
            <p v-if="(item.managers || []).length" class="mt-1 text-xs text-muted">Managers: <span class="font-medium text-navy">{{ (item.managers || []).map(manager => manager.full_name || manager.username).join(', ') }}</span></p>
            <div class="mt-2">
              <button type="button" class="text-[11px] font-semibold text-primary" @click="togglePerms(item.id)">{{ expandedPerms[item.id] ? 'Hide permissions' : `Permissions (${permCount(item)})` }}</button>
              <div v-if="expandedPerms[item.id]" class="mt-1.5 flex flex-wrap gap-1">
                <span v-if="item.is_super_admin" class="rounded border border-primary/20 bg-primary/10 px-1.5 py-0.5 text-[10px] text-primary">All permissions</span>
                <span v-for="code in (item.permission_codes || [])" :key="code" class="rounded border border-border bg-canvas px-1.5 py-0.5 font-mono text-[10px] text-navy">{{ code }}</span>
                <span v-if="!item.is_super_admin && !(item.permission_codes || []).length" class="text-[10px] text-muted">No permissions</span>
              </div>
            </div>
          </div>
        </div>
        <!-- Phones: one row of equal buttons under the details. Wider screens: a column on the right. -->
        <div class="user-actions mt-3 grid grid-cols-3 gap-1.5 sm:mt-0 sm:flex sm:shrink-0 sm:flex-col sm:items-end">
          <button v-if="canEditUser && item.can_manage !== false" class="action-chip action-edit" @click="beginEdit(item)"><Pencil :size="13" /> Edit</button>
          <button v-if="canManageUsers && item.can_manage !== false && item.id !== currentUserId" class="action-chip action-pin" @click="openPasswordChange(item)"><KeyRound :size="13" /> Password</button>
          <button v-if="canManageUsers && item.can_manage !== false && item.id !== currentUserId" class="action-chip" :class="item.is_active ? 'action-danger' : 'action-success'" :disabled="saving" @click="toggleActive(item)"><Power :size="13" /> {{ item.is_active ? 'Deactivate' : 'Activate' }}</button>
        </div>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">{{ search ? `No users match “${search}”.` : 'No users' }}</li>
    </ul>

    <div v-if="passwordTarget" class="fixed inset-0 z-[100] flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-sm sm:items-center sm:p-4" @click.self="closePasswordChange">
      <section class="w-full max-w-md rounded-t-2xl border border-border bg-white p-5 shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" aria-labelledby="change-user-pin-title">
        <header class="mb-4 flex items-start justify-between gap-3">
          <div><h2 id="change-user-pin-title" class="text-base font-bold text-navy">Change {{ passwordTarget.full_name }}’s password</h2><p class="mt-1 text-xs text-muted">Enter a new password of {{ MIN_PASSWORD_LENGTH }} to {{ MAX_PASSWORD_LENGTH }} characters: letters, digits and signs. Their active sessions will be revoked.</p></div>
          <button type="button" class="text-xs text-muted" :disabled="saving" @click="closePasswordChange">Close</button>
        </header>
        <form class="space-y-3" novalidate @submit.prevent="changeUserPassword">
          <div class="block space-y-1 text-xs font-medium text-muted"><span>New password</span><PasswordInput v-model="passwordForm.password" aria-label="New password" :aria-invalid="passwordCheck.invalid('password')" /></div>
          <AppAlert v-if="passwordError || passwordCheck.messages.length" compact :message="passwordError || passwordCheck.messages[0]" />
          <footer class="flex flex-col-reverse gap-2 border-t border-border pt-3 sm:flex-row sm:justify-end">
            <button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="closePasswordChange">Cancel</button>
            <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Update password' }}</button>
          </footer>
        </form>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { branchLabel } from '~/composables/useBranchOptions'
import { KeyRound, Pencil, Power, RotateCcw, UserPen, UserPlus } from 'lucide-vue-next'
import type { AdminUser, Branch, Role } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { MAX_PASSWORD_LENGTH, MIN_PASSWORD_LENGTH, rule } from '~/utils/validation'
import { useFormErrors } from '~/composables/useFormErrors'

const { api } = useApi()
const { user: currentUser } = useAuth()
const { can, canCreateUser, hasFullAccess, permissions } = usePermissions()
const { activeBranchId } = useActiveBranch()
const canCreate = canCreateUser
const canEditUser = computed(() => can('users.edit', 'users.manage'))
const canManageUsers = computed(() => can('users.manage'))
const currentUserId = computed(() => currentUser.value?.id || '')

const items = ref<AdminUser[]>([])
const roles = ref<Role[]>([])
const branches = ref<Branch[]>([])
const loading = ref(true)
const error = ref('')
const showCreate = ref(false)
const saving = ref(false)
const createError = ref('')
const editError = ref('')
const toast = useToast()
const editingUser = ref<AdminUser | null>(null)
const passwordTarget = ref<AdminUser | null>(null)
const passwordError = ref('')
const expandedPerms = ref<Record<string, boolean>>({})
const createForm = reactive({ full_name: '', info: '', password: '', role_code: '', branch_id: activeBranchId.value, manager_ids: [] as string[] })
const editForm = reactive({ full_name: '', info: '', username: '', role_codes: [] as string[], branch_ids: [] as string[], manager_ids: [] as string[] })

// The username is the Name and the branch, each in small letters without spaces: "dara@phnompenh"
// ('' until both are there). A user in several branches goes by their default one, else the first.
function usernameFor(name: string, branchId: string | undefined) {
  const branch = branches.value.find(b => b.id === branchId)
  const part = (text: string) => text.toLowerCase().replace(/[\s@]+/g, '')
  return part(name) && branch ? `${part(name)}@${part(branch.name)}` : ''
}
const createUsername = computed(() => usernameFor(createForm.full_name, createForm.branch_id))
function usernameBranch(branchIds: string[]) {
  const own = editingUser.value?.default_branch_id
  return own && branchIds.includes(own) ? own : branchIds[0]
}
// On Edit the username follows the Name and the branches once either is changed (opening the
// form leaves it as it is).
let editStart = { name: '', branches: '' }
watch(() => [editForm.full_name, editForm.branch_ids.join(',')], ([name, ids]) => {
  if (!editingUser.value) return
  if (name === editStart.name && ids === editStart.branches) {
    editForm.username = editingUser.value.username
    return
  }
  const next = usernameFor(editForm.full_name, usernameBranch(editForm.branch_ids))
  if (next) editForm.username = next
})
const passwordForm = reactive({ password: '' })

// Same rules as the API (CreateUserInput, UpdateUserInput, the roles/branches and password endpoints).
const createCheck = reactive(useFormErrors(
  () => ({ ...createForm, username: createUsername.value, role_codes: createForm.role_code, branch_ids: createForm.branch_id }),
  () => ({
    full_name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 128)],
    username: [rule.minLength('username', 3), rule.maxLength('username', 64)],
    info: [rule.maxLength('info', 255)],
    password: [rule.required('password'), rule.password('password')],
    role_codes: [rule.required('role')],
    branch_ids: [rule.required('branch')],
  }),
))
const editCheck = reactive(useFormErrors(
  () => ({ ...editForm }),
  () => ({
    full_name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 128)],
    username: [rule.required('username'), rule.minLength('username', 3), rule.maxLength('username', 64)],
    info: [rule.maxLength('info', 255)],
    ...(canManageUsers.value && !isEditingSelf.value ? { role_codes: [rule.chooseOne('roles')] } : {}),
    ...(canManageUsers.value && !isEditingSelf.value && branches.value.length ? { branch_ids: [rule.chooseOne('branches')] } : {}),
  }),
))
const passwordCheck = reactive(useFormErrors(
  () => ({ ...passwordForm }),
  () => ({ password: [rule.required('password'), rule.password('password')] }),
))
// Branch-level managers can only hand out roles whose permissions they hold themselves (the API enforces this too).
const assignableRoles = computed(() => roles.value.filter((role) => {
  if (role.code === 'super_admin') return !!currentUser.value?.is_super_admin
  if (hasFullAccess.value || !role.permissions) return true
  return role.permissions.every(permission => permissions.value.includes(permission.code))
}))
// A new user's branch: the one picked in the top bar, or the only one there is.
function defaultBranchId() {
  return activeBranchId.value || (branches.value.length === 1 ? branches.value[0]!.id : '')
}
// The roles of the branches the user is in (a branch role is only for its branch). The system roles
// stay out (the Super Admin clones one into a branch on Roles to give it there), except the Super
// Admin role for the Super Admin. Someone who already holds one keeps it (editRoleOptions).
function roleOptionsFor(branchIds: string[]) {
  return assignableRoles.value
    .filter(role => !role.is_system || role.code === 'super_admin')
    .filter(role => !role.branch_id || branchIds.includes(role.branch_id))
    .map(role => ({ value: role.code, label: role.branch_id ? `${role.name} (${role.branch?.name || 'branch'})` : role.name }))
}
const createRoleOptions = computed(() => roleOptionsFor(createForm.branch_id ? [createForm.branch_id] : []))
// A role the editor can't give stays on the list for someone who already has it, so it still shows.
const editRoleOptions = computed(() => {
  const options = roleOptionsFor(editForm.branch_ids)
  const held = (editingUser.value?.roles || []).filter(role => !options.some(option => option.value === role.code))
  return [...options, ...held.map(role => ({ value: role.code, label: role.name }))]
})
// A branch role no longer fits once its branch is taken off the user.
// Only branch roles are dropped; other roles the editor can't hand out stay as they are.
function fitsBranches(code: string, branchIds: string[]) {
  const role = roles.value.find(r => r.code === code)
  return !role?.branch_id || branchIds.includes(role.branch_id)
}
watch(() => createForm.branch_id, () => { if (createForm.role_code && !fitsBranches(createForm.role_code, createForm.branch_id ? [createForm.branch_id] : [])) createForm.role_code = '' })
watch(() => [...editForm.branch_ids], () => { editForm.role_codes = editForm.role_codes.filter(code => fitsBranches(code, editForm.branch_ids)) })
const branchOptions = computed(() => branches.value.map(branch => ({ value: branch.id, label: branchLabel(branch) })))
const isEditingSelf = computed(() => editingUser.value?.id === currentUserId.value)

function togglePerms(userId: string) {
  expandedPerms.value[userId] = !expandedPerms.value[userId]
}

function initials(name: string) {
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map(part => part[0]!.toUpperCase()).join('')
}

// "All branches" (owner/admin), or the names of the user’s own branches.
function scopeText(item: AdminUser) {
  if (item.is_super_admin || item.data_scope?.all) return 'All branches'
  const ids = item.data_scope?.branch_ids || []
  if (!ids.length) return 'No branches'
  const codes = ids.map(id => branches.value.find(branch => branch.id === id)?.name).filter(Boolean)
  const hidden = ids.length - codes.length
  return [codes.join(', '), hidden ? `${hidden} more` : ''].filter(Boolean).join(' + ')
}

function scopeTitle(item: AdminUser) {
  return item.is_super_admin || item.data_scope?.all ? 'Owner/admin access (Manage Branches): every branch' : 'Only the branches this user is assigned to'
}

function permCount(item: AdminUser) {
  const codes = item.permission_codes ?? item.roles.flatMap((role) => role.permission_codes ?? [])
  return new Set(codes).size
}

// Search box and branch / role filters
const search = ref('')
const branchFilter = ref('')
const roleFilter = ref('')
const filterBranchOptions = computed(() => [{ value: '', label: 'All branches' }, ...branchOptions.value])
const filterRoleOptions = computed(() => [{ value: '', label: 'All roles' }, ...roles.value.map(role => ({ value: role.code, label: role.name }))])
const userFiltersActive = computed(() => !!branchFilter.value || !!roleFilter.value)
const userFilterSummary = computed(() => {
  const branch = branches.value.find(item => item.id === branchFilter.value)
  const role = roles.value.find(item => item.code === roleFilter.value)
  return `${branch ? branch.name : 'All branches'} · ${role ? role.name : 'All roles'}`
})

function fetchUsers() {
  return api<{ items: AdminUser[] }>('/users', {
    query: { page_size: 100, search: search.value || undefined, branch_id: branchFilter.value || undefined, role: roleFilter.value || undefined },
  })
}

let userRequest = 0
watch([search, branchFilter, roleFilter], async () => {
  const request = ++userRequest
  loading.value = true
  error.value = ''
  try {
    const result = await fetchUsers()
    if (request === userRequest) items.value = result.items || []
  } catch (e: any) {
    if (request === userRequest) error.value = e?.data?.error || 'Could not load users.'
  } finally {
    if (request === userRequest) loading.value = false
  }
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [userResult, roleResult, branchResult, everyone] = await Promise.all([
      fetchUsers(),
      // include_super_admin: the Super Admin can give the Super Admin role (the API leaves it out for anyone else).
      api<{ items: Role[] }>('/roles', { everyBranch: true, query: { include_super_admin: 'true' } }).catch(() => ({ items: [] })),
      api<{ items: Branch[] }>('/branches?page_size=100').catch(() => ({ items: [] })),
      // Unfiltered list for the Managers picker (the visible list may be filtered).
      api<{ items: AdminUser[] }>('/users', { query: { page_size: 100, is_active: 'true' }, everyBranch: true }).catch(() => ({ items: [] })),
    ])
    items.value = userResult.items || []
    roles.value = roleResult.items || []
    branches.value = branchResult.items || []
    if (!createForm.branch_id) createForm.branch_id = defaultBranchId()
    allUsers.value = everyone.items || []
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load users.'
  } finally {
    loading.value = false
  }
}

function toggleCreate() {
  createError.value = ''
  createCheck.reset()
  showCreate.value = true
}

function openPasswordChange(item: AdminUser) {
  passwordTarget.value = item
  passwordForm.password = ''
  passwordError.value = ''
  passwordCheck.reset()
}

function closePasswordChange() {
  if (saving.value) return
  passwordTarget.value = null
  passwordError.value = ''
}

async function changeUserPassword() {
  if (!passwordTarget.value) return
  passwordError.value = ''
  if (!passwordCheck.check()) return
  saving.value = true
  try {
    await api(`/users/${passwordTarget.value.id}/password`, { method: 'POST', body: { password: passwordForm.password } })
    toast.success(`${passwordTarget.value.full_name} can sign in with the new password.`, { title: 'Password updated' })
    passwordTarget.value = null
    passwordForm.password = ''
  } catch (e: any) {
    if (!passwordCheck.fromApi(e)) passwordError.value = e?.data?.error || 'Could not change this user’s password.'
  } finally {
    saving.value = false
  }
}

// Managers: people who may approve/reject (or owners/admins), working in one of the user's branches.
const allUsers = ref<AdminUser[]>([])
const APPROVER_CODES = ['transactions.approve', 'transactions.reject']
function codesOf(user: AdminUser) {
  return user.permission_codes ?? (user.roles || []).flatMap(role => role.permission_codes || [])
}
function managerOptions(forUser: AdminUser | null, branchIds: string[]) {
  return allUsers.value
    .filter((candidate) => {
      if (!candidate.is_active || candidate.id === forUser?.id) return false
      const codes = codesOf(candidate)
      const full = candidate.is_super_admin || codes.includes('branches.manage')
      if (!full && !APPROVER_CODES.some(code => codes.includes(code))) return false
      return full || (candidate.branches || []).some(branch => branchIds.includes(branch.id))
    })
    .map(candidate => ({ value: candidate.id, label: `${candidate.full_name} (${candidate.username})` }))
}
// Drop managers who stop qualifying when the user's branches change.
watch(() => [...editForm.branch_ids], () => {
  if (!editingUser.value) return
  const allowed = new Set(managerOptions(editingUser.value, editForm.branch_ids).map(option => option.value))
  editForm.manager_ids = editForm.manager_ids.filter(id => allowed.has(id))
})
watch(() => createForm.branch_id, () => {
  const allowed = new Set(managerOptions(null, createForm.branch_id ? [createForm.branch_id] : []).map(option => option.value))
  createForm.manager_ids = createForm.manager_ids.filter(id => allowed.has(id))
})

async function create() {
  createError.value = ''
  if (!createCheck.check()) return
  saving.value = true
  try {
    const username = createUsername.value
    await api('/users', { method: 'POST', body: {
      username, full_name: createForm.full_name.trim(), info: createForm.info.trim(),
      password: createForm.password, role_codes: [createForm.role_code], branch_ids: [createForm.branch_id],
      manager_ids: createForm.manager_ids,
    } })
    const name = createForm.full_name.trim()
    Object.assign(createForm, { full_name: '', info: '', password: '', role_code: '', branch_id: defaultBranchId(), manager_ids: [] })
    showCreate.value = false
    toast.success(`${name} can sign in with ${username} and their password.`, { title: 'User created' })
    await load()
  } catch (e: any) {
    if (!createCheck.fromApi(e)) createError.value = e?.data?.error || 'Could not create this user.'
  } finally {
    saving.value = false
  }
}

function beginEdit(item: AdminUser) {
  editingUser.value = item
  editStart = { name: item.full_name, branches: (item.branches || []).map(branch => branch.id).join(',') }
  Object.assign(editForm, {
    full_name: item.full_name,
    info: item.info || '',
    username: item.username,
    role_codes: (item.roles || []).map((role) => role.code),
    branch_ids: (item.branches || []).map((branch) => branch.id),
    manager_ids: (item.managers || []).map((manager) => manager.id),
  })
  editError.value = ''
  editCheck.reset()
}

function cancelEdit() {
  editingUser.value = null
  editError.value = ''
}

async function saveEdit() {
  if (!editingUser.value) return
  editError.value = ''
  if (!editCheck.check()) return
  const target = editingUser.value
  saving.value = true
  try {
    const renamed = editForm.username !== target.username
    await api(`/users/${target.id}`, { method: 'PATCH', body: { full_name: editForm.full_name.trim(), info: editForm.info.trim(), ...(renamed ? { username: editForm.username } : {}) } })
    if (canManageUsers.value && target.id !== currentUserId.value) {
      // Branches first: a role for a branch needs the user in that branch.
      if (branches.value.length) await api(`/users/${target.id}/branches`, { method: 'POST', body: { branch_ids: editForm.branch_ids } })
      await api(`/users/${target.id}/roles`, { method: 'POST', body: { role_codes: editForm.role_codes } })
      await api(`/users/${target.id}/managers`, { method: 'POST', body: { manager_ids: editForm.manager_ids } })
    }
    editingUser.value = null
    toast.success(`${editForm.full_name.trim() || target.full_name}’s details are saved.${renamed ? ` They now sign in with ${editForm.username}.` : ''}`, { title: 'User updated' })
    await load()
  } catch (e: any) {
    if (!editCheck.fromApi(e)) editError.value = e?.data?.error || 'Could not update this user.'
  } finally {
    saving.value = false
  }
}

async function toggleActive(item: AdminUser) {
  saving.value = true
  try {
    await api(`/users/${item.id}/status`, { method: 'PATCH', body: { is_active: !item.is_active } })
    toast.success(item.is_active ? `${item.full_name} can no longer sign in.` : `${item.full_name} can sign in again.`, { title: item.is_active ? 'User deactivated' : 'User activated' })
    await load()
  } catch (e: any) {
    toast.error(e?.data?.error || 'Please try again.', { title: 'Couldn’t change the status' })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.user-actions:empty { display: none; }
@media (max-width: 639px) {
  .user-actions .action-chip { justify-content: center; min-height: 36px; padding: 0 6px; }
}
</style>
