<template>
  <div ref="root" class="date-picker" :class="{ open }">
    <button type="button" class="date-trigger" :aria-label="ariaLabel" :aria-expanded="open" aria-haspopup="dialog" :disabled="disabled" @click="toggle">
      <CalendarDays :size="16" />
      <span>{{ label }}</span>
      <small v-if="modelValue === todayKey" class="date-today">Today</small>
      <ChevronDown :size="15" class="date-chevron" />
    </button>
    <!-- Teleported so a scrolling parent (e.g. a modal body) cannot clip the calendar. -->
    <Teleport to="body">
      <div v-if="open && isMobile" class="date-backdrop" @click="close"></div>
      <div v-if="open" ref="popover" class="date-popover" :style="popoverStyle" role="dialog" :aria-label="ariaLabel || 'Choose a date'" @keydown.esc="close">
        <div v-if="isMobile" class="date-sheet-header">
          <span class="date-sheet-handle"></span>
          <strong>Select date</strong>
          <small>{{ longLabel }}</small>
        </div>
        <div class="date-shortcuts">
          <button v-for="shortcut in shortcuts" :key="shortcut.label" type="button" :class="{ 'is-selected': modelValue === shortcut.value }" @click="pick(shortcut.value)">{{ shortcut.label }}</button>
        </div>
        <VDatePicker :model-value="pickerValue" color="primary" :width="isMobile ? '100%' : 312" hide-header @update:model-value="onPickerUpdate" />
        <div v-if="isMobile" class="date-actions">
          <button type="button" @click="close">Close</button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useEventListener, useMediaQuery } from '@vueuse/core'
import { CalendarDays, ChevronDown } from 'lucide-vue-next'
import { VDatePicker } from 'vuetify/components/VDatePicker'

const props = withDefaults(defineProps<{
  modelValue: string
  ariaLabel?: string
  placeholder?: string
  disabled?: boolean
}>(), {
  ariaLabel: '',
  placeholder: 'Select a date',
  disabled: false,
})
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const root = ref<HTMLElement | null>(null)
const popover = ref<HTMLElement | null>(null)
const open = ref(false)
const popoverStyle = ref<Record<string, string>>({})
// Same breakpoint as DateRangePicker: phones get a bottom sheet.
const isMobile = useMediaQuery('(max-width: 700px)')

const todayKey = dateKey(new Date())
const shortcuts = computed(() => {
  const now = new Date()
  return [
    { label: 'Today', value: todayKey },
    { label: 'Yesterday', value: dateKey(new Date(now.getFullYear(), now.getMonth(), now.getDate() - 1)) },
  ]
})
const pickerValue = computed(() => props.modelValue ? new Date(`${props.modelValue}T12:00:00`) : null)
const label = computed(() => formatDate({ month: 'short', day: 'numeric', year: 'numeric' }))
const longLabel = computed(() => formatDate({ weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' }))

function formatDate(options: Intl.DateTimeFormatOptions) {
  if (!props.modelValue) return props.placeholder
  const parsed = new Date(`${props.modelValue}T00:00:00`)
  return Number.isNaN(parsed.valueOf()) ? props.modelValue : new Intl.DateTimeFormat('en-US', options).format(parsed)
}

function dateKey(value: Date | string) {
  if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}/.test(value)) return value.slice(0, 10)
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.valueOf())) return ''
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function position() {
  if (!open.value || isMobile.value || !root.value) {
    popoverStyle.value = {}
    return
  }
  const rect = root.value.getBoundingClientRect()
  const width = popover.value?.offsetWidth || 336
  const height = popover.value?.offsetHeight || 420
  const spaceBelow = window.innerHeight - rect.bottom
  const top = spaceBelow < height + 12 && rect.top > spaceBelow
    ? Math.max(8, rect.top - height - 6)
    : rect.bottom + 6
  const left = Math.min(Math.max(8, rect.left), window.innerWidth - width - 8)
  popoverStyle.value = { top: `${top}px`, left: `${left}px` }
}

async function toggle() {
  if (props.disabled) return
  open.value = !open.value
  if (open.value) {
    position()
    await nextTick()
    position()
  }
}

function close() {
  open.value = false
}

function pick(value: string) {
  if (value) emit('update:modelValue', value)
  close()
}

function onPickerUpdate(value: unknown) {
  const picked = Array.isArray(value) ? value[0] : value
  if (picked instanceof Date || typeof picked === 'string') pick(dateKey(picked))
}

useEventListener(window, 'resize', position)
useEventListener(window, 'scroll', position, { capture: true, passive: true })
onClickOutside(root, () => { if (!isMobile.value) close() }, { ignore: [popover] })
</script>

<style scoped>
.date-picker { position: relative; min-width: 0; }
.date-trigger { display: flex; width: 100%; min-height: 44px; align-items: center; gap: 9px; border: 1px solid #dfe7f1; border-radius: 10px; padding: 10px 12px; color: #263f65; background: #fff; font: inherit; font-size: 14px; text-align: left; cursor: pointer; transition: border-color .15s ease, box-shadow .15s ease; }
.date-trigger:focus-visible, .date-picker.open .date-trigger { outline: none; border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.date-trigger:disabled { color: #8e9cb0; background: #f6f8fb; cursor: not-allowed; }
.date-trigger > svg:first-child { flex: 0 0 auto; color: #2979ed; }
.date-trigger > span { overflow: hidden; flex: 1; text-overflow: ellipsis; white-space: nowrap; }
.date-today { flex: 0 0 auto; border-radius: 999px; padding: 2px 8px; color: #1769e8; background: #eef5ff; font-size: 11px; font-weight: 700; }
.date-chevron { flex: 0 0 auto; color: #7b8da7; }
.date-popover { position: fixed; z-index: 130; overflow: hidden; border: 1px solid #dfe7f1; border-radius: 14px; padding: 10px 10px 4px; background: #fff; box-shadow: 0 16px 36px rgb(20 48 86 / 20%); }
.date-shortcuts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 6px; }
.date-shortcuts button { border: 1px solid #e5ebf3; border-radius: 8px; padding: 7px 8px; color: #536b8d; background: #f9fbfd; font-size: 12px; font-weight: 600; cursor: pointer; }
.date-shortcuts button:hover { border-color: #bdd5f7; color: #1769d2; background: #f0f6ff; }
.date-shortcuts button.is-selected { border-color: #8ab6fa; color: #1769e8; background: #eef5ff; }
.date-popover :deep(.v-date-picker-controls) { font-size: 14px; }
.date-popover :deep(.v-date-picker-month__weekday) { color: #7184a2; font-size: 12px; }
.date-popover :deep(.v-date-picker-month__day-btn) { font-size: 13px; }
.date-backdrop { position: fixed; inset: 0; z-index: 129; background: rgb(10 35 75 / 42%); backdrop-filter: blur(2px); }
@media (max-width: 700px) {
  .date-trigger { min-height: 48px; font-size: 15px; }
  .date-popover { right: 0; bottom: 0; left: 0; display: flex; max-height: calc(100dvh - 40px); flex-direction: column; overflow-y: auto; border: 0; border-radius: 20px 20px 0 0; padding: 8px 16px calc(14px + env(safe-area-inset-bottom)); box-shadow: 0 -12px 32px rgb(20 48 86 / 18%); animation: date-sheet-up .22s ease-out; }
  .date-sheet-header { display: flex; flex-direction: column; align-items: center; gap: 4px; padding-bottom: 12px; text-align: center; }
  .date-sheet-handle { width: 38px; height: 4px; margin-bottom: 8px; border-radius: 999px; background: #d6dfeb; }
  .date-sheet-header strong { color: #162a4d; font-size: 16px; font-weight: 750; }
  .date-sheet-header small { color: #6d809e; font-size: 13px; }
  .date-shortcuts { gap: 8px; }
  .date-shortcuts button { min-height: 40px; border-radius: 10px; font-size: 13px; }
  .date-popover :deep(.v-date-picker-month) { padding-inline: 0; }
  .date-popover :deep(.v-date-picker-controls) { padding-inline: 4px; font-size: 16px; }
  .date-popover :deep(.v-date-picker-month__weekday) { font-size: 13px; }
  .date-popover :deep(.v-date-picker-month__day-btn) { font-size: 15px; }
  .date-actions { display: flex; padding-top: 12px; border-top: 1px solid #edf1f6; }
  .date-actions button { flex: 1; min-height: 44px; border: 1px solid #e0e7f1; border-radius: 10px; color: #667b99; background: #fff; font-size: 14px; font-weight: 700; cursor: pointer; }
}
@keyframes date-sheet-up { from { transform: translateY(100%); } }
</style>
