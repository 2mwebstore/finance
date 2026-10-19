import { createVuetify } from 'vuetify'
import { VDateRangePicker } from 'vuetify/labs/VDateRangePicker'

export default defineNuxtPlugin((nuxtApp) => {
  const vuetify = createVuetify({
    components: { VDateRangePicker },
    theme: {
      defaultTheme: 'light',
      themes: {
        light: { colors: { primary: '#1769e8' } },
      },
    },
  })

  nuxtApp.vueApp.use(vuetify)
})
