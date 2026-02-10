// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: true },
  ssr: true,

  modules: [
    '@ant-design-vue/nuxt',
  ],

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080',
      mockMode: process.env.NUXT_PUBLIC_MOCK_MODE || 'false',
    },
  },

  antd: {
    extractStyle: true,
  },

  app: {
    head: {
      title: 'OA智审 - 流程智能审核平台',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'OA流程智能审核平台' },
      ],
    },
  },
})
