/**
 * useTokenGuard — 主动 Token 过期检测与自动续期。
 *
 * 解决的问题：用户停留在同一页面不切换路由时，Token 过期后页面不会自动刷新，
 * 左下角仍显示用户信息，localStorage 中 auth_state 不会被清除。
 *
 * 策略：
 * 1. 每 5 分钟检查一次 access_token 是否即将过期（剩余 < 5 分钟）
 * 2. 即将过期时主动调用 refresh 续期
 * 3. refresh 也失败则清除本地状态并跳转登录页
 * 4. 页面从后台切回前台时（visibilitychange）立即检查一次
 */

// Token 剩余有效期低于此阈值时触发主动刷新（秒）
const REFRESH_THRESHOLD_SECONDS = 5 * 60
// 定时检查间隔（毫秒）
const CHECK_INTERVAL_MS = 5 * 60 * 1000

/**
 * 解析 JWT payload 中的 exp 字段（秒级时间戳）。
 */
function parseJwtExp(token: string): number | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) return null
    const payload = JSON.parse(atob(parts[1].replace(/-/g, '+').replace(/_/g, '/')))
    return typeof payload.exp === 'number' ? payload.exp : null
  } catch { return null }
}

export const useTokenGuard = () => {
  const { token, refreshToken, doRefreshToken, logout, isAuthenticated } = useAuth()

  let intervalId: ReturnType<typeof setInterval> | null = null

  /**
   * 检查 access_token 是否即将过期，若是则尝试刷新。
   * 若 refresh 也失败，则登出用户。
   */
  const checkAndRefresh = async () => {
    // 未登录状态无需检查
    if (!isAuthenticated.value) return

    const t = token.value || localStorage.getItem('token')
    if (!t) {
      // token 丢失但状态显示已登录，尝试用 refresh_token 恢复
      const rt = refreshToken.value || localStorage.getItem('refresh_token')
      if (rt) {
        const rtExp = parseJwtExp(rt)
        if (rtExp && rtExp > Date.now() / 1000) {
          const ok = await doRefreshToken()
          if (!ok) await logout()
        } else {
          await logout()
        }
      } else {
        await logout()
      }
      return
    }

    const exp = parseJwtExp(t)
    if (!exp) return

    const remainingSeconds = exp - Date.now() / 1000

    // Token 已过期或即将过期，尝试刷新
    if (remainingSeconds < REFRESH_THRESHOLD_SECONDS) {
      const ok = await doRefreshToken()
      if (!ok) {
        await logout()
      }
    }
  }

  /**
   * 页面可见性变化时的处理：从后台切回前台立即检查 token 状态。
   */
  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible') {
      checkAndRefresh()
    }
  }

  /**
   * 启动 Token 守卫：注册定时器和 visibilitychange 监听。
   * 应在应用初始化时调用一次（如 app.vue 的 onMounted）。
   */
  const startGuard = () => {
    if (import.meta.server) return // SSR 环境不启动

    // 避免重复注册
    stopGuard()

    // 立即检查一次
    checkAndRefresh()

    // 定时检查
    intervalId = setInterval(checkAndRefresh, CHECK_INTERVAL_MS)

    // 页面可见性变化时检查
    document.addEventListener('visibilitychange', handleVisibilityChange)
  }

  /**
   * 停止 Token 守卫：清除定时器和事件监听。
   */
  const stopGuard = () => {
    if (intervalId) {
      clearInterval(intervalId)
      intervalId = null
    }
    if (typeof document !== 'undefined') {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }

  return { startGuard, stopGuard, checkAndRefresh }
}
