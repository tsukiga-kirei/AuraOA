<script setup lang="ts">
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'
dayjs.locale('zh-cn')

// 启动 Token 主动过期检测守卫
const { startGuard, stopGuard } = useTokenGuard()
onMounted(() => { startGuard() })
onUnmounted(() => { stopGuard() })
</script>

<template>
  <a-config-provider :locale="zhCN">
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </a-config-provider>
</template>

<style>
/*确保 CSS 变量全局可用*/
@import '~/assets/css/variables.css';
@import '~/assets/css/global.css';
</style>
