import type { DiscountMode, DocLineForm } from '~/types/api'
import { discountAmount, lineDiscount, linesSubtotal, roundMoney } from '~/utils/inventory'

export interface PosCart {
  branch_id: string
  customer: string
  /** Off the whole sale, typed as an amount or a percent (sent as an amount). */
  discount: string
  discount_mode: DiscountMode
  /** Newest first. Same shape as the sale form's lines. */
  lines: DocLineForm[]
}

/** A sale parked on the till to finish later (kept in this browser, per user and branch). */
export interface HeldSale {
  id: number
  held_at: string
  currency: string
  total: number
  customer: string
  discount: string
  discount_mode: DiscountMode
  lines: DocLineForm[]
}

/** How a POS sale was paid. */
export interface PosPayment {
  payment_method_id: string
  method_name: string
  reference: string
  /** Cash only: what the customer handed over and the change given back. */
  cash?: { received_usd: number; received_khr: number; change_usd: number; change_khr: number; change: number }
}

const emptyCart = (branchId = ''): PosCart => ({ branch_id: branchId, customer: '', discount: '', discount_mode: 'amount', lines: [] })

/**
 * The POS cart. It lives for the browser session, so leaving the till for the back office and
 * coming back keeps it. Totals follow the sale form: subtotal (lines after their discounts) −
 * sale discount = total.
 */
export function usePosCart(currency: () => string) {
  const cart = useState<PosCart>('pos-cart', () => emptyCart())
  let nextKey = Date.now()

  const itemCount = computed(() => cart.value.lines.reduce((sum, l) => sum + (Number(l.quantity) || 0), 0))
  const gross = computed(() => roundMoney(cart.value.lines.reduce((sum, l) => sum + (Number(l.quantity) || 0) * (Number(l.price) || 0), 0), currency()))
  const subtotal = computed(() => linesSubtotal(cart.value.lines, currency()))
  const lineDiscounts = computed(() => roundMoney(gross.value - subtotal.value, currency()))
  const saleDiscount = computed(() => discountAmount(subtotal.value, cart.value.discount, cart.value.discount_mode, currency()))
  const total = computed(() => roundMoney(subtotal.value - saleDiscount.value, currency()))

  function lineTotal(line: DocLineForm) {
    return roundMoney((Number(line.quantity) || 0) * (Number(line.price) || 0) - lineDiscount(line, currency()), currency())
  }

  // The product picked lands on top; one already in the cart gets one more and moves to the top.
  function add(productId: string, price: number) {
    const lines = cart.value.lines
    const index = lines.findIndex(l => l.product_id === productId)
    if (index >= 0) {
      const line = lines[index]!
      cart.value.lines = [{ ...line, quantity: String((Number(line.quantity) || 0) + 1) }, ...lines.filter((_, i) => i !== index)]
      return
    }
    cart.value.lines = [{ key: nextKey++, product_id: productId, quantity: '1', price: price > 0 ? String(price) : '0', discount: '', discount_mode: 'amount', note: '' }, ...lines]
  }

  function update(key: number, patch: Partial<DocLineForm>) {
    cart.value.lines = cart.value.lines.map(l => l.key === key ? { ...l, ...patch } : l)
  }

  // The steppers don't reorder: a row moving away under the cashier's finger is worse than useful.
  function step(key: number, by: number) {
    const line = cart.value.lines.find(l => l.key === key)
    if (!line) return
    const next = (Number(line.quantity) || 0) + by
    if (next <= 0) remove(key)
    else update(key, { quantity: String(next) })
  }

  function remove(key: number) {
    cart.value.lines = cart.value.lines.filter(l => l.key !== key)
  }

  function clear(branchId = cart.value.branch_id) {
    cart.value = emptyCart(branchId)
  }

  /** Puts a held sale back in the cart (only the lines whose product is still sold here). */
  function restore(held: HeldSale, keep: (productId: string) => boolean) {
    cart.value = {
      branch_id: cart.value.branch_id, customer: held.customer, discount: held.discount, discount_mode: held.discount_mode,
      lines: held.lines.filter(l => keep(l.product_id)).map(l => ({ ...l, key: nextKey++ })),
    }
    return held.lines.length - cart.value.lines.length
  }

  return { cart, itemCount, gross, subtotal, lineDiscounts, saleDiscount, total, lineTotal, add, update, step, remove, clear, restore }
}

/** Held sales for one user and branch, kept in this browser so a reload doesn't lose them. */
export function useHeldSales(key: () => string) {
  const list = ref<HeldSale[]>([])
  const storageKey = () => `pos-held:${key()}`

  function read() {
    try {
      list.value = JSON.parse(localStorage.getItem(storageKey()) || '[]') as HeldSale[]
    } catch {
      list.value = []
    }
  }
  function write() {
    try {
      localStorage.setItem(storageKey(), JSON.stringify(list.value))
    } catch {
      // Private window or full storage: held sales last until the page closes.
    }
  }
  watch(key, read, { immediate: true })

  function hold(sale: Omit<HeldSale, 'id' | 'held_at'>) {
    const id = Math.max(0, ...list.value.map(h => h.id)) + 1
    list.value = [{ id, held_at: new Date().toISOString(), ...sale }, ...list.value]
    write()
  }

  /** Takes a held sale off the list (to resume it, or to throw it away). */
  function take(id: number) {
    const held = list.value.find(h => h.id === id)
    list.value = list.value.filter(h => h.id !== id)
    write()
    return held
  }

  return { list, hold, take }
}
