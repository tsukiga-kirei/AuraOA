<script setup lang="ts">
import zhCN from 'ant-design-vue/es/locale/zh_CN'
</script>

<template>
  <a-config-provider :locale="zhCN">
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </a-config-provider>
</template>

<style>
/* Ensure CSS variables are available globally */
@import '~/assets/css/variables.css';
@import '~/assets/css/global.css';
</style>
