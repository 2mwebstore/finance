export function useApiBase() {
  const config = useRuntimeConfig()
  const configuredBase = String(config.public.apiBase || '').replace(/\/$/, '')
  if (configuredBase) return configuredBase

  if (import.meta.client) {
    return `${window.location.protocol}//${window.location.hostname}:8080/api/v1`
  }

  return 'http://localhost:8080/api/v1'
}
