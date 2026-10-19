import { useApiBase } from '~/composables/useApiBase'

export function useApi() {
  const apiBase = useApiBase()
  const { getToken, tryRefresh, logout } = useAuth()

  async function api<T>(path: string, opts: {
    method?: string
    body?: unknown
    query?: Record<string, string | number | undefined>
  } = {}): Promise<T> {
    const headers: Record<string, string> = {}
    const token = getToken()
    if (token) headers.Authorization = `Bearer ${token}`

    const query = opts.query
      ? '?' + Object.entries(opts.query)
          .filter(([, v]) => v !== undefined && v !== '')
          .map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`)
          .join('&')
      : ''

    try {
      return await $fetch<T>(`${apiBase}${path}${query}`, {
        method: (opts.method || 'GET') as any,
        body: opts.body as any,
        headers,
        credentials: 'include',
      })
    } catch (e: any) {
      if (e?.status === 401 || e?.statusCode === 401) {
        const ok = await tryRefresh()
        if (ok) {
          const t2 = getToken()
          if (t2) headers.Authorization = `Bearer ${t2}`
          return await $fetch<T>(`${apiBase}${path}${query}`, {
            method: (opts.method || 'GET') as any,
            body: opts.body as any,
            headers,
            credentials: 'include',
          })
        }
        await logout()
        if (import.meta.client) {
          navigateTo('/login')
        }
      }
      throw e
    }
  }

  return { api, apiBase }
}
