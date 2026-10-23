import { sentence } from '~/composables/useToast'

/** Input name (as the API names it, e.g. `branch_id`) → message. */
export type FieldErrors = Record<string, string>
export type Check = (value: unknown) => string
export type Schema = Record<string, Check[]>

const EMAIL = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

/** A PIN is 4 to 6 digits (API: auth.MinPINLength / MaxPINLength). */
export const MIN_PIN_LENGTH = 4
export const MAX_PIN_LENGTH = 6

function isBlank(value: unknown) {
  if (Array.isArray(value)) return value.length === 0
  return value === null || value === undefined || String(value).trim() === ''
}

/**
 * The same rules and wording as the API's checks (backend/internal/controllers/validation.go),
 * so a message reads the same whether the browser or the server catches it.
 * Only `required` / `chooseOne` complain about empty values; the rest skip them.
 */
export const rule = {
  required: (label: string): Check => value => isBlank(value) ? `${label} is required` : '',
  chooseOne: (label: string): Check => value => isBlank(value) ? `choose at least one of the ${label}` : '',
  minLength: (label: string, min: number): Check => value =>
    !isBlank(value) && String(value).trim().length < min ? `${label} must be at least ${min} characters` : '',
  maxLength: (label: string, max: number): Check => value =>
    !isBlank(value) && String(value).trim().length > max ? `${label} must be at most ${max} characters` : '',
  email: (label: string): Check => value =>
    !isBlank(value) && !EMAIL.test(String(value).trim()) ? `${label} must be a valid email address` : '',
  /** PIN of 4 to 6 digits (API: min=4,max=6,numeric, checked in that order). */
  pin: (label: string): Check => (value) => {
    if (isBlank(value)) return ''
    const pin = String(value)
    if (pin.length < MIN_PIN_LENGTH || pin.length > MAX_PIN_LENGTH) return `${label} must be ${MIN_PIN_LENGTH} to ${MAX_PIN_LENGTH} digits`
    return /^\d+$/.test(pin) ? '' : `${label} must contain digits only`
  },
  positiveAmount: (label: string): Check => value =>
    !isBlank(value) && !(Number(value) > 0) ? `${label} must be greater than 0` : '',
}

/** Runs each field's checks in order and keeps the first problem per field. */
export function validate(values: Record<string, unknown>, schema: Schema): FieldErrors {
  const errors: FieldErrors = {}
  for (const [field, checks] of Object.entries(schema)) {
    for (const check of checks) {
      const message = check(values[field])
      if (message) {
        errors[field] = sentence(message)
        break
      }
    }
  }
  return errors
}

/** The API answers `{ "error": "...", "fields": { "branch_id": "branch is required" } }`. */
export function apiFieldErrors(e: any): FieldErrors {
  const fields = e?.data?.fields
  if (!fields || typeof fields !== 'object') return {}
  return Object.fromEntries(Object.entries(fields).map(([field, message]) => [field, sentence(String(message))]))
}
