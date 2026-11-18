import type { WeekSheetRow } from '~/types/api'
import { byAgentCode, byCurrency, currencyTables } from '~/utils/winlose'

export const AGENT_SORTS = [
  { value: 'name', label: 'A to Z' },
  { value: 'desc', label: 'Z to A' },
] as const
export type AgentSort = (typeof AGENT_SORTS)[number]['value']

/** A filter button: a currency or a code start, with how many agents have it. */
export interface AgentChoice { value: string; label: string; count: number }

/** Where a row keeps the agent's code (user), name and currency. */
export interface AgentFields<R> { code: (row: R) => string; name: (row: R) => string; currency: (row: R) => string }
/** An entry's rows (Data entry, an entry's View page). */
export const SHEET_ROW_FIELDS: AgentFields<WeekSheetRow> = { code: r => r.platform_code, name: r => r.platform_name, currency: r => r.currency }

// A code's start: up to its first "_" (SBK_, SBU_, 12K_ …), in capitals; '' when it has none.
const prefixOf = (code: string) => {
  const at = code.indexOf('_')
  return at > 0 ? code.slice(0, at + 1).toUpperCase() : ''
}

/**
 * Agents as Data entry shows them (New entry, Update, an entry's View page, the Agents list), like the
 * Report: a table per currency (USD first, then KHR), its agents in the order chosen and numbered in
 * it; then only those of the currency and the code start chosen (All: every one) matching the search
 * by user or name, each keeping its number.
 */
export function useAgentFilter<R>(rows: () => R[], fields: AgentFields<R>) {
  const search = ref('')
  const sort = ref<AgentSort>('name')

  // The currencies the entry's agents are in (USD first), each with how many: buttons only with two.
  const currencies = computed<AgentChoice[]>(() => {
    const counts = new Map<string, number>()
    for (const row of rows()) counts.set(fields.currency(row), (counts.get(fields.currency(row)) || 0) + 1)
    if (counts.size < 2) return []
    return [...counts].sort(([a], [b]) => byCurrency(a, b)).map(([value, count]) => ({ value, label: value, count }))
  })
  const chosenCurrency = ref('')
  // What was chosen, while the entry's agents still have it.
  const currency = computed({
    get: () => (currencies.value.some(c => c.value === chosenCurrency.value) ? chosenCurrency.value : ''),
    set: (value: string) => { chosenCurrency.value = value },
  })

  // The starts the entry's agents have, A to Z, each with how many: buttons only when they split the
  // agents (not when every agent has the same one).
  const prefixes = computed<AgentChoice[]>(() => {
    const list = rows()
    const counts = new Map<string, number>()
    for (const row of list) {
      const p = prefixOf(fields.code(row))
      if (p) counts.set(p, (counts.get(p) || 0) + 1)
    }
    if (counts.size === 1 && [...counts.values()][0] === list.length) return []
    return [...counts].sort(([a], [b]) => byAgentCode(a, b)).map(([value, count]) => ({ value, label: value, count }))
  })
  const chosenPrefix = ref('')
  const prefix = computed({
    get: () => (prefixes.value.some(p => p.value === chosenPrefix.value) ? chosenPrefix.value : ''),
    set: (value: string) => { chosenPrefix.value = value },
  })

  const tables = computed(() => {
    const list = [...rows()].sort((a, b) => byAgentCode(fields.code(a), fields.code(b)))
    if (sort.value === 'desc') list.reverse()
    const q = search.value.trim().toLowerCase()
    const shown = (row: R) => (!currency.value || fields.currency(row) === currency.value)
      && (!prefix.value || prefixOf(fields.code(row)) === prefix.value)
      && (!q || fields.code(row).toLowerCase().includes(q) || (fields.name(row) || '').toLowerCase().includes(q))
    return currencyTables(list, fields.currency)
      .map(t => ({ currency: t.currency, rows: t.rows.map((row, i) => ({ row, no: i + 1 })).filter(({ row }) => shown(row)) }))
      .filter(t => t.rows.length)
  })

  const noMatch = computed(() => {
    const q = search.value.trim()
    const within = [currency.value, prefix.value].filter(Boolean).join(' · ')
    return `No agent matches${q ? ` “${q}”` : ''}${within ? ` in ${within}` : ''}.`
  })

  // Every agent, A to Z.
  function reset() {
    search.value = ''
    chosenPrefix.value = ''
    chosenCurrency.value = ''
    sort.value = 'name'
  }

  return { search, sort, currencies, currency, prefixes, prefix, tables, noMatch, reset }
}
