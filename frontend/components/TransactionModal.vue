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
      <form v-else class="transaction-form" @submit.prevent="submit">
        <div class="form-body">
          <div class="amount-card">
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
              <MoneyInput v-model="form.amount" :currency="form.currency" aria-label="Transaction amount" required />
            </div>
            <small v-if="amountAtLimit" class="field-hint">Maximum is {{ form.currency === 'USD' ? '$999,999,999.99' : '៛999,999,999,999' }}.</small>
            <small v-if="mode === 'edit'" class="field-hint">The currency can't be changed after a transaction is created.</small>
          </div>

          <div class="form-grid">
            <div class="form-field" :class="{ 'full-field': mode === 'edit' && transactionType === 'income' }">
              <span>Date</span>
              <DatePicker v-model="form.transaction_date" aria-label="Transaction date" />
            </div>

            <div v-if="mode === 'create'" class="form-field">
              <span>Branch</span>
              <SearchSelect v-model="form.branch_id" :options="branchOptions" placeholder="Select a branch" search-placeholder="Search branches…" aria-label="Branch" />
              <small v-if="!branches.length" class="field-hint">No accessible branches are available.</small>
            </div>

            <div v-if="transactionType === 'expense'" class="form-field" :class="{ 'full-field': mode === 'create' }">
              <span>Category</span>
              <SearchSelect v-model="form.category_id" :options="categoryOptions" placeholder="Select a category" search-placeholder="Search categories…" aria-label="Expense category" :disabled="!categories.length" />
              <small v-if="!categories.length && !categoryLoading" class="field-hint">There are no expense categories available.</small>
            </div>

            <label class="form-field full-field">
              <span>Customer or supplier <small>Optional</small></span>
              <input v-model.trim="form.counterparty" maxlength="255" placeholder="Name of payer or payee" />
            </label>

            <label class="form-field full-field">
              <span>Description <small>Optional</small></span>
              <textarea v-model.trim="form.description" rows="2" maxlength="1024" placeholder="What is this transaction for?" />
            </label>
          </div>

          <label v-if="mode === 'create'" class="pending-option">
            <span><strong>Submit for approval</strong><small>This transaction stays pending until approved.</small></span>
            <input v-model="form.pending" type="checkbox" role="switch" class="pending-switch" />
          </label>

          <p v-if="error" class="form-error" role="alert">{{ error }}</p>
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
import { ArrowDownLeft, ArrowUpRight, X } from 'lucide-vue-next'
import type { Branch, Category, Transaction } from '~/types/api'

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
const { canViewBranches } = usePermissions()
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
  counterparty: '',
  pending: false,
})
const loadingOptions = ref(true)
const categoryLoading = ref(false)
const saving = ref(false)
const error = ref('')
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
const submitDisabled = computed(() => saving.value
  || (props.mode === 'create' && (!branches.value.length || !form.branch_id))
  || (transactionType.value === 'expense' && (!categories.value.length || !form.category_id)))

function localDate() {
  const now = new Date()
  return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`
}

function close() {
  if (!saving.value) emit('update:modelValue', false)
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
  form.counterparty = txn?.counterparty || ''
  form.pending = false
  error.value = ''
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
  saving.value = true
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
          counterparty: form.counterparty,
        },
      })
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
          counterparty: form.counterparty,
          pending: form.pending,
        },
      })
    }
    emit('update:modelValue', false)
    emit('saved', saved)
  } catch (e: any) {
    error.value = e?.data?.error || (props.mode === 'edit' ? 'Could not update this transaction.' : 'Could not save this transaction.')
  } finally {
    saving.value = false
  }
}

watch(() => [props.modelValue, props.mode, props.type, props.transaction?.id], ([isOpen]) => {
  if (isOpen) prepareModal()
})
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
.form-body { overflow-y: auto; padding: 20px 24px 6px; }

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

.pending-option { display: flex; align-items: center; justify-content: space-between; gap: 14px; margin-top: 18px; border: 1px solid #e5ebf4; border-radius: 12px; padding: 13px 14px; background: #f8fafd; cursor: pointer; }
.pending-option > span { display: flex; flex-direction: column; gap: 3px; }
.pending-option strong { color: #2c4468; font-size: 13px; font-weight: 700; }
.pending-option small { color: #7b8ca5; font-size: 12px; }
.pending-switch { position: relative; flex: 0 0 auto; width: 42px; height: 24px; margin: 0; border-radius: 999px; background: #cfd9e6; cursor: pointer; appearance: none; transition: background .15s ease; }
.pending-switch::before { content: ''; position: absolute; top: 3px; left: 3px; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgb(16 40 80 / 25%); transition: transform .15s ease; }
.pending-switch:checked { background: #1769e8; }
.pending-switch:checked::before { transform: translateX(18px); }
.pending-switch:focus-visible { outline: 0; box-shadow: 0 0 0 3px rgb(37 99 235 / 20%); }

.form-error { margin: 14px 0 0; border-radius: 10px; padding: 10px 12px; color: #b9384c; background: #fff0f2; font-size: 13px; }
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
  .form-grid { grid-template-columns: minmax(0, 1fr); gap: 15px; }
  .full-field { grid-column: auto; }
  .form-field input, .form-field textarea, .form-field :deep(.search-select-trigger) { min-height: 48px; font-size: 15px; }
  .amount-input input { font-size: 21px; }
  .modal-actions { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr); padding: 12px 18px calc(12px + env(safe-area-inset-bottom)); box-shadow: 0 -6px 16px rgb(20 48 86 / 6%); }
  .modal-actions button { min-width: 0; min-height: 48px; }
}
@keyframes modal-sheet-up { from { transform: translateY(100%); } }
</style>
