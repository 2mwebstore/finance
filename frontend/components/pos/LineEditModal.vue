<template>
  <AppModal :model-value="!!line" form :title="title" eyebrow="ITEM" :subtitle="line ? `${formatQty(line.quantity)} × ${formatMoney(Number(line.price) || 0, currency)}` : ''" width="420px" @update:model-value="(open) => { if (!open) emit('close') }" @submit="save">
    <div class="space-y-4">
      <div class="space-y-1.5 text-xs font-semibold text-navy">
        Discount
        <span class="font-normal text-muted">(10% = percent, 10 = amount)</span>
        <InventoryDiscountInput ref="discountEl" v-model:value="discount" v-model:mode="mode" class="font-normal" :currency="(currency as Currency)" aria-label="Discount" :invalid="!!error" />
        <p v-if="off" class="text-[11px] font-normal text-muted">−{{ formatMoney(off, currency) }} on this item</p>
      </div>
      <label class="block space-y-1.5 text-xs font-semibold text-navy">
        Note <span class="font-normal text-muted">(optional)</span>
        <input v-model="note" type="text" maxlength="255" autocomplete="off" class="w-full rounded-lg border border-border px-3 py-2 text-sm font-normal text-navy" placeholder="e.g. less sugar, gift wrap" />
      </label>
      <AppAlert v-if="error" compact :message="error" />
    </div>
    <template #footer>
      <button type="button" class="border border-border bg-white text-muted" @click="emit('close')">Cancel</button>
      <button type="submit" class="bg-primary text-white">Save</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import type { Currency, DiscountMode, DocLineForm } from '~/types/api'
import { sentence } from '~/composables/useToast'
import { formatMoney } from '~/utils/money'
import { formatQty, lineDiscount, linesCheck } from '~/utils/inventory'

/** A cart line's discount (percent or amount) and note, edited together. */
const props = defineProps<{ line: DocLineForm | null; title: string; currency: string }>()
const emit = defineEmits<{ close: []; save: [patch: Pick<DocLineForm, 'discount' | 'discount_mode' | 'note'>] }>()

const discount = ref('')
const mode = ref<DiscountMode>('amount')
const note = ref('')
const error = ref('')
const discountEl = ref<{ focus: () => void } | null>(null)
const off = computed(() => props.line ? lineDiscount({ ...props.line, discount: discount.value, discount_mode: mode.value }, props.currency) : 0)

watch(() => props.line, (line) => {
  if (!line) return
  discount.value = line.discount
  mode.value = line.discount_mode
  note.value = line.note || ''
  error.value = ''
  nextTick(() => discountEl.value?.focus())
}, { immediate: true })

function save() {
  if (!props.line) return
  // Same rules as the sale form's lines (a number, at most 100% or the line's amount).
  error.value = sentence(linesCheck([{ ...props.line, discount: discount.value, discount_mode: mode.value }]))
  if (error.value) return
  emit('save', { discount: discount.value, discount_mode: mode.value, note: note.value.trim() })
}
</script>
