<template>
  <div ref="root" class="date-range-picker" :class="{ open }">
    <button type="button" class="range-trigger" :aria-expanded="open" aria-haspopup="dialog" @click="toggle">
      <CalendarDays :size="15" />
      <span>{{ rangeLabel }}</span>
      <ChevronDown :size="14" class="range-chevron" />
    </button>
    <div v-if="open && isMobile" class="range-backdrop" @click="close"></div>
    <div v-if="open" class="range-popover" :class="{ 'align-right': alignRight }" role="dialog" aria-label="Choose date range">
      <div v-if="isMobile" class="range-sheet-header">
        <span class="range-sheet-handle"></span>
        <strong>Select date range</strong>
        <small>{{ draftLabel }}</small>
      </div>
      <div class="range-presets">
        <button v-for="preset in presets" :key="preset.label" type="button" @click="setDraft(preset.getRange())">{{ preset.label }}</button>
      </div>
      <div class="range-picker-scroll">
        <VDatePicker v-if="isMobile" :model-value="draftDates" multiple="range" color="primary" width="100%" hide-header @update:model-value="onPickerUpdate" />
        <VDateRangePicker v-else :model-value="draftDates" color="primary" :width="690" hide-header @update:model-value="onPickerUpdate" />
      </div>
      <p v-if="invalidRange" class="range-error">End date must be on or after the start date.</p>
      <div class="range-actions">
        <button type="button" class="range-clear" @click="clear">Clear</button>
        <span class="range-hint">Choose a start and end date</span>
        <button type="button" class="range-cancel" @click="close">Close</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useMediaQuery } from '@vueuse/core'
import { CalendarDays, ChevronDown } from 'lucide-vue-next'
import { VDatePicker } from 'vuetify/components/VDatePicker'
import { VDateRangePicker } from 'vuetify/labs/VDateRangePicker'
import type { DateRangeValue } from '~/utils/dateRanges'
import { currentMonthRange, formatDateKey } from '~/utils/dateRanges'

const props = defineProps<{ modelValue: DateRangeValue }>()
const emit = defineEmits<{ 'update:modelValue': [value: DateRangeValue] }>()
const root = ref<HTMLElement | null>(null)
const open = ref(false)
const alignRight = ref(false)
// Phones get a single-month bottom sheet; the two-month picker needs ~690px.
const isMobile = useMediaQuery('(max-width: 700px)')
const draftDates = ref<Array<Date | string>>([])
const selectedKeys = computed(() => draftDates.value.map(dateKey).filter(Boolean))
const invalidRange = computed(() => selectedKeys.value.length > 1 && selectedKeys.value.at(-1)! < selectedKeys.value[0]!)
const rangeLabel = computed(() => props.modelValue.from && props.modelValue.to
  ? `${formatDateKey(props.modelValue.from)} – ${formatDateKey(props.modelValue.to)}`
  : 'Select date range')
const draftLabel = computed(() => {
  const [from, to] = [selectedKeys.value[0], selectedKeys.value.at(-1)]
  if (!from) return 'Tap a start date'
  if (selectedKeys.value.length < 2) return `${formatDateKey(from)} – tap an end date`
  return `${formatDateKey(from)} – ${formatDateKey(to!)}`
})

const presets = [
  { label: 'This month', getRange: () => currentMonthRange() },
  { label: 'Last month', getRange: () => monthRange(-1) },
  { label: 'Last 30 days', getRange: () => recentRange(30) },
  { label: 'Today', getRange: () => recentRange(1) },
]

function dateKey(value: Date | string) {
  if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}/.test(value)) return value.slice(0, 10)
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.valueOf())) return ''
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function pickerDates(range: DateRangeValue): Array<Date | string> {
  if (!range.from || !range.to) return []
  return [range.from, range.to].map(value => new Date(`${value}T12:00:00`))
}

function monthRange(offset: number): DateRangeValue {
  const now = new Date()
  const first = new Date(now.getFullYear(), now.getMonth() + offset, 1)
  const last = new Date(first.getFullYear(), first.getMonth() + 1, 0)
  return { from: dateKey(first), to: dateKey(last) }
}

function recentRange(days: number): DateRangeValue {
  const end = new Date()
  const start = new Date(end.getFullYear(), end.getMonth(), end.getDate() - days + 1)
  return { from: dateKey(start), to: dateKey(end) }
}

function toggle() {
  open.value = !open.value
  // Open leftwards when a trigger near the right edge (e.g. the dashboard filter panel) has no room.
  const left = root.value?.getBoundingClientRect().left ?? 0
  alignRight.value = !isMobile.value && left + 720 > window.innerWidth
  if (open.value) draftDates.value = pickerDates(props.modelValue)
}

function setDraft(range: DateRangeValue) {
  draftDates.value = pickerDates(range)
  applyDraft()
}

function onPickerUpdate(value: Array<Date | string>) {
  draftDates.value = value
  if (value.length >= 2) applyDraft()
}

function applyDraft() {
  if (invalidRange.value || selectedKeys.value.length < 2) return
  emit('update:modelValue', { from: selectedKeys.value[0]!, to: selectedKeys.value.at(-1)! })
  open.value = false
}

function clear() {
  emit('update:modelValue', { from: '', to: '' })
  draftDates.value = []
  open.value = false
}

function close() {
  open.value = false
}

watch(() => props.modelValue, range => { draftDates.value = pickerDates(range) }, { deep: true })
onClickOutside(root, () => { open.value = false })
</script>

<style scoped>
.date-range-picker { position: relative; min-width: 0; }
.range-trigger { display: flex; width: 100%; min-height: 37px; align-items: center; gap: 8px; border: 1px solid #dfe7f1; border-radius: 8px; padding: 8px 10px; color: #3e587b; background: #fff; font: inherit; font-size: 12px; text-align: left; cursor: pointer; }
.range-trigger > svg:first-child { flex: 0 0 auto; color: #2979ed; }
.range-trigger > span { overflow: hidden; flex: 1; text-overflow: ellipsis; white-space: nowrap; }
.range-chevron { flex: 0 0 auto; color: #7b8da7; }
.range-popover { position: absolute; z-index: 90; top: calc(100% + 6px); left: 0; width: min(720px, calc(100vw - 24px)); overflow: hidden; border: 1px solid #dfe7f1; border-radius: 12px; padding: 11px; background: #fff; box-shadow: 0 16px 36px rgb(20 48 86 / 17%); }
.range-popover.align-right { right: 0; left: auto; }
.range-presets { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 5px; }
.range-presets button { border: 1px solid #e5ebf3; border-radius: 7px; padding: 7px 8px; color: #536b8d; background: #f9fbfd; font-size: 11px; text-align: center; cursor: pointer; }
.range-presets button:hover { border-color: #bdd5f7; color: #1769d2; background: #f0f6ff; }
.range-picker-scroll { max-width: 100%; margin-top: 8px; overflow: auto; }
.range-picker-scroll :deep(.v-date-range-picker) { max-width: none; }
.range-picker-scroll :deep(.v-date-picker-controls) { font-size: 14px; }
.range-picker-scroll :deep(.v-date-picker-month__weekday) { color: #7184a2; font-size: 12px; }
.range-picker-scroll :deep(.v-date-picker-month__day-btn) { font-size: 13px; }
.range-error { margin: 8px 0 0; color: #c33d50; font-size: 11px; }
.range-actions { display: flex; justify-content: flex-end; gap: 7px; margin-top: 5px; padding-top: 9px; border-top: 1px solid #edf1f6; }
.range-actions button { border: 1px solid #e0e7f1; border-radius: 7px; padding: 7px 11px; color: #667b99; background: #fff; font-size: 11px; font-weight: 700; cursor: pointer; }
.range-actions .range-clear { margin-right: auto; }
.range-hint { margin-right: auto; color: #93a1b5; font-size: 11px; }
.range-backdrop { position: fixed; inset: 0; z-index: 89; background: rgb(10 35 75 / 42%); backdrop-filter: blur(2px); }
@media (max-width: 700px) {
  .range-trigger { min-height: 42px; padding: 10px 12px; font-size: 13px; }
  .range-popover { position: fixed; top: auto; right: 0; bottom: 0; left: 0; display: flex; width: auto; max-height: calc(100dvh - 40px); flex-direction: column; overflow-y: auto; border: 0; border-radius: 20px 20px 0 0; padding: 8px 16px calc(14px + env(safe-area-inset-bottom)); box-shadow: 0 -12px 32px rgb(20 48 86 / 18%); animation: sheet-up .22s ease-out; }
  .range-sheet-header { display: flex; flex-direction: column; align-items: center; gap: 4px; padding-bottom: 12px; text-align: center; }
  .range-sheet-handle { width: 38px; height: 4px; margin-bottom: 8px; border-radius: 999px; background: #d6dfeb; }
  .range-sheet-header strong { color: #162a4d; font-size: 16px; font-weight: 750; }
  .range-sheet-header small { color: #6d809e; font-size: 13px; }
  .range-presets { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
  .range-presets button { min-height: 40px; border-radius: 10px; font-size: 13px; font-weight: 600; }
  .range-picker-scroll { flex: 0 0 auto; margin-top: 6px; overflow: visible; }
  .range-picker-scroll :deep(.v-date-picker-month) { padding-inline: 0; }
  .range-picker-scroll :deep(.v-date-picker-controls) { padding-inline: 4px; font-size: 16px; }
  .range-picker-scroll :deep(.v-date-picker-month__weekday) { font-size: 13px; }
  .range-picker-scroll :deep(.v-date-picker-month__day-btn) { font-size: 15px; }
  .range-error { font-size: 13px; text-align: center; }
  .range-actions { gap: 10px; padding-top: 12px; }
  .range-actions button { flex: 1; min-height: 44px; border-radius: 10px; font-size: 14px; }
  .range-actions .range-clear { margin-right: 0; }
  .range-hint { display: none; }
}
@keyframes sheet-up { from { transform: translateY(100%); } }
</style>
