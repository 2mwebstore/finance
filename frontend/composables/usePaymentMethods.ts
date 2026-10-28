/**
 * Active payment methods for the sale, purchase and expense forms, with Cash as the default
 * (else the first one). The list is managed under Financial activity → Categories.
 */
export function usePaymentMethods() {
  const { api } = useApi()
  const methods = ref<{ id: string; code: string; name: string }[]>([])
  const options = computed(() => methods.value.map(pm => ({ value: pm.id, label: pm.name })))
  const defaultId = computed(() => (methods.value.find(pm => pm.code === 'cash') || methods.value[0])?.id || '')

  async function load() {
    if (methods.value.length) return
    try {
      methods.value = (await api<{ items: { id: string; code: string; name: string }[] }>('/payment-methods', { query: { active: 'true' } })).items || []
    } catch {
      methods.value = []
    }
  }

  return { methods, options, defaultId, load }
}
