/**
 * useSidebarMenu — Centralized sidebar menu driven purely by user permissions.
 *
 * Sidebar always shows ALL sections the user has access to, regardless of
 * which page they're currently on. No route-context switching.
 *
 * Login always lands on /overview (the overview dashboard).
 * User dropdown only shows "Personal Settings" and "Logout" (no duplicate nav).
 *
 * All labels use i18n keys for internationalization.
 */
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
  /** i18n key for the label */
  labelKey: string
  badge?: number
}

export interface SidebarSection {
  id: string
  /** i18n key for the section title */
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
  const { userPermissions, currentUser, activeRole } = useAuth()
  const { members, roles } = useOrgApi()

  /** Merged page permissions from the current user's business roles */
  const businessPagePerms = computed<Set<string>>(() => {
    const uname = currentUser.value?.username
    if (!uname) return new Set()
    const member = members.value.find(m => m.username === uname)
    if (!member) return new Set()
    const rIds = member.role_ids
    const perms = new Set<string>()
    roles.value.filter(r => rIds.includes(r.id)).forEach(r => r.page_permissions.forEach(p => perms.add(p)))
    return perms
  })

  /** Sidebar sections — permission-driven, business items filtered by page_permissions */
  const sections = computed<SidebarSection[]>(() => {
    const perms = userPermissions.value
    const result: SidebarSection[] = []

    // Overview dashboard is always visible to all authenticated users
    result.push({ id: 'overview', titleKey: 'sidebar.section.overview', items: OVERVIEW_ITEMS })

    if (perms.includes('business')) {
      const pagePerms = businessPagePerms.value
      const filtered = BUSINESS_ITEMS.filter(item => pagePerms.has(item.key))
      if (filtered.length) {
        result.push({ id: 'business', titleKey: 'sidebar.section.business', items: filtered })
      }
    }
    if (perms.includes('tenant_admin')) {
      // Filter tenant admin items by the tenant_admin business role's page_permissions
      const pagePerms = businessPagePerms.value
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

  /** Check if a menu item is active */
  const isMenuActive = (itemKey: string) => {
    const path = route.path
    if (itemKey === '/admin/tenant/rules' || itemKey === '/dashboard' || itemKey === '/overview') {
      return path === itemKey
    }
    return path.startsWith(itemKey)
  }

  /** Logo always goes to overview dashboard */
  const logoTarget = '/overview'

  return {
    sections,
    isMenuActive,
    logoTarget,
  }
}
