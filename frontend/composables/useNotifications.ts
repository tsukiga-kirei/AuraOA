import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/en'

import type { UserNotificationItem, UserNotificationListResponse } from '~/types/user-notifications'

dayjs.extend(relativeTime)

// 未读数量轮询间隔（毫秒）
const POLL_MS = 120_000

export function useNotifications() {
  const { authFetch, isAuthenticated, activeRole, userLocale, token } = useAuth()

  /** 通知列表（响应式，供模板直接绑定） */
  const items = ref<UserNotificationItem[]>([])
  /** 通知总数 */
  const total = ref(0)
  /** 未读通知数量（用于头部角标展示） */
  const unreadCount = ref(0)
  /** 列表加载状态 */
  const listLoading = ref(false)

  /**
   * 从后端拉取当前用户的未读通知数量。
   * 未登录或无激活角色时重置为 0。
   */
  async function refreshUnread() {
    if (!isAuthenticated.value || !activeRole.value?.id) {
      unreadCount.value = 0
      return
    }
    try {
      const data = await authFetch<{ count: number }>('/api/auth/notifications/unread-count')
      unreadCount.value = Number(data?.count) || 0
    } catch {
      unreadCount.value = 0
    }
  }

  /**
   * 从后端拉取最近 30 条通知列表。
   * 未登录或无激活角色时清空列表。
   */
  async function refreshList() {
    if (!isAuthenticated.value || !activeRole.value?.id) {
      items.value = []
      total.value = 0
      return
    }
    listLoading.value = true
    try {
      const data = await authFetch<UserNotificationListResponse>('/api/auth/notifications', {
        query: { limit: 30, offset: 0 },
      })
      items.value = data?.items ?? []
      total.value = Number(data?.total) || 0
    } catch {
      items.value = []
      total.value = 0
    } finally {
      listLoading.value = false
    }
  }

  /**
   * 将指定通知标记为已读，并刷新未读数量。
   * @param id 通知 ID
   */
  async function markOneRead(id: string) {
    try {
      await authFetch(`/api/auth/notifications/${id}/read`, { method: 'PUT' })
      const row = items.value.find(i => i.id === id)
      if (row) row.read = true
      await refreshUnread()
    } catch { /* 标记失败时静默忽略 */ }
  }

  /**
   * 将当前用户所有通知标记为已读，并清零未读角标。
   */
  async function markAllRead() {
    try {
      await authFetch('/api/auth/notifications/read-all', { method: 'PUT' })
      items.value = items.value.map(i => ({ ...i, read: true }))
      unreadCount.value = 0
    } catch { /* 标记失败时静默忽略 */ }
  }

  /**
   * 将 ISO 时间字符串格式化为相对时间（如"3 分钟前"）。
   * 根据当前用户语言偏好自动切换中英文。
   * @param iso ISO 格式时间字符串
   */
  function formatRelative(iso: string) {
    const loc = userLocale.value?.toLowerCase().startsWith('en') ? 'en' : 'zh-cn'
    dayjs.locale(loc)
    return dayjs(iso).fromNow()
  }

  // 登录态或激活角色变化时，重置列表并重新拉取未读数
  watch(
    () => [token.value, activeRole.value?.id] as const,
    () => {
      items.value = []
      total.value = 0
      if (isAuthenticated.value && activeRole.value?.id) {
        refreshUnread()
      } else {
        unreadCount.value = 0
      }
    },
    { immediate: true },
  )

  // 定时轮询未读数量，组件卸载时清除定时器
  let pollTimer: ReturnType<typeof setInterval> | null = null
  onMounted(() => {
    pollTimer = setInterval(() => {
      if (isAuthenticated.value && activeRole.value?.id) refreshUnread()
    }, POLL_MS)
  })
  onUnmounted(() => {
    if (pollTimer) clearInterval(pollTimer)
  })

  return {
    items,
    total,
    unreadCount,
    listLoading,
    refreshUnread,
    refreshList,
    markOneRead,
    markAllRead,
    formatRelative,
  }
}
