// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2024-11-01',
  devtools: { enabled: true },
  ssr: false,
  experimental: {
    appManifest: false,
  },

  routeRules: {
    '/api/embed/**': {
      cors: true,
      headers: {
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
        'Access-Control-Allow-Headers': '*',
      },
    },
  },

  devServer: {
    host: 'localhost',
    port: 3000,
  },

  modules: [
    '@ant-design-vue/nuxt',
  ],

  css: [
    '~/assets/css/variables.css',
    '~/assets/css/global.css',
  ],

  runtimeConfig: {
    // Nuxt 服务端代理访问 Go 的内部地址；Docker 与浏览器公开地址需分离。
    internalApiBase: process.env.NUXT_INTERNAL_API_BASE
      || process.env.NUXT_PUBLIC_API_BASE
      || 'http://localhost:8080',
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE ?? '',
      timeZone: process.env.NUXT_PUBLIC_TIME_ZONE || process.env.APP_TIMEZONE || 'Asia/Shanghai',
    },
  },

  antd: {
    extractStyle: true,
  },

  app: {
    head: {
      title: 'AuraOA — 极简 AI 驱动的企业 OA 流程协同与智能治理中枢',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1, maximum-scale=1' },
        { name: 'description', content: '极简 AI 驱动的企业 OA 流程协同与智能治理中枢 — 智能体协同、流程总结与合规审核' },
        { name: 'theme-color', content: '#5b5bd6' },
      ],
      link: [
        { rel: 'icon', type: 'image/png', href: '/favicon.png' },
        { rel: 'preconnect', href: 'https://fonts.googleapis.com' },
        { rel: 'preconnect', href: 'https://fonts.gstatic.com', crossorigin: '' },
      ],
    },
    pageTransition: { name: 'page', mode: 'out-in' },
  },
})
