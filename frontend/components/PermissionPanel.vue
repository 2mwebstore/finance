<template>
  <div class="permission-panel">
    <div class="panel-toolbar">
      <label class="panel-search">
        <Search :size="15" />
        <input v-model="search" type="search" placeholder="Search permissions…" aria-label="Search permissions" />
      </label>
      <span class="panel-count"><strong>{{ modelValue.length }}</strong> of {{ permissions.length }} selected</span>
      <div class="panel-bulk">
        <button type="button" :disabled="disabled || allVisibleSelected" @click="setMany(visibleCodes, true)">Select all</button>
        <button type="button" :disabled="disabled || !anyVisibleSelected" @click="setMany(visibleCodes, false)">Clear</button>
      </div>
    </div>

    <div v-if="!groups.length" class="panel-empty">{{ permissions.length ? 'No permissions match your search.' : 'No permission catalog is available.' }}</div>
    <div class="panel-grid">
      <section v-for="group in groups" :key="group.module" class="module-card" :class="{ 'is-full': group.selected === group.items.length }">
        <label class="module-header">
          <input
            type="checkbox"
            class="panel-checkbox"
            :checked="group.selected === group.items.length"
            :indeterminate="group.selected > 0 && group.selected < group.items.length"
            :disabled="disabled"
            :aria-label="`Select all ${group.module} permissions`"
            @change="setMany(group.items.map(item => item.code), ($event.target as HTMLInputElement).checked)"
          />
          <span class="module-name">{{ group.module }}</span>
          <span class="module-count">{{ group.selected }}/{{ group.items.length }}</span>
        </label>
        <label v-for="permission in group.items" :key="permission.code" class="permission-row" :class="{ 'is-checked': selected.has(permission.code) }">
          <input
            type="checkbox"
            class="panel-checkbox"
            :checked="selected.has(permission.code)"
            :disabled="disabled"
            @change="setMany([permission.code], ($event.target as HTMLInputElement).checked)"
          />
          <span class="permission-copy">
            <strong>{{ permission.name }}</strong>
            <small>{{ permission.description || permission.code }}</small>
          </span>
        </label>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Search } from 'lucide-vue-next'
import type { Permission } from '~/types/api'

const props = withDefaults(defineProps<{
  modelValue: string[]
  permissions: Pick<Permission, 'code' | 'name' | 'module' | 'description'>[]
  disabled?: boolean
}>(), { disabled: false })
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()

const search = ref('')
const selected = computed(() => new Set(props.modelValue))
const visible = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  if (!query) return props.permissions
  return props.permissions.filter(permission => [permission.name, permission.code, permission.module || '', permission.description || '']
    .some(text => text.toLocaleLowerCase().includes(query)))
})
const visibleCodes = computed(() => visible.value.map(permission => permission.code))
const allVisibleSelected = computed(() => visibleCodes.value.length > 0 && visibleCodes.value.every(code => selected.value.has(code)))
const anyVisibleSelected = computed(() => visibleCodes.value.some(code => selected.value.has(code)))
const groups = computed(() => {
  const byModule = new Map<string, typeof props.permissions>()
  for (const permission of visible.value) {
    const module = permission.module || 'General'
    byModule.set(module, [...(byModule.get(module) || []), permission])
  }
  return [...byModule.entries()].map(([module, items]) => ({
    module,
    items,
    selected: items.filter(item => selected.value.has(item.code)).length,
  }))
})

function setMany(codes: string[], checked: boolean) {
  if (props.disabled) return
  const next = new Set(props.modelValue)
  for (const code of codes) {
    if (checked) next.add(code)
    else next.delete(code)
  }
  // Keep catalog order so saved roles read the same way every time.
  emit('update:modelValue', props.permissions.map(permission => permission.code).filter(code => next.has(code)))
}
</script>

<style scoped>
.permission-panel { display: flex; flex-direction: column; gap: 12px; }
.panel-toolbar { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; border: 1px solid #e2eaf5; border-radius: 12px; padding: 10px; background: #f8fafd; }
.panel-search { display: flex; flex: 1 1 220px; min-width: 0; align-items: center; gap: 8px; border: 1px solid #dfe7f1; border-radius: 9px; padding: 0 10px; color: #7d90aa; background: #fff; }
.panel-search input { width: 100%; min-width: 0; border: 0; outline: 0; padding: 9px 0; color: #263f65; background: transparent; font-size: 13px; }
.panel-count { color: #6f819c; font-size: 12px; white-space: nowrap; }
.panel-count strong { color: #1769e8; }
.panel-bulk { display: flex; gap: 6px; margin-left: auto; }
.panel-bulk button { min-height: 34px; border: 1px solid #dfe7f1; border-radius: 8px; padding: 0 12px; color: #3d5a82; background: #fff; font-size: 12px; font-weight: 650; cursor: pointer; }
.panel-bulk button:disabled { color: #aab5c3; cursor: default; }
.panel-empty { border-radius: 10px; padding: 14px; color: #8091a9; background: #f5f8fc; font-size: 13px; text-align: center; }
.panel-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 12px; }
.module-card { overflow: hidden; border: 1px solid #e2eaf5; border-radius: 12px; background: #fff; transition: border-color .15s ease; }
.module-card.is-full { border-color: #b9d3f8; }
.module-header { display: flex; align-items: center; gap: 10px; border-bottom: 1px solid #edf1f7; padding: 10px 12px; background: #f8fafd; cursor: pointer; }
.module-name { flex: 1; color: #1b3156; font-size: 13px; font-weight: 750; }
.module-count { border-radius: 999px; padding: 2px 8px; color: #5f7391; background: #eef2f8; font-size: 11px; font-weight: 700; font-variant-numeric: tabular-nums; }
.module-card.is-full .module-count { color: #1769e8; background: #e7f0ff; }
.permission-row { display: flex; align-items: flex-start; gap: 10px; padding: 10px 12px; border-bottom: 1px solid #f1f4f9; cursor: pointer; transition: background .12s ease; }
.permission-row:last-child { border-bottom: 0; }
.permission-row:hover { background: #fafcff; }
.permission-row.is-checked { background: #f3f8ff; }
.permission-copy { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
.permission-copy strong { color: #223b60; font-size: 13px; font-weight: 650; }
.permission-copy small { color: #8193ab; font-size: 11px; line-height: 1.4; }
.panel-checkbox { flex: 0 0 auto; width: 18px; height: 18px; margin: 1px 0 0; accent-color: #1769e8; cursor: pointer; }
.panel-checkbox:disabled { cursor: not-allowed; }
@media (max-width: 700px) {
  .panel-count { order: 3; }
  .panel-bulk { margin-left: 0; }
  .panel-grid { grid-template-columns: 1fr; }
  .permission-row { padding: 12px; }
}
</style>
