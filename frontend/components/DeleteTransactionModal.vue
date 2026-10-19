<template>
  <div v-if="modelValue && transaction" class="delete-backdrop" @click.self="close">
    <section class="delete-dialog" role="alertdialog" aria-modal="true" aria-labelledby="delete-title">
      <div class="delete-icon"><Trash2 :size="21" /></div>
      <button type="button" class="close-button" aria-label="Close dialog" @click="close"><X :size="18" /></button>
      <p class="delete-eyebrow">REMOVE TRANSACTION</p>
      <h2 id="delete-title">Delete this transaction?</h2>
      <p class="delete-copy">
        <strong>{{ transaction.transaction_number }}</strong> will be voided and removed from active financial totals. Its record stays in the audit history.
      </p>

      <label class="reason-field">
        <span>Reason <small>Optional</small></span>
        <textarea v-model.trim="reason" rows="2" maxlength="512" placeholder="Add a short reason" />
      </label>

      <p v-if="error" class="delete-error" role="alert">{{ error }}</p>
      <div class="delete-actions">
        <button type="button" class="cancel-button" :disabled="deleting" @click="close">Cancel</button>
        <button type="button" class="confirm-button" :disabled="deleting" @click="confirmDelete">
          <Trash2 :size="15" />{{ deleting ? 'Deleting…' : 'Delete transaction' }}
        </button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { Trash2, X } from 'lucide-vue-next'
import type { Transaction } from '~/types/api'

const props = defineProps<{
  modelValue: boolean
  transaction: Transaction | null
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  deleted: [transaction: Transaction]
}>()

const { api } = useApi()
const reason = ref('')
const error = ref('')
const deleting = ref(false)

watch(() => [props.modelValue, props.transaction?.id], ([isOpen]) => {
  if (isOpen) {
    reason.value = ''
    error.value = ''
  }
})

function close() {
  if (!deleting.value) emit('update:modelValue', false)
}

async function confirmDelete() {
  if (!props.transaction) return
  deleting.value = true
  error.value = ''
  try {
    const updated = await api<Transaction>(`/transactions/${props.transaction.id}/void`, {
      method: 'POST',
      body: { reason: reason.value },
    })
    emit('update:modelValue', false)
    emit('deleted', updated)
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not delete this transaction.'
  } finally {
    deleting.value = false
  }
}
</script>

<style scoped>
.delete-backdrop { position: fixed; inset: 0; z-index: 110; display: flex; align-items: center; justify-content: center; padding: 18px; background: rgb(8 24 48 / 56%); backdrop-filter: blur(4px); }
.delete-dialog { position: relative; width: min(420px, 100%); border: 1px solid #e5eaf2; border-radius: 15px; padding: 23px; background: #fff; box-shadow: 0 24px 72px rgb(8 24 48 / 28%); }
.delete-icon { display: flex; align-items: center; justify-content: center; width: 43px; height: 43px; margin-bottom: 16px; border-radius: 12px; color: #d83d52; background: #fff0f2; }
.close-button { position: absolute; top: 16px; right: 16px; display: flex; align-items: center; justify-content: center; width: 31px; height: 31px; border: 0; border-radius: 8px; color: #8492a7; background: #f5f7fa; cursor: pointer; }
.delete-eyebrow { margin: 0 0 6px; color: #9a6b73; font-size: 9px; font-weight: 800; letter-spacing: .12em; }
.delete-dialog h2 { margin: 0; color: #1a2f50; font-size: 17px; font-weight: 750; }
.delete-copy { margin: 9px 0 18px; color: #71819a; font-size: 11px; line-height: 1.65; }
.delete-copy strong { color: #334c70; }
.reason-field { display: flex; flex-direction: column; gap: 6px; color: #3a5275; font-size: 10px; font-weight: 700; }
.reason-field span small { margin-left: 4px; color: #93a0b2; font-size: 9px; font-weight: 450; }
.reason-field textarea { width: 100%; resize: vertical; border: 1px solid #e0e7f0; border-radius: 8px; outline: none; padding: 9px 10px; color: #344d70; font: inherit; font-size: 10px; font-weight: 450; }
.reason-field textarea:focus { border-color: #97bafa; box-shadow: 0 0 0 3px rgb(37 99 235 / 9%); }
.delete-error { margin: 12px 0 0; border-radius: 8px; padding: 9px 10px; color: #b33d50; background: #fff0f2; font-size: 10px; }
.delete-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; padding-top: 14px; border-top: 1px solid #edf1f7; }
.delete-actions button { display: inline-flex; align-items: center; justify-content: center; gap: 7px; min-height: 36px; border-radius: 8px; padding: 0 12px; font-size: 10px; font-weight: 700; cursor: pointer; }
.cancel-button { border: 1px solid #dfe6ef; color: #657993; background: #fff; }
.confirm-button { border: 0; color: #fff; background: #dc3f55; }
.confirm-button:hover { background: #c93449; }
.delete-actions button:disabled { opacity: .6; cursor: wait; }
@media (max-width: 480px) { .delete-dialog { padding: 20px; } }
</style>
