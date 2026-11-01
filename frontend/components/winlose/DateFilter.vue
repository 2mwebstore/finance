<template>
  <div ref="root" class="relative min-w-0">
    <button type="button" class="flex h-10 w-full items-center gap-2 rounded-lg border border-border bg-white px-3 text-left text-sm text-navy hover:bg-canvas" :aria-label="label" :aria-expanded="open" aria-haspopup="dialog" @click="toggle">
      <CalendarDays :size="15" class="shrink-0 text-muted" />
      <span class="min-w-0 flex-1 truncate tabular-nums">{{ triggerText }}</span>
      <ChevronDown :size="14" class="shrink-0 text-muted transition-transform" :class="open ? 'rotate-180' : ''" />
    </button>

    <!-- The ranges; Custom Range adds the calendars (two months). On phones a sheet above the menu
         bar: the ranges, or the calendar (one month) once Custom Range is chosen. -->
    <div v-if="open" class="fixed inset-0 z-40 bg-navy/20 sm:hidden" @click="close" />
    <div v-if="open" class="z-50 flex flex-col overflow-hidden rounded-xl border border-border bg-white shadow-lg max-sm:fixed max-sm:inset-x-3 max-sm:bottom-24 max-sm:max-h-[calc(100dvh-8rem)] max-sm:overflow-y-auto sm:absolute sm:top-full sm:mt-2 sm:flex-row" :class="alignRight ? 'sm:right-0' : 'sm:left-0'" role="dialog" :aria-label="label">
      <ul class="py-1 sm:w-40 sm:shrink-0" :class="custom ? 'max-sm:hidden sm:border-r sm:border-border' : ''">
        <li v-for="p in presetList" :key="p.value">
          <button type="button" class="w-full whitespace-nowrap px-4 py-2.5 text-left text-sm" :class="p.value === active ? 'bg-primary font-semibold text-white' : 'text-navy hover:bg-canvas'" @click="pick(p.value)">{{ p.label }}</button>
        </li>
      </ul>
      <div v-if="custom" class="p-3">
        <div class="flex gap-6">
          <div v-for="(m, i) in shownMonths" :key="m.key" class="w-full sm:w-[232px]">
            <div class="mb-1 flex items-center justify-between">
              <button v-if="i === 0" type="button" class="grid h-8 w-8 place-items-center rounded-lg text-navy hover:bg-canvas" aria-label="Previous month" @click="shift(-1)"><ChevronLeft :size="16" /></button>
              <span v-else class="w-8" />
              <strong class="text-sm text-navy">{{ m.title }}</strong>
              <button v-if="i === shownMonths.length - 1" type="button" class="grid h-8 w-8 place-items-center rounded-lg text-navy hover:bg-canvas" aria-label="Next month" @click="shift(1)"><ChevronRight :size="16" /></button>
              <span v-else class="w-8" />
            </div>
            <div class="grid grid-cols-7 text-center">
              <span v-for="d in WEEKDAYS" :key="d" class="py-1 text-[11px] font-semibold text-slate-500">{{ d }}</span>
              <button v-for="day in m.days" :key="day.key" type="button" class="h-8 text-xs tabular-nums" :class="dayClass(day)" :aria-label="us(day.key)" @click="pickDay(day.key)">{{ day.n }}</button>
            </div>
          </div>
        </div>
        <div class="mt-3 flex flex-wrap items-center justify-end gap-2 border-t border-border pt-3">
          <span class="mr-auto text-xs tabular-nums text-muted">{{ draftText }}</span>
          <button type="button" class="rounded-lg px-3 py-1.5 text-sm font-semibold text-muted hover:bg-canvas" @click="close">Cancel</button>
          <button type="button" class="rounded-lg bg-primary px-4 py-1.5 text-sm font-semibold text-white hover:bg-primary-hover disabled:opacity-50" :disabled="!draft.from" @click="apply">Apply</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useEventListener, useMediaQuery } from '@vueuse/core'
import { CalendarDays, ChevronDown, ChevronLeft, ChevronRight } from 'lucide-vue-next'
import type { DateRangeValue } from '~/utils/dateRanges'
import { todayISO } from '~/utils/inventory'
import { mondayOf, plusDays } from '~/utils/winlose'

/**
 * A date filter: Today, Yesterday, This Week, Last Week, This Month, Last Month (weeks run Monday
 * to Sunday) or Custom Range, picked on a calendar and applied. With `allowAll`, "All dates" is
 * an empty range.
 */
const props = withDefaults(defineProps<{ modelValue: DateRangeValue; allowAll?: boolean; label?: string }>(), { allowAll: false, label: 'Dates' })
const emit = defineEmits<{ 'update:modelValue': [value: DateRangeValue] }>()

type Preset = 'today' | 'yesterday' | 'this_week' | 'last_week' | 'this_month' | 'last_month'
const WEEKDAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa']

const dayKey = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
const parse = (key: string) => new Date(`${key}T12:00:00`)
// 2026-11-01 → 11/01/2026
const us = (key: string) => key ? `${key.slice(5, 7)}/${key.slice(8, 10)}/${key.slice(0, 4)}` : ''
function monthRange(offset: number): DateRangeValue {
  const now = new Date()
  return { from: dayKey(new Date(now.getFullYear(), now.getMonth() + offset, 1)), to: dayKey(new Date(now.getFullYear(), now.getMonth() + offset + 1, 0)) }
}
const ranges = computed<Record<Preset, DateRangeValue>>(() => {
  const today = todayISO()
  const monday = mondayOf(today)
  return {
    today: { from: today, to: today },
    yesterday: { from: plusDays(today, -1), to: plusDays(today, -1) },
    this_week: { from: monday, to: plusDays(monday, 6) },
    last_week: { from: plusDays(monday, -7), to: plusDays(monday, -1) },
    this_month: monthRange(0),
    last_month: monthRange(-1),
  }
})
const presetList = computed(() => [
  ...(props.allowAll ? [{ value: 'all', label: 'All dates' }] : []),
  { value: 'today', label: 'Today' },
  { value: 'yesterday', label: 'Yesterday' },
  { value: 'this_week', label: 'This Week' },
  { value: 'last_week', label: 'Last Week' },
  { value: 'this_month', label: 'This Month' },
  { value: 'last_month', label: 'Last Month' },
  { value: 'custom', label: 'Custom Range' },
])

// The range chosen: a preset when it matches one, else custom.
const matched = computed(() => {
  const { from, to } = props.modelValue
  if (!from && !to) return props.allowAll ? 'all' : 'custom'
  return (Object.keys(ranges.value) as Preset[]).find(p => ranges.value[p].from === from && ranges.value[p].to === to) || 'custom'
})
const triggerText = computed(() => {
  const { from, to } = props.modelValue
  return from || to ? `${us(from)} - ${us(to || from)}` : props.allowAll ? 'All dates' : 'Choose dates'
})

const root = ref<HTMLElement | null>(null)
const open = ref(false)
const custom = ref(false)
const alignRight = ref(false)
const active = computed(() => custom.value ? 'custom' : matched.value)
const isPhone = useMediaQuery('(max-width: 639px)')
const draft = reactive({ from: '', to: '' })
const view = ref(new Date()) // the first month shown

function toggle() {
  if (open.value) return close()
  draft.from = props.modelValue.from
  draft.to = props.modelValue.to
  custom.value = matched.value === 'custom'
  const first = draft.from ? parse(draft.from) : new Date()
  view.value = new Date(first.getFullYear(), first.getMonth(), 1)
  // Near the right edge the panel opens to the left (presets and two months: about 680px).
  const rect = root.value?.getBoundingClientRect()
  alignRight.value = !!rect && rect.left + 680 > window.innerWidth
  open.value = true
}
function close() {
  open.value = false
  custom.value = false
}
onClickOutside(root, () => { if (open.value) close() })
useEventListener('keydown', (e: KeyboardEvent) => { if (e.key === 'Escape' && open.value) close() })

function pick(value: string) {
  if (value === 'custom') {
    custom.value = true
    return
  }
  emit('update:modelValue', value === 'all' ? { from: '', to: '' } : { ...ranges.value[value as Preset] })
  close()
}

// The first day tapped starts the range, the next one ends it (an earlier one starts it again).
function pickDay(key: string) {
  if (!draft.from || draft.to || key < draft.from) {
    draft.from = key
    draft.to = ''
  } else {
    draft.to = key
  }
}
function apply() {
  if (!draft.from) return
  emit('update:modelValue', { from: draft.from, to: draft.to || draft.from })
  close()
}
const draftText = computed(() => draft.from ? `${us(draft.from)} - ${draft.to ? us(draft.to) : '…'}` : 'Choose the first day')

function shift(months: number) {
  view.value = new Date(view.value.getFullYear(), view.value.getMonth() + months, 1)
}
// Each month as six weeks from Sunday; days of the months around it are greyed.
const shownMonths = computed(() => Array.from({ length: isPhone.value ? 1 : 2 }, (_, i) => {
  const first = new Date(view.value.getFullYear(), view.value.getMonth() + i, 1)
  const start = new Date(first.getFullYear(), first.getMonth(), 1 - first.getDay())
  const days = Array.from({ length: 42 }, (_, n) => {
    const d = new Date(start.getFullYear(), start.getMonth(), start.getDate() + n)
    return { key: dayKey(d), n: d.getDate(), inMonth: d.getMonth() === first.getMonth() }
  })
  return { key: dayKey(first), title: first.toLocaleDateString('en-US', { month: 'short', year: 'numeric' }), days }
}))

function dayClass(day: { key: string; inMonth: boolean }) {
  const end = draft.to || draft.from
  if (day.key === draft.from || day.key === end) return 'rounded-lg bg-primary font-semibold text-white'
  if (draft.from && day.key > draft.from && day.key < end) return 'bg-primary/10 text-navy'
  const today = day.key === todayISO() ? ' font-bold underline' : ''
  return (day.inMonth ? 'rounded-lg text-navy hover:bg-canvas' : 'rounded-lg text-slate-300 hover:bg-canvas') + today
}
</script>
