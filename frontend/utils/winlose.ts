import { formatMoney } from '~/utils/money'

/** A rate as a percent label: "35%" (rates are stored as percents). */
export function formatRate(value: string | number | null | undefined) {
  return `${new Intl.NumberFormat('en-US', { maximumFractionDigits: 4 }).format(Number(value || 0))}%`
}

/** The rates a report line used: "35%", or "35%–50%" when the rate changed in the range. */
export function rateRange(min: string, max: string) {
  return Number(min) === Number(max) ? formatRate(min) : `${formatRate(min)}–${formatRate(max)}`
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
