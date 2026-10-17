import type { InjectionKey, Ref } from 'vue'
import type { AuditResult } from '~/types/audit'
import type { EmbedContextResponse } from '~/types/embed'
import type { useExperienceApi } from '~/composables/useExperienceApi'

/** 预览页提供内存适配器，正式嵌入页没有提供者，始终使用原有鉴权与 API。 */
export interface EmbedAuditPreview {
  context: Ref<EmbedContextResponse>
  execute: (progress: (result: Partial<AuditResult>) => void, personal: boolean, latest: boolean) => Promise<AuditResult>
  feedbackApi: Pick<ReturnType<typeof useExperienceApi>, 'interactions' | 'comment' | 'updateComment' | 'deleteComment'>
}

export const embedAuditPreviewKey: InjectionKey<EmbedAuditPreview> = Symbol('embed-audit-preview')
