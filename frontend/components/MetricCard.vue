<template>
  <div class="bg-white rounded-xl border border-border p-4 shadow-sm">
    <div class="text-xs text-muted font-medium">{{ label }}</div>
    <div
      class="mt-1 text-lg font-bold tabular-nums"
      :class="{
        'text-income': tone === 'income',
        'text-expense': tone === 'expense',
        'text-navy': tone === 'neutral',
      }"
    >
      {{ value }}
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label: string
  value: string
  tone: 'income' | 'expense' | 'neutral'
}>()
</script>
