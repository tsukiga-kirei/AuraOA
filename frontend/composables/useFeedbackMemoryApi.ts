/** 反馈记忆 API：/api/tenant/feedback-memory，流程选择、素材、评论导入、生成、分页版本与发布/停用。 */
import type { PagedResult } from "~/types/admin-data";
import type {
  FeedbackMemoryScope,
  FeedbackMemoryWorkspace,
  FeedbackMemoryConfig,
  FeedbackMemoryComments,
  FeedbackMemoryEntryInput,
  FeedbackMemoryVersion,
  FeedbackMemoryVersionSummary,
  FeedbackMemoryItem,
  FeedbackMemoryInputPreview,
} from "~/types/feedback-memory";
export function useFeedbackMemoryApi() {
  const { authFetch } = useAuth();
  const root = "/api/tenant/feedback-memory";
  return {
    configs: (module: string) =>
      authFetch<FeedbackMemoryConfig[]>(`${root}/configs`, {
        query: { module },
      }),
    workspace: (scope: FeedbackMemoryScope) =>
      authFetch<FeedbackMemoryWorkspace>(root, { query: scope }),
    comments: (
      scope: FeedbackMemoryScope,
      page: number,
      page_size: number,
      keyword: string,
      feedback?: string,
      deprecation = "active",
    ) =>
      authFetch<FeedbackMemoryComments>(`${root}/comments`, {
        query: { ...scope, page, page_size, keyword, feedback, deprecation },
      }),
    setCommentDeprecated: (
      scope: FeedbackMemoryScope,
      id: string,
      deprecated: boolean,
    ) =>
      authFetch(`${root}/comments/${id}`, {
        method: "PUT",
        body: { ...scope, deprecated },
      }),
    importComments: (scope: FeedbackMemoryScope, comment_ids: string[]) =>
      authFetch(`${root}/import`, {
        method: "POST",
        body: { ...scope, comment_ids },
      }),
    saveEntry: (
      scope: FeedbackMemoryScope,
      input: FeedbackMemoryEntryInput,
      id?: string,
    ) =>
      authFetch(`${root}/entries${id ? `/${id}` : ""}`, {
        method: id ? "PUT" : "POST",
        body: { ...scope, ...input },
      }),
    deleteEntry: (scope: FeedbackMemoryScope, id: string) =>
      authFetch(`${root}/entries/${id}`, { method: "DELETE", query: scope }),
    generate: (scope: FeedbackMemoryScope) =>
      authFetch<FeedbackMemoryVersion>(`${root}/generate`, {
        method: "POST",
        body: scope,
        timeout: 600000,
      }),
    preview: (scope: FeedbackMemoryScope) =>
      authFetch<FeedbackMemoryInputPreview>(`${root}/preview`, {
        method: "POST",
        body: scope,
      }),
    saveDraft: (
      scope: FeedbackMemoryScope,
      version: FeedbackMemoryVersion,
      items: FeedbackMemoryItem[],
    ) =>
      authFetch<FeedbackMemoryVersion>(`${root}/versions/${version.id}`, {
        method: "PUT",
        body: { ...scope, items, edit_revision: version.edit_revision },
      }),
    versions: (scope: FeedbackMemoryScope, page: number, page_size: number) =>
      authFetch<PagedResult<FeedbackMemoryVersionSummary>>(`${root}/versions`, {
        query: { ...scope, page, page_size },
      }),
    version: (scope: FeedbackMemoryScope, id: string) =>
      authFetch<FeedbackMemoryVersion>(`${root}/versions/${id}`, {
        query: scope,
      }),
    publish: (scope: FeedbackMemoryScope, id: string, edit_revision: number) =>
      authFetch(`${root}/versions/${id}/publish`, {
        method: "POST",
        body: { ...scope, edit_revision },
      }),
    disable: (scope: FeedbackMemoryScope) =>
      authFetch(`${root}/disable`, { method: "POST", body: scope }),
    enable: (scope: FeedbackMemoryScope) =>
      authFetch(`${root}/enable`, { method: "POST", body: scope }),
  };
}
