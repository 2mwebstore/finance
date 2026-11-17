<template>
  <!-- "10%" is a percent, "10" (no %) an amount in the currency; the sign on the right says which. -->
  <span class="flex overflow-hidden rounded-lg border bg-white focus-within:border-primary" :class="invalid ? 'border-expense' : 'border-border'">
    <input
      ref="inputEl"
      :value="text"
      type="text"
      inputmode="decimal"
      autocomplete="off"
      placeholder="0 or 10%"
      class="min-w-0 flex-1 border-0 bg-transparent px-2.5 text-sm text-navy outline-none"
      :class="inputClass"
      :aria-label="ariaLabel"
      :aria-invalid="invalid ? 'true' : undefined"
      @input="onInput(($event.target as HTMLInputElement).value)"
    />
    <!-- Phones' number pads have no % key: tapping the sign switches. -->
    <button type="button" class="discount-sign" :class="mode === 'percent' ? 'is-percent' : ''" :title="mode === 'percent' ? 'Percent (type a number without % for an amount, or tap)' : 'Amount (type 10% for a percent, or tap)'" :aria-label="mode === 'percent' ? 'Percent discount: switch to an amount' : 'Amount discount: switch to a percent'" @click="switchMode">{{ mode === 'percent' ? '%' : symbol }}</button>
  </span>
</template>

<script setup lang="ts">
import type { Currency, DiscountMode } from '~/types/api'
import { currencyMeta } from '~/utils/money'

/**
 * A discount typed as text: a number ending in % ("10%") is a percent, a plain number ("10") an
 * amount. It gives back the number (without %) and the mode, like the old amount/% switch.
 */
const props = withDefaults(defineProps<{
  value: string
  mode: DiscountMode
  currency: Currency
  ariaLabel?: string
  invalid?: boolean
  /** Height and padding of the box, e.g. "py-2" or "h-9". */
  inputClass?: string
}>(), { ariaLabel: 'Discount', invalid: false, inputClass: 'py-2' })
// `set` gives both at once, for a parent that copies a whole list on each change (two separate
// updates there would each start from the old list, and the second would undo the first).
const emit = defineEmits<{ 'update:value': [value: string]; 'update:mode': [mode: DiscountMode]; set: [discount: { value: string; mode: DiscountMode }] }>()

const symbol = computed(() => currencyMeta(props.currency).symbol.trim())
const inputEl = ref<HTMLInputElement | null>(null)
defineExpose({ focus: () => inputEl.value?.focus() })

function show(value: string, mode: DiscountMode) {
  return value ? (mode === 'percent' ? `${value}%` : value) : ''
}
function parse(raw: string): { value: string; mode: DiscountMode } {
  const t = raw.replace(/\s+/g, '')
  return t.endsWith('%') ? { value: t.slice(0, -1), mode: 'percent' } : { value: t, mode: 'amount' }
}

// What is typed stays as typed; it follows the numbers when they change from outside (a loaded
// document, or prices converted to another currency).
const text = ref(show(props.value, props.mode))
watch(() => [props.value, props.mode] as const, ([value, mode]) => {
  const now = parse(text.value)
  if (now.value !== value || now.mode !== mode) text.value = show(value, mode)
})

function onInput(raw: string) {
  text.value = raw
  const next = parse(raw)
  emit('update:value', next.value)
  if (next.mode !== props.mode) emit('update:mode', next.mode)
  emit('set', next)
}

function switchMode() {
  const mode: DiscountMode = props.mode === 'percent' ? 'amount' : 'percent'
  text.value = show(props.value, mode) || (mode === 'percent' ? '%' : '')
  emit('update:mode', mode)
  emit('set', { value: props.value, mode })
}
</script>

<style scoped>
.discount-sign { flex: 0 0 auto; min-width: 34px; border-left: 1px solid #e2e8f0; background: #f5f8ff; padding: 0 10px; color: #64748b; font-size: 12px; font-weight: 700; }
.discount-sign.is-percent { color: #2563eb; }
</style>
