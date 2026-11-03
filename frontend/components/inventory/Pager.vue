<template>
  <!-- Phones: the count and Rows on one line, then Previous / Page / Next across the full width;
       wider screens: everything in one row. -->
  <div v-if="total > 0" class="flex flex-col gap-3 text-xs text-muted sm:flex-row sm:flex-wrap sm:items-center sm:justify-between sm:gap-2">
    <div class="flex items-center justify-between gap-3 sm:justify-start sm:gap-x-3">
      <span>Showing {{ first }}–{{ last }} of {{ total }}</span>
      <!-- Rows per page (the list's limit), when the page offers a choice -->
      <label v-if="sizes?.length" class="flex items-center gap-1.5">
        <span>Rows</span>
        <span class="w-28 sm:w-24"><SearchSelect :model-value="String(pageSize)" :options="sizeOptions" :searchable="false" aria-label="Rows per page" @update:model-value="(value: string) => emit('update:pageSize', Number(value))" /></span>
      </label>
    </div>
    <div class="grid grid-cols-[1fr_auto_1fr] items-center gap-2 sm:flex">
      <button type="button" class="pager-button" :disabled="page <= 1" @click="emit('update:page', page - 1)"><ChevronLeft :size="15" class="sm:hidden" /> Previous</button>
      <span class="whitespace-nowrap text-center">Page {{ page }} of {{ pages }}</span>
      <button type="button" class="pager-button" :disabled="page >= pages" @click="emit('update:page', page + 1)">Next <ChevronRight :size="15" class="sm:hidden" /></button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'

const props = defineProps<{ page: number; pageSize: number; total: number; sizes?: number[] }>()
const emit = defineEmits<{ 'update:page': [page: number]; 'update:pageSize': [size: number] }>()
const pages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const first = computed(() => (props.page - 1) * props.pageSize + 1)
const last = computed(() => Math.min(props.total, props.page * props.pageSize))
const sizeOptions = computed(() => (props.sizes || []).map(size => ({ value: String(size), label: `${size} rows` })))
</script>

<style scoped>
.pager-button { display: inline-flex; align-items: center; justify-content: center; gap: 4px; min-height: 44px; border: 1px solid #dfe7f1; border-radius: 10px; padding: 0 14px; color: #1b3156; background: #fff; font: inherit; font-weight: 600; cursor: pointer; }
.pager-button:disabled { opacity: .5; cursor: not-allowed; }
@media (min-width: 640px) {
  .pager-button { min-height: 0; border-radius: 8px; padding: 6px 12px; }
}
</style>
