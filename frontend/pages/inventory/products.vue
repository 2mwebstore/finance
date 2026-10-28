<template>
  <section class="space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Products</h1>
        <p class="mt-1 text-xs text-muted">The products every branch buys and sells. Stock shown is for {{ branchFilter ? labelForBranch(branchFilter) : 'all your branches' }}.</p>
      </div>
      <div v-if="canManage" class="flex flex-wrap gap-2">
        <button type="button" class="inline-flex items-center gap-1.5 rounded-lg border border-border bg-white px-3 py-2 text-sm font-medium text-navy" @click="categoriesOpen = true"><Tags :size="16" /> Categories</button>
        <button type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> Add product</button>
      </div>
    </header>

    <div class="flex flex-col gap-2 sm:flex-row sm:items-center">
      <div class="min-w-0 flex-1"><ListSearch v-model="search" placeholder="Search by SKU, name or barcode…" /></div>
      <div class="grid grid-cols-2 gap-2" :class="showBranchFilter ? 'sm:w-[540px] sm:grid-cols-3' : 'sm:w-[360px]'">
        <SearchSelect v-if="showBranchFilter" v-model="branchFilter" :options="branchOptions" aria-label="Branch stock" />
        <SearchSelect v-model="categoryFilter" :options="categoryFilterOptions" aria-label="Category" />
        <SearchSelect v-model="activeFilter" :options="activeOptions" :searchable="false" aria-label="Status" />
      </div>
    </div>

    <div v-if="loading" class="text-sm text-muted">Loading products…</div>
    <AppAlert v-else-if="error" title="Couldn’t load products" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <!-- Under 768px: a 2-column grid of product cards. -->
    <div v-else-if="isPhone" class="keep-cols grid grid-cols-2 gap-2.5">
      <article v-for="product in items" :key="product.id" class="relative flex flex-col overflow-hidden rounded-xl border border-border bg-white" :class="!product.is_active ? 'opacity-70' : ''">
        <button v-if="product.has_image || product.image_url" type="button" class="block w-full" :aria-label="`Show the picture of ${product.name}`" @click="previewing = product"><InventoryProductImage :product="product" size="card" /></button>
        <InventoryProductImage v-else :product="product" size="card" />
        <span v-if="!product.is_active" class="absolute left-1.5 top-1.5 rounded-full bg-slate-700/85 px-2 py-0.5 text-[9px] font-bold uppercase text-white">Inactive</span>
        <span v-else-if="isLow(product)" class="absolute left-1.5 top-1.5 rounded-full bg-amber-500/90 px-2 py-0.5 text-[9px] font-bold uppercase text-white">Low stock</span>
        <span v-else-if="product.type === 'service'" class="absolute left-1.5 top-1.5 rounded-full bg-violet-600/90 px-2 py-0.5 text-[9px] font-bold uppercase text-white">Service</span>
        <button v-if="canManage" type="button" class="absolute right-1.5 top-1.5 grid h-8 w-8 place-items-center rounded-lg bg-white/90 text-primary shadow-sm" :aria-label="`Edit ${product.name}`" @click="openEdit(product)"><Pencil :size="14" /></button>
        <div class="flex flex-1 flex-col gap-0.5 p-2.5">
          <h2 class="line-clamp-2 text-[13px] font-semibold leading-snug text-navy">{{ product.name }}</h2>
          <p class="truncate font-mono text-[10px] text-muted">{{ product.sku }}</p>
          <p class="truncate text-[11px] text-muted">{{ product.category_name || 'No category' }}</p>
          <div class="mt-auto flex items-end justify-between gap-1 pt-1.5">
            <div class="min-w-0">
              <p class="truncate text-sm font-bold text-navy">{{ formatMoney(product.sale_price, 'USD') }}</p>
              <p v-if="altCurrency" class="truncate text-[10px] text-muted">{{ formatMoney(fromUSD(product.sale_price, altCurrency), altCurrency) }}</p>
            </div>
            <p v-if="product.type === 'service'" class="shrink-0 text-right text-[11px] font-semibold leading-tight text-violet-700">No<br><span class="font-normal">stock</span></p>
            <p v-else class="shrink-0 text-right text-[11px] font-semibold leading-tight" :class="isLow(product) ? 'text-amber-700' : 'text-muted'">{{ formatQty(product.quantity) }}<br><span class="font-normal">{{ product.unit }}</span></p>
          </div>
        </div>
      </article>
      <p v-if="!items.length" class="col-span-2 rounded-xl border border-border bg-white p-6 text-center text-sm text-muted">{{ search ? `No products match “${search}”.` : 'No products yet.' }}</p>
    </div>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="product in items" :key="product.id" class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
        <div class="flex min-w-0 flex-1 items-start gap-3">
          <button v-if="product.has_image || product.image_url" type="button" class="shrink-0 rounded-xl transition hover:ring-2 hover:ring-primary/40" :aria-label="`Show the picture of ${product.name}`" @click="previewing = product"><InventoryProductImage :product="product" /></button>
          <InventoryProductImage v-else :product="product" />
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="text-sm font-semibold text-navy">{{ product.name }}</h2>
              <span v-if="!product.is_active" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Inactive</span>
              <span v-else-if="isLow(product)" class="rounded-full bg-amber-50 px-2 py-0.5 text-[10px] font-bold uppercase text-amber-700">Low stock</span>
              <span v-if="product.type === 'service'" class="rounded-full bg-violet-50 px-2 py-0.5 text-[10px] font-bold uppercase text-violet-700">Service</span>
            </div>
            <p class="mt-0.5 font-mono text-[11px] text-muted">{{ product.sku }}<template v-if="product.barcode"> · {{ product.barcode }}</template></p>
            <p class="mt-0.5 text-xs text-muted">{{ product.category_name || 'No category' }} · per {{ product.unit }}</p>
          </div>
        </div>
        <div class="flex items-center justify-between gap-4 sm:justify-end">
          <div class="text-left sm:text-right">
            <p class="text-sm font-bold text-navy">{{ formatMoney(product.sale_price, 'USD') }}</p>
            <p class="text-[11px] text-muted"><template v-if="altCurrency">{{ formatMoney(fromUSD(product.sale_price, altCurrency), altCurrency) }}</template><template v-if="altCurrency && product.type !== 'service'"> · </template><template v-if="product.type !== 'service'">cost {{ formatMoney(product.cost_price, 'USD') }}</template></p>
          </div>
          <div v-if="product.type === 'service'" class="text-right">
            <p class="text-sm font-bold text-muted">—</p>
            <p class="text-[11px] text-muted">no stock</p>
          </div>
          <div v-else class="text-right">
            <p class="text-sm font-bold" :class="isLow(product) ? 'text-amber-700' : 'text-navy'">{{ formatQty(product.quantity) }}</p>
            <p class="text-[11px] text-muted">in stock</p>
          </div>
          <button v-if="canManage" type="button" class="action-chip action-edit" @click="openEdit(product)"><Pencil :size="13" /> Edit</button>
        </div>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">{{ search ? `No products match “${search}”.` : 'No products yet.' }}</li>
    </ul>
    <InventoryPager v-model:page="page" :page-size="pageSize" :total="total" />

    <AppModal :model-value="formOpen" form :title="editing ? `Edit ${editing.name}` : 'Add product'" :eyebrow="editing ? 'EDIT PRODUCT' : 'NEW PRODUCT'" subtitle="Shared by every branch; each branch keeps its own stock." :busy="saving" width="640px" @update:model-value="(open) => { if (!open) closeForm() }" @submit="save">
      <template #icon><Package :size="20" /></template>
      <div class="mb-3 flex items-start gap-3 rounded-xl border border-border bg-canvas p-3">
        <InventoryProductImage :product="previewProduct" size="lg" />
        <div class="min-w-0 flex-1 space-y-2">
          <div class="flex flex-wrap gap-1 text-[11px]" role="tablist" aria-label="Picture">
            <button v-for="m in pictureModes" :key="m.value" type="button" class="rounded-full border px-2.5 py-1 font-semibold" :class="pictureMode === m.value ? 'border-primary bg-white text-primary' : 'border-border text-muted'" @click="pictureMode = m.value">{{ m.label }}</button>
          </div>
          <div v-if="pictureMode === 'upload'" class="space-y-1">
            <label class="inline-flex cursor-pointer items-center gap-1.5 rounded-lg border border-dashed border-primary/50 bg-white px-3 py-2 text-xs font-semibold text-primary"><ImagePlus :size="14" /> {{ pendingFile ? pendingFile.name : 'Choose a picture' }}<input type="file" accept="image/jpeg,image/png,image/webp,image/gif" class="sr-only" @change="pickFile" /></label>
            <p class="text-[11px] text-muted">JPG, PNG, WebP or GIF, up to 5 MB. Stored with your files (R2).</p>
          </div>
          <input v-else-if="pictureMode === 'url'" v-model.trim="form.image_url" type="url" maxlength="1024" :aria-invalid="formCheck.invalid('image_url')" class="w-full rounded-lg border border-border bg-white px-3 py-2 text-sm text-navy" placeholder="https://…/picture.jpg" />
          <p v-else class="text-[11px] text-muted">{{ editing && (editing.has_image || editing.image_url) ? 'The picture is removed when you save.' : 'No picture.' }}</p>
        </div>
      </div>
      <div class="mb-3 space-y-1 text-xs font-medium text-muted">
        Type
        <div class="keep-cols grid grid-cols-2 gap-2" role="radiogroup" aria-label="Product type">
          <button v-for="t in productTypes" :key="t.value" type="button" role="radio" :aria-checked="form.type === t.value" class="rounded-lg border px-3 py-2 text-left" :class="form.type === t.value ? 'border-primary bg-blue-50 ring-1 ring-primary' : 'border-border bg-white'" @click="form.type = t.value">
            <span class="block text-sm font-semibold text-navy">{{ t.label }}</span>
            <span class="block text-[11px] font-normal text-muted">{{ t.hint }}</span>
          </button>
        </div>
        <small v-if="editing" class="block font-normal">The type can’t change once the product has been bought, sold or stocked.</small>
      </div>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="space-y-1 text-xs font-medium text-muted">SKU<p class="rounded-lg border border-dashed border-border bg-canvas px-3 py-2 font-mono text-sm text-navy">{{ editing ? editing.sku : 'Generated when you save' }}</p></div>
        <label class="space-y-1 text-xs font-medium text-muted">Name<input v-model.trim="form.name" maxlength="191" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. Coffee beans 250 g" /></label>
        <div class="space-y-1 text-xs font-medium text-muted">Category<SearchSelect v-model="form.category_id" :options="categoryOptions" placeholder="Choose a category" aria-label="Category" :aria-invalid="formCheck.invalid('category_id')" /></div>
        <label class="space-y-1 text-xs font-medium text-muted">Unit<input v-model.trim="form.unit" maxlength="32" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="pcs, kg, box…" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Barcode <span class="font-normal">(optional)</span><input v-model.trim="form.barcode" maxlength="64" class="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm text-navy" /></label>
        <label class="space-y-1 text-xs font-medium text-muted">Sale price (USD)<MoneyInput v-model="form.sale_price" currency="USD" :aria-invalid="formCheck.invalid('sale_price')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" aria-label="Sale price in USD" /><small v-if="Number(form.sale_price) && altCurrency" class="block font-normal">≈ {{ formatMoney(fromUSD(form.sale_price, altCurrency), altCurrency) }}</small></label>
        <label v-if="form.type === 'standard'" class="space-y-1 text-xs font-medium text-muted">Usual cost (USD)<MoneyInput v-model="form.cost_price" currency="USD" :aria-invalid="formCheck.invalid('cost_price')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" aria-label="Usual cost in USD" /><small v-if="Number(form.cost_price) && altCurrency" class="block font-normal">≈ {{ formatMoney(fromUSD(form.cost_price, altCurrency), altCurrency) }}</small></label>
        <p class="rounded-lg bg-canvas px-3 py-2 text-[11px] text-muted sm:col-span-2"><template v-if="form.type === 'service'">A service is only sold: it has no stock and no cost. </template>Prices are in USD. A sale or purchase in another currency converts them at the rate set in Manage → Currency.</p>
        <label v-if="form.type === 'standard'" class="space-y-1 text-xs font-medium text-muted">Low-stock level<input v-model.trim="form.reorder_level" type="text" inputmode="decimal" :aria-invalid="formCheck.invalid('reorder_level')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="0 = no alert" /></label>
        <label v-if="editing" class="flex items-center gap-2 self-end pb-2 text-xs font-medium text-navy"><input v-model="form.is_active" type="checkbox" class="accent-primary" /> Active (can be bought and sold)</label>
        <label class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Description <span class="font-normal">(optional)</span><textarea v-model.trim="form.description" rows="2" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" /></label>
      </div>
      <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save this product' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="closeForm">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : editing ? 'Save changes' : 'Add product' }}</button>
      </template>
    </AppModal>

    <AppModal :model-value="!!previewing" :title="previewing?.name || ''" eyebrow="PRODUCT PICTURE" :subtitle="previewing ? `${previewing.sku} · ${formatMoney(previewing.sale_price, 'USD')}` : ''" width="640px" @update:model-value="(open) => { if (!open) previewing = null }">
      <template #icon><ImageIcon :size="20" /></template>
      <InventoryProductImage v-if="previewing" :product="previewing" size="xl" />
      <template #footer>
        <a v-if="previewing?.image_url" :href="previewing.image_url" target="_blank" rel="noopener noreferrer" class="border border-border bg-white text-muted">Open original</a>
        <button v-if="canManage && previewing" type="button" class="border border-border bg-white text-primary" @click="openEdit(previewing); previewing = null">Change picture</button>
        <button type="button" class="bg-primary text-white" @click="previewing = null">Close</button>
      </template>
    </AppModal>

    <AppModal v-model="categoriesOpen" title="Product categories" eyebrow="INVENTORY" subtitle="Group products for filters and reports." width="560px">
      <template #icon><Tags :size="20" /></template>
      <form class="grid grid-cols-1 gap-2 rounded-xl border border-border bg-canvas p-3 sm:grid-cols-[1fr_1fr_auto]" novalidate @submit.prevent="saveCategory">
        <input v-model.trim="categoryForm.code" maxlength="64" placeholder="Code (e.g. drinks)" :aria-invalid="categoryCheck.invalid('code')" class="rounded-lg border border-border px-3 py-2 font-mono text-sm" />
        <input v-model.trim="categoryForm.name" maxlength="128" placeholder="Name (e.g. Drinks)" :aria-invalid="categoryCheck.invalid('name')" class="rounded-lg border border-border px-3 py-2 text-sm" />
        <div class="flex gap-2">
          <button type="submit" class="rounded-lg bg-primary px-3 py-2 text-xs font-semibold text-white disabled:opacity-60" :disabled="savingCategory">{{ categoryForm.id ? 'Save' : 'Add' }}</button>
          <button v-if="categoryForm.id" type="button" class="rounded-lg border border-border bg-white px-3 py-2 text-xs font-semibold text-muted" @click="resetCategory">Cancel</button>
        </div>
      </form>
      <AppAlert v-if="categoryError || categoryCheck.messages.length" class="mt-2" compact :message="categoryError" :items="categoryCheck.messages" />
      <ul class="mt-3 divide-y divide-border rounded-xl border border-border">
        <li v-for="cat in allCategories" :key="cat.id" class="flex items-center justify-between gap-2 px-3 py-2">
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold text-navy">{{ cat.name }} <span v-if="!cat.is_active" class="ml-1 rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Inactive</span></p>
            <p class="font-mono text-[11px] text-muted">{{ cat.code }}</p>
          </div>
          <div class="flex shrink-0 gap-1.5">
            <button type="button" class="action-chip action-edit" @click="editCategory(cat)"><Pencil :size="12" /> Edit</button>
            <button type="button" class="action-chip" :class="cat.is_active ? 'action-danger' : 'action-success'" :disabled="savingCategory" @click="toggleCategory(cat)">{{ cat.is_active ? 'Hide' : 'Show' }}</button>
          </div>
        </li>
        <li v-if="!allCategories.length" class="p-4 text-center text-xs text-muted">No categories yet.</li>
      </ul>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { ImageIcon, ImagePlus, Package, Pencil, Plus, Tags } from 'lucide-vue-next'
import { useMediaQuery } from '@vueuse/core'
import type { Paged, Product, ProductCategory, ProductType } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { rule } from '~/utils/validation'
import { formatMoney } from '~/utils/money'
import { formatQty } from '~/utils/inventory'
import { useCurrency } from '~/composables/useCurrency'
import { useBranchOptions } from '~/composables/useBranchOptions'

const { api } = useApi()
const { can } = usePermissions()
const toast = useToast()
const canManage = computed(() => can('inventory.manage'))
// The product whose picture is shown large.
const previewing = ref<Product | null>(null)
// Phones and small tablets (under 768px) get a 2-column card grid instead of rows.
const isPhone = useMediaQuery('(max-width: 767px)')
const { fromUSD, load: loadCurrency, defaultCurrency, active: activeCurrencies } = useCurrency()
// The other currency to show prices in: the default one, else the first other active currency.
const altCurrency = computed(() => defaultCurrency.value !== 'USD' ? defaultCurrency.value : activeCurrencies.value.find(c => !c.base)?.code || '')

const items = ref<Product[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(true)
const error = ref('')
const search = ref('')
const categoryFilter = ref('')
const activeFilter = ref('true')
// Stock of one branch, or the total of the user's branches.
const branchFilter = ref('')
const { options: branchOptions, showFilter: showBranchFilter, labelFor: labelForBranch } = useBranchOptions('All my branches')
const allCategories = ref<ProductCategory[]>([])

const activeOptions = [{ value: 'true', label: 'Active' }, { value: 'false', label: 'Inactive' }, { value: '', label: 'All' }]
const categoryFilterOptions = computed(() => [{ value: '', label: 'All categories' }, ...allCategories.value.map(c => ({ value: c.id, label: c.name }))])
const categoryOptions = computed(() => allCategories.value.filter(c => c.is_active || c.id === form.category_id).map(c => ({ value: c.id, label: c.name })))
const productTypes = [
  { value: 'standard', label: 'Standard', hint: 'Stocked; has a cost and a sale price' },
  { value: 'service', label: 'Service', hint: 'Only sold; a sale price, no stock' },
] as const

// Low when the shown stock (one branch, or all of them) is at or under the product's low-stock level.
function isLow(product: Product) {
  return product.type !== 'service' && Number(product.reorder_level) > 0 && Number(product.quantity) <= Number(product.reorder_level)
}

let loadRequest = 0
async function load() {
  const request = ++loadRequest
  loading.value = true
  error.value = ''
  try {
    const result = await api<Paged<Product>>('/inventory/products', {
      query: { page: page.value, page_size: pageSize, search: search.value || undefined, category_id: categoryFilter.value || undefined, active: activeFilter.value || undefined, branch_id: branchFilter.value || undefined },
    })
    if (request !== loadRequest) return
    items.value = result.items || []
    total.value = result.total || 0
  } catch (e: any) {
    if (request === loadRequest) error.value = e?.data?.error || 'Could not load products.'
  } finally {
    if (request === loadRequest) loading.value = false
  }
}

async function loadCategories() {
  try {
    allCategories.value = (await api<{ items: ProductCategory[] }>('/inventory/categories', { query: { include_inactive: canManage.value ? 'true' : undefined } })).items || []
  } catch {
    allCategories.value = []
  }
}

watch([search, categoryFilter, activeFilter, branchFilter], () => { page.value = 1; load() })
watch(page, load)
onMounted(() => { load(); loadCategories(); loadCurrency() })

// --- product form ---
const formOpen = ref(false)
const editing = ref<Product | null>(null)
const saving = ref(false)
const formError = ref('')
const emptyForm = () => ({ type: 'standard' as ProductType, name: '', category_id: '', unit: 'pcs', barcode: '', sale_price: '', cost_price: '', reorder_level: '', description: '', image_url: '', is_active: true })
const form = reactive(emptyForm())

// Picture: an uploaded file (stored in R2) or a web address; "none" removes it.
const pictureModes = [{ value: 'upload', label: 'Upload' }, { value: 'url', label: 'Web address' }, { value: 'none', label: 'No picture' }] as const
const pictureMode = ref<'upload' | 'url' | 'none'>('upload')
const pendingFile = ref<File | null>(null)
const pendingPreview = ref('')
const previewProduct = computed(() => {
  if (pictureMode.value === 'none') return { id: 'preview', name: form.name }
  if (pictureMode.value === 'url') return { id: 'preview', name: form.name, image_url: /^https?:\/\//.test(form.image_url) ? form.image_url : '' }
  if (pendingPreview.value) return { id: 'preview', name: form.name, image_url: pendingPreview.value }
  return editing.value && !editing.value.image_url ? editing.value : { id: 'preview', name: form.name }
})

function pickFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 5 * 1024 * 1024) { formError.value = 'The picture is larger than 5 MB.'; return }
  if (!['image/jpeg', 'image/png', 'image/webp', 'image/gif'].includes(file.type)) { formError.value = 'Use a JPG, PNG, WebP or GIF picture.'; return }
  formError.value = ''
  if (pendingPreview.value) URL.revokeObjectURL(pendingPreview.value)
  pendingFile.value = file
  pendingPreview.value = URL.createObjectURL(file)
}

function resetPicture(product: Product | null) {
  if (pendingPreview.value) URL.revokeObjectURL(pendingPreview.value)
  pendingFile.value = null
  pendingPreview.value = ''
  pictureMode.value = product?.image_url ? 'url' : 'upload'
}

// Same rules as the API (services.ProductInput / SaveProduct).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 191)],
    category_id: [rule.required('category')],
    // Every product needs a sale price; a standard (stocked) product also needs its usual cost.
    sale_price: [rule.required('sale price'), rule.positiveAmount('sale price')],
    cost_price: [(value: unknown) => form.type === 'service' ? '' : rule.required('usual cost')(value) || rule.positiveAmount('usual cost')(value)],
    image_url: [(value: unknown) => pictureMode.value !== 'url' || !value || /^https?:\/\/[^\s/]+/.test(String(value)) ? '' : 'enter a picture address starting with https://'],
    reorder_level: [(value: unknown) => value === '' || (Number(value) >= 0 && /^\d+(\.\d{1,4})?$/.test(String(value))) ? '' : 'low-stock level must be 0 or more'],
  }),
))

function openCreate() {
  editing.value = null
  Object.assign(form, emptyForm())
  resetPicture(null)
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

function openEdit(product: Product) {
  editing.value = product
  resetPicture(product)
  Object.assign(form, {
    image_url: product.image_url || '', type: product.type || 'standard', name: product.name, category_id: product.category_id || '', unit: product.unit, barcode: product.barcode || '',
    sale_price: String(Number(product.sale_price)), cost_price: String(Number(product.cost_price)),
    reorder_level: Number(product.reorder_level) ? String(Number(product.reorder_level)) : '', description: product.description || '', is_active: product.is_active,
  })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

function closeForm() {
  if (!saving.value) formOpen.value = false
}

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const body = { ...form, image_url: pictureMode.value === 'url' ? form.image_url : '', is_active: editing.value ? form.is_active : true }
    const saved = editing.value
      ? await api<Product>(`/inventory/products/${editing.value.id}`, { method: 'PATCH', body })
      : await api<Product>('/inventory/products', { method: 'POST', body })
    // The picture: upload the chosen file, or remove the old one.
    const hadPicture = !!(editing.value?.has_image || editing.value?.image_url)
    if (pictureMode.value === 'upload' && pendingFile.value) {
      const upload = new FormData()
      upload.append('file', pendingFile.value)
      try {
        await api(`/inventory/products/${saved.id}/image`, { method: 'POST', body: upload })
      } catch (e: any) {
        toast.error(e?.data?.error || 'The picture could not be uploaded.', { title: 'Product saved without its picture' })
      }
    } else if (pictureMode.value === 'none' && hadPicture) {
      await api(`/inventory/products/${saved.id}/image`, { method: 'DELETE' })
    }
    formOpen.value = false
    toast.success(`${form.name} is saved.`, { title: editing.value ? 'Product updated' : 'Product added' })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this product.'
  } finally {
    saving.value = false
  }
}

// --- categories ---
const categoriesOpen = ref(false)
const savingCategory = ref(false)
const categoryError = ref('')
const categoryForm = reactive({ id: '', code: '', name: '', is_active: true })
const categoryCheck = reactive(useFormErrors(
  () => ({ ...categoryForm }),
  () => ({
    code: [rule.required('code'), rule.minLength('code', 2), rule.maxLength('code', 64)],
    name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 128)],
  }),
))

function resetCategory() {
  Object.assign(categoryForm, { id: '', code: '', name: '', is_active: true })
  categoryCheck.reset()
  categoryError.value = ''
}

function editCategory(cat: ProductCategory) {
  Object.assign(categoryForm, { id: cat.id, code: cat.code, name: cat.name, is_active: cat.is_active })
  categoryCheck.reset()
}

async function writeCategory(id: string, body: Record<string, unknown>) {
  savingCategory.value = true
  categoryError.value = ''
  try {
    if (id) await api(`/inventory/categories/${id}`, { method: 'PATCH', body })
    else await api('/inventory/categories', { method: 'POST', body })
    await loadCategories()
    return true
  } catch (e: any) {
    if (!categoryCheck.fromApi(e)) categoryError.value = e?.data?.error || 'Could not save this category.'
    return false
  } finally {
    savingCategory.value = false
  }
}

async function saveCategory() {
  if (!categoryCheck.check()) return
  if (await writeCategory(categoryForm.id, { code: categoryForm.code, name: categoryForm.name, is_active: categoryForm.is_active })) resetCategory()
}

function toggleCategory(cat: ProductCategory) {
  writeCategory(cat.id, { code: cat.code, name: cat.name, description: cat.description || '', is_active: !cat.is_active })
}

watch(categoriesOpen, (open) => { if (!open) resetCategory() })
</script>
