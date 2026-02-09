<script setup lang="ts">
import { ref } from 'vue'
import {
  DashboardOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  SettingOutlined,
  MonitorOutlined,
  BulbOutlined,
  BulbFilled,
  LogoutOutlined,
} from '@ant-design/icons-vue'

const collapsed = ref(false)
const selectedKeys = ref<string[]>(['/dashboard'])

const { isDark, toggle: toggleTheme, restore: restoreTheme } = useTheme()
const { logout } = useAuth()

onMounted(() => {
  restoreTheme()
})

const menuItems = [
  { key: '/dashboard', icon: DashboardOutlined, label: '审核工作台' },
  { key: '/cron', icon: ClockCircleOutlined, label: '定时任务' },
  { key: '/archive', icon: FolderOpenOutlined, label: '归档复盘' },
  { key: '/admin/tenant', icon: SettingOutlined, label: '租户配置' },
  { key: '/admin/system', icon: SettingOutlined, label: '系统管理' },
  { key: '/admin/monitor', icon: MonitorOutlined, label: '全局监控' },
]
</script>

<template>
  <a-config-provider :theme="{ token: isDark ? { colorBgBase: '#141414', colorTextBase: '#ffffffd9' } : {} }">
    <a-layout style="min-height: 100vh">
      <a-layout-sider
        v-model:collapsed="collapsed"
        collapsible
        breakpoint="lg"
        :theme="isDark ? 'dark' : 'dark'"
      >
        <div style="height: 32px; margin: 16px; color: #fff; text-align: center; font-weight: bold; font-size: 16px; line-height: 32px;">
          {{ collapsed ? '智审' : 'OA智审平台' }}
        </div>
        <a-menu v-model:selectedKeys="selectedKeys" theme="dark" mode="inline">
          <a-menu-item v-for="item in menuItems" :key="item.key">
            <NuxtLink :to="item.key" style="color: inherit; text-decoration: none;">
              <component :is="item.icon" />
              <span>{{ item.label }}</span>
            </NuxtLink>
          </a-menu-item>
        </a-menu>
      </a-layout-sider>

      <a-layout>
        <a-layout-header
          :style="{
            background: isDark ? '#1f1f1f' : '#fff',
            padding: '0 16px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
          }"
        >
          <span :style="{ fontSize: '18px', fontWeight: 600, color: isDark ? '#ffffffd9' : '#1a1a2e' }">
            OA流程智能审核平台
          </span>
          <a-space>
            <a-tooltip :title="isDark ? '切换亮色模式' : '切换暗色模式'">
              <a-button type="text" @click="toggleTheme">
                <BulbFilled v-if="isDark" style="color: #faad14;" />
                <BulbOutlined v-else />
              </a-button>
            </a-tooltip>
            <a-tooltip title="退出登录">
              <a-button type="text" @click="logout">
                <LogoutOutlined />
              </a-button>
            </a-tooltip>
          </a-space>
        </a-layout-header>

        <a-layout-content style="margin: 16px;">
          <div
            :style="{
              padding: '24px',
              background: isDark ? '#1f1f1f' : '#fff',
              minHeight: '360px',
              borderRadius: '8px',
            }"
          >
            <slot />
          </div>
        </a-layout-content>

        <a-layout-footer :style="{ textAlign: 'center', color: isDark ? '#ffffff73' : undefined }">
          OA智审 ©2025
        </a-layout-footer>
      </a-layout>
    </a-layout>
  </a-config-provider>
</template>
