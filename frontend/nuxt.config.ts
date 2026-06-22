import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  css: ['~/assets/css/main.css'],
  ssr: false,

  app: {
    head: {
      title: 'Massa',
      link: [
        { rel: 'icon', type: 'image/svg+xml', href: '/favicon.svg' },
        { rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }
      ],
      meta: [
        { name: 'description', content: 'Massa is a personal weight and BMI tracker that syncs with Google Health.' },
        { property: 'og:title', content: 'Massa' },
        { property: 'og:description', content: 'Weight and BMI, tracked quietly.' },
        { property: 'og:image', content: '/og-image.png' },
        { property: 'og:image:type', content: 'image/png' },
        { property: 'og:image:width', content: '1200' },
        { property: 'og:image:height', content: '630' },
        { name: 'twitter:card', content: 'summary_large_image' }
      ]
    }
  },

  vite: {
    plugins: [tailwindcss()]
  },

  modules: ['@nuxt/eslint', '@pinia/nuxt'],

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE
        || (process.env.NODE_ENV === 'production' ? '' : 'http://localhost:8080')
    }
  }
})