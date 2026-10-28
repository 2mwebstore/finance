import { currencyMeta } from '~/utils/money'
import type { Product, StockMovementType } from '~/types/api'

/** Quantities show up to 4 decimals, without trailing zeros: 12, 1.5, 0.125. */
export function formatQty(value: string | number | null | undefined) {
  return new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 }).format(Number(value || 0))
}

/** 24 Oct 2026 */
export function formatDocDate(value: string | null | undefined) {
  if (!value) return ''
  return new Date(value).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', year: 'numeric' })
}

/** 24 Oct 2026, 14:05 */
export function formatDocDateTime(value: string | null | undefined) {
  if (!value) return ''
  return new Date(value).toLocaleString('en-GB', { day: '2-digit', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit' })
}

/** Today as YYYY-MM-DD in the browser's time zone (for date inputs). */
export function todayISO() {
  const now = new Date()
  return new Date(now.getTime() - now.getTimezoneOffset() * 60000).toISOString().slice(0, 10)
}

/** A YYYY-MM-DD day `days` later (for due dates). */
export function addDaysISO(day: string, days: number) {
  const [y, m, d] = day.split('-').map(Number)
  const date = new Date(y || 0, (m || 1) - 1, (d || 1) + days)
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

export const MOVEMENT_LABELS: Record<StockMovementType, string> = {
  sale: 'Sale',
  sale_void: 'Sale voided',
  sale_edit: 'Sale changed',
  purchase: 'Purchase',
  purchase_void: 'Purchase voided',
  purchase_edit: 'Purchase changed',
  transfer_out: 'Sent to branch',
  transfer_in: 'Received from branch',
  transfer_void: 'Transfer voided',
  adjustment: 'Adjustment',
}

/** A service has no stock, so it never runs out. */
export function isService(product: Product | undefined) {
  return product?.type === 'service'
}

/** What a branch has of a product, plus what an edited sale already holds (`held`). */
export function availableQty(product: Product | undefined, held?: Record<string, number>) {
  if (!product) return 0
  return Number(product.quantity || 0) + (held?.[product.id] || 0)
}

/** Units of a product across a document's lines. */
export function cartQty(lines: { product_id: string; quantity: string }[], productId: string) {
  return lines.filter(l => l.product_id === productId).reduce((sum, l) => sum + (Number(l.quantity) || 0), 0)
}

/** Rounds to the currency's decimals (cents for USD, whole riel for KHR…). */
export function roundMoney(value: number, currency: string) {
  const places = currencyMeta(currency).decimals
  const f = 10 ** places
  return Math.round((value + Number.EPSILON) * f) / f
}

/**
 * A discount typed as an amount or a percent of `base`, as an amount (what the API stores).
 * Percent is capped at 100; an amount at `base`.
 */
export function discountAmount(base: number, value: string, mode: 'amount' | 'percent', currency: string) {
  const n = Number(value)
  if (!value || !Number.isFinite(n) || n <= 0) return 0
  const amount = mode === 'percent' ? base * Math.min(n, 100) / 100 : n
  return roundMoney(Math.min(amount, Math.max(base, 0)), currency)
}

/** A form line's discount as an amount (percent of quantity × price, or the amount typed). */
export function lineDiscount(line: { quantity: string; price: string; discount: string; discount_mode: 'amount' | 'percent' }, currency: string) {
  const gross = (Number(line.quantity) || 0) * (Number(line.price) || 0)
  return discountAmount(gross, line.discount, line.discount_mode, currency)
}

/** Subtotal of form lines after their discounts. */
export function linesSubtotal(lines: { quantity: string; price: string; discount: string; discount_mode: 'amount' | 'percent' }[], currency: string) {
  return roundMoney(lines.reduce((sum, l) => sum + (Number(l.quantity) || 0) * (Number(l.price) || 0) - lineDiscount(l, currency), 0), currency)
}

const QTY = /^\d+(\.\d{1,4})?$/
const NUMBER = /^\d+(\.\d+)?$/

/** Same rules as the API for document lines (products, quantities, discounts). */
export function linesCheck(value: unknown) {
  const lines = value as { product_id: string; quantity: string; discount: string; discount_mode: 'amount' | 'percent'; price: string }[]
  if (!lines.length) return 'add at least one product'
  if (lines.some(l => !l.product_id)) return 'choose a product on every line'
  if (lines.some(l => !(Number(l.quantity) > 0) || !QTY.test(l.quantity))) return 'quantity must be a number above 0 (at most 4 decimals) on every line'
  if (lines.some(l => l.discount && !NUMBER.test(l.discount))) return 'a line discount must be a number of 0 or more'
  if (lines.some(l => l.discount && l.discount_mode === 'percent' && Number(l.discount) > 100)) return 'a line discount can be at most 100%'
  if (lines.some(l => l.discount && l.discount_mode === 'amount' && Number(l.discount) > (Number(l.quantity) || 0) * (Number(l.price) || 0))) return 'a line discount is more than the line'
  return ''
}

/** The discount on a document's total: a number of 0 or more, at most 100 as a percent. */
export function docDiscountCheck(mode: () => 'amount' | 'percent') {
  return (value: unknown) => {
    const text = String(value || '')
    if (!text) return ''
    if (!NUMBER.test(text)) return 'the discount must be a number of 0 or more'
    return mode() === 'percent' && Number(text) > 100 ? 'the discount can be at most 100%' : ''
  }
}
