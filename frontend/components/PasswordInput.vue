<template>
  <div class="pin-input">
    <input
      :value="modelValue"
      :type="visible ? 'text' : 'password'"
      inputmode="numeric"
      :pattern="`[0-9]{${MIN_PIN_LENGTH},${MAX_PIN_LENGTH}}`"
      :minlength="MIN_PIN_LENGTH"
      :maxlength="MAX_PIN_LENGTH"
      :autocomplete="existing ? 'current-password' : 'new-password'"
      required
      :aria-label="ariaLabel"
      :placeholder="'•'.repeat(MIN_PIN_LENGTH)"
      @input="onInput"
    />
    <!-- There is no confirm field, so let people check what they typed. -->
    <button type="button" class="pin-toggle" :aria-label="visible ? 'Hide PIN' : 'Show PIN'" :aria-pressed="visible" @click="visible = !visible">
      <EyeOff v-if="visible" :size="17" />
      <Eye v-else :size="17" />
    </button>
  </div>
</template>

<script setup lang="ts">
import { Eye, EyeOff } from 'lucide-vue-next'
import { MAX_PIN_LENGTH, MIN_PIN_LENGTH } from '~/utils/validation'

// `existing`: the PIN someone already has (for the browser's password manager).
defineProps<{ modelValue: string; ariaLabel?: string; existing?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
const visible = ref(false)

function onInput(event: Event) {
  const input = event.target as HTMLInputElement
  const digits = input.value.replace(/\D/g, '').slice(0, MAX_PIN_LENGTH)
  if (digits !== input.value) input.value = digits
  emit('update:modelValue', digits)
}
</script>

<style scoped>
.pin-input { position: relative; }
.pin-input input { width: 100%; min-height: 46px; border: 1px solid #dfe7f1; border-radius: 10px; padding: 10px 46px 10px 14px; color: #1b3156; background: #fff; font-size: 20px; font-weight: 700; letter-spacing: .5em; font-variant-numeric: tabular-nums; outline: none; transition: border-color .15s ease, box-shadow .15s ease; }
.pin-input input::placeholder { color: #b7c3d4; letter-spacing: .5em; }
.pin-input input:focus { border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.pin-toggle { position: absolute; top: 50%; right: 8px; display: grid; width: 32px; height: 32px; place-items: center; border: 0; border-radius: 8px; color: #6f83a2; background: transparent; cursor: pointer; transform: translateY(-50%); }
.pin-toggle:hover { color: #1d365c; background: #f2f6fc; }
</style>
