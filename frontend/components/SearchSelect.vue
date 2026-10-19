<template>
  <div ref="root" class="search-select" :class="{ 'is-open': open }">
    <button
      type="button"
      class="search-select-trigger"
      role="combobox"
      :aria-expanded="open"
      :aria-label="ariaLabel || placeholder"
      :disabled="disabled"
      @click="toggle"
    >
      <span :class="hasSelection ? 'selected-label' : 'placeholder-label'">{{ selectedLabel }}</span>
      <ChevronDown :size="15" />
    </button>
    <div v-if="open" class="search-select-menu" :class="{ 'no-search': !searchable, 'drop-up': dropUp }">
      <div v-if="searchable" class="search-input-wrap">
        <Search :size="14" />
        <input ref="searchInput" v-model="search" type="search" :placeholder="searchPlaceholder" @keydown.esc="open = false" />
      </div>
      <div class="search-select-options" role="listbox" :aria-multiselectable="multiple">
        <button
          v-for="option in filteredOptions"
          :key="option.value"
          type="button"
          role="option"
          :aria-selected="isSelected(option)"
          :disabled="option.disabled"
          class="search-select-option"
          :class="{ selected: isSelected(option) }"
          @click="select(option)"
        >
          <span>{{ option.label }}</span>
          <Check v-if="isSelected(option)" :size="14" />
        </button>
        <p v-if="!filteredOptions.length" class="search-select-empty">{{ emptyText }}</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import { Check, ChevronDown, Search } from 'lucide-vue-next'

interface SearchSelectOption {
  value: string
  label: string
  disabled?: boolean
}

const props = withDefaults(defineProps<{
  modelValue: string | string[]
  options: SearchSelectOption[]
  multiple?: boolean
  placeholder?: string
  searchPlaceholder?: string
  emptyText?: string
  ariaLabel?: string
  disabled?: boolean
  /** Short lists (currency, rows per page) read better without a search box. */
  searchable?: boolean
}>(), {
  searchable: true,
  multiple: false,
  placeholder: 'Select an option',
  searchPlaceholder: 'Search…',
  emptyText: 'No matching options',
  ariaLabel: '',
  disabled: false,
})
const emit = defineEmits<{ 'update:modelValue': [value: any] }>()

const root = ref<HTMLElement | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)
const open = ref(false)
const dropUp = ref(false)
const search = ref('')
const selectedValues = computed(() => Array.isArray(props.modelValue) ? props.modelValue : props.modelValue ? [props.modelValue] : [])
const selectedOption = computed(() => props.options.find(option => option.value === props.modelValue))
const hasSelection = computed(() => props.multiple ? selectedValues.value.length > 0 : !!selectedOption.value)
const selectedLabel = computed(() => {
  if (props.multiple) {
    const labels = props.options.filter(option => selectedValues.value.includes(option.value)).map(option => option.label)
    return labels.length ? labels.join(', ') : props.placeholder
  }
  return selectedOption.value?.label || props.placeholder
})
const filteredOptions = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  if (!query) return props.options
  return props.options.filter(option => option.label.toLocaleLowerCase().includes(query))
})

function toggle() {
  if (props.disabled) return
  open.value = !open.value
  if (open.value) {
    search.value = ''
    // Open upwards when the trigger sits near the bottom of the screen (e.g. "Rows" in a pager).
    const rect = root.value?.getBoundingClientRect()
    const spaceBelow = rect ? window.innerHeight - rect.bottom : Infinity
    dropUp.value = !!rect && spaceBelow < 240 && rect.top > spaceBelow
  }
}

function select(option: SearchSelectOption) {
  if (option.disabled) return
  if (props.multiple) {
    const current = selectedValues.value
    emit('update:modelValue', current.includes(option.value)
      ? current.filter(value => value !== option.value)
      : [...current, option.value])
    return
  }
  emit('update:modelValue', option.value)
  open.value = false
}

function isSelected(option: SearchSelectOption) {
  if (!props.multiple) return props.modelValue === option.value
  return selectedValues.value.includes(option.value)
}

watch(open, async (isOpen) => {
  if (isOpen) {
    await nextTick()
    searchInput.value?.focus()
  }
})

onClickOutside(root, () => { open.value = false })
</script>

<style scoped>
.search-select { position: relative; width: 100%; }
.search-select-trigger { display: flex; width: 100%; align-items: center; justify-content: space-between; gap: 9px; min-height: 37px; border: 1px solid #dfe7f1; border-radius: 8px; padding: 9px 11px; color: #263f65; background: #fff; font: inherit; font-size: 11px; font-weight: 450; text-align: left; cursor: pointer; }
.search-select-trigger:focus-visible { outline: none; border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 9%); }
.search-select-trigger:disabled { color: #8e9cb0; background: #f6f8fb; cursor: not-allowed; }
.search-select-trigger > span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.search-select-trigger > svg { flex: 0 0 auto; color: #7d90aa; }
.placeholder-label { color: #8292aa; }
.search-select-menu { position: absolute; z-index: 30; top: calc(100% + 5px); left: 0; width: 100%; overflow: hidden; border: 1px solid #dfe7f1; border-radius: 9px; background: #fff; box-shadow: 0 12px 28px rgb(20 48 86 / 15%); }
.search-input-wrap { display: flex; align-items: center; gap: 7px; margin: 7px; border: 1px solid #e3eaf3; border-radius: 7px; padding: 0 8px; color: #8394ac; }
.search-input-wrap input { width: 100%; min-width: 0; border: 0; outline: 0; padding: 8px 0; color: #2c4468; background: transparent; font: inherit; font-size: 10px; }
.search-select-options { max-height: 190px; overflow-y: auto; padding: 0 4px 5px; }
.no-search .search-select-options { padding-top: 5px; }
.search-select-menu.drop-up { top: auto; bottom: calc(100% + 5px); box-shadow: 0 -12px 28px rgb(20 48 86 / 15%); }
.search-select-option { display: flex; width: 100%; align-items: center; justify-content: space-between; gap: 8px; border: 0; border-radius: 6px; padding: 8px; color: #405a7e; background: #fff; font: inherit; font-size: 10px; text-align: left; cursor: pointer; }
.search-select-option:hover, .search-select-option.selected { color: #1769d2; background: #f0f6ff; }
.search-select-option:disabled { color: #a3afbf; cursor: not-allowed; }
.search-select-empty { margin: 0; padding: 10px 8px; color: #8b9ab0; font-size: 10px; text-align: center; }
@media (max-width: 700px) {
  .search-select-trigger { min-height: 42px; padding: 10px 12px; font-size: 13px; }
  .search-input-wrap input { padding: 10px 0; font-size: 13px; }
  .search-select-option { padding: 11px 10px; font-size: 13px; }
  .search-select-empty { font-size: 12px; }
}
</style>
