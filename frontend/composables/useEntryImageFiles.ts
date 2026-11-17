/**
 * Group summary pictures kept for RP's History: fetched with the access token (the API keeps them
 * behind sign-in) and shown from blob URLs, each once per page session. thumb: the small copy for
 * the list (the picture itself when there is none).
 */
export function useEntryImageFiles() {
  const { apiBase } = useApi()
  const { getToken, tryRefresh } = useAuth()
  const cache = useState<Record<string, string>>('entry-image-cache', () => ({}))

  async function fileUrl(id: string, thumb: boolean): Promise<string> {
    const key = `${id}:${thumb ? 'thumb' : 'full'}`
    const known = cache.value[key]
    if (known) return known
    const url = `${apiBase}/winlose/images/${id}/file${thumb ? '?thumb=1' : ''}`
    const get = () => {
      const token = getToken()
      return fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {}, credentials: 'include' })
    }
    let res = await get()
    if (res.status === 401 && await tryRefresh()) res = await get()
    if (!res.ok) throw new Error(`image ${res.status}`)
    const objectUrl = URL.createObjectURL(await res.blob())
    cache.value[key] = objectUrl
    return objectUrl
  }

  return { fileUrl }
}
