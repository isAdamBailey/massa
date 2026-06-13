import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  css: ['~/assets/css/main.css'],
  ssr: false,

  vite: {
    plugins: [tailwindcss()]
  },

  modules: ['@nuxt/eslint', '@pinia/nuxt'],

  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080'
    }
  }
})