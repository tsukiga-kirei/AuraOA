<script setup lang="ts">
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import dayjs from 'dayjs'
import utc from 'dayjs/plugin/utc'
import timezone from 'dayjs/plugin/timezone'
import 'dayjs/locale/zh-cn'
dayjs.extend(utc)
dayjs.extend(timezone)
dayjs.locale('zh-cn')
dayjs.tz.setDefault(String(useRuntimeConfig().public.timeZone || 'Asia/Shanghai'))

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
