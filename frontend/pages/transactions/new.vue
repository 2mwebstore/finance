<template>
  <div class="new-transaction-route">
    <TransactionModal
      :model-value="modalOpen"
      :type="transactionType"
      mode="create"
      @update:model-value="onModalChange"
    />
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const { canCreateIncome, canCreateExpense, canViewTransactions, canViewDashboard } = usePermissions()
const modalOpen = ref(true)

const transactionType = computed<'income' | 'expense'>(() => {
  if (route.query.type === 'expense' && canCreateExpense.value) return 'expense'
  if (route.query.type === 'income' && canCreateIncome.value) return 'income'
  if (canCreateIncome.value) return 'income'
  return 'expense'
})

function returnToPreviousPage() {
  if (canViewTransactions.value) return navigateTo('/transactions')
  if (canViewDashboard.value) return navigateTo('/dashboard')
  return navigateTo('/more')
}

function onModalChange(open: boolean) {
  modalOpen.value = open
  if (!open) returnToPreviousPage()
}
</script>

<style scoped>
.new-transaction-route { min-height: calc(100vh - 110px); }
</style>
