import type { Branch } from '~/types/api'

/**
 * Branches the signed-in user can see (their roles' data scope), for branch filters on lists.
 * GET /branches is limited the same way; people who cannot call it use scope_branches from
 * their profile.
 */
export function useBranchOptions(allLabel = 'All branches') {
  const { api } = useApi()
  const { user } = useAuth()
  const { can } = usePermissions()
  const branches = ref<Pick<Branch, 'id' | 'code' | 'name'>[]>([...(user.value?.scope_branches || user.value?.branches || [])])

  const options = computed(() => [
    { value: '', label: allLabel },
    ...branches.value.map(branch => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })),
  ])
  // A filter only helps when there is more than one branch to choose from.
  const showFilter = computed(() => branches.value.length > 1)

  function labelFor(id: string) {
    const branch = branches.value.find(item => item.id === id)
    return branch ? branch.code : allLabel
  }

  async function load() {
    if (!can('branches.view', 'branches.manage', 'users.create', 'users.manage')) return
    try {
      const result = await api<{ items: Branch[] }>('/branches', { query: { page_size: 100 } })
      if (result.items?.length) branches.value = result.items
    } catch {
      // Keep the profile branches.
    }
  }

  onMounted(load)
  return { branches, options, showFilter, labelFor }
}
