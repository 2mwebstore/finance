<template>
  <div class="password-input">
    <!-- The full keyboard (letters, digits and signs); no capitals or corrections added by phones -->
    <input
      :value="modelValue"
      :type="visible ? 'text' : 'password'"
      :minlength="MIN_PASSWORD_LENGTH"
      :maxlength="MAX_PASSWORD_LENGTH"
      :autocomplete="existing ? 'current-password' : 'new-password'"
      autocapitalize="off"
      autocorrect="off"
      spellcheck="false"
      required
      :aria-label="ariaLabel"
      @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    />
    <!-- There is no confirm field, so let people check what they typed. -->
    <button type="button" class="password-toggle" :aria-label="visible ? 'Hide password' : 'Show password'" :aria-pressed="visible" @click="visible = !visible">
      <EyeOff v-if="visible" :size="17" />
      <Eye v-else :size="17" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Eye, EyeOff } from 'lucide-vue-next'
import { MAX_PASSWORD_LENGTH, MIN_PASSWORD_LENGTH } from '~/utils/validation'

// A password: letters, digits and signs (a PIN of digits still is one).
// `existing`: the password someone already has (for the browser's password manager).
defineProps<{ modelValue: string; ariaLabel?: string; existing?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const visible = ref(false)
</script>

<style scoped>
.password-input { position: relative; }
/* 16px so phones don't zoom in on it */
.password-input input { width: 100%; min-height: 46px; border: 1px solid #dfe7f1; border-radius: 10px; padding: 10px 46px 10px 14px; color: #1b3156; background: #fff; font-size: 16px; font-weight: 600; outline: none; transition: border-color .15s ease, box-shadow .15s ease; }
.password-input input:focus { border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.password-toggle { position: absolute; top: 50%; right: 8px; display: grid; width: 32px; height: 32px; place-items: center; border: 0; border-radius: 8px; color: #6f83a2; background: transparent; cursor: pointer; transform: translateY(-50%); }
.password-toggle:hover { color: #1d365c; background: #f2f6fc; }
</style>
