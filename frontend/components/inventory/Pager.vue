<template>
  <div v-if="total > 0" class="flex flex-wrap items-center justify-between gap-2 text-xs text-muted">
    <span>Showing {{ first }}–{{ last }} of {{ total }}</span>
    <div class="flex items-center gap-2">
      <button type="button" class="rounded-lg border border-border bg-white px-3 py-1.5 font-semibold text-navy disabled:opacity-50" :disabled="page <= 1" @click="emit('update:page', page - 1)">Previous</button>
      <span>Page {{ page }} of {{ pages }}</span>
      <button type="button" class="rounded-lg border border-border bg-white px-3 py-1.5 font-semibold text-navy disabled:opacity-50" :disabled="page >= pages" @click="emit('update:page', page + 1)">Next</button>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ page: number; pageSize: number; total: number }>()
const emit = defineEmits<{ 'update:page': [page: number] }>()
const pages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
const first = computed(() => (props.page - 1) * props.pageSize + 1)
const last = computed(() => Math.min(props.total, props.page * props.pageSize))
</script>
