/**
 * useSidebarMenu — 完全由用户权限驱动的集中侧边栏菜单。
 *
 * 侧边栏始终显示用户有权访问的所有部分，无论
 * 他们当前所在的页面。没有路由上下文切换。
 *
 * 登录始终位于 /overview（概览仪表板）。
 * 用户下拉列表仅显示"个人设置"和"注销"（无重复导航）。
 *
 * 所有标签都使用 i18n 键进行国际化。*/
import {
  DashboardOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  AppstoreOutlined,
  ApartmentOutlined,
  DatabaseOutlined,
  TeamOutlined,
  SettingOutlined,
  PieChartOutlined,
} from '@ant-design/icons-vue'
import type { Component } from 'vue'

export interface SidebarMenuItem {
  key: string
  icon: Component
  /** 标签的 i18n 键*/
  labelKey: string
  badge?: number
}

export interface SidebarSection {
  id: string
  /** 章节标题的 i18n 键*/
  titleKey: string
  items: SidebarMenuItem[]
}

const OVERVIEW_ITEMS: SidebarMenuItem[] = [
  { key: '/overview', icon: PieChartOutlined, labelKey: 'menu.overview' },
]

const BUSINESS_ITEMS: SidebarMenuItem[] = [
  { key: '/dashboard', icon: DashboardOutlined, labelKey: 'menu.dashboard', badge: 6 },
  { key: '/cron', icon: ClockCircleOutlined, labelKey: 'menu.cron' },
  { key: '/archive', icon: FolderOpenOutlined, labelKey: 'menu.archive' },
]

const TENANT_ITEMS: SidebarMenuItem[] = [
  { key: '/admin/tenant/rules', icon: AppstoreOutlined, labelKey: 'menu.tenant.rules' },
  { key: '/admin/tenant/org', icon: ApartmentOutlined, labelKey: 'menu.tenant.org' },
  { key: '/admin/tenant/data', icon: DatabaseOutlined, labelKey: 'menu.tenant.data' },
  { key: '/admin/tenant/user-configs', icon: SettingOutlined, labelKey: 'menu.tenant.userConfigs' },
]

const SYSTEM_ITEMS: SidebarMenuItem[] = [
  { key: '/admin/system/tenants', icon: TeamOutlined, labelKey: 'menu.system.tenants' },
  { key: '/admin/system/settings', icon: SettingOutlined, labelKey: 'menu.system.settings' },
]

export const useSidebarMenu = () => {
  const route = useRoute()
  const { userPermissions, menus } = useAuth()

  /**
   * 从认证菜单（GetMenu API 返回）构建页面权限集合。
   * 适用于所有角色，包括业务用户，无需访问 org API。
   */
  const menuPagePerms = computed<Set<string>>(() => {
    const perms = new Set<string>()
    menus.value.forEach(m => {
      if (m.path) perms.add(m.path)
    })
    return perms
  })

  /** 侧边栏分区 — 由权限和菜单驱动 */
  const sections = computed<SidebarSection[]>(() => {
    const perms = userPermissions.value
    const result: SidebarSection[] = []

    //所有已认证用户始终可见概览仪表板
    result.push({ id: 'overview', titleKey: 'sidebar.section.overview', items: OVERVIEW_ITEMS })

    if (perms.includes('business')) {
      const pagePerms = menuPagePerms.value
      //有菜单数据时按权限过滤，否则不显示（等待菜单加载）
      const filtered = pagePerms.size > 0
        ? BUSINESS_ITEMS.filter(item => pagePerms.has(item.key))
        : []
      if (filtered.length) {
        result.push({ id: 'business', titleKey: 'sidebar.section.business', items: filtered })
      }
    }

    if (perms.includes('tenant_admin')) {
      const pagePerms = menuPagePerms.value
      //GetMenu 返回的路径与 TENANT_ITEMS key 完全匹配（如 /admin/tenant/org）
      //有菜单数据时按权限过滤，否则显示全部（tenant_admin 菜单固定，不依赖 org API）
      const filtered = pagePerms.size > 0
        ? TENANT_ITEMS.filter(item => pagePerms.has(item.key))
        : TENANT_ITEMS
      if (filtered.length) {
        result.push({ id: 'tenant', titleKey: 'sidebar.section.tenant', items: filtered })
      }
    }

    if (perms.includes('system_admin')) {
      result.push({ id: 'system', titleKey: 'sidebar.section.system', items: SYSTEM_ITEMS })
    }

    return result
  })

  /** 检查菜单项是否处于活动状态*/
  const isMenuActive = (itemKey: string) => {
    const path = route.path
    if (itemKey === '/admin/tenant/rules' || itemKey === '/dashboard' || itemKey === '/overview') {
      return path === itemKey
    }
    return path.startsWith(itemKey)
  }

  /** 徽标始终显示在概览仪表板上*/
  const logoTarget = '/overview'

  return {
    sections,
    isMenuActive,
    logoTarget,
  }
}
