import { apiFieldErrors, validate, type FieldErrors, type Schema } from '~/utils/validation'

/**
 * Field errors for one form. `check()` runs before submitting; after the first attempt the
 * errors update as the person types, so each one clears as soon as it is fixed.
 * Show `messages` in the form's <AppAlert> and mark inputs with `invalid(field)`.
 */
export function useFormErrors(values: () => Record<string, unknown>, schema: () => Schema) {
  const errors = ref<FieldErrors>({})
  const attempted = ref(false)
  const messages = computed(() => Object.values(errors.value))

  function check() {
    attempted.value = true
    errors.value = validate(values(), schema())
    return messages.value.length === 0
  }

  /** Marks the inputs the API rejected; returns true when it named any. */
  function fromApi(e: unknown) {
    errors.value = apiFieldErrors(e)
    return messages.value.length > 0
  }

  function reset() {
    attempted.value = false
    errors.value = {}
  }

  function invalid(field: string) {
    return errors.value[field] ? 'true' : undefined
  }

  watch(values, () => {
    if (attempted.value) errors.value = validate(values(), schema())
  }, { deep: true })

  return { errors, messages, check, fromApi, reset, invalid }
}
