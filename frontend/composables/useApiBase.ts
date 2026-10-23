/** Where the browser calls the API: /api/v1 on this app's own origin unless configured otherwise. */
export function useApiBase() {
  const config = useRuntimeConfig()
  return String(config.public.apiBase || '/api/v1').replace(/\/$/, '')
}
