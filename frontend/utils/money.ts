/** A currency code from Manage → Currency (USD, KHR, THB…). */
export type SupportedCurrency = string

export interface CurrencyMeta {
  symbol: string
  decimals: number
}

// Symbols and decimals per currency, filled from the API by useCurrency (USD and KHR until then).
const registry: Record<string, CurrencyMeta> = {
  USD: { symbol: '$', decimals: 2 },
  KHR: { symbol: '៛', decimals: 0 },
}

export function setCurrencyMeta(list: { code: string; symbol: string; decimals: number }[]) {
  for (const c of list) registry[c.code] = { symbol: c.symbol, decimals: c.decimals }
}

export function currencyMeta(code: string): CurrencyMeta {
  return registry[code] || { symbol: code ? `${code} ` : '', decimals: 2 }
}

export function formatMoney(amount: string | number | null | undefined, currency: SupportedCurrency): string {
  const parsed = typeof amount === 'number' ? amount : Number(amount || 0)
  const value = Number.isFinite(parsed) ? parsed : 0
  const { symbol, decimals } = currencyMeta(currency)
  const formatted = new Intl.NumberFormat('en-US', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  }).format(Math.abs(value))
  return `${value < 0 ? '-' : ''}${symbol}${formatted}`
}
