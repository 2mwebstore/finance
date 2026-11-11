<template>
  <input
    v-bind="$attrs"
    ref="input"
    :value="displayValue"
    type="text"
    :inputmode="keypad ? 'none' : 'decimal'"
    autocomplete="off"
    :placeholder="decimalsOf(currency) ? `0.${'0'.repeat(decimalsOf(currency))}` : '0'"
    :aria-label="ariaLabel"
    @focus="onFocus"
    @input="onInput"
    @blur="onBlur"
  />
  <!-- A phone's keypad for a signed amount: big keys, digits and the signs only -->
  <Teleport v-if="keypad" to="body">
    <div v-if="open" ref="pad" class="amount-keypad" role="group" :aria-label="`Keypad${ariaLabel ? ` for ${ariaLabel}` : ''}`" @pointerdown.prevent @mousedown.prevent>
      <div class="amount-keypad-keys">
        <button
          v-for="k in KEYS" :key="k.key" type="button" tabindex="-1" class="amount-key"
          :class="[k.class, k.key === '-' && typed.startsWith('-') ? 'is-on' : '']"
          :aria-label="k.label" :aria-pressed="k.key === '-' ? typed.startsWith('-') : undefined"
          :disabled="k.key === '.' && !decimalsOf(currency)"
          @pointerdown.prevent="k.key !== 'done' && press(k.key)" @click="(e: MouseEvent) => { if (k.key === 'done' || !e.detail) press(k.key) }"
        >
          <Delete v-if="k.key === 'back'" :size="24" aria-hidden="true" /><template v-else>{{ k.text }}</template>
        </button>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { Delete } from 'lucide-vue-next'
import { currencyMeta, type SupportedCurrency } from '~/utils/money'

// Decimals come from Manage → Currency (USD 2, KHR 0…). Whole digits stop at 12 for currencies
// without decimals and 9 otherwise (inside the API's maximum).
function decimalsOf(currency: SupportedCurrency) {
  return currencyMeta(currency).decimals
}
function wholeLimit(currency: SupportedCurrency) {
  return decimalsOf(currency) ? 9 : 12
}

// The page's class, disabled and aria-* go on the box (the keypad beside it takes none).
defineOptions({ inheritAttrs: false })

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

// A signed amount on a phone: phones' big number pads have no minus (an iPhone's has only digits
// and a point), and their other keyboards have letters. So the phone's keyboard stays closed
// (inputmode none) and the box opens the app's own keypad: 1–9, 0, a point (not for KHR), a minus
// that makes the amount a loss and back, delete and Done. An amount already in the box is chosen
// whole, so the first key starts it again (delete clears it). Computers keep the text box.
const keypad = props.signed && import.meta.client && !!window.matchMedia?.('(pointer: coarse)').matches
const KEYS = [
  ...['1', '2', '3', '4', '5', '6', '7', '8', '9'].map(d => ({ key: d, text: d, label: d, class: '' })),
  { key: '-', text: '−', label: 'Minus (a loss)', class: 'is-sign' },
  { key: '0', text: '0', label: '0', class: '' },
  { key: '.', text: '.', label: 'Point', class: 'is-sign' },
  { key: 'done', text: 'Done', label: 'Done', class: 'is-done' },
  { key: 'back', text: '', label: 'Delete', class: 'is-sign' },
]
const open = ref(false)
const pad = ref<HTMLElement | null>(null)
// What is typed with the keypad (a lone minus too), and whether the box's amount is still chosen whole.
const typed = ref('')
let whole = false
function openKeypad() {
  const el = input.value
  if (!keypad || !el || el.disabled || el.readOnly) return
  typed.value = normalize(props.modelValue, props.currency)
  whole = !!typed.value
  open.value = true
  document.addEventListener('click', tapOutside, true)
  nextTick(() => {
    if (whole) el.setSelectionRange(0, el.value.length)
    // The keypad's height is kept free under the page, so the last agents can scroll above it, and
    // the box stays in sight.
    const space = pad.value?.offsetHeight || 330
    document.body.style.paddingBottom = `${space}px`
    const box = el.getBoundingClientRect()
    const free = window.innerHeight - space - 12
    if (box.bottom > free) window.scrollBy({ top: box.bottom - free, behavior: 'smooth' })
    else if (box.top < 72) window.scrollBy({ top: box.top - 72, behavior: 'smooth' })
  })
}
function closeKeypad() {
  if (!open.value) return
  open.value = false
  document.removeEventListener('click', tapOutside, true)
  document.body.style.paddingBottom = ''
}
// A tap anywhere else (a Save button, the page) leaves the box and closes the keypad; scrolling
// the page doesn't.
function tapOutside(event: MouseEvent) {
  const target = event.target as Node | null
  if (target && (input.value?.contains(target) || pad.value?.contains(target))) return
  input.value?.blur()
}
// Keys act as soon as they are touched (and from a keyboard's Enter). Done waits for the tap to end:
// the keypad closing under the finger would pass the tap on to what is beneath (the bottom bar).
function press(key: string) {
  const el = input.value
  if (!el) return
  if (key === 'done') {
    el.blur()
    return
  }
  let negative = typed.value.startsWith('-')
  let amount = typed.value.replace(/^-/, '')
  // The first key replaces an amount chosen whole.
  if (whole) {
    negative = false
    amount = ''
    whole = false
  }
  if (key === 'back') {
    if (amount) amount = amount.slice(0, -1)
    else negative = false
  } else if (key === '-') {
    negative = !negative
  } else if (key === '.') {
    if (decimalsOf(props.currency) && !amount.includes('.')) amount = `${amount || '0'}.`
  } else {
    const next = amount === '0' ? key : amount + key
    if (!overLimit(next, props.currency)) amount = next
  }
  typed.value = `${negative ? '-' : ''}${amount}`
  const shown = formatEntry(typed.value, props.currency)
  displayValue.value = shown
  el.value = shown
  el.setSelectionRange(shown.length, shown.length)
  emit('update:modelValue', amount ? normalize(typed.value, props.currency) : '')
}
onBeforeUnmount(closeKeypad)
// Digits past the currency's limits (whole digits, decimals).
function overLimit(value: string, currency: SupportedCurrency) {
  const [digits = '', fraction] = value.replace(/^-/, '').split('.')
  return digits.replace(/^0+(?=\d)/, '').length > wholeLimit(currency) || (fraction !== undefined && fraction.length > decimalsOf(currency))
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

function onFocus() {
  focused.value = true
  openKeypad()
}

function onInput(event: Event) {
  const element = event.target as HTMLInputElement
  // A key typed some other way (a phone's own or a plugged-in keyboard) while the keypad is open.
  whole = false
  const rawCaret = rawCaretLength(element.value.slice(0, element.selectionStart ?? element.value.length))
  const raw = normalize(element.value, props.currency)
  const formatted = formatEntry(raw, props.currency)
  displayValue.value = formatted
  // A refused character leaves the value as it was, so put the box back by hand.
  if (element.value !== formatted) element.value = formatted
  emit('update:modelValue', raw === '-' ? '' : raw)
  typed.value = raw
  nextTick(() => {
    const caret = caretForRawLength(formatted, rawCaret)
    element.setSelectionRange(caret, caret)
  })
}

function onBlur() {
  focused.value = false
  closeKeypad()
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

<style scoped>
/* Like a phone's number pad: big keys on grey, above the app's bottom bar, under pop-ups. */
.amount-keypad { position: fixed; right: 0; bottom: 0; left: 0; z-index: 70; padding: 6px 6px calc(6px + env(safe-area-inset-bottom)); background: #d1d5db; box-shadow: 0 -4px 16px rgb(15 23 42 / 12%); touch-action: manipulation; user-select: none; -webkit-user-select: none; }
.amount-keypad-keys { display: grid; grid-template-columns: repeat(3, 1fr); gap: 6px; max-width: 520px; margin: 0 auto; }
.amount-key { display: flex; align-items: center; justify-content: center; height: 52px; border-radius: 8px; background: #fff; color: #0f172a; font-size: 26px; font-weight: 500; box-shadow: 0 1px 0 rgb(15 23 42 / 25%); -webkit-tap-highlight-color: transparent; }
.amount-key:active { background: #e2e8f0; }
.amount-key:disabled { opacity: .35; }
.amount-key.is-sign { background: #e5e7eb; }
.amount-key.is-on { background: #e2445c; color: #fff; }
.amount-key.is-done { grid-column: span 2; background: #2563eb; color: #fff; font-size: 18px; font-weight: 700; }
.amount-key.is-done:active { background: #1d4ed8; }
</style>
