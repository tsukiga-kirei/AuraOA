/**
 * useSidebarMenu — Centralized sidebar menu driven purely by user permissions.
 *
 * Sidebar always shows ALL sections the user has access to, regardless of
 * which page they're currently on. No route-context switching.
 *
 * Login always lands on /dashboard (the first menu item).
 * User dropdown only shows "个人设置" and "退出登录" (no duplicate nav).
 */
import {
  DashboardOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  AppstoreOutlined,
  ApartmentOutlined,
  DatabaseOutlined,
  MonitorOutlined,
  TeamOutlined,
  SettingOutlined,
} from '@ant-design/icons-vue'
import type { Component } from 'vue'

export interface SidebarMenuItem {
  key: string
  icon: Component
  label: string
  badge?: number
}

export interface SidebarSection {
  id: string
  title: string
  items: SidebarMenuItem[]
}

const BUSINESS_ITEMS: SidebarMenuItem[] = [
  { key: '/dashboard', icon: DashboardOutlined, label: '审核工作台', badge: 6 },
  { key: '/cron', icon: ClockCircleOutlined, label: '定时任务' },
  { key: '/archive', icon: FolderOpenOutlined, label: '归档复盘' },
]

const TENANT_ITEMS: SidebarMenuItem[] = [
  { key: '/admin/tenant', icon: AppstoreOutlined, label: '规则配置' },
  { key: '/admin/tenant/org', icon: ApartmentOutlined, label: '组织人员' },
  { key: '/admin/tenant/data', icon: DatabaseOutlined, label: '数据信息' },
]

const SYSTEM_ITEMS: SidebarMenuItem[] = [
  { key: '/admin/system', icon: MonitorOutlined, label: '全局监控' },
  { key: '/admin/system/tenants', icon: TeamOutlined, label: '租户管理' },
  { key: '/admin/system/settings', icon: SettingOutlined, label: '系统设置' },
]

export const useSidebarMenu = () => {
  const route = useRoute()
  const { userPermissions } = useAuth()

  /** Sidebar sections — purely permission-driven, always the same regardless of route */
  const sections = computed<SidebarSection[]>(() => {
    const perms = userPermissions.value
    const result: SidebarSection[] = []

    if (perms.includes('business')) {
      result.push({ id: 'business', title: '工作台', items: BUSINESS_ITEMS })
    }
    if (perms.includes('tenant_admin')) {
      result.push({ id: 'tenant', title: '租户管理', items: TENANT_ITEMS })
    }
    if (perms.includes('system_admin')) {
      result.push({ id: 'system', title: '系统管理', items: SYSTEM_ITEMS })
    }

    return result
  })

  /** Check if a menu item is active */
  const isMenuActive = (itemKey: string) => {
    const path = route.path
    if (itemKey === '/admin/system' || itemKey === '/admin/tenant' || itemKey === '/dashboard') {
      return path === itemKey
    }
    return path.startsWith(itemKey)
  }

  /** Logo always goes to dashboard */
  const logoTarget = '/dashboard'

  return {
    sections,
    isMenuActive,
    logoTarget,
  }
}
