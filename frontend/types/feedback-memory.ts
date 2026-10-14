import type { PagedResult } from "./admin-data";
export interface FeedbackMemoryScope {
  module: "audit" | "archive";
  process_type: string;
}
export interface FeedbackMemoryConfig {
  process_type: string;
  process_type_label: string;
}
export interface FeedbackMemoryComment {
  id: string;
  audit_log_id: string;
  process_id: string;
  title: string;
  username: string;
  content: string;
  feedback: "like" | "dislike" | null;
  updated_at: string;
  imported: boolean;
  deprecated: boolean;
}
export interface FeedbackMemoryEntry {
  id: string;
  memory_id: string;
  source_type: "manual" | "comment";
  source_id: string | null;
  condition: string;
  content: string;
  enabled: boolean;
  source_snapshot: Partial<FeedbackMemoryComment>;
  created_by: string;
  created_at: string;
  updated_at: string;
  implanted_version_id: string | null;
}
export interface FeedbackMemoryItem {
  condition: string;
  content: string;
  source_refs: string[];
}
export interface FeedbackMemorySource {
  ref: string;
  source_type: "manual" | "comment" | "previous";
  condition: string;
  content: string;
  snapshot: FeedbackMemoryEntry | FeedbackMemoryItem;
  inherited?: boolean;
}
export interface FeedbackMemoryVersion {
  id: string;
  memory_id: string;
  version_no: number;
  base_version_id: string | null;
  input_revision: number;
  status: "draft" | "published";
  created_by: string;
  created_at: string;
  published_at: string | null;
  sources: FeedbackMemorySource[];
  items: FeedbackMemoryItem[];
  warnings: string[];
  system_prompt: string;
  user_prompt: string;
  prompt_text: string;
  generated_items: FeedbackMemoryItem[];
  edit_revision: number;
  edited_by: string;
  edited_at: string | null;
  runtime_system_prompt: string;
  runtime_user_prompt: string;
}
export type FeedbackMemoryVersionSummary = Pick<
  FeedbackMemoryVersion,
  | "id"
  | "memory_id"
  | "version_no"
  | "base_version_id"
  | "input_revision"
  | "status"
  | "created_by"
  | "created_at"
  | "published_at"
>;
export interface FeedbackMemoryWorkspace {
  memory: {
    id: string;
    revision: number;
    last_version: number;
    published_version_id: string | null;
    enabled: boolean;
  } | null;
  entries: FeedbackMemoryEntry[];
  published: FeedbackMemoryVersion | null;
  baseline: FeedbackMemoryVersion | null;
  system_prompt: string;
  user_prompt_template: string;
}
export type FeedbackMemoryEntryInput = Pick<
  FeedbackMemoryEntry,
  "condition" | "content" | "enabled"
>;
export type FeedbackMemoryComments = PagedResult<FeedbackMemoryComment>;
export interface FeedbackMemoryInputPreview {
  system_prompt: string;
  user_prompt: string;
  base_version_id: string | null;
}
