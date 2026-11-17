<template>
  <div class="print-shell">
    <div class="print-toolbar">
      <NuxtLink :to="{ path: '/inventory/quotations', query: quote ? { open: quote.id } : {} }" class="toolbar-button"><ArrowLeft :size="15" /> Back</NuxtLink>
      <button type="button" class="toolbar-button primary" :disabled="!quote" @click="print"><Printer :size="15" /> Print or save as PDF</button>
    </div>

    <p v-if="loading" class="print-message">Loading the quotation…</p>
    <p v-else-if="error" class="print-message error">{{ error }}</p>
    <article v-else-if="quote" class="sheet" aria-label="Quotation">
      <header class="sheet-head">
        <div>
          <p class="company">{{ quote.seller?.company || quote.branch_name }}</p>
          <p v-if="quote.seller?.branch && quote.seller.branch !== quote.seller.company" class="muted">{{ quote.seller.branch }}</p>
          <p v-if="quote.seller?.address" class="muted pre">{{ quote.seller.address }}</p>
          <p v-if="contactLine" class="muted">{{ contactLine }}</p>
        </div>
        <div class="doc-title">
          <h1>QUOTATION</h1>
          <p class="mono">{{ quote.number }}</p>
          <p v-if="quote.status === 'converted'" class="stamp">Turned into sale {{ quote.sale_number }}</p>
          <p v-else-if="quote.status === 'declined'" class="stamp declined">Declined</p>
        </div>
      </header>

      <section class="meta">
        <div>
          <p class="label">For</p>
          <p class="strong">{{ quote.customer }}</p>
          <p v-if="quote.contact" class="pre">{{ quote.contact }}</p>
        </div>
        <dl>
          <div><dt>Date</dt><dd>{{ formatDocDate(quote.quote_date) }}</dd></div>
          <div v-if="quote.valid_until"><dt>Valid until</dt><dd>{{ formatDocDate(quote.valid_until) }}</dd></div>
          <div><dt>Currency</dt><dd>{{ quote.currency }}</dd></div>
          <div><dt>Prepared by</dt><dd>{{ quote.created_by }}</dd></div>
        </dl>
      </section>

      <table class="lines">
        <thead>
          <tr><th class="num">#</th><th>Item</th><th class="right">Qty</th><th class="right">Unit price</th><th v-if="lineDiscounts" class="right">Discount</th><th class="right">Amount</th></tr>
        </thead>
        <tbody>
          <tr v-for="(line, i) in quote.lines" :key="i">
            <td class="num">{{ i + 1 }}</td>
            <td>
              <span class="strong">{{ line.name }}</span>
              <span class="sku mono">{{ line.sku }}</span>
              <span v-if="line.note" class="note">{{ line.note }}</span>
            </td>
            <td class="right nowrap">{{ formatQty(line.quantity) }} {{ line.unit }}</td>
            <td class="right nowrap">{{ formatMoney(line.unit_price, quote.currency) }}</td>
            <td v-if="lineDiscounts" class="right nowrap">{{ Number(line.discount) ? `−${formatMoney(line.discount, quote.currency)}` : '' }}</td>
            <td class="right nowrap strong">{{ formatMoney(line.line_total, quote.currency) }}</td>
          </tr>
        </tbody>
      </table>

      <section class="totals">
        <div><span>Subtotal</span><span>{{ formatMoney(quote.subtotal, quote.currency) }}</span></div>
        <div v-if="Number(quote.discount)"><span>Discount</span><span>−{{ formatMoney(quote.discount, quote.currency) }}</span></div>
        <div v-if="Number(quote.shipping)"><span>Shipping</span><span>+{{ formatMoney(quote.shipping, quote.currency) }}</span></div>
        <div class="grand"><span>Total</span><span>{{ formatMoney(quote.total, quote.currency) }}</span></div>
        <div v-if="quote.currency !== 'USD'" class="muted small"><span>≈ in USD</span><span>{{ formatMoney(quote.total_base, 'USD') }}</span></div>
      </section>

      <section v-if="quote.note" class="terms">
        <p class="label">Notes</p>
        <div class="rich-text-content" v-html="sanitizeRichText(quote.note)"></div>
      </section>

      <footer class="sheet-foot">
        <div class="sign"><span></span><p>Customer’s signature &amp; date</p></div>
        <div class="sign"><span></span><p>For {{ quote.seller?.company || quote.branch_name }}</p></div>
      </footer>
      <p class="fine">This quotation is a price offer, not an invoice{{ quote.valid_until ? `; prices hold until ${formatDocDate(quote.valid_until)}` : '' }}.</p>
    </article>
  </div>
</template>

<script setup lang="ts">
import { ArrowLeft, Printer } from 'lucide-vue-next'
import type { Quotation } from '~/types/api'
import { formatMoney } from '~/utils/money'
import { sanitizeRichText } from '~/utils/richText'
import { formatDocDate, formatQty } from '~/utils/inventory'

/** A quotation as an A4 page to print or save as PDF (the browser's print dialog). */
definePageMeta({ layout: false })

const route = useRoute()
const { api } = useApi()
const quote = ref<Quotation | null>(null)
const loading = ref(true)
const error = ref('')

useHead(() => ({ title: quote.value ? `${quote.value.number} · ${quote.value.customer}` : 'Quotation' }))

// The Discount column only when a line has one.
const lineDiscounts = computed(() => !!quote.value?.lines.some(l => Number(l.discount)))
const contactLine = computed(() => [quote.value?.seller?.phone, quote.value?.seller?.email].filter(Boolean).join(' · '))

onMounted(async () => {
  try {
    quote.value = await api<Quotation>(`/inventory/quotations/${String(route.params.id)}`)
  } catch (e: any) {
    error.value = e?.statusCode === 404 || e?.status === 404 ? 'This quotation wasn’t found.' : e?.data?.error || 'Could not open this quotation.'
  } finally {
    loading.value = false
  }
})

function print() {
  window.print()
}
</script>

<style scoped>
.print-shell { min-height: 100vh; background: #eef2f7; padding: 16px; color: #111827; }
.print-toolbar { display: flex; max-width: 210mm; margin: 0 auto 12px; justify-content: space-between; gap: 8px; }
.toolbar-button { display: inline-flex; align-items: center; gap: 6px; border: 1px solid #e2e8f0; border-radius: 10px; background: #fff; padding: 8px 14px; font-size: 13px; font-weight: 600; color: #111827; }
.toolbar-button.primary { border-color: #2563eb; background: #2563eb; color: #fff; }
.toolbar-button:disabled { opacity: 0.5; }
.print-message { max-width: 210mm; margin: 40px auto; text-align: center; font-size: 14px; color: #64748b; }
.print-message.error { color: #dc2626; }

.sheet { box-sizing: border-box; max-width: 210mm; min-height: 297mm; margin: 0 auto; background: #fff; padding: 16mm 15mm; font-size: 12px; line-height: 1.45; box-shadow: 0 4px 24px rgba(15, 23, 42, 0.08); }
.sheet-head { display: flex; justify-content: space-between; gap: 24px; border-bottom: 2px solid #111827; padding-bottom: 12px; }
.company { font-size: 18px; font-weight: 800; }
.doc-title { text-align: right; }
.doc-title h1 { margin: 0; font-size: 22px; font-weight: 800; letter-spacing: 0.12em; }
.stamp { display: inline-block; margin-top: 6px; border: 1.5px solid #7c3aed; border-radius: 6px; padding: 2px 8px; font-size: 10px; font-weight: 800; text-transform: uppercase; color: #7c3aed; }
.stamp.declined { border-color: #dc2626; color: #dc2626; }
.meta { display: flex; justify-content: space-between; gap: 24px; margin: 16px 0; }
.meta dl { display: grid; gap: 2px; margin: 0; }
.meta dl div { display: flex; justify-content: space-between; gap: 18px; }
.meta dt { color: #64748b; }
.meta dd { margin: 0; font-weight: 600; text-align: right; }
.label { margin-bottom: 2px; font-size: 10px; font-weight: 700; letter-spacing: 0.1em; text-transform: uppercase; color: #64748b; }
.strong { font-weight: 700; }
.muted { color: #64748b; }
.small { font-size: 11px; }
.pre { white-space: pre-line; }
.mono { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; }
.nowrap { white-space: nowrap; }
.lines { width: 100%; border-collapse: collapse; }
.lines th { border-bottom: 1px solid #cbd5e1; background: #f8fafc; padding: 6px 8px; text-align: left; font-size: 10px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: #475569; }
.lines td { border-bottom: 1px solid #e2e8f0; padding: 7px 8px; vertical-align: top; }
.lines .right { text-align: right; }
.lines .num { width: 24px; color: #94a3b8; }
.sku { display: block; font-size: 10px; color: #94a3b8; }
.note { display: block; font-size: 11px; font-style: italic; color: #64748b; }
.totals { display: grid; width: 260px; margin: 12px 0 0 auto; gap: 3px; }
.totals div { display: flex; justify-content: space-between; gap: 12px; }
.totals .grand { margin-top: 4px; border-top: 2px solid #111827; padding-top: 6px; font-size: 15px; font-weight: 800; }
.terms { margin-top: 20px; border-radius: 8px; background: #f8fafc; padding: 10px 12px; }
.sheet-foot { display: grid; grid-template-columns: 1fr 1fr; gap: 40px; margin-top: 48px; }
.sign span { display: block; height: 40px; border-bottom: 1px solid #94a3b8; }
.sign p { margin-top: 4px; font-size: 11px; color: #64748b; }
.fine { margin-top: 24px; text-align: center; font-size: 10px; color: #94a3b8; }

@media (max-width: 640px) {
  .print-shell { padding: 8px; }
  .sheet { min-height: 0; padding: 18px 14px; }
  .sheet-head, .meta { flex-direction: column; gap: 10px; }
  .doc-title { text-align: left; }
  .lines { display: block; overflow-x: auto; }
  .totals { width: 100%; }
}

@media print {
  @page { size: A4; margin: 0; }
  .print-shell { padding: 0; background: #fff; }
  .print-toolbar { display: none; }
  .sheet { max-width: none; min-height: 0; box-shadow: none; }
  .lines tr { break-inside: avoid; }
}
</style>
