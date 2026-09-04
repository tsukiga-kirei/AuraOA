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
let fetchPromise: Promise<OAJumpConfig | null> | null = null

export const useOAJump = () => {
  const { authFetch } = useAuth()

  const loadOAJumpConfig = async (force = false): Promise<OAJumpConfig | null> => {
    if (configState.value && !force) return configState.value
    if (fetchPromise && !force) return fetchPromise

    loadingState.value = true
    fetchPromise = (async () => {
      try {
        const data = await authFetch<OAJumpConfig>('/api/tenant/settings/oa-jump-config')
        configState.value = data
        return data
      } catch {
        configState.value = {
          enabled: false,
          oa_base_url: '',
          process_url_template: '',
          resolved_template: '',
        }
        return configState.value
      } finally {
        loadingState.value = false
        fetchPromise = null
      }
    })()

    return fetchPromise
  }

  const canJumpToOA = computed(() => !!configState.value?.enabled)

  const buildTargetURL = (processId: string): string => {
    if (!configState.value?.enabled) return ''
    const pid = String(processId || '').trim()
    if (!pid) return ''

    const template = configState.value.resolved_template || configState.value.process_url_template
    if (template) {
      return template
        .replace(/\{process_id\}/gi, pid)
        .replace(/\{requestid\}/gi, pid)
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
