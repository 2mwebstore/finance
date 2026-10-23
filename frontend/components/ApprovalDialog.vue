<template>
  <div v-if="modelValue && transaction" class="approval-backdrop" @click.self="close">
    <section class="approval-dialog" :class="isApprove ? 'is-approve' : 'is-reject'" role="alertdialog" aria-modal="true" aria-labelledby="approval-title">
      <span class="sheet-handle" aria-hidden="true"></span>
      <div class="approval-icon"><CheckCircle2 v-if="isApprove" :size="22" /><XCircle v-else :size="22" /></div>
      <h2 id="approval-title">{{ isApprove ? 'Approve this transaction?' : 'Reject this transaction?' }}</h2>
      <p class="approval-summary">
        <strong>{{ transaction.transaction_number }}</strong> · {{ formatMoney(transaction.amount, transaction.currency) }} {{ transaction.type }}
        <template v-if="transaction.created_by?.name"> · by {{ transaction.created_by.name }}</template>
      </p>
      <p class="approval-copy">{{ isApprove ? 'It will count toward totals and reports.' : 'It stays in the history but never counts toward totals.' }}</p>
      <div class="approval-files">
        <span class="approval-files-label">Attachments</span>
        <div v-if="files.length" class="approval-file-list">
          <button v-for="(file, i) in files" :key="file.id" type="button" class="approval-file" :title="`Preview ${file.name}`" @click="emit('preview', i)">
            <ImageIcon v-if="file.type.startsWith('image/')" :size="15" /><FileText v-else :size="15" />
            <span>{{ file.name }}</span>
            <Eye :size="14" class="approval-file-eye" />
          </button>
        </div>
        <p v-else class="approval-files-empty">No files attached.</p>
        <AppAlert v-if="needsReceipt" type="warning" compact message="Attach a receipt (image or PDF) before approving this expense." />
        <button v-if="canAddFile && files.length < MAX_ATTACHMENTS" type="button" class="approval-add-file" :disabled="uploading || saving" @click="fileInput?.click()">
          <Paperclip :size="15" /> {{ uploading ? 'Uploading…' : files.length ? 'Add another file' : 'Attach a receipt' }}
        </button>
        <input ref="fileInput" type="file" class="sr-only" :accept="ACCEPTED_TYPES.join(',')" tabindex="-1" aria-hidden="true" @change="onFile" />
      </div>
      <label class="approval-note">
        <span>{{ isApprove ? 'Note' : 'Reason' }} <small>Optional</small></span>
        <textarea v-model.trim="notes" rows="2" maxlength="500" :placeholder="isApprove ? 'Anything to add?' : 'Why is it rejected?'" />
      </label>
      <AppAlert v-if="error" class="approval-error" compact :message="error" />
      <div class="approval-actions">
        <button type="button" class="cancel-button" :disabled="saving" @click="close">Cancel</button>
        <button type="button" class="confirm-button" :disabled="saving || uploading || needsReceipt" @click="confirm">
          <Check v-if="isApprove" :size="16" /><X v-else :size="16" />
          {{ saving ? (isApprove ? 'Approving…' : 'Rejecting…') : (isApprove ? 'Approve' : 'Reject') }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { Check, CheckCircle2, Eye, FileText, ImageIcon, Paperclip, X, XCircle } from 'lucide-vue-next'
import { MAX_ATTACHMENTS, type Transaction } from '~/types/api'
import { formatMoney } from '~/utils/money'

const props = defineProps<{
  modelValue: boolean
  transaction: Transaction | null
  action: 'approve' | 'reject'
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  done: [transaction: Transaction]
  /** A file was attached here; the updated transaction. */
  filesChanged: [transaction: Transaction]
  /** Open the file viewer at this file. */
  preview: [index: number]
}>()

const { api } = useApi()
const { canEditIncome, canEditExpense, canCreateIncome, canCreateExpense } = usePermissions()
const notes = ref('')
const error = ref('')
const saving = ref(false)
const uploading = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)
const ACCEPTED_TYPES = ['image/jpeg', 'image/png', 'image/gif', 'image/webp', 'application/pdf']
const MAX_FILE_BYTES = 10 * 1024 * 1024
const isApprove = computed(() => props.action === 'approve')
const files = computed(() => props.transaction?.attachments || [])
// The API only approves an expense that has its receipt.
const needsReceipt = computed(() => isApprove.value && props.transaction?.type === 'expense' && !files.value.length)
// Same rule as the API for a pending transaction: create or edit rights for its type.
const canAddFile = computed(() => {
  const type = props.transaction?.type
  if (!type) return false
  return type === 'income' ? canCreateIncome.value || canEditIncome.value : canCreateExpense.value || canEditExpense.value
})

async function onFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !props.transaction) return
  error.value = ''
  if (!ACCEPTED_TYPES.includes(file.type)) {
    error.value = 'Choose an image (JPG, PNG, GIF, WebP) or a PDF.'
    return
  }
  if (file.size > MAX_FILE_BYTES) {
    error.value = 'The file is larger than 10 MB.'
    return
  }
  uploading.value = true
  try {
    const body = new FormData()
    body.append('file', file)
    const updated = await api<Transaction>(`/transactions/${props.transaction.id}/attachments`, { method: 'POST', body })
    emit('filesChanged', updated)
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not attach the file.'
  } finally {
    uploading.value = false
  }
}

watch(() => [props.modelValue, props.transaction?.id, props.action], ([open]) => {
  if (open) {
    notes.value = ''
    error.value = ''
  }
})

function close() {
  if (!saving.value && !uploading.value) emit('update:modelValue', false)
}

async function confirm() {
  if (!props.transaction) return
  saving.value = true
  error.value = ''
  try {
    const updated = await api<Transaction>(`/transactions/${props.transaction.id}/${props.action}`, {
      method: 'POST',
      body: { notes: notes.value },
    })
    emit('update:modelValue', false)
    emit('done', updated)
  } catch (e: any) {
    error.value = e?.data?.error || `Could not ${props.action} this transaction.`
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.approval-backdrop { position: fixed; inset: 0; z-index: 110; display: flex; align-items: center; justify-content: center; padding: 18px; background: rgb(8 24 48 / 56%); backdrop-filter: blur(4px); }
.approval-dialog { --accent: #13966e; --accent-soft: #e6f8f0; position: relative; width: min(430px, 100%); border: 1px solid #e5eaf2; border-radius: 18px; padding: 24px; background: #fff; box-shadow: 0 24px 72px rgb(8 24 48 / 28%); }
.approval-dialog.is-reject { --accent: #dc3f55; --accent-soft: #fff0f2; }
.sheet-handle { display: none; }
.approval-icon { display: grid; width: 46px; height: 46px; margin-bottom: 14px; place-items: center; border-radius: 13px; color: var(--accent); background: var(--accent-soft); }
.approval-dialog h2 { margin: 0; color: #172b4e; font-size: 19px; font-weight: 750; letter-spacing: -.02em; }
.approval-summary { margin: 8px 0 0; color: #50668a; font-size: 13px; }
.approval-summary strong { color: #1d365c; }
.approval-copy { margin: 4px 0 16px; color: #7b8ca5; font-size: 13px; }
.approval-files { display: flex; flex-direction: column; gap: 8px; margin-bottom: 16px; }
.approval-files-label { color: #324b70; font-size: 13px; font-weight: 650; }
.approval-file-list { display: flex; flex-direction: column; gap: 6px; }
.approval-file { display: flex; min-width: 0; align-items: center; gap: 8px; border: 1px solid #dfe7f1; border-radius: 10px; padding: 9px 11px; color: #3d5a82; background: #f8fbff; font-size: 13px; font-weight: 600; text-align: left; cursor: pointer; }
.approval-file span { min-width: 0; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.approval-file:hover { border-color: #b9d2f7; background: #eef5ff; }
.approval-file-eye { flex: 0 0 auto; color: #1769e8; }
.approval-files-empty { margin: 0; color: #8193ab; font-size: 13px; }
.approval-add-file { display: inline-flex; align-self: flex-start; align-items: center; gap: 6px; border: 1px dashed #b9cbe4; border-radius: 9px; padding: 8px 12px; color: #1769e8; background: #fff; font-size: 13px; font-weight: 700; cursor: pointer; }
.approval-add-file:hover:not(:disabled) { background: #f2f7ff; }
.approval-add-file:disabled { opacity: .6; cursor: wait; }
.approval-note { display: flex; flex-direction: column; gap: 6px; color: #324b70; font-size: 13px; font-weight: 650; }
.approval-note small { margin-left: 4px; color: #94a2b7; font-size: 11px; font-weight: 500; }
.approval-note textarea { width: 100%; min-height: 70px; resize: vertical; border: 1px solid #dfe7f1; border-radius: 10px; outline: none; padding: 10px 12px; color: #263f65; font: inherit; font-size: 14px; font-weight: 450; }
.approval-note textarea:focus { border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.approval-error { margin-top: 12px; }
.approval-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; padding-top: 16px; border-top: 1px solid #edf1f7; }
.approval-actions button { display: inline-flex; min-height: 44px; align-items: center; justify-content: center; gap: 7px; border-radius: 10px; padding: 0 18px; font-size: 14px; font-weight: 700; cursor: pointer; }
.cancel-button { border: 1px solid #dfe6ef; color: #536984; background: #fff; }
.confirm-button { border: 0; color: #fff; background: var(--accent); }
.confirm-button:hover:not(:disabled) { filter: brightness(.95); }
.approval-actions button:disabled { opacity: .55; cursor: not-allowed; }
@media (max-width: 540px) {
  .approval-backdrop { align-items: flex-end; padding: 0; }
  .approval-dialog { width: 100%; border: 0; border-radius: 22px 22px 0 0; padding: 22px 18px calc(16px + env(safe-area-inset-bottom)); }
  .sheet-handle { position: absolute; top: 8px; left: 50%; display: block; width: 38px; height: 4px; border-radius: 999px; background: #d6dfeb; transform: translateX(-50%); }
  .approval-actions { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr); }
}
</style>
