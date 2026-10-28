<template>
  <AppModal :model-value="open" form title="Take payment" eyebrow="POS" width="480px" :busy="submitting" @update:model-value="(value) => { if (!value) emit('close') }" @submit="confirm">
    <div class="space-y-5">
      <div class="text-center">
        <p class="text-xs font-semibold text-muted">Amount due</p>
        <p class="mt-1 text-4xl font-bold text-navy">{{ formatMoney(total, currency) }}</p>
        <p v-if="otherCurrency" class="text-sm text-muted">≈ {{ formatMoney(convert(total, currency, otherCurrency), otherCurrency) }}</p>
      </div>

      <div class="flex flex-wrap gap-2" role="radiogroup" aria-label="Payment method">
        <button
          v-for="method in choices"
          :key="method.id"
          type="button"
          role="radio"
          :aria-checked="methodId === method.id"
          class="rounded-lg border px-3.5 py-2 text-sm font-semibold"
          :class="methodId === method.id ? 'border-primary bg-primary text-white' : 'border-border bg-white text-navy hover:bg-canvas'"
          @click="methodId = method.id"
        >{{ method.name }}</button>
      </div>

      <div v-if="isCash" class="space-y-4">
        <div class="grid gap-3" :class="khrActive ? 'grid-cols-2' : 'grid-cols-1'">
          <label class="block space-y-1 text-xs font-semibold text-muted">
            Received in USD
            <MoneyInput ref="usdEl" v-model="tenderUSD" currency="USD" class="w-full rounded-lg border border-border px-3 py-2.5 text-lg font-semibold text-navy" aria-label="Cash received in USD" />
          </label>
          <label v-if="khrActive" class="block space-y-1 text-xs font-semibold text-muted">
            Received in KHR
            <MoneyInput v-model="tenderKHR" currency="KHR" class="w-full rounded-lg border border-border px-3 py-2.5 text-lg font-semibold text-navy" aria-label="Cash received in riel" />
          </label>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="rounded-lg border border-border bg-white px-3 py-1.5 text-sm font-semibold text-navy hover:bg-canvas" @click="setExact">Exact</button>
          <button v-for="amount in [1, 5, 10, 20, 50, 100]" :key="amount" type="button" class="rounded-lg border border-border bg-white px-3 py-1.5 text-sm font-semibold text-navy hover:bg-canvas" @click="addUSD(amount)">+${{ amount }}</button>
        </div>
        <div class="flex items-center justify-between rounded-xl border px-3 py-3" :class="short > 0 ? 'border-rose-200 bg-rose-50' : 'border-emerald-200 bg-emerald-50'">
          <span class="text-sm font-semibold" :class="short > 0 ? 'text-expense' : 'text-income'">{{ short > 0 ? 'Still to pay' : 'Change' }}</span>
          <span class="text-right text-lg font-bold" :class="short > 0 ? 'text-expense' : 'text-income'">
            <template v-if="short > 0">{{ formatMoney(short, currency) }}<span v-if="otherCurrency" class="ml-1 text-sm font-medium">({{ formatMoney(convert(short, currency, otherCurrency), otherCurrency) }})</span></template>
            <template v-else>{{ changeLabel }}</template>
          </span>
        </div>
      </div>

      <label v-else class="block space-y-1 text-xs font-semibold text-muted">
        Reference <span class="font-normal">(optional)</span>
        <input v-model.trim="reference" type="text" maxlength="120" autocomplete="off" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm font-normal text-navy" placeholder="e.g. the bank transaction ID" />
      </label>
      <AppAlert v-if="error" compact :message="error" />
    </div>
    <template #footer>
      <button type="button" class="border border-border bg-white text-muted" :disabled="submitting" @click="emit('close')">Cancel</button>
      <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="submitting || (isCash && short > 0)">{{ submitting ? 'Saving…' : isCash ? 'Confirm cash' : `Confirm ${methodName}` }}</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import type { PosPayment } from '~/composables/usePosCart'
import { useCurrency } from '~/composables/useCurrency'
import { formatMoney } from '~/utils/money'
import { roundMoney } from '~/utils/inventory'

/**
 * Pay the cart: cash (any mix of USD and riel, with the change worked out) or another payment
 * method with an optional reference. Change is whole dollars plus the rest in riel, rounded to
 * ៛100; a riel sale gives its change in riel.
 */
const props = defineProps<{
  open: boolean
  total: number
  currency: string
  methods: { id: string; code: string; name: string }[]
  defaultMethodId: string
  submitting?: boolean
  /** Why the last try didn't save (e.g. not enough stock): the window stays open to fix it. */
  error?: string
}>()
const emit = defineEmits<{ close: []; paid: [payment: PosPayment] }>()

const { active, convert, rateOf } = useCurrency()
const khrActive = computed(() => active.value.some(c => c.code === 'KHR'))
const otherCurrency = computed(() => props.currency === 'USD' ? (khrActive.value ? 'KHR' : '') : 'USD')

// No methods set up: the API books an empty payment method as cash.
const choices = computed(() => props.methods.length ? props.methods : [{ id: '', code: 'cash', name: 'Cash' }])
const methodId = ref('')
const method = computed(() => choices.value.find(m => m.id === methodId.value))
const methodName = computed(() => method.value?.name || 'payment')
const isCash = computed(() => method.value?.code === 'cash')

const tenderUSD = ref('')
const tenderKHR = ref('')
const reference = ref('')
const usdEl = ref<{ $el?: HTMLElement } | null>(null)

watch(() => props.open, (open) => {
  if (!open) return
  methodId.value = choices.value.some(m => m.id === props.defaultMethodId) ? props.defaultMethodId : choices.value[0]!.id
  tenderUSD.value = ''
  tenderKHR.value = ''
  reference.value = ''
  nextTick(() => (usdEl.value?.$el as HTMLInputElement | undefined)?.focus())
}, { immediate: true })

const received = computed(() => roundMoney(convert(tenderUSD.value || 0, 'USD', props.currency) + (khrActive.value ? convert(tenderKHR.value || 0, 'KHR', props.currency) : 0), props.currency))
const balance = computed(() => roundMoney(received.value - props.total, props.currency))
const short = computed(() => balance.value < 0 ? -balance.value : 0)

// The change to hand back, split into the notes the till has.
const change = computed(() => {
  const amount = Math.max(balance.value, 0)
  if (props.currency === 'KHR') return { usd: 0, khr: Math.round(amount / 100) * 100 }
  if (props.currency !== 'USD' || !khrActive.value) return { usd: amount, khr: 0 }
  let usd = Math.floor(amount)
  let khr = Math.round(((amount - usd) * rateOf('KHR')) / 100) * 100
  if (khr >= rateOf('KHR')) { usd += 1; khr -= rateOf('KHR') }
  return { usd, khr }
})
const changeLabel = computed(() => {
  const { usd, khr } = change.value
  if (props.currency !== 'USD' && props.currency !== 'KHR') return formatMoney(usd, props.currency)
  const parts = [usd || !khr ? formatMoney(usd, 'USD') : '', khr ? formatMoney(khr, 'KHR') : ''].filter(Boolean)
  return parts.join(' + ')
})

function setExact() {
  if (props.currency === 'KHR') {
    tenderUSD.value = ''
    tenderKHR.value = String(props.total)
  } else {
    tenderUSD.value = String(convert(props.total, props.currency, 'USD'))
    tenderKHR.value = ''
  }
}

function addUSD(amount: number) {
  tenderUSD.value = String(roundMoney((Number(tenderUSD.value) || 0) + amount, 'USD'))
}

function confirm() {
  if (props.submitting || !method.value) return
  if (!isCash.value) {
    emit('paid', { payment_method_id: method.value.id, method_name: method.value.name, reference: reference.value })
    return
  }
  if (short.value > 0) return
  emit('paid', {
    payment_method_id: method.value.id,
    method_name: method.value.name,
    reference: '',
    cash: { received_usd: Number(tenderUSD.value) || 0, received_khr: khrActive.value ? Number(tenderKHR.value) || 0 : 0, change_usd: change.value.usd, change_khr: change.value.khr, change: Math.max(balance.value, 0) },
  })
}
</script>
