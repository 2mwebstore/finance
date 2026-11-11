<template>
  <section class="mx-auto max-w-2xl space-y-5">
    <header>
      <p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCOUNT</p>
      <h1 class="text-xl font-bold text-navy">Your profile</h1>
      <p class="mt-1 text-sm text-muted">Manage your account details and sign-in password.</p>
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

    <form class="space-y-4 rounded-xl border border-border bg-white p-4 sm:p-5" novalidate @submit.prevent="changePassword">
      <div><h2 class="text-sm font-semibold text-navy">Change password</h2><p class="mt-1 text-xs text-muted">Enter your current password, then a new one of {{ MIN_PASSWORD_LENGTH }} to {{ MAX_PASSWORD_LENGTH }} characters: letters, digits and signs. Your other devices will be signed out.</p></div>
      <div class="grid max-w-md grid-cols-1 gap-3 sm:grid-cols-2">
        <div class="block space-y-1 text-xs font-medium text-muted"><span>Current password</span><PasswordInput v-model="currentPin" existing aria-label="Current password" :aria-invalid="pinCheck.invalid('current_password')" /></div>
        <div class="block space-y-1 text-xs font-medium text-muted"><span>New password</span><PasswordInput v-model="newPin" aria-label="New password" :aria-invalid="pinCheck.invalid('new_password')" /></div>
      </div>
      <AppAlert v-if="error || pinCheck.messages.length" class="max-w-md" compact :message="error" :items="pinCheck.messages" />
      <button type="submit" :disabled="saving" class="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-60 sm:w-auto">{{ saving ? 'Updating…' : 'Update password' }}</button>
    </form>
  </section>
</template>

<script setup lang="ts">
import { useToast } from '~/composables/useToast'
import { MAX_PASSWORD_LENGTH, MIN_PASSWORD_LENGTH, rule } from '~/utils/validation'
import { useFormErrors } from '~/composables/useFormErrors'
const { api } = useApi()
const { user, fetchMe } = useAuth()
const currentPin = ref('')
const newPin = ref('')
const saving = ref(false)
const error = ref('')
const toast = useToast()
// Same rules as the API (changePasswordRequest: current password required, new one 4 to 64 characters).
const pinCheck = reactive(useFormErrors(
  () => ({ current_password: currentPin.value, new_password: newPin.value }),
  () => ({
    current_password: [rule.required('current password')],
    new_password: [rule.required('new password'), rule.password('new password')],
  }),
))

async function changePassword() {
  error.value = ''
  if (!pinCheck.check()) return
  saving.value = true
  try {
    await api('/auth/change-password', { method: 'POST', body: { current_password: currentPin.value, new_password: newPin.value } })
    currentPin.value = ''
    newPin.value = ''
    pinCheck.reset()
    // Picks up the account as it is now.
    await fetchMe()
    toast.success('Use it the next time you sign in. Your other devices were signed out.', { title: 'Your password was updated' })
  } catch (e: any) {
    if (!pinCheck.fromApi(e)) error.value = e?.data?.error || 'Could not change your password.'
  } finally {
    saving.value = false
  }
}
</script>
