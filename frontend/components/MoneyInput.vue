<template>
  <input
    ref="input"
    :value="displayValue"
    type="text"
    inputmode="decimal"
    autocomplete="off"
    :placeholder="currency === 'USD' ? '0.00' : '0'"
    :aria-label="ariaLabel"
    @focus="focused = true"
    @input="onInput"
    @blur="onBlur"
  />
</template>

<script setup lang="ts">
import type { SupportedCurrency } from '~/utils/money'

const props = defineProps<{
  modelValue: string
  currency: SupportedCurrency
  ariaLabel?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const input = ref<HTMLInputElement | null>(null)
const focused = ref(false)
const displayValue = ref(formatEntry(props.modelValue, props.currency, false))

function normalize(value: string, currency: SupportedCurrency) {
  const cleaned = value.replace(/,/g, '').replace(/[^\d.]/g, '')
  if (currency === 'KHR') return cleaned.split('.')[0].replace(/\D/g, '')
  const dot = cleaned.indexOf('.')
  if (dot < 0) return cleaned
  const whole = cleaned.slice(0, dot).replace(/\./g, '')
  const fraction = cleaned.slice(dot + 1).replace(/\./g, '').slice(0, 2)
  return `${whole}.${fraction}`
}

function groupDigits(value: string) {
  return value.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

function formatEntry(value: string, currency: SupportedCurrency, complete = false) {
  const raw = normalize(value, currency)
  if (!raw) return ''
  if (currency === 'KHR') return groupDigits(raw)

  const hasDot = raw.includes('.')
  let [whole, fraction = ''] = raw.split('.')
  if (!whole) whole = '0'
  if (complete) fraction = fraction.padEnd(2, '0').slice(0, 2)
  return `${groupDigits(whole)}${hasDot || complete ? `.${fraction}` : ''}`
}

function rawCaretLength(value: string) {
  return value.replace(/,/g, '').length
}

function caretForRawLength(value: string, target: number) {
  if (target <= 0) return 0
  let rawCount = 0
  for (let i = 0; i < value.length; i += 1) {
    if (value[i] !== ',') rawCount += 1
    if (rawCount >= target) return i + 1
  }
  return value.length
}

function onInput(event: Event) {
  const element = event.target as HTMLInputElement
  const rawCaret = rawCaretLength(element.value.slice(0, element.selectionStart ?? element.value.length))
  const raw = normalize(element.value, props.currency)
  const formatted = formatEntry(raw, props.currency)
  displayValue.value = formatted
  emit('update:modelValue', raw)
  nextTick(() => {
    const caret = caretForRawLength(formatted, rawCaret)
    element.setSelectionRange(caret, caret)
  })
}

function onBlur() {
  focused.value = false
  const formatted = formatEntry(props.modelValue, props.currency, true)
  displayValue.value = formatted
  if (formatted) emit('update:modelValue', normalize(formatted, props.currency))
}

watch(() => [props.modelValue, props.currency], ([value, currency]) => {
  if (!focused.value) displayValue.value = formatEntry(String(value), currency as SupportedCurrency, true)
})

watch(() => props.currency, (currency, previousCurrency) => {
  if (previousCurrency && currency !== previousCurrency) {
    const amount = normalize(props.modelValue, currency)
    emit('update:modelValue', amount)
    displayValue.value = formatEntry(amount, currency)
  }
})
</script>
