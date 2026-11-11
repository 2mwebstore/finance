<template>
  <input
    ref="input"
    :value="displayValue"
    :type="asNumber ? 'number' : 'text'"
    :inputmode="asNumber ? undefined : 'decimal'"
    :step="asNumber ? 'any' : undefined"
    autocomplete="off"
    :placeholder="decimalsOf(currency) ? `0.${'0'.repeat(decimalsOf(currency))}` : '0'"
    :aria-label="ariaLabel"
    @touchstart.passive="onTouchStart"
    @touchend.passive="backIfNotTyping()"
    @touchcancel.passive="backIfNotTyping()"
    @beforeinput="onBeforeInput"
    @focus="onFocus"
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

// A signed amount on a phone: phones' number pads have no minus (iPhones' none at all), so while
// one is typed the box is a number box, which opens the keyboard of digits and signs
// (1 2 3 … - / : ; ( ) $ & @ "). It is plain digits while typed (-12000) and goes back to
// text with separators (-12,000) when left. Computers keep the text box.
const touch = import.meta.client && !!window.matchMedia?.('(pointer: coarse)').matches
const asNumber = ref(false)
function toNumber() {
  const el = input.value
  if (!props.signed || !touch || asNumber.value || !el || el.disabled || el.readOnly) return
  const raw = normalize(props.modelValue, props.currency)
  asNumber.value = true
  displayValue.value = raw === '-' ? '' : raw
  // Now, before the keyboard opens (the page updates a moment later).
  el.type = 'number'
  el.removeAttribute('inputmode')
  el.value = displayValue.value
}
function toText() {
  asNumber.value = false
  const formatted = formatEntry(props.modelValue, props.currency, true)
  displayValue.value = formatted
  const el = input.value
  if (!el) return
  el.type = 'text'
  el.setAttribute('inputmode', 'decimal')
  if (el.value !== formatted) el.value = formatted
}
// A touch that scrolled the page instead of opening the box puts it back. Browsers that take
// over a scroll may not send its end, so a touch is also checked a moment after it starts (a tap
// has opened the box by then; a slow one switches again when it does).
function backIfNotTyping(after = 400) {
  setTimeout(() => { if (asNumber.value && document.activeElement !== input.value) toText() }, after)
}
function onTouchStart() {
  toNumber()
  backIfNotTyping(700)
}
function onFocus() {
  focused.value = true
  toNumber() // also when reached with the keyboard's arrows
}
// A number box takes digits, one minus and one point; other keys (a 1,000 comma, + * …) do nothing.
function onBeforeInput(event: Event) {
  const data = (event as InputEvent).data
  if (asNumber.value && data && /[^\d.-]/.test(data)) event.preventDefault()
}
// Digits past the currency's limits (whole digits, decimals) in a number box.
function overLimit(typed: string, currency: SupportedCurrency) {
  const [whole = '', fraction] = typed.replace(/^-/, '').split('.')
  return whole.replace(/^0+(?=\d)/, '').length > wholeLimit(currency) || (fraction !== undefined && fraction.length > decimalsOf(currency))
}
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
  if (asNumber.value) {
    // Not a number yet (a lone minus): nothing until the digits come.
    if (element.validity.badInput) {
      displayValue.value = ''
      emit('update:modelValue', '')
      return
    }
    const raw = normalize(element.value, props.currency)
    if (overLimit(element.value, props.currency)) element.value = raw
    displayValue.value = element.value
    emit('update:modelValue', raw === '-' ? '' : raw)
    return
  }
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
  if (asNumber.value) toText()
  const formatted = formatEntry(props.modelValue, props.currency, true)
  displayValue.value = formatted
  if (input.value && input.value.value !== formatted) input.value.value = formatted
  if (formatted) emit('update:modelValue', normalize(formatted, props.currency))
}

watch(() => [props.modelValue, props.currency], ([value, currency]) => {
  if (!focused.value && !asNumber.value) displayValue.value = formatEntry(String(value), currency as SupportedCurrency, true)
})

watch(() => props.currency, (currency, previousCurrency) => {
  if (previousCurrency && currency !== previousCurrency) {
    const amount = normalize(props.modelValue, currency)
    emit('update:modelValue', amount)
    displayValue.value = formatEntry(amount, currency)
  }
})
</script>
