import type { Ref } from 'vue'

/**
 * A list that loads a page at a time as the person scrolls (see <InfiniteLoader />): the first page
 * on reset(), the next pages appended by loadMore(). refresh() fetches again the pages already
 * loaded and swaps them in, so a save or delete keeps the scroll place.
 */
export function useInfiniteList<T>(
  fetchPage: (page: number, size: number) => Promise<{ items: T[]; total: number }>,
  keyOf: (item: T) => string,
  size = 30,
  failure = 'Could not load the list.',
) {
  const items = ref([]) as Ref<T[]>
  const total = ref(0)
  const page = ref(0)
  const loading = ref(false) // the first page
  const loadingMore = ref(false)
  const error = ref('')
  const hasMore = computed(() => items.value.length < total.value)
  // Each reset or refresh makes the answers still on the way stale.
  let request = 0

  function describe(e: any) {
    return e?.data?.error || failure
  }

  async function reset() {
    const mine = ++request
    page.value = 0
    items.value = []
    total.value = 0
    loading.value = true
    loadingMore.value = false
    error.value = ''
    try {
      const result = await fetchPage(1, size)
      if (mine !== request) return
      items.value = result.items || []
      total.value = result.total || 0
      page.value = 1
    } catch (e) {
      if (mine === request) error.value = describe(e)
    } finally {
      if (mine === request) loading.value = false
    }
  }

  async function loadMore() {
    if (loading.value || loadingMore.value || !hasMore.value || error.value) return
    const mine = request
    loadingMore.value = true
    try {
      const result = await fetchPage(page.value + 1, size)
      if (mine !== request) return
      const seen = new Set(items.value.map(keyOf))
      items.value = [...items.value, ...(result.items || []).filter(item => !seen.has(keyOf(item)))]
      total.value = result.total || 0
      page.value += 1
    } catch (e) {
      if (mine === request) error.value = describe(e)
    } finally {
      if (mine === request) loadingMore.value = false
    }
  }

  async function refresh() {
    const mine = ++request
    const pages = Math.max(page.value, 1)
    error.value = ''
    try {
      let all: T[] = []
      let count = 0
      let got = 0
      for (let n = 1; n <= pages; n++) {
        const result = await fetchPage(n, size)
        if (mine !== request) return
        all = [...all, ...(result.items || [])]
        count = result.total || 0
        got = n
        if (!result.items?.length) break
      }
      const seen = new Set<string>()
      items.value = all.filter(item => !seen.has(keyOf(item)) && !!seen.add(keyOf(item)))
      total.value = count
      page.value = got
    } catch (e) {
      if (mine === request) error.value = describe(e)
    } finally {
      if (mine === request) loading.value = false
    }
  }

  return { items, total, loading, loadingMore, error, hasMore, reset, loadMore, refresh }
}
