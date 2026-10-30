<template>
  <div class="grid min-w-0 grid-cols-1 gap-2" :class="mode === 'custom' ? 'sm:grid-cols-2' : ''">
    <SearchSelect :model-value="mode" :options="options" :searchable="false" :aria-label="label" @update:model-value="choose" />
    <DateRangePicker v-if="mode === 'custom'" :model-value="modelValue" @update:model-value="(range) => emit('update:modelValue', range)" />
  </div>
</template>

<script setup lang="ts">
import type { DateRangeValue } from '~/utils/dateRanges'
import { todayISO } from '~/utils/inventory'
import { mondayOf, plusDays } from '~/utils/winlose'

/**
 * A date filter: Today, Yesterday, This week, Last week, This month, Last month or a custom range
 * (weeks run Monday to Sunday). With `allowAll`, "All dates" is an empty range.
 */
const props = withDefaults(defineProps<{ modelValue: DateRangeValue; allowAll?: boolean; label?: string }>(), { allowAll: false, label: 'Dates' })
const emit = defineEmits<{ 'update:modelValue': [value: DateRangeValue] }>()

type Preset = 'today' | 'yesterday' | 'this_week' | 'last_week' | 'this_month' | 'last_month'

const dayKey = (d: Date) => `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
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
const options = computed(() => [
  ...(props.allowAll ? [{ value: 'all', label: 'All dates' }] : []),
  { value: 'today', label: 'Today' },
  { value: 'yesterday', label: 'Yesterday' },
  { value: 'this_week', label: 'This week' },
  { value: 'last_week', label: 'Last week' },
  { value: 'this_month', label: 'This month' },
  { value: 'last_month', label: 'Last month' },
  { value: 'custom', label: 'Custom range' },
])

// The preset the range matches, unless the person chose Custom range (then the calendar shows).
const customChosen = ref(false)
const mode = computed(() => {
  const { from, to } = props.modelValue
  if (customChosen.value) return 'custom'
  if (!from && !to) return props.allowAll ? 'all' : 'custom'
  const match = (Object.keys(ranges.value) as Preset[]).find(p => ranges.value[p].from === from && ranges.value[p].to === to)
  return match || 'custom'
})

function choose(value: string | string[]) {
  const picked = String(value)
  customChosen.value = picked === 'custom'
  if (picked === 'custom') return // the current range stays; the calendar next to it changes it
  emit('update:modelValue', picked === 'all' ? { from: '', to: '' } : { ...ranges.value[picked as Preset] })
}
</script>
