<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">{{ copy.eyebrow }}</p>
        <h1 class="text-xl font-bold text-navy">{{ copy.title }}</h1>
        <p class="mt-1 text-xs text-muted">{{ copy.intro }} {{ hasFullAccess ? 'Shared categories appear in every branch; branch categories only in their branch.' : 'Add categories for your branch. Shared categories are managed by the owner or an admin.' }}</p>
      </div>
      <button v-if="canManage" class="inline-flex items-center gap-2 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate">
        <Plus :size="16" /> Add category
      </button>
    </header>

    <ListSearch v-model="search" placeholder="Search by name, code or description…" />
    <FilterCard :summary="categoryFilterSummary" :active="categoryFiltersActive">
      <div class="filter-field">
        <span>Applies to</span>
        <SearchSelect v-model="scopeFilter" :options="scopeFilterOptions" search-placeholder="Search branches…" aria-label="Filter categories by branch" />
      </div>
      <div class="filter-field">
        <span>Status</span>
        <SearchSelect v-model="statusFilter" :options="statusOptions" :searchable="false" aria-label="Filter categories by status" />
      </div>
      <button type="button" class="filter-reset" :disabled="!categoryFiltersActive" @click="scopeFilter = ''; statusFilter = ''"><RotateCcw :size="15" /> Reset</button>
    </FilterCard>

    <div v-if="loading" class="rounded-xl border border-border bg-white p-6 text-center text-sm text-muted">Loading categories…</div>
    <div v-else-if="error" class="flex items-center justify-between gap-3 rounded-xl border border-red-100 bg-white p-4 text-sm text-expense">
      <span>{{ error }}</span><button class="font-semibold text-primary" @click="load">Try again</button>
    </div>
    <div v-else class="overflow-hidden rounded-xl border border-border bg-white">
      <article v-for="category in visibleCategories" :key="category.id" class="flex items-start justify-between gap-4 border-b border-border p-4 last:border-0">
        <div class="min-w-0">
          <h2 class="truncate text-sm font-semibold text-navy">{{ category.name }}</h2>
          <div class="mt-1.5 flex flex-wrap items-center gap-1.5">
            <span class="rounded-full px-2 py-0.5 text-[9px] font-bold uppercase" :class="category.is_active ? 'bg-green-50 text-income' : 'bg-slate-100 text-muted'">{{ category.is_active ? 'Active' : 'Inactive' }}</span>
            <span v-if="category.is_system" class="rounded-full bg-blue-50 px-2 py-0.5 text-[9px] font-bold uppercase text-primary">System</span>
            <span class="rounded-full px-2 py-0.5 text-[9px] font-bold uppercase" :class="category.branch_id ? 'bg-amber-50 text-amber-700' : 'bg-indigo-50 text-indigo-600'">{{ category.branch_id ? (category.branch_name || 'Branch') : 'Shared' }}</span>
          </div>
          <p class="mt-1.5 font-mono text-[10px] text-muted">{{ category.code }} · {{ category.type }}</p>
          <p v-if="category.description" class="mt-1 text-xs text-muted">{{ category.description }}</p>
        </div>
        <button v-if="canEdit(category)" class="action-chip action-edit shrink-0" @click="openEdit(category)"><Pencil :size="13" /> Edit</button>
      </article>
      <div v-if="!categories.length && !search" class="p-8 text-center text-sm text-muted">No categories yet. Add a category to make it available on {{ copy.forms }}.</div>
      <div v-else-if="!visibleCategories.length" class="p-8 text-center text-sm text-muted">{{ search ? `No categories match “${search}”.` : 'No categories match these filters.' }}</div>
    </div>

    <div v-if="modalOpen" class="fixed inset-0 z-[90] flex items-end justify-center bg-slate-950/50 p-0 backdrop-blur-sm sm:items-center sm:p-5" @click.self="close">
      <section class="w-full max-w-lg rounded-t-2xl border border-border bg-white shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" :aria-label="formMode === 'create' ? 'Add category' : 'Edit category'">
        <header class="flex items-center justify-between border-b border-border px-5 py-4">
          <div><p class="text-[9px] font-extrabold tracking-[0.14em] text-slate-400">{{ formMode === 'create' ? 'NEW CATEGORY' : 'UPDATE CATEGORY' }}</p><h2 class="mt-1 text-base font-bold text-navy">{{ formMode === 'create' ? 'Add category' : 'Edit category' }}</h2></div>
          <button class="grid h-8 w-8 place-items-center rounded-lg bg-canvas text-muted hover:text-navy" aria-label="Close" @click="close"><X :size="17" /></button>
        </header>
        <form class="space-y-3 p-5" novalidate @submit.prevent="save">
          <label class="block space-y-1.5 text-xs font-semibold text-navy">
            <span>Name</span><input v-model.trim="form.name" required minlength="2" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2.5 font-normal" placeholder="e.g. Office supplies" />
          </label>
          <label class="block space-y-1.5 text-xs font-semibold text-navy">
            <span>Code</span><input v-model.trim="form.code" required minlength="2" maxlength="64" :aria-invalid="formCheck.invalid('code')" class="w-full rounded-lg border border-border px-3 py-2.5 font-mono font-normal" placeholder="e.g. office-supplies" />
            <small class="block font-normal text-muted">{{ formMode === 'create' ? 'Generated from the name; you can adjust it before saving.' : 'Changing the code keeps existing transactions linked to this category.' }}</small>
          </label>
          <div class="block space-y-1.5 text-xs font-semibold text-navy">
            <span>Applies to</span>
            <SearchSelect v-if="formMode === 'create'" v-model="form.branch_id" :options="scopeOptions" search-placeholder="Search branches…" aria-label="Category applies to" />
            <p v-else class="rounded-lg border border-border bg-canvas px-3 py-2.5 font-normal text-muted">{{ editing?.branch_id ? (editing.branch_name || 'One branch') : 'Shared — every branch' }}</p>
          </div>
          <label class="block space-y-1.5 text-xs font-semibold text-navy">
            <span>Description <small class="font-normal text-muted">Optional</small></span><textarea v-model.trim="form.description" rows="2" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2.5 font-normal" placeholder="Add a short description" />
          </label>
          <label v-if="formMode === 'edit'" class="flex items-center gap-2 rounded-lg border border-border bg-canvas px-3 py-2.5 text-xs text-navy">
            <input v-model="form.is_active" type="checkbox" class="accent-primary" /> Category is active and available on new forms
          </label>
          <AppAlert v-if="formError || formCheck.messages.length" :title="formError ? 'Couldn’t save this category' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
          <footer class="flex justify-end gap-2 border-t border-border pt-4">
            <button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted" :disabled="saving" @click="close">Cancel</button>
            <button type="submit" class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save category' }}</button>
          </footer>
        </form>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { Pencil, Plus, RotateCcw, X } from 'lucide-vue-next'
import type { Branch, Category } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { rule } from '~/utils/validation'
import { useFormErrors } from '~/composables/useFormErrors'

/**
 * Categories of one kind: financial (income/expense ledger) expense categories, or inventory
 * expense categories. They are separate lists on separate pages.
 */
const props = defineProps<{ type: 'expense' | 'inv_expense' }>()
const copy = computed(() => props.type === 'inv_expense'
  ? { eyebrow: 'INVENTORY', title: 'Expense categories', intro: 'Categories for inventory expenses, like shipping, packaging and storage.', forms: 'inventory expense forms' }
  : { eyebrow: 'FINANCIAL ACTIVITY', title: 'Categories', intro: 'Categories for expense transactions.', forms: 'expense forms' })

const { api } = useApi()
const { user } = useAuth()
const { can, hasFullAccess } = usePermissions()
const canManage = computed(() => can(props.type === 'inv_expense' ? 'inventory.manage' : 'categories.manage'))
const branches = ref<Branch[]>([])
// Shared categories belong to the owner/admin; branch managers only work in their branches.
const SHARED = ''
const scopeOptions = computed(() => [
  ...(hasFullAccess.value ? [{ value: SHARED, label: 'Shared — every branch' }] : []),
  ...branches.value.map(branch => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })),
])

// Filters (client-side: the list is already limited to what this user may see)
const SHARED_ONLY = 'shared'
const scopeFilter = ref('')
const statusFilter = ref('')
const scopeFilterOptions = computed(() => [
  { value: '', label: 'All categories' },
  { value: SHARED_ONLY, label: 'Shared only' },
  ...branches.value.map(branch => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })),
])
const statusOptions = [
  { value: '', label: 'Any status' },
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
]
const visibleCategories = computed(() => categories.value.filter((category) => {
  if (scopeFilter.value === SHARED_ONLY && category.branch_id) return false
  if (scopeFilter.value && scopeFilter.value !== SHARED_ONLY && category.branch_id !== scopeFilter.value) return false
  if (statusFilter.value === 'active' && !category.is_active) return false
  if (statusFilter.value === 'inactive' && category.is_active) return false
  return true
}))
const categoryFiltersActive = computed(() => !!scopeFilter.value || !!statusFilter.value)
const categoryFilterSummary = computed(() => {
  const scope = scopeFilterOptions.value.find(option => option.value === scopeFilter.value)?.label || 'All categories'
  const status = statusOptions.find(option => option.value === statusFilter.value)?.label || 'Any status'
  return `${scope.split(' — ')[0]} · ${status}`
})

function canEdit(category: Category) {
  return canManage.value && (hasFullAccess.value || !!category.branch_id)
}
const categories = ref<Category[]>([])
const loading = ref(true)
const error = ref('')
const modalOpen = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const editing = ref<Category | null>(null)
const saving = ref(false)
const formError = ref('')
const toast = useToast()
const codeEdited = ref(false)
const form = reactive({ name: '', code: '', description: '', is_active: true, branch_id: SHARED })
// Same rules as the API (CreateCategoryInput / UpdateCategoryInput).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 128)],
    code: [rule.required('code'), rule.minLength('code', 2), rule.maxLength('code', 64)],
  }),
))

watch(() => form.name, (name) => {
  if (formMode.value === 'create' && !codeEdited.value) form.code = slugify(name)
})
watch(() => form.code, (code) => {
  if (formMode.value === 'create' && code && code !== slugify(form.name)) codeEdited.value = true
})

function slugify(value: string) {
  return value.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

const search = ref('')
let loadRequest = 0

async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<{ items: Category[] }>('/categories', { query: { type: props.type, search: search.value || undefined } })
    if (request !== loadRequest) return
    // Shared first, then each branch's own categories.
    categories.value = (result.items || [])
      .filter((category) => category.type === props.type)
      .sort((a, b) => Number(!!a.branch_id) - Number(!!b.branch_id) || (a.branch_name || '').localeCompare(b.branch_name || ''))
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load categories.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}
watch(search, () => load())


function openCreate() {
  formMode.value = 'create'
  editing.value = null
  Object.assign(form, { name: '', code: '', description: '', is_active: true, branch_id: hasFullAccess.value ? SHARED : branches.value[0]?.id || SHARED })
  codeEdited.value = false
  formError.value = ''
  formCheck.reset()
  modalOpen.value = true
}

function openEdit(category: Category) {
  formMode.value = 'edit'
  editing.value = category
  Object.assign(form, { name: category.name, code: category.code, description: category.description || '', is_active: category.is_active })
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
      await api('/categories', { method: 'POST', body: { type: props.type, name: form.name, code: form.code, description: form.description, branch_id: form.branch_id || undefined } })
    } else if (editing.value) {
      await api(`/categories/${editing.value.id}`, { method: 'PATCH', body: { code: form.code, name: form.name, description: form.description, is_active: form.is_active } })
    }
    modalOpen.value = false
    toast.success(`${form.name} is available on ${copy.value.forms}.`, { title: formMode.value === 'create' ? 'Category created' : 'Category updated' })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this category.'
  } finally {
    saving.value = false
  }
}

async function loadBranches() {
  // The user's own branches; owners/admins load every branch.
  branches.value = [...(user.value?.scope_branches || user.value?.branches || [])]
  if (!hasFullAccess.value) return
  try {
    const result = await api<{ items: Branch[] }>('/branches', { query: { page_size: 100 } })
    branches.value = result.items || branches.value
  } catch {
    // The profile's scope branches remain as a fallback.
  }
}

onMounted(() => {
  void load()
  void loadBranches()
})
</script>
