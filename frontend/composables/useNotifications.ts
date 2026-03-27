import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import 'dayjs/locale/en'

import type { UserNotificationItem, UserNotificationListResponse } from '~/types/user-notifications'

dayjs.extend(relativeTime)

const POLL_MS = 120_000

export function useNotifications() {
  const { authFetch, isAuthenticated, activeRole, userLocale, token } = useAuth()

  const items = ref<UserNotificationItem[]>([])
  const total = ref(0)
  const unreadCount = ref(0)
  const listLoading = ref(false)

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

  async function markOneRead(id: string) {
    try {
      await authFetch(`/api/auth/notifications/${id}/read`, { method: 'PUT' })
      const row = items.value.find(i => i.id === id)
      if (row) row.read = true
      await refreshUnread()
    } catch { /* 忽略 */ }
  }

  async function markAllRead() {
    try {
      await authFetch('/api/auth/notifications/read-all', { method: 'PUT' })
      items.value = items.value.map(i => ({ ...i, read: true }))
      unreadCount.value = 0
    } catch { /* 忽略 */ }
  }

  function formatRelative(iso: string) {
    const loc = userLocale.value?.toLowerCase().startsWith('en') ? 'en' : 'zh-cn'
    dayjs.locale(loc)
    return dayjs(iso).fromNow()
  }

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
