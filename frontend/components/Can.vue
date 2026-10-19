<template>
  <slot v-if="allowed" />
  <slot v-else name="fallback" />
</template>

<script setup lang="ts">
/**
 * Declarative permission gate.
 * <Can perm="income.create">...</Can>
 * <Can :any="['income.create','expense.create']">...</Can>
 * <Can :all="['users.edit','users.manage']">...</Can>
 */
const props = defineProps<{
  perm?: string
  any?: string[]
  all?: string[]
}>()

const { can, canAll } = usePermissions()

const allowed = computed(() => {
  if (props.all?.length) return canAll(...props.all)
  if (props.any?.length) return can(...props.any)
  if (props.perm) return can(props.perm)
  return true
})
</script>
