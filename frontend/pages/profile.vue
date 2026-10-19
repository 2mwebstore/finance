<template>
  <section class="mx-auto max-w-2xl space-y-5">
    <header>
      <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCOUNT</p>
      <h1 class="text-xl font-bold text-navy">Your profile</h1>
      <p class="mt-1 text-sm text-muted">Manage your account details and sign-in PIN.</p>
    </header>

    <article class="rounded-xl border border-border bg-white p-4 sm:p-5">
      <h2 class="text-sm font-semibold text-navy">Account details</h2>
      <dl class="mt-3 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2">
        <div><dt class="text-xs text-muted">Full name</dt><dd class="mt-1 font-medium text-navy">{{ user?.full_name || '—' }}</dd></div>
        <div><dt class="text-xs text-muted">Username</dt><dd class="mt-1 font-medium text-navy">{{ user?.username || '—' }}</dd></div>
        <div><dt class="text-xs text-muted">Email</dt><dd class="mt-1 break-all font-medium text-navy">{{ user?.email || '—' }}</dd></div>
        <div><dt class="text-xs text-muted">Roles</dt><dd class="mt-1 font-medium text-navy">{{ (user?.roles || []).join(', ') || '—' }}</dd></div>
      </dl>
    </article>

    <form class="space-y-4 rounded-xl border border-border bg-white p-4 sm:p-5" @submit.prevent="changePassword">
      <div><h2 class="text-sm font-semibold text-navy">Change PIN</h2><p class="mt-1 text-xs text-muted">Enter your new 4-digit PIN. Tap the eye to check it before saving.</p></div>
      <div class="block max-w-xs space-y-1 text-xs font-medium text-muted"><span>New PIN</span><PinInput v-model="newPin" aria-label="New PIN" /></div>
      <p v-if="error" class="rounded-lg bg-red-50 px-3 py-2 text-xs text-expense" role="alert">{{ error }}</p>
      <p v-if="notice" class="rounded-lg bg-green-50 px-3 py-2 text-xs text-income" role="status">{{ notice }}</p>
      <button type="submit" :disabled="saving || newPin.length !== 4" class="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-60 sm:w-auto">{{ saving ? 'Updating…' : 'Update PIN' }}</button>
    </form>
  </section>
</template>

<script setup lang="ts">
const { api } = useApi()
const { user } = useAuth()
const newPin = ref('')
const saving = ref(false)
const error = ref('')
const notice = ref('')

async function changePassword() {
  error.value = ''
  notice.value = ''
  if (!/^\d{4}$/.test(newPin.value)) {
    error.value = 'Your PIN must be exactly 4 digits.'
    return
  }
  saving.value = true
  try {
    await api('/auth/change-password', { method: 'POST', body: { new_password: newPin.value } })
    newPin.value = ''
    notice.value = 'Your PIN was updated.'
  } catch (e: any) {
    error.value = e?.data?.error || 'Could not change your PIN.'
  } finally {
    saving.value = false
  }
}
</script>
