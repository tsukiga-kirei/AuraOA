<script setup lang="ts">
/**
 * Root index page — 统一跳转 /overview，auth middleware 会负责未登录拦截和权限检查。
 */
definePageMeta({ middleware: 'auth', layout: 'default' })

onMounted(() => {
  navigateTo('/overview')
})
</script>

<template>
  <div style="display: flex; align-items: center; justify-content: center; height: 60vh;">
    <a-spin size="large" tip="Loading..." />
  </div>
</template>
