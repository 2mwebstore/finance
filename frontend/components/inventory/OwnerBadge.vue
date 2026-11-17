<template>
  <!-- Shared by every branch, or one branch's own. -->
  <span class="inline-flex shrink-0 items-center rounded-full px-2 py-0.5 text-[10px] font-bold uppercase" :class="branchId ? 'bg-amber-50 text-amber-700' : 'bg-indigo-50 text-indigo-600'" :title="branchId ? `Only ${branchName || 'one branch'} sees and uses it` : 'Every branch sees and uses it'">{{ branchId ? (branchName || 'One branch') : 'Shared' }}</span>
</template>

<script setup lang="ts">
defineProps<{ branchId?: string | null; branchName?: string }>()
</script>
