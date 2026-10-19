import { joinURL } from 'ufo'

/**
 * Same-origin API: the browser calls /api/v1/* on this app and we forward it to the Go API
 * (dev and production). One origin keeps the HttpOnly refresh cookie first-party in every browser.
 * The target is read at runtime: NUXT_API_PROXY_TARGET (e.g. http://api.railway.internal:8080).
 */
export default defineEventHandler((event) => {
  const target = useRuntimeConfig(event).apiProxyTarget
  return proxyRequest(event, joinURL(target, event.path))
})
