import { formatMoney } from '~/utils/money'

/** A rate as a percent label: "35%" (rates are stored as percents). */
export function formatRate(value: string | number | null | undefined) {
  return `${new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 }).format(Number(value || 0))}%`
}

/** Agents A to Z by code, ignoring case (the API orders them the same way). */
export function byAgentCode(a: string, b: string) {
  const x = a.toLowerCase(), y = b.toLowerCase()
  return x < y ? -1 : x > y ? 1 : a < b ? -1 : a > b ? 1 : 0
}

/** A rate, or the range it moved in: "10%" or "10%–20%". */
export function rateLabel(min: string, max: string) {
  return Number(min) === Number(max) ? formatRate(min) : `${formatRate(min)}–${formatRate(max)}`
}

/**
 * The rate a Total is worked out at: under v3 what is left of 100% after the agent's rate (60% →
 * 40%), before that the rate itself. The rate as saved when the weeks' formulas differ.
 */
export function usedRate(min: string, max: string, formula: string) {
  return formula === 'v3' ? rateLabel(String(100 - Number(max)), String(100 - Number(min))) : rateLabel(min, max)
}

/** A week as "26 Oct – 01 Nov 2026". */
export function weekLabel(start: string, end: string) {
  const day = (key: string, year: boolean) => new Date(`${key}T12:00:00`).toLocaleDateString('en-GB', { day: '2-digit', month: 'short', ...(year ? { year: 'numeric' } : {}) })
  return `${day(start, false)} – ${day(end, true)}`
}

/** An amount in its currency; losses show with a minus. */
export function formatAmount(value: string | number | null | undefined, currency: string) {
  return formatMoney(Number(value || 0), currency)
}

function iso(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

/** The Monday of a YYYY-MM-DD day's week (weeks run Monday to Sunday). */
export function mondayOf(day: string) {
  const [y, m, d] = day.split('-').map(Number)
  const date = new Date(y || 0, (m || 1) - 1, d || 1)
  date.setDate(date.getDate() - ((date.getDay() + 6) % 7))
  return iso(date)
}

/** YYYY-MM-DD plus some days. */
export function plusDays(day: string, days: number) {
  const [y, m, d] = day.split('-').map(Number)
  return iso(new Date(y || 0, (m || 1) - 1, (d || 1) + days))
}

/** Downloads rows as a CSV file that Excel opens (UTF-8 with a BOM), like the other reports. */
export function downloadCsv(filename: string, header: string[], rows: (string | number)[][]) {
  const escape = (value: string | number) => {
    const text = String(value)
    return /[",\n]/.test(text) ? `"${text.replace(/"/g, '""')}"` : text
  }
  const lines = [header.map(escape).join(','), ...rows.map(row => row.map(escape).join(','))]
  const url = URL.createObjectURL(new Blob(['﻿' + lines.join('\n')], { type: 'text/csv;charset=utf-8' }))
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.click()
  URL.revokeObjectURL(url)
}
