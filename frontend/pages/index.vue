<script setup lang="ts">
// 根路由页面：挂载后立即跳转到 /overview，未登录时由 auth 中间件拦截并重定向到登录页
definePageMeta({ middleware: 'auth', layout: 'default' })

onMounted(() => {
  // 页面挂载后直接跳转到概览页
  navigateTo('/overview')
})
</script>

<template>
  <div style="display: flex; align-items: center; justify-content: center; height: 60vh;">
    <a-spin size="large" :tip="'加载中...'" />
  </div>
</template>
