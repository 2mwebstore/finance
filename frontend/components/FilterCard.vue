<template>
  <div class="filter-card" :class="{ 'is-collapsed': !open }">
    <!-- Phones: collapsed behind a summary row (same pattern as the dashboard filter). -->
    <button type="button" class="filter-card-toggle" :aria-expanded="open" @click="open = !open">
      <span class="filter-card-icon"><ListFilter :size="18" /></span>
      <span class="filter-card-copy"><strong>{{ title }}</strong><small>{{ summary }}</small></span>
      <span v-if="active" class="filter-card-dot" title="Filters applied"></span>
      <ChevronDown :size="17" class="filter-card-chevron" />
    </button>
    <div class="filter-card-body"><slot /></div>
  </div>
</template>

<script setup lang="ts">
import { ChevronDown, ListFilter } from 'lucide-vue-next'

withDefaults(defineProps<{
  summary: string
  active?: boolean
  title?: string
}>(), {
  active: false,
  title: 'Filter',
})
const open = ref(false)
</script>

<style scoped>
.filter-card { margin-bottom: 14px; border: 1px solid #e2eaf5; border-radius: 14px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.filter-card-toggle { display: none; }
.filter-card-body { display: flex; align-items: flex-end; gap: 12px; padding: 14px 16px; }
.filter-card-body :slotted(.filter-field) { display: flex; flex: 1 1 170px; min-width: 140px; max-width: 240px; flex-direction: column; gap: 7px; color: #4f6688; font-size: 12px; font-weight: 650; }
.filter-card-body :slotted(.date-range-filter) { flex-basis: 260px; min-width: 230px; max-width: 320px; }
.filter-card-body :deep(.search-select-trigger) { font-size: 12px; }
.filter-card-body :slotted(.filter-reset) { display: inline-flex; min-height: 38px; align-items: center; justify-content: center; gap: 6px; margin-left: auto; border: 1px solid #dfe7f1; border-radius: 9px; padding: 0 14px; color: #5e7392; background: #fff; font-size: 12px; font-weight: 650; cursor: pointer; }
.filter-card-body :slotted(.filter-reset:hover:not(:disabled)) { filter: brightness(.97); }
.filter-card-body :slotted(.filter-reset:disabled) { color: #aab5c3; background: #f8fafc; cursor: default; }
@media (max-width: 900px) {
  .filter-card-body { flex-wrap: wrap; }
  .filter-card-body :slotted(.date-range-filter) { max-width: none; }
}
@media (max-width: 700px) {
  .filter-card-toggle { display: flex; width: 100%; min-height: 60px; align-items: center; gap: 10px; border: 0; border-radius: 14px; padding: 10px 14px; color: #162a4d; background: transparent; text-align: left; cursor: pointer; }
  .filter-card-icon { display: grid; flex: 0 0 auto; width: 34px; height: 34px; place-items: center; border-radius: 9px; color: #1769e8; background: #edf5ff; }
  .filter-card-copy { display: flex; flex: 1; min-width: 0; flex-direction: column; gap: 3px; }
  .filter-card-copy strong { font-size: 14px; font-weight: 720; }
  .filter-card-copy small { overflow: hidden; color: #7f90aa; font-size: 12px; text-overflow: ellipsis; white-space: nowrap; }
  .filter-card-dot { flex: 0 0 auto; width: 8px; height: 8px; border-radius: 50%; background: #1769e8; }
  .filter-card-chevron { flex: 0 0 auto; color: #8293ad; transform: rotate(180deg); transition: transform .18s ease; }
  .filter-card.is-collapsed .filter-card-chevron { transform: none; }
  .filter-card.is-collapsed .filter-card-body { display: none; }
  /* First field (the date range) gets its own row; the next field shares a row with Reset. */
  .filter-card-body { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: end; gap: 12px 10px; border-top: 1px solid #edf1f7; padding: 12px 14px 14px; }
  /* .filter-wide fields (e.g. Branch) also get a row of their own. */
  .filter-card-body > :slotted(:first-child), .filter-card-body > :slotted(.filter-wide) { grid-column: 1 / -1; width: 100%; }
  .filter-card-body :slotted(.filter-field) { min-width: 0; max-width: none; }
  .filter-card-body :slotted(.filter-reset) { min-height: 42px; margin-left: 0; font-size: 13px; }
}
</style>
