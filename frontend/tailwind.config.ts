import type { Config } from 'tailwindcss'

export default {
  content: [
    './components/**/*.{js,vue,ts}',
    './layouts/**/*.vue',
    './pages/**/*.vue',
    './plugins/**/*.{js,ts}',
    './app.vue',
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          DEFAULT: '#2563EB',
          hover: '#1D4ED8',
        },
        navy: {
          dark: '#0B1220',
          DEFAULT: '#111827',
        },
        surface: '#FFFFFF',
        canvas: '#F5F8FF',
        muted: '#64748B',
        income: '#16A34A',
        expense: '#DC2626',
        border: '#E2E8F0',
      },
    },
  },
  plugins: [],
} satisfies Config
