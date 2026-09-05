import { ref, computed, readonly } from 'vue'

/**
 * OA 流程详情跳转逻辑与租户配置
 * 对接端点：GET /api/tenant/settings/oa-jump-config
 */
export interface OAJumpConfig {
  enabled: boolean
  oa_base_url: string
  process_url_template: string
  resolved_template: string
}

const configState = ref<OAJumpConfig | null>(null)
const loadingState = ref(false)
let cachedIdentity = ''
let fetchPromise: Promise<OAJumpConfig | null> | null = null

export const useOAJump = () => {
  const { authFetch, activeRole } = useAuth()
  const identity = computed(() => JSON.stringify(activeRole.value))
  watch(identity, () => { configState.value = null; fetchPromise = null; loadingState.value = false }, { flush: 'sync' })

  const loadOAJumpConfig = async (force = false): Promise<OAJumpConfig | null> => {
    if (cachedIdentity !== identity.value) { configState.value = null; fetchPromise = null; cachedIdentity = identity.value }
    if (configState.value && !force) return configState.value
    if (fetchPromise && !force) return fetchPromise

    const requestIdentity = identity.value
    loadingState.value = true
    fetchPromise = (async () => {
      try {
        const data = await authFetch<OAJumpConfig>('/api/tenant/settings/oa-jump-config')
        if (requestIdentity !== identity.value) return null
        configState.value = data
        return data
      } catch {
        if (requestIdentity !== identity.value) return null
        configState.value = {
          enabled: false,
          oa_base_url: '',
          process_url_template: '',
          resolved_template: '',
        }
        return configState.value
      } finally {
        if (requestIdentity === identity.value) { loadingState.value = false; fetchPromise = null }
      }
    })()

    return fetchPromise
  }

  const canJumpToOA = computed(() => cachedIdentity === identity.value && !!configState.value?.enabled)

  const buildTargetURL = (processId: string): string => {
    if (cachedIdentity !== identity.value || !configState.value?.enabled) return ''
    const pid = String(processId || '').trim()
    if (!pid) return ''

    const template = configState.value.resolved_template || configState.value.process_url_template
    if (template) {
      return template
        .replace(/\{process_id\}/gi, encodeURIComponent(pid))
        .replace(/\{requestid\}/gi, encodeURIComponent(pid))
    }

    let base = (configState.value.oa_base_url || '').trim()
    if (!base) return ''
    if (!/^https?:\/\//i.test(base)) {
      base = `http://${base}`
    }
    base = base.replace(/\/+$/, '')
    return `${base}/workflow/request/ViewRequestForwardSPA.jsp?requestid=${encodeURIComponent(pid)}`
  }

  const jumpToOA = (processId: string): boolean => {
    const url = buildTargetURL(processId)
    if (!url) return false
    if (typeof window !== 'undefined') {
      window.open(url, '_blank', 'noopener,noreferrer')
      return true
    }
    return false
  }

  return {
    oaJumpConfig: readonly(configState),
    loading: readonly(loadingState),
    canJumpToOA,
    loadOAJumpConfig,
    buildTargetURL,
    jumpToOA,
  }
}
