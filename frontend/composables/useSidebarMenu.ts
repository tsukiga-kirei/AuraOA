/**
 * useSidebarMenu — 基于用户权限动态构建侧边栏菜单。
 *
 * 菜单分区完全由用户权限组（userPermissions）和后端菜单权限（menus）驱动，
 * 无需感知当前路由，确保侧边栏始终展示用户有权访问的所有入口。
 *
 * 登录后默认落地页为 /overview（概览仪表板）。
 * 所有菜单标签通过 i18n 键国际化。
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
  /** 菜单标签的 i18n 键 */
  labelKey: string
  badge?: number
}

export interface SidebarSection {
  id: string
  /** 分区标题的 i18n 键 */
  titleKey: string
  items: SidebarMenuItem[]
}

/** 概览分区菜单项（所有已登录用户可见） */
const OVERVIEW_ITEMS: SidebarMenuItem[] = [
  { key: '/overview', icon: PieChartOutlined, labelKey: 'menu.overview' },
]

/** 业务用户菜单项（需要 business 权限组） */
const BUSINESS_ITEMS: SidebarMenuItem[] = [
  { key: '/dashboard', icon: DashboardOutlined, labelKey: 'menu.dashboard' },
  { key: '/cron', icon: ClockCircleOutlined, labelKey: 'menu.cron' },
  { key: '/archive', icon: FolderOpenOutlined, labelKey: 'menu.archive' },
]

/** 租户管理员菜单项（需要 tenant_admin 权限组） */
const TENANT_ITEMS: SidebarMenuItem[] = [
  { key: '/admin/tenant/rules', icon: AppstoreOutlined, labelKey: 'menu.tenant.rules' },
  { key: '/admin/tenant/org', icon: ApartmentOutlined, labelKey: 'menu.tenant.org' },
  { key: '/admin/tenant/data', icon: DatabaseOutlined, labelKey: 'menu.tenant.data' },
  { key: '/admin/tenant/user-configs', icon: SettingOutlined, labelKey: 'menu.tenant.userConfigs' },
]

/** 系统管理员菜单项（需要 system_admin 权限组） */
const SYSTEM_ITEMS: SidebarMenuItem[] = [
  { key: '/admin/system/tenants', icon: TeamOutlined, labelKey: 'menu.system.tenants' },
  { key: '/admin/system/settings', icon: SettingOutlined, labelKey: 'menu.system.settings' },
]

// 待审核数量轮询间隔（毫秒）
const POLL_INTERVAL_MS = 60_000

export const useSidebarMenu = () => {
  const route = useRoute()
  const { userPermissions, menus, authFetch } = useAuth()

  /** 待审核数量（从后端实时获取，用于仪表盘菜单角标） */
  const pendingAuditCount = ref(0)
  let pollTimer: ReturnType<typeof setInterval> | null = null

  /** 从后端拉取待审核数量，接口失败时不影响侧边栏显示 */
  const fetchPendingCount = async () => {
    try {
      const stats = await authFetch<{ pending_ai_count: number }>('/api/audit/stats')
      pendingAuditCount.value = stats.pending_ai_count ?? 0
    } catch {
      // 接口失败时静默忽略，不影响侧边栏正常显示
    }
  }

  // 组件挂载时立即拉取一次，并启动定时轮询
  onMounted(() => {
    fetchPendingCount()
    pollTimer = setInterval(fetchPendingCount, POLL_INTERVAL_MS)
  })

  // 组件卸载时清除定时器，避免内存泄漏
  onUnmounted(() => {
    if (pollTimer) clearInterval(pollTimer)
  })

  /**
   * 从认证菜单（GetMenu API 返回）构建页面权限集合。
   * 适用于所有角色，无需额外调用组织架构接口。
   */
  const menuPagePerms = computed<Set<string>>(() => {
    const perms = new Set<string>()
    menus.value.forEach(m => {
      if (m.path) perms.add(m.path)
    })
    return perms
  })

  /**
   * 根据用户权限组和后端菜单权限动态计算侧边栏分区列表。
   * 菜单未加载完成时对应分区不显示，等待加载后自动更新。
   */
  const sections = computed<SidebarSection[]>(() => {
    const perms = userPermissions.value
    const result: SidebarSection[] = []

    // 概览仪表板对所有已登录用户可见
    result.push({ id: 'overview', titleKey: 'sidebar.section.overview', items: OVERVIEW_ITEMS })

    if (perms.includes('business')) {
      const pagePerms = menuPagePerms.value
      // 有菜单数据时按权限过滤，菜单未加载时不显示（避免闪烁）
      const filtered = pagePerms.size > 0
        ? BUSINESS_ITEMS.filter(item => pagePerms.has(item.key)).map(item => {
            // 仪表盘菜单项附加待审核数量角标
            if (item.key === '/dashboard' && pendingAuditCount.value > 0) {
              return { ...item, badge: pendingAuditCount.value }
            }
            return item
          })
        : []
      if (filtered.length) {
        result.push({ id: 'business', titleKey: 'sidebar.section.business', items: filtered })
      }
    }

    if (perms.includes('tenant_admin')) {
      const pagePerms = menuPagePerms.value
      // 从后端 menus 过滤，菜单未加载时不显示
      const filtered = pagePerms.size > 0
        ? TENANT_ITEMS.filter(item => pagePerms.has(item.key))
        : []
      if (filtered.length) {
        result.push({ id: 'tenant', titleKey: 'sidebar.section.tenant', items: filtered })
      }
    }

    if (perms.includes('system_admin')) {
      const pagePerms = menuPagePerms.value
      const filtered = pagePerms.size > 0
        ? SYSTEM_ITEMS.filter(item => pagePerms.has(item.key))
        : []
      if (filtered.length) {
        result.push({ id: 'system', titleKey: 'sidebar.section.system', items: filtered })
      }
    }

    return result
  })

  /**
   * 判断菜单项是否处于激活状态（高亮显示）。
   * 精确匹配规则页、仪表盘、概览；其余路径使用前缀匹配。
   * @param itemKey 菜单项路径
   */
  const isMenuActive = (itemKey: string) => {
    const path = route.path
    if (itemKey === '/admin/tenant/rules' || itemKey === '/dashboard' || itemKey === '/overview') {
      return path === itemKey
    }
    return path.startsWith(itemKey)
  }

  /** Logo 点击跳转目标（始终为概览仪表板） */
  const logoTarget = '/overview'

  return {
    sections,
    isMenuActive,
    logoTarget,
  }
}
