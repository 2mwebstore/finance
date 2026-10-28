<template>
  <section class="mx-auto max-w-3xl space-y-4">
    <header class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">INVENTORY</p>
        <h1 class="text-xl font-bold text-navy">Payment methods</h1>
        <p class="mt-1 text-xs text-muted">How money is paid or received: used on sales, purchases, inventory expenses and transactions. New forms start with Cash.</p>
      </div>
      <button type="button" class="inline-flex items-center gap-1.5 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover" @click="openCreate"><Plus :size="16" /> Add method</button>
    </header>

    <div v-if="loading" class="text-sm text-muted">Loading…</div>
    <AppAlert v-else-if="error" title="Couldn’t load payment methods" :message="error"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>
    <ul v-else class="divide-y divide-border rounded-xl border border-border bg-white">
      <li v-for="method in items" :key="method.id" class="flex items-center justify-between gap-3 p-4">
        <div class="flex min-w-0 items-center gap-3">
          <span class="grid h-10 w-10 shrink-0 place-items-center rounded-xl" :class="method.is_active ? 'bg-blue-50 text-primary' : 'bg-slate-100 text-slate-400'"><Wallet :size="18" /></span>
          <div class="min-w-0">
            <div class="flex flex-wrap items-center gap-2">
              <h2 class="truncate text-sm font-semibold" :class="method.is_active ? 'text-navy' : 'text-slate-400'">{{ method.name }}</h2>
              <span v-if="method.code === 'cash'" class="rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-bold uppercase text-emerald-700">Default</span>
              <span v-if="method.is_system" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Built-in</span>
              <span v-if="!method.is_active" class="rounded-full bg-slate-100 px-2 py-0.5 text-[10px] font-bold uppercase text-muted">Off</span>
            </div>
            <p class="font-mono text-[11px] text-muted">{{ method.code }}</p>
          </div>
        </div>
        <div class="flex shrink-0 gap-1.5">
          <button type="button" class="action-chip action-edit" @click="openEdit(method)"><Pencil :size="13" /> Edit</button>
          <button type="button" class="action-chip" :class="method.is_active ? 'action-danger' : 'action-success'" :disabled="busyId === method.id" @click="toggle(method)">{{ method.is_active ? 'Turn off' : 'Turn on' }}</button>
        </div>
      </li>
      <li v-if="!items.length" class="p-6 text-center text-sm text-muted">No payment methods yet.</li>
    </ul>

    <AppModal :model-value="formOpen" form :title="editing ? `Edit ${editing.name}` : 'Add payment method'" eyebrow="PAYMENT METHOD" subtitle="Shown on sales, purchases, expenses and transactions." :busy="saving" width="460px" @update:model-value="(open) => { if (!open && !saving) formOpen = false }" @submit="save">
      <template #icon><Wallet :size="20" /></template>
      <div class="space-y-3">
        <label class="block space-y-1 text-xs font-medium text-muted">Name<input v-model.trim="form.name" maxlength="128" :aria-invalid="formCheck.invalid('name')" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="e.g. ABA Bank, Wing, KHQR" /></label>
        <label class="block space-y-1 text-xs font-medium text-muted">Code
          <input v-model.trim="form.code" :disabled="!!editing" maxlength="64" :aria-invalid="formCheck.invalid('code')" class="w-full rounded-lg border border-border px-3 py-2 font-mono text-sm text-navy disabled:bg-canvas" placeholder="e.g. aba-bank" @input="codeEdited = true" />
          <small class="block font-normal">{{ editing ? 'The code stays the same so saved records keep their method.' : 'Made from the name; you can change it before saving.' }}</small>
        </label>
      </div>
      <AppAlert v-if="formError || formCheck.messages.length" class="mt-3" :title="formError ? 'Couldn’t save' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="saving" @click="formOpen = false">Cancel</button>
        <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="saving">{{ saving ? 'Saving…' : 'Save' }}</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { Pencil, Plus, Wallet } from 'lucide-vue-next'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { rule } from '~/utils/validation'

interface PaymentMethod { id: string; code: string; name: string; is_active: boolean; is_system: boolean }

const { api } = useApi()
const toast = useToast()
const items = ref<PaymentMethod[]>([])
const loading = ref(true)
const error = ref('')
const busyId = ref('')

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = (await api<{ items: PaymentMethod[] }>('/payment-methods')).items || []
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not load payment methods.'
  } finally {
    loading.value = false
  }
}
onMounted(load)

async function toggle(method: PaymentMethod) {
  busyId.value = method.id
  try {
    await api(`/payment-methods/${method.id}`, { method: 'PATCH', body: { is_active: !method.is_active } })
    toast.success(`${method.name} is ${method.is_active ? 'off' : 'on'}.`, { title: 'Payment method updated' })
    await load()
  } catch (e: any) {
    toast.error(e?.data?.fields?.is_active || e?.data?.error || 'Please try again.', { title: 'Couldn’t change it' })
  } finally {
    busyId.value = ''
  }
}

// --- add / edit ---
const formOpen = ref(false)
const editing = ref<PaymentMethod | null>(null)
const saving = ref(false)
const formError = ref('')
const codeEdited = ref(false)
const form = reactive({ name: '', code: '' })
// Same rules as the API (Create/UpdatePaymentMethodInput).
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    name: [rule.required('name'), rule.minLength('name', 2), rule.maxLength('name', 128)],
    ...(editing.value ? {} : { code: [rule.required('code'), rule.minLength('code', 2), rule.maxLength('code', 64)] }),
  }),
))
watch(() => form.name, (name) => {
  if (!editing.value && !codeEdited.value) form.code = name.toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
})

function openCreate() {
  editing.value = null
  Object.assign(form, { name: '', code: '' })
  codeEdited.value = false
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

function openEdit(method: PaymentMethod) {
  editing.value = method
  Object.assign(form, { name: method.name, code: method.code })
  formError.value = ''
  formCheck.reset()
  formOpen.value = true
}

async function save() {
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    if (editing.value) await api(`/payment-methods/${editing.value.id}`, { method: 'PATCH', body: { name: form.name } })
    else await api('/payment-methods', { method: 'POST', body: { name: form.name, code: form.code } })
    formOpen.value = false
    toast.success(`${form.name} is available on forms.`, { title: editing.value ? 'Payment method updated' : 'Payment method added' })
    await load()
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save this payment method.'
  } finally {
    saving.value = false
  }
}
</script>
