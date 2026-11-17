<template>
  <AppModal :model-value="modelValue" form :title="party ? `Edit ${party.name}` : `Add ${noun}`" :eyebrow="noun.toUpperCase()" :subtitle="subtitle" :busy="saving" width="480px" @update:model-value="(open) => { if (!open && !saving) emit('update:modelValue', false) }" @submit="save">
    <template #icon><Contact v-if="kind === 'customers'" :size="20" /><Factory v-else :size="20" /></template>
    <div class="space-y-3">
      <div class="space-y-1 text-xs font-medium text-muted">Who uses them<SearchSelect v-model="form.branch_id" :options="ownerOptions" :searchable="false" aria-label="Who uses them" :aria-invalid="formCheck.invalid('branch_id')" /><small class="block font-normal">{{ form.branch_id ? `Only this branch sees them and picks them on its ${docs}.` : `Every branch sees them and picks them on its ${docs}.` }}<template v-if="!hasFullAccess"> Only the owner or an admin can share someone with every branch.</template></small></div>
      <label class="block space-y-1 text-xs font-medium text-muted">Name<input ref="nameEl" v-model="form.name" maxlength="191" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" :placeholder="kind === 'customers' ? 'Customer or company name' : 'Supplier or company name'" /></label>
      <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
        <label class="block space-y-1 text-xs font-medium text-muted">Phone <span class="font-normal">(optional)</span><input v-model="form.phone" type="tel" maxlength="64" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="012 345 678" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Email <span class="font-normal">(optional)</span><input v-model="form.email" type="email" maxlength="191" :aria-invalid="formCheck.invalid('email')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="name@example.com" /></label>
      </div>
      <label class="block space-y-1 text-xs font-medium text-muted">Address <span class="font-normal">(optional)</span><textarea v-model="form.address" rows="2" maxlength="512" class="w-full resize-y rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Street, city"></textarea></label>
      <label class="block space-y-1 text-xs font-medium text-muted">Note <span class="font-normal">(optional)</span><input v-model="form.note" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" :placeholder="kind === 'customers' ? 'e.g. pays by ABA, VIP' : 'e.g. delivers on Mondays'" /></label>
    </div>
    <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
    <template #footer>
      <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="emit('update:modelValue', false)">Cancel</button>
      <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : party ? 'Save changes' : `Add ${noun}` }}</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import { Contact, Factory } from 'lucide-vue-next'
import type { Party, PartyKind } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { useOwner } from '~/composables/useOwner'
import { rule } from '~/utils/validation'

/** Adds someone to the customer or supplier list, or changes them (party). */
const props = withDefaults(defineProps<{
  modelValue: boolean
  kind: PartyKind
  party?: Party | null
  /** A new one starts with this name (what was typed in a picker). */
  initialName?: string
  /** A new one starts as this branch's own (a document's branch). */
  initialBranch?: string
}>(), { party: null, initialName: '', initialBranch: '' })
const emit = defineEmits<{ 'update:modelValue': [open: boolean]; saved: [party: Party] }>()

const { api } = useApi()
const toast = useToast()
const { ownerOptions, defaultOwner, hasFullAccess } = useOwner()
const noun = computed(() => props.kind === 'customers' ? 'customer' : 'supplier')
const docs = computed(() => props.kind === 'customers' ? 'sales and quotations' : 'purchases')
const subtitle = computed(() => props.party
  ? `Renaming them renames them on their ${props.kind === 'customers' ? 'sales and quotations' : 'purchases'} too.`
  : `Shared by every branch or one branch’s own; ${props.kind === 'customers' ? 'sales, quotations and the POS' : 'purchases'} pick from the list.`)

const saving = ref(false)
const formError = ref('')
const nameEl = ref<HTMLInputElement | null>(null)
const form = reactive({ branch_id: '', name: '', phone: '', email: '', address: '', note: '' })
// Same rules as the API (services.PartyInput).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 191)],
    email: [rule.email('email')],
  }),
))

watch(() => props.modelValue, (open) => {
  if (!open) return
  const p = props.party
  Object.assign(form, p
    ? { branch_id: p.branch_id || '', name: p.name, phone: p.phone, email: p.email, address: p.address, note: p.note }
    : { branch_id: defaultOwner(props.initialBranch), name: props.initialName, phone: '', email: '', address: '', note: '' })
  formError.value = ''
  formCheck.reset()
  nextTick(() => nameEl.value?.focus())
}, { immediate: true })

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const body = { branch_id: form.branch_id, name: form.name.trim(), phone: form.phone.trim(), email: form.email.trim(), address: form.address.trim(), note: form.note.trim() }
    const saved = props.party
      ? await api<Party>(`/inventory/${props.kind}/${props.party.id}`, { method: 'PATCH', body: { ...body, is_active: props.party.is_active } })
      : await api<Party>(`/inventory/${props.kind}`, { method: 'POST', body })
    toast.success(saved.name, { title: props.party ? `${noun.value[0]!.toUpperCase()}${noun.value.slice(1)} changed` : `${noun.value[0]!.toUpperCase()}${noun.value.slice(1)} added` })
    emit('saved', saved)
    emit('update:modelValue', false)
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || `Could not save this ${noun.value}.`
  } finally {
    saving.value = false
  }
}
</script>
