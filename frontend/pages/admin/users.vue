<template>
  <section class="space-y-4">
    <header class="flex items-center justify-between gap-3">
      <div><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCESS CONTROL</p><h1 class="text-xl font-bold text-navy">Users</h1></div>
      <button v-if="canCreate" class="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="toggleCreate">{{ showCreate ? 'Cancel' : '+ User' }}</button>
    </header>

    <form v-if="showCreate" class="space-y-3 rounded-xl border border-border bg-white p-4" @submit.prevent="create">
      <h2 class="text-sm font-semibold text-navy">Create user</h2>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="space-y-1 text-xs font-medium text-muted">Username<input v-model.trim="createForm.username" required minlength="3" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Email<input v-model.trim="createForm.email" type="email" required class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Full name<input v-model.trim="createForm.full_name" required minlength="2" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Temporary 4-digit PIN<input v-model="createForm.password" type="password" inputmode="numeric" pattern="[0-9]{4}" maxlength="4" minlength="4" required autocomplete="new-password" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="4 digits" /></label>
        <div class="space-y-1 text-xs font-medium text-muted"><span class="block">Role</span><SearchSelect v-model="createForm.role_code" :options="roleOptions" placeholder="Select a role" search-placeholder="Search roles…" aria-label="User role" /></div>
        <div class="space-y-1 text-xs font-medium text-muted"><span class="block">Branch</span><SearchSelect v-model="createForm.branch_id" :options="branchOptions" placeholder="Select a branch" search-placeholder="Search branches…" aria-label="User branch" /></div>
        <div class="space-y-1 text-xs font-medium text-muted sm:col-span-2">
          <span class="block">Managers <small>(approve or reject this user's transactions · optional)</small></span>
          <SearchSelect v-model="createForm.manager_ids" :options="managerOptions(null, createForm.branch_id ? [createForm.branch_id] : [])" multiple :disabled="!createForm.branch_id" :placeholder="createForm.branch_id ? 'Anyone with approval rights' : 'Choose a branch first'" search-placeholder="Search people…" aria-label="User managers" />
        </div>
      </div>
      <p v-if="createError" class="text-sm text-expense">{{ createError }}</p>
      <button type="submit" :disabled="saving || !createForm.role_code || !createForm.branch_id" class="rounded-lg bg-primary px-4 py-2 text-sm text-white disabled:opacity-60">{{ saving ? 'Creating…' : 'Create user' }}</button>
    </form>

    <form v-if="editingUser" class="space-y-3 rounded-xl border border-blue-100 bg-white p-4 shadow-sm" @submit.prevent="saveEdit">
      <div class="flex items-start justify-between gap-3"><div><h2 class="text-sm font-semibold text-navy">Edit {{ editingUser.username }}</h2><p class="mt-1 text-xs text-muted">Update profile and access details.</p></div><button type="button" class="text-xs font-medium text-muted hover:text-navy" @click="cancelEdit">Cancel</button></div>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="space-y-1 text-xs font-medium text-muted">Full name<input v-model.trim="editForm.full_name" required minlength="2" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Email<input v-model.trim="editForm.email" type="email" required class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
        <div v-if="canManageUsers && !isEditingSelf" class="space-y-1 text-xs font-medium text-muted"><span class="block">Roles <small>(select one or more)</small></span><SearchSelect v-model="editForm.role_codes" :options="roleOptions" multiple placeholder="Select roles" search-placeholder="Search roles…" aria-label="User roles" /></div>
        <div v-if="canManageUsers && !isEditingSelf && branches.length" class="space-y-1 text-xs font-medium text-muted"><span class="block">Branches <small>(select one or more)</small></span><SearchSelect v-model="editForm.branch_ids" :options="branchOptions" multiple placeholder="Select branches" search-placeholder="Search branches…" aria-label="User branches" /></div>
        <div v-if="canManageUsers && !isEditingSelf" class="space-y-1 text-xs font-medium text-muted sm:col-span-2">
          <span class="block">Managers <small>(only they can approve or reject this user's transactions; empty = anyone with approval rights)</small></span>
          <SearchSelect v-model="editForm.manager_ids" :options="managerOptions(editingUser, editForm.branch_ids)" multiple placeholder="Anyone with approval rights" search-placeholder="Search people…" aria-label="User managers" />
        </div>
      </div>
      <p v-if="editError" class="text-sm text-expense">{{ editError }}</p>
      <button type="submit" :disabled="saving || (canManageUsers && !isEditingSelf && (!editForm.role_codes.length || (branches.length > 0 && !editForm.branch_ids.length)))" class="rounded-lg bg-primary px-4 py-2 text-sm text-white disabled:opacity-60">{{ saving ? 'Saving…' : 'Save changes' }}</button>
    </form>

    <p v-if="notice" class="rounded-lg bg-green-50 px-3 py-2 text-sm text-income">{{ notice }}</p>
    <FilterCard :summary="userFilterSummary" :active="userFiltersActive">
      <div v-if="branches.length > 1" class="filter-field">
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
    <div v-else-if="error" class="text-sm text-expense">{{ error }}</div>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="item in items" :key="item.id" class="p-4">
        <div class="flex flex-col gap-3 sm:flex-row sm:justify-between">
          <div class="min-w-0">
            <div class="text-sm font-semibold text-navy">{{ item.full_name }}</div>
            <div class="text-xs text-muted">@{{ item.username }} · {{ item.email }}</div>
            <div class="mt-2 flex flex-wrap gap-1.5">
              <span v-for="role in item.roles || []" :key="role.id || role.code" class="rounded-full bg-blue-50 px-2 py-1 text-[10px] font-medium text-primary">{{ role.name }}</span>
              <span v-if="!(item.roles || []).length" class="text-xs text-muted">No role assigned</span>
            </div>
            <p class="mt-2 text-xs text-muted">Branches: {{ (item.branches || []).map(branch => branch.code).join(', ') || '—' }}</p>
            <p v-if="(item.managers || []).length" class="mt-1 text-xs text-muted">Managers: <span class="font-medium text-navy">{{ (item.managers || []).map(manager => manager.full_name || manager.username).join(', ') }}</span></p>
            <div class="mt-2">
              <button type="button" class="text-[10px] font-semibold text-primary" @click="togglePerms(item.id)">{{ expandedPerms[item.id] ? 'Hide permissions' : `Permissions (${permCount(item)})` }}</button>
              <div v-if="expandedPerms[item.id]" class="mt-1.5 flex flex-wrap gap-1">
                <span v-if="item.is_super_admin" class="rounded border border-primary/20 bg-primary/10 px-1.5 py-0.5 text-[10px] text-primary">All permissions</span>
                <span v-for="code in (item.permission_codes || [])" :key="code" class="rounded border border-border bg-canvas px-1.5 py-0.5 font-mono text-[10px] text-navy">{{ code }}</span>
                <span v-if="!item.is_super_admin && !(item.permission_codes || []).length" class="text-[10px] text-muted">No permissions</span>
              </div>
            </div>
          </div>
          <div class="flex shrink-0 flex-row flex-wrap items-center gap-2 sm:flex-col sm:items-end">
            <span class="rounded-full px-2 py-1 text-[10px] font-semibold uppercase" :class="item.is_active ? 'bg-green-50 text-income' : 'bg-slate-100 text-muted'">{{ item.is_active ? 'Active' : 'Inactive' }}</span>
            <span v-if="item.is_super_admin" class="text-[10px] font-medium text-primary">Super Admin</span>
            <div class="flex flex-wrap justify-start gap-1.5 sm:justify-end">
              <button v-if="canEditUser && item.can_manage !== false" class="rounded-md border border-border px-2 py-1 text-[10px] font-medium text-navy hover:bg-canvas" @click="beginEdit(item)">Edit</button>
              <button v-if="canManageUsers && item.can_manage !== false && item.id !== currentUserId" class="rounded-md border border-border px-2 py-1 text-[10px] font-medium text-primary hover:bg-blue-50" @click="openPasswordChange(item)">Change PIN</button>
              <button v-if="canManageUsers && item.can_manage !== false && item.id !== currentUserId" class="rounded-md border px-2 py-1 text-[10px] font-medium" :class="item.is_active ? 'border-red-100 text-expense hover:bg-red-50' : 'border-green-100 text-income hover:bg-green-50'" :disabled="saving" @click="toggleActive(item)">{{ item.is_active ? 'Deactivate' : 'Activate' }}</button>
            </div>
          </div>
        </div>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">No users</li>
    </ul>

    <div v-if="passwordTarget" class="fixed inset-0 z-[100] flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-sm sm:items-center sm:p-4" @click.self="closePasswordChange">
      <section class="w-full max-w-md rounded-t-2xl border border-border bg-white p-5 shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" aria-labelledby="change-user-pin-title">
        <header class="mb-4 flex items-start justify-between gap-3">
          <div><h2 id="change-user-pin-title" class="text-base font-bold text-navy">Change {{ passwordTarget.full_name }}’s PIN</h2><p class="mt-1 text-xs text-muted">Enter a new 4-digit numeric PIN. Their active sessions will be revoked.</p></div>
          <button type="button" class="text-xs text-muted" :disabled="saving" @click="closePasswordChange">Close</button>
        </header>
        <form class="space-y-3" @submit.prevent="changeUserPassword">
          <div class="block space-y-1 text-xs font-medium text-muted"><span>New PIN</span><PinInput v-model="passwordForm.password" aria-label="New PIN" /></div>
          <p v-if="passwordError" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-expense" role="alert">{{ passwordError }}</p>
          <footer class="flex flex-col-reverse gap-2 border-t border-border pt-3 sm:flex-row sm:justify-end">
            <button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="closePasswordChange">Cancel</button>
            <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving || passwordForm.password.length !== 4">{{ saving ? 'Saving…' : 'Update PIN' }}</button>
          </footer>
        </form>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { RotateCcw } from 'lucide-vue-next'
import type { AdminUser, Branch, Role } from '~/types/api'

const { api } = useApi()
const { user: currentUser } = useAuth()
const { can, canCreateUser, hasFullAccess, permissions } = usePermissions()
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
const notice = ref('')
const editingUser = ref<AdminUser | null>(null)
const passwordTarget = ref<AdminUser | null>(null)
const passwordError = ref('')
const expandedPerms = ref<Record<string, boolean>>({})
const createForm = reactive({ username: '', email: '', full_name: '', password: '', role_code: '', branch_id: '', manager_ids: [] as string[] })
const editForm = reactive({ email: '', full_name: '', role_codes: [] as string[], branch_ids: [] as string[], manager_ids: [] as string[] })
const passwordForm = reactive({ password: '' })
// Branch-level managers can only hand out roles whose permissions they hold themselves (the API enforces this too).
const assignableRoles = computed(() => roles.value.filter((role) => {
  if (role.code === 'super_admin') return !!currentUser.value?.is_super_admin
  if (hasFullAccess.value || !role.permissions) return true
  return role.permissions.every(permission => permissions.value.includes(permission.code))
}))
const roleOptions = computed(() => assignableRoles.value.map(role => ({ value: role.code, label: role.name })))
const branchOptions = computed(() => branches.value.map(branch => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })))
const isEditingSelf = computed(() => editingUser.value?.id === currentUserId.value)

function togglePerms(userId: string) {
  expandedPerms.value[userId] = !expandedPerms.value[userId]
}

function permCount(item: AdminUser) {
  const codes = item.permission_codes ?? item.roles.flatMap((role) => role.permission_codes ?? [])
  return new Set(codes).size
}

// Branch and role filters
const branchFilter = ref('')
const roleFilter = ref('')
const filterBranchOptions = computed(() => [{ value: '', label: 'All branches' }, ...branchOptions.value])
const filterRoleOptions = computed(() => [{ value: '', label: 'All roles' }, ...roles.value.map(role => ({ value: role.code, label: role.name }))])
const userFiltersActive = computed(() => !!branchFilter.value || !!roleFilter.value)
const userFilterSummary = computed(() => {
  const branch = branches.value.find(item => item.id === branchFilter.value)
  const role = roles.value.find(item => item.code === roleFilter.value)
  return `${branch ? branch.code : 'All branches'} · ${role ? role.name : 'All roles'}`
})

function fetchUsers() {
  return api<{ items: AdminUser[] }>('/users', {
    query: { page_size: 100, branch_id: branchFilter.value || undefined, role: roleFilter.value || undefined },
  })
}

let userRequest = 0
watch([branchFilter, roleFilter], async () => {
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
      api<{ items: Role[] }>('/roles').catch(() => ({ items: [] })),
      api<{ items: Branch[] }>('/branches?page_size=100').catch(() => ({ items: [] })),
      // Unfiltered list for the Managers picker (the visible list may be filtered).
      api<{ items: AdminUser[] }>('/users', { query: { page_size: 100, is_active: 'true' } }).catch(() => ({ items: [] })),
    ])
    items.value = userResult.items || []
    roles.value = roleResult.items || []
    branches.value = branchResult.items || []
    allUsers.value = everyone.items || []
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load users.'
  } finally {
    loading.value = false
  }
}

function toggleCreate() {
  showCreate.value = !showCreate.value
  createError.value = ''
}

function openPasswordChange(item: AdminUser) {
  passwordTarget.value = item
  passwordForm.password = ''
  passwordError.value = ''
}

function closePasswordChange() {
  if (saving.value) return
  passwordTarget.value = null
  passwordError.value = ''
}

async function changeUserPassword() {
  if (!passwordTarget.value) return
  passwordError.value = ''
  if (!/^\d{4}$/.test(passwordForm.password)) {
    passwordError.value = 'The PIN must be exactly 4 digits.'
    return
  }
  saving.value = true
  try {
    await api(`/users/${passwordTarget.value.id}/password`, { method: 'POST', body: { password: passwordForm.password } })
    notice.value = `${passwordTarget.value.full_name}’s PIN was updated.`
    passwordTarget.value = null
    passwordForm.password = ''
  } catch (e: any) {
    passwordError.value = e?.data?.error || 'Could not change this user’s PIN.'
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
    .map(candidate => ({ value: candidate.id, label: `${candidate.full_name} (@${candidate.username})` }))
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
  if (!createForm.role_code || !createForm.branch_id) return
  saving.value = true
  createError.value = ''
  notice.value = ''
  try {
    await api('/users', { method: 'POST', body: {
      username: createForm.username.trim(), email: createForm.email.trim(), full_name: createForm.full_name.trim(),
      password: createForm.password, role_codes: [createForm.role_code], branch_ids: [createForm.branch_id],
      manager_ids: createForm.manager_ids,
    } })
    Object.assign(createForm, { username: '', email: '', full_name: '', password: '', role_code: '', branch_id: '', manager_ids: [] })
    showCreate.value = false
    notice.value = 'User created.'
    await load()
  } catch (e: any) {
    createError.value = e?.data?.error || 'Could not create this user.'
  } finally {
    saving.value = false
  }
}

function beginEdit(item: AdminUser) {
  editingUser.value = item
  Object.assign(editForm, {
    email: item.email,
    full_name: item.full_name,
    role_codes: (item.roles || []).map((role) => role.code),
    branch_ids: (item.branches || []).map((branch) => branch.id),
    manager_ids: (item.managers || []).map((manager) => manager.id),
  })
  editError.value = ''
  notice.value = ''
}

function cancelEdit() {
  editingUser.value = null
  editError.value = ''
}

async function saveEdit() {
  if (!editingUser.value) return
  if (canManageUsers.value && !isEditingSelf.value && (!editForm.role_codes.length || (branches.value.length > 0 && !editForm.branch_ids.length))) return
  const target = editingUser.value
  saving.value = true
  editError.value = ''
  notice.value = ''
  try {
    await api(`/users/${target.id}`, { method: 'PATCH', body: { email: editForm.email.trim(), full_name: editForm.full_name.trim() } })
    if (canManageUsers.value && target.id !== currentUserId.value) {
      await api(`/users/${target.id}/roles`, { method: 'POST', body: { role_codes: editForm.role_codes } })
      if (branches.value.length) await api(`/users/${target.id}/branches`, { method: 'POST', body: { branch_ids: editForm.branch_ids } })
      await api(`/users/${target.id}/managers`, { method: 'POST', body: { manager_ids: editForm.manager_ids } })
    }
    editingUser.value = null
    notice.value = 'User updated.'
    await load()
  } catch (e: any) {
    editError.value = e?.data?.error || 'Could not update this user.'
  } finally {
    saving.value = false
  }
}

async function toggleActive(item: AdminUser) {
  saving.value = true
  notice.value = ''
  error.value = ''
  try {
    await api(`/users/${item.id}/status`, { method: 'PATCH', body: { is_active: !item.is_active } })
    notice.value = `${item.full_name} ${item.is_active ? 'deactivated' : 'activated'}.`
    await load()
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not change user status.'
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>
