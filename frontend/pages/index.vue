<script setup lang="ts">
const { initAuth, user } = useAuth()
const { canViewDashboard, canViewTransactions } = usePermissions()

if (import.meta.client) {
  await initAuth()

  const isMobile = window.matchMedia('(max-width: 760px)').matches
  if (!user.value) {
    await navigateTo('/login')
  } else if (isMobile && canViewTransactions.value) {
    await navigateTo('/transactions')
  } else if (canViewDashboard.value) {
    await navigateTo('/dashboard')
  } else if (canViewTransactions.value) {
    await navigateTo('/transactions')
  } else {
    await navigateTo('/more')
  }
}
</script>
