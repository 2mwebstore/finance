<template>
  <span class="inline-flex items-center rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wide" :class="tone">{{ label }}</span>
</template>

<script setup lang="ts">
const props = defineProps<{ status: string }>()
const tone = computed(() => ({
  pending: 'bg-amber-50 text-amber-700',
  approved: 'bg-emerald-50 text-emerald-700',
  completed: 'bg-emerald-50 text-emerald-700',
  rejected: 'bg-rose-50 text-rose-700',
  voided: 'bg-slate-100 text-slate-500',
}[props.status] || 'bg-slate-100 text-slate-600'))
const label = computed(() => ({ pending: 'Waiting for approval', completed: 'Done' }[props.status] || props.status))
</script>
