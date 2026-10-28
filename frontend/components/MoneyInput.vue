<template>
  <input
    ref="input"
    :value="displayValue"
    type="text"
    inputmode="decimal"
    autocomplete="off"
    :placeholder="decimalsOf(currency) ? `0.${'0'.repeat(decimalsOf(currency))}` : '0'"
    :aria-label="ariaLabel"
    @focus="focused = true"
    @input="onInput"
    @blur="onBlur"
  />
</template>

<script setup lang="ts">
import { currencyMeta, type SupportedCurrency } from '~/utils/money'

// Decimals come from Manage → Currency (USD 2, KHR 0…). Whole digits stop at 12 for currencies
// without decimals and 9 otherwise (inside the API's maximum).
function decimalsOf(currency: SupportedCurrency) {
  return currencyMeta(currency).decimals
}
function wholeLimit(currency: SupportedCurrency) {
  return decimalsOf(currency) ? 9 : 12
}

const props = defineProps<{
  modelValue: string
  currency: SupportedCurrency
  ariaLabel?: string
  /** Allows a minus in front (e.g. a loss). */
  signed?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const input = ref<HTMLInputElement | null>(null)
const focused = ref(false)
// Shown finished (e.g. 1,500.00), as when the value changes while the input isn't focused.
const displayValue = ref(formatEntry(props.modelValue, props.currency, true))

// A minus is kept only in front, and only when the input is signed.
function normalize(value: string, currency: SupportedCurrency) {
  const amount = normalizeAmount(value, currency)
  return props.signed && value.trim().startsWith('-') ? `-${amount}` : amount
}

function normalizeAmount(value: string, currency: SupportedCurrency) {
  const cleaned = value.replace(/,/g, '').replace(/[^\d.]/g, '')
  const limit = wholeLimit(currency)
  const places = decimalsOf(currency)
  if (!places) return trimWhole(cleaned.split('.')[0]!.replace(/\D/g, ''), limit)
  const dot = cleaned.indexOf('.')
  if (dot < 0) return trimWhole(cleaned, limit)
  const whole = trimWhole(cleaned.slice(0, dot).replace(/\./g, ''), limit)
  const fraction = cleaned.slice(dot + 1).replace(/\./g, '').slice(0, places)
  return `${whole}.${fraction}`
}

// Drop leading zeros and stop at the currency's digit limit (extra digits are ignored).
function trimWhole(digits: string, limit: number) {
  return digits.replace(/^0+(?=\d)/, '').slice(0, limit)
}

function groupDigits(value: string) {
  return value.replace(/\B(?=(\d{3})+(?!\d))/g, ',')
}

function formatEntry(value: string, currency: SupportedCurrency, complete = false) {
  const raw = normalize(value, currency)
  const amount = raw.replace(/^-/, '')
  // A lone minus stays while typing; a finished zero has none.
  let sign = raw.startsWith('-') ? '-' : ''
  if (!amount) return complete ? '' : sign
  if (complete && !Number(amount)) sign = ''
  if (!decimalsOf(currency)) return sign + groupDigits(amount)

  const hasDot = amount.includes('.')
  let [whole, fraction = ''] = amount.split('.')
  if (!whole) whole = '0'
  if (complete) fraction = fraction.padEnd(decimalsOf(currency), '0').slice(0, decimalsOf(currency))
  return `${sign}${groupDigits(whole)}${hasDot || complete ? `.${fraction}` : ''}`
}

// Characters before the caret that are kept (letters and other refused characters don't count).
function rawCaretLength(value: string) {
  return value.replace(/[^\d.-]/g, '').length
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
  // A refused character leaves the value as it was, so put the box back by hand.
  if (element.value !== formatted) element.value = formatted
  emit('update:modelValue', raw === '-' ? '' : raw)
  nextTick(() => {
    const caret = caretForRawLength(formatted, rawCaret)
    element.setSelectionRange(caret, caret)
  })
}

function onBlur() {
  focused.value = false
  const formatted = formatEntry(props.modelValue, props.currency, true)
  displayValue.value = formatted
  if (input.value && input.value.value !== formatted) input.value.value = formatted
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
