<template>
  <div>
    <SearchSelect
      :model-value="modelValue"
      :options="options"
      :placeholder="shownPlaceholder"
      :search-placeholder="`Search name or phone…`"
      :empty-text="loading ? 'Loading…' : `No ${noun} matches`"
      :aria-label="ariaLabel || noun"
      :aria-invalid="invalid ? 'true' : undefined"
      :create-label="canAdd ? `Add a new ${noun}` : ''"
      @update:model-value="choose"
      @create="startAdd"
    />
    <InventoryPartyModal v-if="canAdd" v-model="adding" :kind="kind" :initial-name="addName" :initial-branch="branchId" @saved="added" />
  </div>
</template>

<script setup lang="ts">
import type { Party, PartyKind } from '~/types/api'

/**
 * Picks a customer or supplier for a document of `branchId` (those shared by every branch and that
 * branch's own; searchable by name or phone), or adds a new one from the dropdown ("+ Add …",
 * with what was typed; it starts as the branch's own). `name` is the document's own name: an old
 * document's turned-off customer still shows, and one typed without the list shows as typed.
 */
const props = withDefaults(defineProps<{
  kind: PartyKind
  modelValue: string
  /** The document's branch: only those shared or this branch's own are offered. */
  branchId?: string
  /** The document's name for this party (shown when they aren't in the active list). */
  name?: string
  /** Adds a "nobody" choice at the top with this label (e.g. "Walk-in"). */
  noneLabel?: string
  placeholder?: string
  ariaLabel?: string
  invalid?: boolean
}>(), { branchId: '', name: '', noneLabel: '', placeholder: '', ariaLabel: '', invalid: false })
const emit = defineEmits<{ 'update:modelValue': [id: string]; pick: [party: Party | null] }>()

const { api } = useApi()
const { can } = usePermissions()
const noun = computed(() => props.kind === 'customers' ? 'customer' : 'supplier')
// Whoever makes these documents can add someone new (changing them needs the Manage permission).
const canAdd = computed(() => props.kind === 'customers'
  ? can('customers.manage', 'sales.create', 'sales.edit', 'quotations.create', 'quotations.edit')
  : can('suppliers.manage', 'purchases.create', 'purchases.edit'))

const items = ref<Party[]>([])
const loading = ref(true)
async function load() {
  loading.value = true
  try {
    // The first 500 (by name) of those turned on that the document's branch can use.
    items.value = (await api<{ items: Party[] }>(`/inventory/${props.kind}`, { query: { active: 'true', page_size: 500, branch_id: props.branchId || undefined } })).items || []
  } catch {
    items.value = []
  } finally {
    loading.value = false
  }
}
onMounted(load)
// Another branch: its list, and someone of the old branch's own can't stay.
watch(() => props.branchId, async (now, before) => {
  if (now === before) return
  await load()
  const picked = items.value.find(p => p.id === props.modelValue)
  if (before && props.modelValue && !picked) {
    emit('update:modelValue', '')
    emit('pick', null)
  }
})

const labelOf = (p: Party) => [p.name, p.phone, p.branch_id ? `${p.branch_name || 'branch'} only` : ''].filter(Boolean).join(' · ')
const options = computed(() => {
  const list = items.value.map(p => ({ value: p.id, label: labelOf(p) }))
  // Someone turned off (or past the first 500) who is already on this document.
  if (props.modelValue && !items.value.some(p => p.id === props.modelValue)) list.unshift({ value: props.modelValue, label: props.name || `This ${noun.value}` })
  // (A name typed without the list shows in the box instead of the "nobody" choice.)
  if (props.noneLabel && !(props.name && !props.modelValue)) list.unshift({ value: '', label: props.noneLabel })
  return list
})
const shownPlaceholder = computed(() => {
  if (!props.modelValue && props.name) return `${props.name} (typed, not in the list)`
  return props.placeholder || `Choose a ${noun.value}`
})

function choose(id: string) {
  emit('update:modelValue', id)
  emit('pick', items.value.find(p => p.id === id) || null)
}

const adding = ref(false)
const addName = ref('')
function startAdd(text: string) {
  addName.value = text
  adding.value = true
}
function added(p: Party) {
  items.value = [...items.value, p].sort((a, b) => a.name.localeCompare(b.name))
  emit('update:modelValue', p.id)
  emit('pick', p)
}
</script>
