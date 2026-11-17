import type { PaymentMethod } from '~/types/api'

/**
 * Active payment methods for the sale, purchase and expense forms, with Cash as the default
 * (else the first one). `branch` is the document's branch: only methods shared by every branch or
 * that branch's own are offered (managed under Inventory → Payment Methods).
 */
export function usePaymentMethods(branch: () => string = () => '') {
  const { api } = useApi()
  const methods = ref<PaymentMethod[]>([])
  const options = computed(() => methods.value.map(pm => ({ value: pm.id, label: pm.branch_id ? `${pm.name} (${pm.branch_name || 'branch'} only)` : pm.name })))
  const defaultId = computed(() => (methods.value.find(pm => pm.code === 'cash') || methods.value[0])?.id || '')

  let loadedFor: string | null = null
  let request = 0
  async function load() {
    const branchId = branch()
    if (loadedFor === branchId && methods.value.length) return
    const current = ++request
    try {
      const items = (await api<{ items: PaymentMethod[] }>('/payment-methods', { query: { active: 'true', branch_id: branchId || undefined } })).items || []
      if (current !== request) return
      methods.value = items
      loadedFor = branchId
    } catch {
      if (current === request) methods.value = []
    }
  }

  /** The method when the branch can use it, else the default. */
  function keep(id: string) {
    return id && methods.value.some(pm => pm.id === id) ? id : defaultId.value
  }

  return { methods, options, defaultId, load, keep }
}
