<template>
  <AppModal :model-value="modelValue" form :title="title" :eyebrow="eyebrow" :subtitle="subtitle" :busy="busy" @update:model-value="(open) => { if (!open) emit('update:modelValue', false) }" @submit="submit">
    <label class="block space-y-1.5 text-xs font-semibold text-navy">
      {{ label }}<span v-if="!required" class="font-normal text-muted"> (optional)</span>
      <textarea v-model.trim="text" rows="3" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm font-normal" :aria-invalid="error ? 'true' : undefined" />
    </label>
    <AppAlert v-if="error || serverError" class="mt-3" compact :message="error || serverError" />
    <template #footer>
      <button type="button" class="border border-border bg-white text-muted" :disabled="busy" @click="emit('update:modelValue', false)">Cancel</button>
      <button type="submit" class="text-white disabled:opacity-60" :class="danger ? 'bg-expense' : 'bg-primary'" :disabled="busy">{{ busy ? 'Saving…' : confirmLabel }}</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
/** Asks for a reason or note before a void, reject or approve; the parent runs the action. */
const props = withDefaults(defineProps<{
  modelValue: boolean
  title: string
  eyebrow?: string
  subtitle?: string
  label?: string
  confirmLabel?: string
  required?: boolean
  danger?: boolean
  busy?: boolean
  serverError?: string
}>(), { eyebrow: '', subtitle: '', label: 'Reason', confirmLabel: 'Confirm', required: true, danger: false, busy: false, serverError: '' })
const emit = defineEmits<{ 'update:modelValue': [open: boolean]; confirm: [text: string] }>()
const text = ref('')
const error = ref('')
watch(() => props.modelValue, (open) => { if (open) { text.value = ''; error.value = '' } })

function submit() {
  // Same rule as the API: a reason of at least 3 characters when one is needed.
  error.value = props.required && text.value.length < 3 ? `${props.label} must be at least 3 characters` : ''
  if (!error.value) emit('confirm', text.value)
}
</script>
