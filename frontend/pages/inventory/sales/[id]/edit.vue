<template>
  <InventorySaleForm :sale-id="String(route.params.id)" />
</template>

<script setup lang="ts">
const route = useRoute()
</script>
