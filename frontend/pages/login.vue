<template>
  <div>
    <form class="bg-white rounded-2xl shadow-xl p-6 space-y-4" novalidate @submit.prevent="onSubmit">
      <div>
        <!-- name@branch for people added on Users (an email still works for those who have one) -->
        <label class="block text-sm font-medium text-navy mb-1">Username</label>
        <input
          v-model="identifier"
          type="text"
          autocomplete="username"
          required
          :aria-invalid="loginCheck.invalid('identifier')"
          class="w-full rounded-lg border border-border px-3 py-2.5 text-base focus:outline-none focus:ring-2 focus:ring-primary"
          placeholder="e.g. dara@phnompenh"
        />
      </div>
      <div>
        <label class="block text-sm font-medium text-navy mb-1">Password</label>
        <div class="relative">
          <input
            v-model="password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="current-password"
            autocapitalize="off"
            autocorrect="off"
            spellcheck="false"
            required
            :aria-invalid="loginCheck.invalid('password')"
            class="w-full rounded-lg border border-border py-2.5 pl-3 pr-11 text-base focus:outline-none focus:ring-2 focus:ring-primary"
            placeholder="Password"
          />
          <button
            type="button"
            class="absolute inset-y-0 right-0 grid w-11 place-items-center rounded-r-lg text-muted hover:text-navy focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary"
            :aria-label="showPassword ? 'Hide password' : 'Show password'"
            :aria-pressed="showPassword"
            @click="showPassword = !showPassword"
          >
            <EyeOff v-if="showPassword" :size="18" />
            <Eye v-else :size="18" />
          </button>
        </div>
      </div>
      <p v-if="showDemoCredentials" class="text-center text-xs text-muted">
        Demo login: <span class="font-medium text-navy">manager1</span> / password <span class="font-medium text-navy">246810</span>
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
import { Eye, EyeOff } from 'lucide-vue-next'
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
const showPassword = ref(false)
// Same required fields as the API (loginRequest).
const loginCheck = reactive(useFormErrors(
  () => ({ identifier: identifier.value, password: password.value }),
  () => ({ identifier: [rule.required('username')], password: [rule.required('password')] }),
))

onMounted(() => {
  if (user.value) navigateTo(defaultLandingPath())
})

function defaultLandingPath() {
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
