<template>
  <div v-if="modelValue" class="modal-backdrop" @click.self="close">
    <section class="transaction-modal" :class="transactionType === 'income' ? 'is-income' : 'is-expense'" role="dialog" aria-modal="true" :aria-label="`${actionLabel} ${transactionType}`">
      <header class="modal-header">
        <span class="sheet-handle" aria-hidden="true"></span>
        <div class="modal-title-group">
          <span class="modal-icon" :class="transactionType === 'income' ? 'income-icon' : 'expense-icon'">
            <ArrowDownLeft v-if="transactionType === 'income'" :size="20" />
            <ArrowUpRight v-else :size="20" />
          </span>
          <div>
            <p class="modal-eyebrow">{{ mode === 'edit' ? 'UPDATE RECORD' : 'NEW RECORD' }}</p>
            <h2>{{ actionLabel }} {{ transactionType === 'income' ? 'Income' : 'Expense' }}</h2>
          </div>
        </div>
        <button type="button" class="icon-button" aria-label="Close dialog" @click="close"><X :size="19" /></button>
      </header>

      <div v-if="loadingOptions" class="modal-loading">Loading form options…</div>
      <form v-else class="transaction-form" novalidate @submit.prevent="submit">
        <div class="form-body">
          <div class="amount-card" :class="{ 'has-error': errors.amount }">
            <div class="amount-card-top">
              <span class="field-label">Amount</span>
              <div class="currency-toggle" role="radiogroup" aria-label="Currency">
                <button
                  v-for="option in currencyOptions"
                  :key="option.value"
                  type="button"
                  role="radio"
                  :aria-checked="form.currency === option.value"
                  :class="{ 'is-selected': form.currency === option.value }"
                  :disabled="mode === 'edit'"
                  @click="form.currency = option.value"
                >{{ option.value }}</button>
              </div>
            </div>
            <div class="amount-input">
              <span class="amount-symbol" aria-hidden="true">{{ currencySymbol }}</span>
              <MoneyInput v-model="form.amount" :currency="form.currency" aria-label="Transaction amount" required :aria-invalid="invalid('amount')" />
            </div>
            <small v-if="errors.amount" class="field-error">{{ errors.amount }}</small>
            <small v-if="amountAtLimit" class="field-hint">Maximum is {{ form.currency === 'USD' ? '$999,999,999.99' : '៛999,999,999,999' }}.</small>
            <small v-if="mode === 'edit'" class="field-hint">The currency can't be changed after a transaction is created.</small>
          </div>

          <div class="form-grid">
            <div class="form-field" :class="{ 'full-field': mode === 'edit' && transactionType === 'income', 'has-error': errors.transaction_date }">
              <span>Date <b class="required-mark" aria-hidden="true">*</b></span>
              <DatePicker v-model="form.transaction_date" aria-label="Transaction date" />
              <small v-if="errors.transaction_date" class="field-error">{{ errors.transaction_date }}</small>
            </div>

            <div v-if="mode === 'create'" class="form-field" :class="{ 'has-error': errors.branch_id }">
              <span>Branch <b class="required-mark" aria-hidden="true">*</b></span>
              <SearchSelect v-model="form.branch_id" :options="branchOptions" placeholder="Select a branch" search-placeholder="Search branches…" aria-label="Branch" />
              <small v-if="errors.branch_id" class="field-error">{{ errors.branch_id }}</small>
              <small v-else-if="!branches.length" class="field-hint">No accessible branches are available.</small>
            </div>

            <div v-if="transactionType === 'expense'" class="form-field" :class="{ 'full-field': mode === 'create', 'has-error': errors.category_id }">
              <span>Category <b class="required-mark" aria-hidden="true">*</b></span>
              <SearchSelect v-model="form.category_id" :options="categoryOptions" placeholder="Select a category" search-placeholder="Search categories…" aria-label="Expense category" :disabled="!categories.length" />
              <small v-if="errors.category_id" class="field-error">{{ errors.category_id }}</small>
              <small v-else-if="!categories.length && !categoryLoading" class="field-hint">There are no expense categories available.</small>
            </div>

            <div class="form-field full-field" :class="{ 'has-error': errors.description }">
              <span>Description <small>Optional</small></span>
              <RichText v-model="form.description" placeholder="What is this transaction for?" aria-label="Description" :max-length="DESCRIPTION_MAX" />
              <small v-if="errors.description" class="field-error">{{ errors.description }}</small>
            </div>

            <div class="form-field full-field">
              <span>Attachments <small>Optional · up to {{ MAX_ATTACHMENTS }} images or PDFs, 10 MB each</small></span>
              <div class="attachment-list">
                <div v-for="file in keptAttachments" :key="file.id" class="attachment-chip">
                  <span class="attachment-icon"><ImageIcon v-if="file.type.startsWith('image/')" :size="20" /><FileText v-else :size="20" /></span>
                  <span class="attachment-meta"><strong>{{ file.name }}</strong><small>{{ formatBytes(file.size) }} · attached</small></span>
                  <button type="button" class="attachment-remove" :aria-label="`Remove ${file.name}`" title="Remove" @click="removedIds.push(file.id)"><Trash2 :size="15" /></button>
                </div>
                <div v-for="item in picked" :key="item.key" class="attachment-chip">
                  <img v-if="item.previewUrl" :src="item.previewUrl" alt="" class="attachment-thumb" />
                  <span v-else class="attachment-icon"><FileText :size="20" /></span>
                  <span class="attachment-meta"><strong>{{ item.file.name }}</strong><small>{{ formatBytes(item.file.size) }} · {{ item.uploaded ? 'uploaded' : 'ready to upload' }}</small></span>
                  <button v-if="!item.uploaded" type="button" class="attachment-remove" :aria-label="`Remove ${item.file.name}`" title="Remove" @click="removePicked(item)"><X :size="16" /></button>
                </div>
                <button
                  v-if="freeSlots > 0"
                  type="button"
                  class="attachment-drop"
                  :class="{ 'is-dragging': dragging }"
                  @click="pickFile"
                  @dragover.prevent="dragging = true"
                  @dragleave="dragging = false"
                  @drop.prevent="onDrop"
                >
                  <span class="attachment-drop-icon"><Paperclip :size="18" /></span>
                  <span class="attachment-meta">
                    <strong>{{ keptAttachments.length || picked.length ? 'Add another file' : 'Attach a receipt or photo' }}</strong>
                    <small>{{ removedIds.length ? `${removedIds.length === 1 ? '1 file' : `${removedIds.length} files`} will be deleted when you save` : `Tap to choose, or drop ${freeSlots === 1 ? 'a file' : 'files'} here` }}</small>
                  </span>
                </button>
              </div>
              <input ref="fileInput" type="file" class="sr-only" multiple :accept="ACCEPTED_TYPES.join(',')" tabindex="-1" aria-hidden="true" @change="onFileChange" />
              <small v-if="attachmentError" class="field-error" role="alert">{{ attachmentError }}</small>
              <small v-else-if="transactionType === 'expense' && !keptAttachments.length && !picked.length" class="field-hint">An expense needs a receipt before a manager can approve it.</small>
            </div>
          </div>

          <label v-if="mode === 'create' && canApprove" class="pending-option">
            <span><strong>Submit for approval</strong><small>This transaction stays pending until approved.</small></span>
            <input v-model="form.pending" type="checkbox" role="switch" class="pending-switch" />
          </label>
          <!-- Without approve permission the API always saves it as pending. -->
          <div v-else-if="mode === 'create'" class="pending-option is-locked" role="note">
            <i class="pending-icon" aria-hidden="true"><Clock3 :size="17" :stroke-width="2.3" /></i>
            <span><strong>Sent for approval</strong><small>It stays pending until a manager approves it.</small></span>
          </div>
          <!-- The API sends an approved transaction back for approval when its money details change. -->
          <div v-else-if="transaction?.status === 'approved' && !canApprove" class="pending-option is-locked" role="note">
            <i class="pending-icon" aria-hidden="true"><Clock3 :size="17" :stroke-width="2.3" /></i>
            <span><strong>Needs approval again</strong><small>Changing the amount, date or category sends it back to a manager.</small></span>
          </div>
        </div>

        <!-- Pinned above the buttons so it is visible however far the form is scrolled. -->
        <div v-if="error || fieldMessages.length" class="form-alert">
          <AppAlert
            compact
            :type="partialSave ? 'warning' : 'error'"
            :title="alertTitle"
            :message="error"
            :items="fieldMessages"
            dismissible
            @dismiss="dismissAlert"
          />
        </div>
        <footer class="modal-actions">
          <button type="button" class="cancel-button" :disabled="saving" @click="close">Cancel</button>
          <button type="submit" class="save-button" :disabled="submitDisabled">
            {{ saving ? 'Saving…' : mode === 'edit' ? 'Save changes' : `Save ${transactionType}` }}
          </button>
        </footer>
      </form>
    </section>
  </div>
</template>

<script setup lang="ts">
import { ArrowDownLeft, ArrowUpRight, Clock3, FileText, ImageIcon, Paperclip, Trash2, X } from 'lucide-vue-next'
import { MAX_ATTACHMENTS, type Branch, type Category, type Transaction } from '~/types/api'
import { richTextToPlain } from '~/utils/richText'
import { rule } from '~/utils/validation'
import { useFormErrors } from '~/composables/useFormErrors'

type TransactionType = 'income' | 'expense'
type TransactionCurrency = 'USD' | 'KHR'

const props = withDefaults(defineProps<{
  modelValue: boolean
  mode?: 'create' | 'edit'
  type?: TransactionType
  transaction?: Transaction | null
}>(), {
  mode: 'create',
  type: 'income',
  transaction: null,
})
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  saved: [transaction: Transaction]
}>()

const { api } = useApi()
const { user } = useAuth()
const { canViewBranches, canApprove } = usePermissions()
const branches = ref<Branch[]>([])
const categories = ref<Category[]>([])
const form = reactive({
  type: 'income' as TransactionType,
  currency: 'USD' as TransactionCurrency,
  amount: '',
  transaction_date: localDate(),
  branch_id: '',
  category_id: '',
  description: '',
  pending: false,
})

const DESCRIPTION_MAX = 1000
// Up to MAX_ATTACHMENTS files, sent after the transaction itself is saved.
const MAX_FILE_BYTES = 10 * 1024 * 1024
const ACCEPTED_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'application/pdf']
const fileInput = ref<HTMLInputElement | null>(null)
const dragging = ref(false)
type PickedFile = { key: number; file: File; previewUrl: string; uploaded: boolean }
let pickedKey = 0
const picked = ref<PickedFile[]>([])
// Saved files the person removed; they are deleted when the form is saved.
const removedIds = ref<string[]>([])
// Removals already done (kept so a retry after a failed upload doesn't redo them).
const deletedIds = ref<string[]>([])
const attachmentError = ref('')
const keptAttachments = computed(() => props.mode === 'edit'
  ? (props.transaction?.attachments || []).filter(file => !removedIds.value.includes(file.id))
  : [])
const freeSlots = computed(() => MAX_ATTACHMENTS - keptAttachments.value.length - picked.value.length)
// Set once a new transaction is created, so a failed upload can be retried without creating it twice.
const createdTransaction = ref<Transaction | null>(null)
const loadingOptions = ref(true)
const categoryLoading = ref(false)
const saving = ref(false)
const error = ref('')
// The record was saved but its file upload failed.
const partialSave = ref(false)
const branchOptions = computed(() => branches.value.map(branch => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })))
const categoryOptions = computed(() => categories.value.map(category => ({
  value: category.id,
  label: `${category.name}${category.is_active ? '' : ' (inactive)'}`,
})))
const currencyOptions: { value: TransactionCurrency; symbol: string }[] = [
  { value: 'USD', symbol: '$' },
  { value: 'KHR', symbol: '៛' },
]
// Mirrors MoneyInput's digit limit so people see why extra digits are ignored.
const amountAtLimit = computed(() => (form.amount.split('.')[0] || '').length >= (form.currency === 'USD' ? 9 : 12))
const currencySymbol = computed(() => currencyOptions.find(option => option.value === form.currency)?.symbol || '')

const transactionType = computed<TransactionType>(() => props.mode === 'edit'
  ? props.transaction?.type || props.type
  : props.type)
const actionLabel = computed(() => props.mode === 'edit' ? 'Edit' : 'Add')
// The button stays enabled: pressing it explains what is missing instead of silently doing nothing.
const submitDisabled = computed(() => saving.value)

// Same required fields as the API (CreateTransactionInput / UpdateTransactionInput).
const { errors, messages: fieldMessages, check, fromApi, reset: resetErrors, invalid } = useFormErrors(
  () => ({
    amount: form.amount,
    transaction_date: form.transaction_date,
    branch_id: form.branch_id,
    category_id: form.category_id,
    description: richTextToPlain(form.description),
  }),
  () => ({
    amount: [rule.required('amount'), rule.positiveAmount('amount')],
    transaction_date: [rule.required('date')],
    ...(props.mode === 'create' ? { branch_id: [rule.required('branch')] } : {}),
    ...(transactionType.value === 'expense' ? { category_id: [rule.required('category')] } : {}),
    description: [rule.maxLength('description', DESCRIPTION_MAX)],
  }),
)
const alertTitle = computed(() => {
  if (partialSave.value) return 'Saved, but the files weren’t all updated'
  if (!error.value) return 'Check the highlighted fields'
  return props.mode === 'edit' ? 'Couldn’t update this transaction' : 'Couldn’t save this transaction'
})

function dismissAlert() {
  error.value = ''
  resetErrors()
}

function localDate() {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}

function close() {
  if (!saving.value) emit('update:modelValue', false)
}

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function pickFile() {
  fileInput.value?.click()
}

function addFiles(files: File[]) {
  attachmentError.value = ''
  for (const file of files) {
    if (freeSlots.value <= 0) {
      attachmentError.value = `A transaction can have up to ${MAX_ATTACHMENTS} files.`
      return
    }
    if (!ACCEPTED_TYPES.includes(file.type)) {
      attachmentError.value = `${file.name}: choose an image (JPG, PNG, GIF, WebP) or a PDF.`
      continue
    }
    if (file.size > MAX_FILE_BYTES) {
      attachmentError.value = `${file.name} is larger than 10 MB.`
      continue
    }
    picked.value.push({ key: ++pickedKey, file, previewUrl: file.type.startsWith('image/') ? URL.createObjectURL(file) : '', uploaded: false })
  }
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  addFiles(Array.from(input.files || []))
  input.value = ''
}

function onDrop(event: DragEvent) {
  dragging.value = false
  addFiles(Array.from(event.dataTransfer?.files || []))
}

function removePicked(item: PickedFile) {
  if (item.previewUrl) URL.revokeObjectURL(item.previewUrl)
  picked.value = picked.value.filter(other => other.key !== item.key)
  attachmentError.value = ''
}

function clearPicked() {
  for (const item of picked.value) {
    if (item.previewUrl) URL.revokeObjectURL(item.previewUrl)
  }
  picked.value = []
}

function resetAttachment() {
  clearPicked()
  removedIds.value = []
  deletedIds.value = []
  attachmentError.value = ''
  createdTransaction.value = null
}

function initializeForm() {
  const txn = props.mode === 'edit' ? props.transaction : null
  form.type = transactionType.value
  form.currency = txn?.currency || 'USD'
  form.amount = txn?.amount || ''
  form.transaction_date = txn?.transaction_date?.slice(0, 10) || localDate()
  form.branch_id = txn?.branch_id || user.value?.branches?.[0]?.id || ''
  form.category_id = txn?.category_id || ''
  form.description = txn?.description || ''
  form.pending = false
  error.value = ''
  resetErrors()
  resetAttachment()
}

async function loadCategories() {
  if (transactionType.value !== 'expense') {
    categories.value = []
    categoryLoading.value = false
    form.category_id = ''
    return
  }
  categoryLoading.value = true
  try {
    // Shared categories plus the ones that belong to this transaction's branch.
    const branchId = props.mode === 'edit' ? props.transaction?.branch_id : form.branch_id
    const result = await api<{ items: Category[] }>('/categories', {
      query: {
        type: transactionType.value,
        active: props.mode === 'edit' ? undefined : 'true',
        branch_id: branchId || undefined,
      },
    })
    categories.value = (result.items || []).filter((category) => category.type === transactionType.value)
    if (props.mode === 'create' && form.category_id && !categories.value.some((category) => category.id === form.category_id)) {
      form.category_id = ''
    }
  } catch (e: any) {
    categories.value = []
    error.value = e?.data?.error || 'Could not load categories.'
  } finally {
    categoryLoading.value = false
  }
}

async function loadBranches() {
  branches.value = user.value?.branches || []
  if (canViewBranches.value) {
    try {
      const result = await api<{ items: Branch[] }>('/branches', { query: { is_active: true, page_size: 100 } })
      branches.value = result.items || branches.value
    } catch {
      // The assigned branches from the user's profile remain available as a fallback.
    }
  }
  if (!form.branch_id && branches.value.length) form.branch_id = branches.value[0].id
}

async function prepareModal() {
  initializeForm()
  loadingOptions.value = true
  await Promise.all([loadCategories(), props.mode === 'create' ? loadBranches() : Promise.resolve()])
  loadingOptions.value = false
}

async function submit() {
  error.value = ''
  partialSave.value = false
  if (!createdTransaction.value && !check()) return
  saving.value = true
  let recordSaved = false
  try {
    let saved: Transaction
    if (props.mode === 'edit' && props.transaction) {
      saved = await api<Transaction>(`/transactions/${props.transaction.id}`, {
        method: 'PATCH',
        body: {
          amount: form.amount,
          transaction_date: form.transaction_date,
          ...(transactionType.value === 'expense' ? { category_id: form.category_id } : {}),
          description: form.description,
        },
      })
    } else if (createdTransaction.value) {
      saved = createdTransaction.value
    } else {
      saved = await api<Transaction>('/transactions', {
        method: 'POST',
        body: {
          type: transactionType.value,
          currency: form.currency,
          amount: form.amount,
          transaction_date: form.transaction_date,
          branch_id: form.branch_id,
          ...(transactionType.value === 'expense' ? { category_id: form.category_id } : {}),
          description: form.description,
          pending: form.pending || !canApprove.value,
        },
      })
      createdTransaction.value = saved
    }
    recordSaved = true

    // Delete removed files first so their slots are free for the new ones.
    for (const id of removedIds.value) {
      if (deletedIds.value.includes(id)) continue
      saved = await api<Transaction>(`/transactions/${saved.id}/attachments/${id}`, { method: 'DELETE' })
      deletedIds.value.push(id)
    }
    for (const item of picked.value) {
      if (item.uploaded) continue
      const body = new FormData()
      body.append('file', item.file)
      saved = await api<Transaction>(`/transactions/${saved.id}/attachments`, { method: 'POST', body })
      item.uploaded = true
    }

    resetAttachment()
    emit('update:modelValue', false)
    emit('saved', saved)
  } catch (e: any) {
    const reason = e?.data?.error || e?.message || ''
    partialSave.value = recordSaved
    if (recordSaved) error.value = `A file couldn't be saved${reason ? ` (${reason})` : ''}. Try again, or remove that file and save.`
    // When the API names the fields, the list says it all; otherwise show its message.
    else if (!fromApi(e)) error.value = reason || (props.mode === 'edit' ? 'Could not update this transaction.' : 'Could not save this transaction.')
  } finally {
    saving.value = false
  }
}

watch(() => [props.modelValue, props.mode, props.type, props.transaction?.id], ([isOpen]) => {
  if (isOpen) prepareModal()
  else resetAttachment()
})
onBeforeUnmount(clearPicked)
// Branch categories differ per branch, so refresh the list when another branch is picked.
watch(() => form.branch_id, (branchId, previous) => {
  if (props.modelValue && props.mode === 'create' && !loadingOptions.value && branchId && previous && branchId !== previous) void loadCategories()
})
onMounted(() => {
  if (props.modelValue) prepareModal()
})
</script>

<style scoped>
.modal-backdrop { position: fixed; inset: 0; z-index: 100; display: flex; align-items: center; justify-content: center; padding: 20px; background: rgb(8 24 48 / 55%); backdrop-filter: blur(4px); }
.transaction-modal { --accent: #13966e; --accent-soft: #effaf5; --accent-border: #cdeedf; --accent-ring: rgb(19 150 110 / 14%); display: flex; width: min(560px, 100%); max-height: min(92vh, 780px); flex-direction: column; overflow: hidden; border: 1px solid #e2eaf5; border-radius: 18px; background: #fff; box-shadow: 0 24px 75px rgb(8 24 48 / 28%); }
.transaction-modal.is-expense { --accent: #e24458; --accent-soft: #fff4f5; --accent-border: #f8d6dc; --accent-ring: rgb(226 68 88 / 13%); }
.sheet-handle { display: none; }
.modal-header { position: relative; display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 20px 24px 16px; border-bottom: 1px solid #edf1f7; }
.modal-title-group { display: flex; align-items: center; gap: 12px; }
.modal-icon { display: flex; align-items: center; justify-content: center; width: 44px; height: 44px; border-radius: 13px; }
.income-icon { color: #0d946b; background: #e2f8ef; }
.expense-icon { color: #dc3f55; background: #ffe8eb; }
.modal-eyebrow { margin: 0 0 4px; color: #8c9ab0; font-size: 10px; font-weight: 800; letter-spacing: .13em; }
.modal-header h2 { margin: 0; color: #172b4e; font-size: 19px; font-weight: 750; letter-spacing: -.02em; }
.icon-button { display: flex; align-items: center; justify-content: center; width: 36px; height: 36px; border: 0; border-radius: 10px; color: #73849d; background: #f3f6fb; cursor: pointer; }
.icon-button:hover { color: #1e3b66; background: #e9eff8; }
.transaction-form { display: flex; min-height: 0; flex: 1; flex-direction: column; }
.form-body { min-height: 0; flex: 1 1 auto; overflow-y: auto; padding: 20px 24px 6px; }

/* Amount: the main input, so it gets the most room. */
.amount-card { margin-bottom: 18px; border: 1px solid var(--accent-border); border-radius: 14px; padding: 14px 16px 12px; background: var(--accent-soft); transition: box-shadow .15s ease, border-color .15s ease; }
.amount-card:focus-within { border-color: var(--accent); box-shadow: 0 0 0 4px var(--accent-ring); }
.amount-card-top { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.field-label { color: #324b70; font-size: 13px; font-weight: 650; }
.currency-toggle { display: inline-flex; gap: 2px; border: 1px solid #e0e7f1; border-radius: 10px; padding: 3px; background: #fff; }
.currency-toggle button { min-width: 52px; border: 0; border-radius: 7px; padding: 6px 10px; color: #657993; background: transparent; font-size: 12px; font-weight: 750; letter-spacing: .02em; cursor: pointer; transition: background .15s ease, color .15s ease; }
.currency-toggle button.is-selected { color: #fff; background: var(--accent); box-shadow: 0 2px 6px rgb(16 40 80 / 14%); }
.currency-toggle button:disabled { cursor: not-allowed; }
.currency-toggle button:disabled:not(.is-selected) { opacity: .45; }
.amount-input { display: flex; align-items: center; gap: 6px; margin-top: 8px; }
.amount-symbol { flex: 0 0 auto; color: var(--accent); font-size: 20px; font-weight: 700; line-height: 1; }
.amount-input input { width: 100%; min-width: 0; border: 0; outline: 0; padding: 2px 0; color: #142a50; background: transparent; font-size: 22px; font-weight: 700; letter-spacing: -.01em; font-variant-numeric: tabular-nums; }
.amount-input input::placeholder { color: #b3c0d2; }
/* The card shows the focus ring; skip the global .page-content input:focus ring. */
.amount-input input:focus { box-shadow: none; }

.form-grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 16px 14px; }
.form-field { display: flex; min-width: 0; flex-direction: column; gap: 7px; color: #324b70; font-size: 13px; font-weight: 650; }
.form-field > span { display: flex; align-items: center; justify-content: space-between; }
.form-field > span small { color: #94a2b7; font-size: 11px; font-weight: 500; }
.form-field input, .form-field textarea { width: 100%; min-width: 0; min-height: 44px; border: 1px solid #dfe7f1; border-radius: 10px; outline: none; padding: 10px 12px; color: #263f65; background: #fff; font: inherit; font-size: 14px; font-weight: 450; transition: border-color .15s ease, box-shadow .15s ease; }
.form-field input::placeholder, .form-field textarea::placeholder { color: #9aa8bc; }
.form-field input:focus, .form-field textarea:focus { border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.form-field textarea { min-height: 76px; resize: vertical; }
.form-field :deep(.search-select-trigger) { min-height: 44px; border-radius: 10px; padding: 10px 12px; font-size: 14px; }
.form-field :deep(.search-select-option) { padding: 10px; font-size: 13px; }
.form-field :deep(.search-input-wrap input) { font-size: 13px; }
.full-field { grid-column: 1 / -1; }
.field-hint { display: block; margin-top: 2px; color: #8e9cb0; font-size: 11px; font-weight: 450; }
.amount-card .field-hint { margin-top: 6px; }

.attachment-list { display: flex; flex-direction: column; gap: 8px; }
.attachment-chip, .attachment-drop { display: flex; width: 100%; min-height: 58px; align-items: center; gap: 11px; border: 1px solid #dfe7f1; border-radius: 12px; padding: 9px 10px; background: #fff; text-align: left; }
.attachment-drop { border-style: dashed; border-color: #c9d6e8; color: #50668a; background: #fafcff; cursor: pointer; transition: border-color .15s ease, background .15s ease; }
.attachment-drop:hover, .attachment-drop.is-dragging { border-color: var(--accent); background: var(--accent-soft); }
.attachment-drop-icon, .attachment-icon { display: grid; flex: 0 0 auto; width: 40px; height: 40px; place-items: center; border-radius: 10px; color: var(--accent); background: var(--accent-soft); }
.attachment-thumb { flex: 0 0 auto; width: 40px; height: 40px; border-radius: 9px; object-fit: cover; background: #f1f5fb; }
.attachment-meta { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 3px; }
.attachment-meta strong { overflow: hidden; color: #1d365c; font-size: 13px; font-weight: 650; text-overflow: ellipsis; white-space: nowrap; }
.attachment-meta small { color: #7b8ca5; font-size: 12px; font-weight: 450; }
.attachment-remove { display: grid; flex: 0 0 auto; width: 32px; height: 32px; place-items: center; border: 0; border-radius: 8px; color: #8193ab; background: #f3f6fb; cursor: pointer; }
.attachment-remove:hover { color: #c93449; background: #fff0f2; }
.field-error { display: block; margin-top: 2px; color: #c33d50; font-size: 12px; font-weight: 550; }
.amount-card .field-error { margin-top: 6px; }
.required-mark { margin-left: 3px; color: #d94357; font-weight: 700; }
.form-field > span .required-mark { margin-right: auto; }
.amount-card.has-error { border-color: #ef8a98; background: #fff6f7; }
.amount-card.has-error:focus-within { border-color: #e2445c; box-shadow: 0 0 0 4px rgb(226 68 92 / 12%); }
.amount-card input[aria-invalid="true"] { border: 0 !important; box-shadow: none !important; }
.has-error :deep(.search-select-trigger), .has-error :deep(.date-trigger), .has-error :deep(.rich-text) { border-color: #e2445c; box-shadow: 0 0 0 3px rgb(226 68 92 / 10%); }
.pending-option { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-top: 18px; border: 1px solid #e5ebf4; border-radius: 12px; padding: 13px 14px; background: #f8fafd; cursor: pointer; }
.pending-option > span { display: flex; flex-direction: column; gap: 3px; }
.pending-option strong { color: #2c4468; font-size: 13px; font-weight: 700; }
.pending-option small { color: #7b8ca5; font-size: 12px; }
.pending-option.is-locked { justify-content: flex-start; gap: 12px; border-color: #f6dfb8; background: #fff8ec; cursor: default; }
.pending-option.is-locked strong { color: #8a5310; }
.pending-option.is-locked small { color: #9b7440; }
.pending-icon { display: grid; flex: 0 0 auto; width: 34px; height: 34px; place-items: center; border-radius: 10px; color: #b86e0c; background: #fdecd0; font-style: normal; }
.pending-icon svg { display: block; }
.pending-switch { position: relative; flex: 0 0 auto; width: 42px; height: 24px; margin: 0; border-radius: 999px; background: #cfd9e6; cursor: pointer; appearance: none; transition: background .15s ease; }
.pending-switch::before { content: ''; position: absolute; top: 3px; left: 3px; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgb(16 40 80 / 25%); transition: transform .15s ease; }
.pending-switch:checked { background: #1769e8; }
.pending-switch:checked::before { transform: translateX(18px); }
.pending-switch:focus-visible { outline: 0; box-shadow: 0 0 0 3px rgb(37 99 235 / 20%); }

.form-alert { flex: 0 0 auto; max-height: 30vh; overflow-y: auto; padding: 10px 24px 0; border-top: 1px solid #edf1f7; }
.form-alert :deep(.app-alert.is-compact) { align-items: flex-start; }
.form-alert + .modal-actions { border-top: 0; padding-top: 10px; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; padding: 14px 24px 18px; border-top: 1px solid #edf1f7; background: #fff; }
.modal-actions button { min-width: 120px; min-height: 44px; border-radius: 10px; padding: 10px 18px; font-size: 14px; font-weight: 700; cursor: pointer; }
.cancel-button { border: 1px solid #dfe7f1; color: #536984; background: #fff; }
.cancel-button:hover { background: #f5f8fc; }
.save-button { border: 0; color: #fff; background: var(--accent); box-shadow: 0 6px 14px var(--accent-ring); text-transform: capitalize; }
.save-button:hover:not(:disabled) { filter: brightness(.95); }
.save-button:disabled { box-shadow: none; opacity: .45; cursor: not-allowed; }
.cancel-button:disabled { opacity: .6; cursor: wait; }
.modal-loading { padding: 48px 20px; color: #8091a9; font-size: 13px; text-align: center; }

/* Phones: a bottom sheet with the buttons pinned under the scrolling form. */
@media (max-width: 540px) {
  .modal-backdrop { align-items: flex-end; padding: 0; }
  .transaction-modal { width: 100%; max-height: 94dvh; border: 0; border-radius: 22px 22px 0 0; animation: modal-sheet-up .22s ease-out; }
  .sheet-handle { position: absolute; top: 7px; left: 50%; display: block; width: 38px; height: 4px; border-radius: 999px; background: #d6dfeb; transform: translateX(-50%); }
  .modal-header { padding: 20px 18px 14px; }
  .modal-header h2 { font-size: 18px; }
  .form-body { padding: 16px 18px 6px; }
  .form-alert { padding: 10px 18px 0; }
  .form-grid { grid-template-columns: minmax(0, 1fr); gap: 15px; }
  .full-field { grid-column: auto; }
  .form-field input, .form-field textarea, .form-field :deep(.search-select-trigger) { min-height: 48px; font-size: 15px; }
  .amount-input input { font-size: 21px; }
  .modal-actions { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr); padding: 12px 18px calc(12px + env(safe-area-inset-bottom)); box-shadow: 0 -6px 16px rgb(20 48 86 / 6%); }
  .modal-actions button { min-width: 0; min-height: 48px; }
}
@keyframes modal-sheet-up { from { transform: translateY(100%); } }
</style>
