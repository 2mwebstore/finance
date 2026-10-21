<template>
  <div>
    <div class="login-heading">
      <p class="login-eyebrow">WELCOME BACK</p>
      <h2>Sign in to your workspace</h2>
      <p>Use your account credentials to continue.</p>
    </div>
    <form class="bg-white rounded-2xl shadow-xl p-6 space-y-4" novalidate @submit.prevent="onSubmit">
      <div>
        <label class="block text-sm font-medium text-navy mb-1">Username or email</label>
        <input
          v-model="identifier"
          type="text"
          autocomplete="username"
          required
          :aria-invalid="loginCheck.invalid('identifier')"
          class="w-full rounded-lg border border-border px-3 py-2.5 text-base focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="manager1"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-navy mb-1">Password</label>
        <input
          v-model="password"
          type="password"
          inputmode="numeric"
          autocomplete="current-password"
          required
          :aria-invalid="loginCheck.invalid('password')"
          class="w-full rounded-lg border border-border px-3 py-2.5 text-base focus:outline-none focus:ring-2 focus:ring-primary"
        />
      </div>
      <p v-if="showDemoCredentials" class="text-center text-xs text-muted">
        Demo login: <span class="font-medium text-navy">manager1</span> / PIN <span class="font-medium text-navy">246810</span>
      </p>
      <AppAlert v-if="error || loginCheck.messages.length" title="Couldn’t sign you in" :message="error" :items="loginCheck.messages" />
      <button
        type="submit"
        :disabled="loading"
        class="w-full rounded-lg bg-primary hover:bg-primary-hover text-white font-medium py-2.5 transition disabled:opacity-60"
      >
        {{ loading ? 'Signing in…' : 'Sign in' }}
      </button>
    </form>
  </div>
</template>

<script setup lang="ts">
import { rule } from '~/utils/validation'
import { useFormErrors } from '~/composables/useFormErrors'

definePageMeta({ layout: 'auth' })

const { login, user } = useAuth()
const { canViewDashboard, canViewTransactions } = usePermissions()
const identifier = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)
const showDemoCredentials = import.meta.dev
// Same required fields as the API (loginRequest).
const loginCheck = reactive(useFormErrors(
  () => ({ identifier: identifier.value, password: password.value }),
  () => ({ identifier: [rule.required('username or email')], password: [rule.required('password')] }),
))

onMounted(() => {
  if (user.value) navigateTo(defaultLandingPath())
})

function defaultLandingPath() {
  if (user.value?.pin_upgrade_required) return '/profile'
  const mobile = window.matchMedia('(max-width: 760px)').matches
  if (mobile && canViewTransactions.value) return '/transactions'
  if (canViewDashboard.value) return '/dashboard'
  if (canViewTransactions.value) return '/transactions'
  return '/more'
}

async function onSubmit() {
  error.value = ''
  if (!loginCheck.check()) return
  loading.value = true
  try {
    await login(identifier.value.trim(), password.value)
    await navigateTo(defaultLandingPath())
  } catch (e: any) {
    if (!loginCheck.fromApi(e)) error.value = e?.data?.error || e?.message || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>
