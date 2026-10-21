import { joinURL } from 'ufo'

/**
 * Same-origin API: the browser calls /api/v1/* on this app and we forward it to the Go API
 * (dev and production). One origin keeps the HttpOnly refresh cookie first-party in every browser.
 * The target is read at runtime: NUXT_API_PROXY_TARGET (e.g. http://api.railway.internal:8080).
 */
export default defineEventHandler(async (event) => {
  const target = joinURL(useRuntimeConfig(event).apiProxyTarget, event.path)
  try {
    return await proxyRequest(event, target)
  } catch (error: any) {
    // A 502 here means the API could not be reached; log why (DNS, refused, timeout…).
    console.error(`[api proxy] ${event.method} ${target} failed:`, error?.cause?.cause?.code || error?.cause?.message || error?.message)
    throw error
  }
})
