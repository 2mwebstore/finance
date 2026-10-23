<template>
  <label class="list-search">
    <Search :size="16" aria-hidden="true" />
    <input v-model="text" type="search" :placeholder="placeholder" :aria-label="placeholder" enterkeyhint="search" @keydown.enter.prevent="commit" />
    <button v-if="text" type="button" class="list-search-clear" aria-label="Clear search" @click="clear"><X :size="15" /></button>
  </label>
</template>

<script setup lang="ts">
import { Search, X } from 'lucide-vue-next'

/**
 * Search box for list pages. v-model updates a moment after typing stops (or on Enter), so a page
 * can reload from the API on every change without a request per keystroke.
 */
const props = withDefaults(defineProps<{ modelValue: string; placeholder?: string; delay?: number }>(), {
  placeholder: 'Search…',
  delay: 300,
})
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const text = ref(props.modelValue)
let timer: ReturnType<typeof setTimeout> | undefined

function commit() {
  if (timer) clearTimeout(timer)
  const value = text.value.trim()
  if (value !== props.modelValue) emit('update:modelValue', value)
}

function clear() {
  text.value = ''
  commit()
}

watch(text, () => {
  if (timer) clearTimeout(timer)
  timer = setTimeout(commit, props.delay)
})
watch(() => props.modelValue, (value) => {
  if (value !== text.value.trim()) text.value = value
})
onBeforeUnmount(() => {
  if (timer) clearTimeout(timer)
})
</script>

<style scoped>
.list-search { display: flex; min-width: 0; align-items: center; gap: 8px; min-height: 42px; border: 1px solid #dfe7f1; border-radius: 11px; padding: 0 8px 0 12px; color: #7d90aa; background: #fff; transition: border-color .15s ease, box-shadow .15s ease; }
.list-search:focus-within { border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.list-search input { width: 100%; min-width: 0; border: 0; outline: 0; padding: 9px 0; color: #263f65; background: transparent; font-size: 13px; box-shadow: none !important; }
.list-search input::-webkit-search-cancel-button { display: none; }
.list-search-clear { display: grid; flex: 0 0 auto; width: 26px; height: 26px; place-items: center; border: 0; border-radius: 7px; color: #8193ab; background: #f3f6fb; cursor: pointer; }
.list-search-clear:hover { color: #1d365c; background: #e9eff8; }
</style>
