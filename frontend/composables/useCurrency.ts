import type { CurrencyInfo, CurrencySettings } from '~/types/api'
import { setCurrencyMeta } from '~/utils/money'

const FALLBACK: CurrencyInfo[] = [
  { code: 'USD', name: 'US Dollar', symbol: '$', decimals: 2, rate: '1', active: true, base: true },
  { code: 'KHR', name: 'Cambodian Riel', symbol: '៛', decimals: 0, rate: '4100', active: true, base: false },
]

/**
 * The currencies from Manage → Currency, loaded once per session. USD is the base: product prices
 * and stock costs are in USD and convert to any other currency at its rate (units per 1 USD).
 */
export function useCurrency() {
  const { api } = useApi()
  const settings = useState<CurrencySettings | null>('currency-settings', () => null)
  const currencies = computed(() => settings.value?.currencies?.length ? settings.value.currencies : FALLBACK)
  const active = computed(() => currencies.value.filter(c => c.active))
  const defaultCurrency = computed(() => settings.value?.default_currency || 'USD')
  /** Choices for currency dropdowns: the active currencies, plus `keep` (a saved record's own). */
  function options(keep?: string) {
    return currencies.value
      .filter(c => c.active || c.code === keep)
      .map(c => ({ value: c.code, label: `${c.code} (${c.symbol.trim()})` }))
  }
  const currencyOptions = computed(() => options())

  function rateOf(code: string) {
    return Number(currencies.value.find(c => c.code === code)?.rate || 0) || (code === 'USD' ? 1 : 0)
  }

  async function load(force = false) {
    if (settings.value && !force) return settings.value
    try {
      settings.value = await api<CurrencySettings>('/currency')
      setCurrencyMeta(settings.value.currencies || [])
    } catch {
      // Keep USD and KHR.
    }
    return settings.value
  }

  /** A USD amount in the given currency, rounded to its decimals. */
  function fromUSD(usd: number | string, code: string) {
    const value = Number(usd) || 0
    const rate = code === 'USD' ? 1 : rateOf(code)
    const decimals = currencies.value.find(c => c.code === code)?.decimals ?? 2
    return roundTo(value * rate, decimals)
  }

  /** An amount in one currency in another, through USD. */
  function convert(amount: number | string, from: string, to: string) {
    const value = Number(amount) || 0
    if (from === to) return value
    const rate = from === 'USD' ? 1 : rateOf(from)
    return fromUSD(rate ? value / rate : 0, to)
  }

  /** "1 USD = ៛4,100" for a non-USD currency. */
  function rateLabel(code: string) {
    const c = currencies.value.find(item => item.code === code)
    if (!c || c.base) return ''
    const amount = new Intl.NumberFormat('en-US', { maximumFractionDigits: Math.max(c.decimals, 2) }).format(Number(c.rate))
    return `1 USD = ${c.symbol}${amount}`
  }

  return { settings, currencies, active, currencyOptions, options, defaultCurrency, load, rateOf, fromUSD, convert, rateLabel }
}

function roundTo(value: number, decimals: number) {
  const f = 10 ** decimals
  return Math.round((value + Number.EPSILON) * f) / f
}
