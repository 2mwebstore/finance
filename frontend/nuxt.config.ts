// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-10-01',
  devtools: { enabled: false },
  // Render in the browser only. Auth lives there: the refresh cookie is scoped
  // to the API's /api/v1/auth path, so the Nuxt server never knows who is
  // signed in and would render a signed-out shell that cannot hydrate cleanly.
  ssr: false,
  modules: ['@nuxtjs/tailwindcss', '@pinia/nuxt'],
  imports: {
    presets: [
      { from: '~/composables/useAuth', imports: ['useAuth'] },
      { from: '~/composables/useApi', imports: ['useApi'] },
      { from: '~/composables/usePermissions', imports: ['usePermissions'] },
      { from: '~/stores/permissions', imports: ['usePermissionsStore'] },
    ],
  },
  css: ['vuetify/styles', '@mdi/font/css/materialdesignicons.css', '~/assets/css/main.css'],
  vite: {
    ssr: {
      // Vuetify Labs imports component CSS from JavaScript. Keep Vuetify in
      // Vite's SSR bundle so Node never tries to load that CSS directly.
      noExternal: ['vuetify'],
    },
  },
  runtimeConfig: {
    // Server-only: where server/routes/api/v1 forwards API calls. Override at runtime with
    // NUXT_API_PROXY_TARGET (Railway: http://<api service>.railway.internal:8080).
    apiProxyTarget: (process.env.NUXT_API_PROXY_TARGET || process.env.API_PROXY_TARGET || 'http://localhost:8080').replace(/\/$/, ''),
    public: {
      // When unset, useApiBase matches the API hostname to the browser host so
      // the refresh cookie remains same-site on localhost and 127.0.0.1.
      apiBase: process.env.NUXT_PUBLIC_API_BASE || '',
    },
  },
  app: {
    head: {
      title: 'Finance — Income & Expense',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1, maximum-scale=1' },
        { name: 'theme-color', content: '#0B1220' },
      ],
    },
  },
  typescript: {
    strict: true,
    typeCheck: false,
  },
})
