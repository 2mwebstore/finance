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

    <div v-if="!sections.length" class="panel-empty">{{ permissions.length ? 'No permissions match your search.' : 'No permission catalog is available.' }}</div>
    <!-- Laid out like the sidebar: its sections, then a card per page with that page's permissions. -->
    <section v-for="section in sections" :key="section.caption || 'main'" class="menu-section">
      <label v-if="section.caption" class="section-header">
        <input
          type="checkbox"
          class="panel-checkbox"
          :checked="section.selected === section.total"
          :indeterminate="section.selected > 0 && section.selected < section.total"
          :disabled="disabled"
          :aria-label="`Select all ${section.caption.toLowerCase()} permissions`"
          @change="setMany(section.codes, ($event.target as HTMLInputElement).checked)"
        />
        <span class="section-caption">{{ section.caption }}</span>
        <span class="section-count">{{ section.selected }}/{{ section.total }}</span>
      </label>
      <div class="panel-grid">
        <section v-for="page in section.pages" :key="page.key" class="module-card" :class="{ 'is-full': page.selected === page.items.length }">
          <label class="module-header">
            <input
              type="checkbox"
              class="panel-checkbox"
              :checked="page.selected === page.items.length"
              :indeterminate="page.selected > 0 && page.selected < page.items.length"
              :disabled="disabled"
              :aria-label="`Select all ${page.label} permissions`"
              @change="setMany(page.items.map(item => item.code), ($event.target as HTMLInputElement).checked)"
            />
            <span class="module-icon"><component :is="page.icon" :size="15" :stroke-width="2" /></span>
            <span class="module-name">{{ page.label }}</span>
            <span class="module-count">{{ page.selected }}/{{ page.items.length }}</span>
          </label>
          <p v-if="page.note" class="module-note">{{ page.note }}</p>
          <label v-for="permission in page.items" :key="permission.code" class="permission-row" :class="{ 'is-checked': selected.has(permission.code) }">
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
              <small v-if="alsoOn(permission.code, page.label)" class="permission-also">Also on {{ alsoOn(permission.code, page.label) }}</small>
            </span>
          </label>
        </section>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { Search } from 'lucide-vue-next'
import type { Permission } from '~/types/api'
import { groupByMenu, pagesOf } from '~/utils/permissionMenu'

const props = withDefaults(defineProps<{
  modelValue: string[]
  permissions: Pick<Permission, 'code' | 'name' | 'module' | 'description'>[]
  disabled?: boolean
}>(), { disabled: false })
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()

const search = ref('')
const selected = computed(() => new Set(props.modelValue))
// Sections and pages as in the sidebar, each page's permissions in the menu's order.
const menu = computed(() => groupByMenu(props.permissions))
// Searching matches a permission, or its page or section by name ("POS", "inventory"…).
const sections = computed(() => {
  const query = search.value.trim().toLocaleLowerCase()
  const matches = (...texts: (string | undefined)[]) => !query || texts.some(text => (text || '').toLocaleLowerCase().includes(query))
  return menu.value
    .map((section) => {
      const pages = section.pages
        .map(page => ({
          ...page,
          items: matches(section.caption, page.label) ? page.items : page.items.filter(p => matches(p.name, p.code, p.description)),
        }))
        .filter(page => page.items.length)
        .map(page => ({ ...page, selected: page.items.filter(item => selected.value.has(item.code)).length }))
      // A permission on several pages counts once.
      const codes = [...new Set(pages.flatMap(page => page.items.map(item => item.code)))]
      return { caption: section.caption, pages, codes, total: codes.length, selected: codes.filter(code => selected.value.has(code)).length }
    })
    .filter(section => section.pages.length)
})
const visibleCodes = computed(() => [...new Set(sections.value.flatMap(section => section.codes))])

// The other sidebar pages a permission also opens (ticking it here ticks it there too).
function alsoOn(code: string, page: string) {
  return pagesOf(code).filter(label => label !== page).join(', ')
}
const allVisibleSelected = computed(() => visibleCodes.value.length > 0 && visibleCodes.value.every(code => selected.value.has(code)))
const anyVisibleSelected = computed(() => visibleCodes.value.some(code => selected.value.has(code)))

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
.menu-section { display: flex; flex-direction: column; gap: 8px; }
.section-header { display: flex; align-items: center; gap: 10px; padding: 4px 2px 0; cursor: pointer; }
.section-caption { flex: 1; color: #8098ba; font-size: 10px; font-weight: 750; letter-spacing: .13em; }
.section-count { color: #8098ba; font-size: 11px; font-weight: 700; font-variant-numeric: tabular-nums; }
.panel-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 12px; }
.module-card { overflow: hidden; border: 1px solid #e2eaf5; border-radius: 12px; background: #fff; transition: border-color .15s ease; }
.module-card.is-full { border-color: #b9d3f8; }
.module-header { display: flex; align-items: center; gap: 10px; border-bottom: 1px solid #edf1f7; padding: 10px 12px; background: #f8fafd; cursor: pointer; }
.module-icon { display: grid; flex: 0 0 auto; width: 26px; height: 26px; place-items: center; border-radius: 7px; color: #fff; background: linear-gradient(180deg, #163a6e, #0d2852); }
.module-card.is-full .module-icon { background: linear-gradient(100deg, #287ef2, #1263df); }
.module-note { margin: 0; border-bottom: 1px solid #f1f4f9; padding: 6px 12px; color: #8a6d1f; background: #fffbeb; font-size: 11px; }
.permission-also { color: #2563eb !important; font-weight: 600; }
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
