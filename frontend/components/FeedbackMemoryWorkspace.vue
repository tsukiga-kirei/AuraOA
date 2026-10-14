<script setup lang="ts">
import { message, Modal } from "ant-design-vue";
import {
  CommentOutlined,
  PlusOutlined,
  RobotOutlined,
  HistoryOutlined,
  SafetyCertificateOutlined,
  EditOutlined,
  DeleteOutlined,
  LinkOutlined,
  ReloadOutlined,
  ApartmentOutlined,
  SearchOutlined,
  EyeOutlined,
  SaveOutlined,
  InfoCircleOutlined,
  StopOutlined,
  UndoOutlined,
} from "@ant-design/icons-vue";
import { useFeedbackMemoryApi } from "~/composables/useFeedbackMemoryApi";
import { appDayjs } from "~/utils/appTime";
import type {
  FeedbackMemoryConfig,
  FeedbackMemoryScope,
  FeedbackMemoryWorkspace,
  FeedbackMemoryEntry,
  FeedbackMemoryEntryInput,
  FeedbackMemoryVersion,
  FeedbackMemoryVersionSummary,
  FeedbackMemoryComment,
  FeedbackMemorySource,
  FeedbackMemoryItem,
  FeedbackMemoryInputPreview,
} from "~/types/feedback-memory";
const { t } = useI18n();
const api = useFeedbackMemoryApi();
const module = ref<"audit" | "archive">("audit");
const processType = ref("");
const processSearch = ref("");
const configs = ref<FeedbackMemoryConfig[]>([]);
const workspace = ref<FeedbackMemoryWorkspace | null>(null);
const loading = ref(false);
const refreshing = ref(false);
const busy = ref(false);
const generating = ref(false);
const error = ref(false);
const selected = ref<FeedbackMemoryVersion | null>(null);
const versionPage = ref(1);
const versionSize = ref(20);
const versions = ref<FeedbackMemoryVersionSummary[]>([]);
const versionTotal = ref(0);
const panel = ref("materials");
const panels = computed(() => [
  {
    key: "materials",
    label: t("feedbackMemory.materials"),
    icon: CommentOutlined,
  },
  {
    key: "memory",
    label: t("feedbackMemory.generatedMemory"),
    icon: RobotOutlined,
  },
  { key: "history", label: t("feedbackMemory.history"), icon: HistoryOutlined },
]);
const filteredConfigs = computed(() => {
  const keyword = processSearch.value.trim().toLocaleLowerCase();
  return configs.value.filter((row) =>
    `${row.process_type_label || ""} ${row.process_type}`
      .toLocaleLowerCase()
      .includes(keyword),
  );
});
const processCategory = computed(
  () =>
    configs.value.find((row) => row.process_type === processType.value)
      ?.process_type_label || "",
);
const promptsOpen = ref(false);
const previewOpen = ref(false);
const integrationPreviewOpen = ref(false);
function openConfirm(
  action: "publish" | "disable" | "delete",
  entryID?: string,
) {
  const target = entryID || selected.value?.id || "";
  const editRevision = selected.value?.edit_revision || 0;
  const targetScope = scope();
  const content = t(
    action === "publish"
      ? selected.value?.items.length
        ? "feedbackMemory.publishConfirm"
        : "feedbackMemory.publishEmptyConfirm"
      : action === "disable"
        ? "feedbackMemory.disableConfirm"
        : "feedbackMemory.deleteConfirm",
  );
  // 与规则配置的发布/切换版本确认框保持相同尺寸与交互。
  Modal.confirm({
    title: t(
      action === "publish"
        ? "feedbackMemory.publish"
        : action === "disable"
          ? "feedbackMemory.disable"
          : "feedbackMemory.delete",
    ),
    content,
    okText: t("feedbackMemory.confirm"),
    cancelText: t("feedbackMemory.cancel"),
    width: 380,
    centered: true,
    maskClosable: true,
    okButtonProps: { danger: action !== "publish" },
    async onOk() {
      busy.value = true;
      try {
        if (action === "publish")
          await api.publish(targetScope, target, editRevision);
        else if (action === "disable") await api.disable(targetScope);
        else await api.deleteEntry(targetScope, target);
        message.success(
          t(
            action === "publish"
              ? "feedbackMemory.published"
              : action === "disable"
                ? "feedbackMemory.disabled"
                : "feedbackMemory.saved",
          ),
        );
        await load();
      } catch (failure) {
        showError(failure);
        throw failure;
      } finally {
        busy.value = false;
      }
    },
  });
}
const previewLoading = ref(false);
const inputPreview = ref<FeedbackMemoryInputPreview | null>(null);
const memoryEditing = ref(false);
const draftItems = ref<FeedbackMemoryItem[]>([]);
const commentDeprecation = ref("active");
const editorOpen = ref(false);
const editing = ref<FeedbackMemoryEntry | null>(null);
const form = reactive<FeedbackMemoryEntryInput>({
  content: "",
  condition: "",
  enabled: true,
});
const commentsOpen = ref(false);
const comments = ref<FeedbackMemoryComment[]>([]);
const commentTotal = ref(0);
const commentPage = ref(1);
const commentSize = ref(20);
const commentKeyword = ref("");
const commentFeedback = ref<string>();
const commentLoading = ref(false);
const selectedCommentIDs = ref<string[]>([]);
const scope = (): FeedbackMemoryScope => ({
  module: module.value,
  process_type: processType.value,
});
const locked = computed(
  () =>
    loading.value ||
    refreshing.value ||
    busy.value ||
    generating.value ||
    previewLoading.value,
);
const enabledCount = computed(
  () =>
    workspace.value?.entries.filter(
      (row) => row.enabled && !row.implanted_version_id,
    ).length || 0,
);
const implantedCount = computed(
  () =>
    workspace.value?.entries.filter((row) => row.implanted_version_id).length ||
    0,
);
const activeSource = ref<FeedbackMemorySource | null>(null);
const stale = computed(
  () =>
    !!selected.value &&
    (selected.value.input_revision !== workspace.value?.memory?.revision ||
      selected.value.id !== workspace.value?.baseline?.id),
);
const current = computed(
  () =>
    !!selected.value &&
    selected.value.id === workspace.value?.memory?.published_version_id,
);
const canPublish = computed(
  () =>
    selected.value?.status === "draft" &&
    !stale.value &&
    (!!selected.value.items?.length || !!selected.value.base_version_id),
);
const canEditMemory = computed(
  () =>
    !!selected.value &&
    selected.value.id === workspace.value?.baseline?.id &&
    (selected.value.status === "published" || !stale.value),
);
const displayedItems = computed(() =>
  memoryEditing.value ? draftItems.value : selected.value?.items || [],
);
const finalPrompt = computed(() =>
  memoryEditing.value
    ? draftItems.value
        .map(
          (item, index) =>
            `${index + 1}. 适用条件：${item.condition.trim() || "仅在当前流程事实支持且不与正式规则冲突时适用"}\n参考记忆：${item.content.trim()}`,
        )
        .join("\n")
        .trim()
    : selected.value?.prompt_text || "",
);
function startMemoryEdit() {
  if (!selected.value) return;
  draftItems.value = selected.value.items.map((item) => ({
    ...item,
    source_refs: [...item.source_refs],
  }));
  memoryEditing.value = true;
}
async function saveMemoryDraft() {
  if (!selected.value) return;
  if (
    draftItems.value.some(
      (item) => !item.content.trim() || !item.source_refs.length,
    )
  ) {
    message.warning(t("feedbackMemory.draftRequired"));
    return;
  }
  await run(async () => {
    selected.value = await api.saveDraft(
      scope(),
      selected.value!,
      draftItems.value,
    );
    memoryEditing.value = false;
  }, "feedbackMemory.draftSaved");
}
async function previewInput() {
  previewLoading.value = true;
  try {
    inputPreview.value = await api.preview(scope());
    integrationPreviewOpen.value = true;
  } catch (failure) {
    showError(failure);
  } finally {
    previewLoading.value = false;
  }
}
async function toggleCommentDeprecated(comment: FeedbackMemoryComment) {
  await run(
    async () => {
      await api.setCommentDeprecated(scope(), comment.id, !comment.deprecated);
      selectedCommentIDs.value = selectedCommentIDs.value.filter(
        (id) => id !== comment.id,
      );
      await loadComments();
    },
    comment.deprecated
      ? "feedbackMemory.restored"
      : "feedbackMemory.deprecatedSaved",
  );
}
const formatDate = (value?: string | null) =>
  value ? appDayjs(value).format("YYYY/MM/DD HH:mm:ss") : "—";
const sourceLabel = (kind: string) => t(`feedbackMemory.source.${kind}`);
const versionStatus = (version: FeedbackMemoryVersionSummary) =>
  version.id === workspace.value?.memory?.published_version_id
    ? t(
        workspace.value.memory.enabled
          ? "feedbackMemory.active"
          : "feedbackMemory.disabledStatus",
      )
    : t(
        version.status === "draft"
          ? "feedbackMemory.draft"
          : "feedbackMemory.historical",
      );
const sourceComment = (source: FeedbackMemorySource) =>
  source.source_type === "comment" && "source_snapshot" in source.snapshot
    ? source.snapshot.source_snapshot
    : null;
function showError(error?: unknown) {
  message.error(
    error instanceof Error ? error.message : t("feedbackMemory.failed"),
  );
}
let loadSeq = 0;
async function loadVersions() {
  const result = await api.versions(
    scope(),
    versionPage.value,
    versionSize.value,
  );
  versions.value = result.items || [];
  versionTotal.value = result.total;
}
async function load() {
  if (!processType.value) return;
  const seq = ++loadSeq;
  loading.value = true;
  error.value = false;
  try {
    const [result, history] = await Promise.all([
      api.workspace(scope()),
      api.versions(scope(), versionPage.value, versionSize.value),
    ]);
    if (seq !== loadSeq) return;
    workspace.value = result;
    versions.value = history.items || [];
    versionTotal.value = history.total;
    if (selected.value)
      selected.value = await api.version(scope(), selected.value.id);
    else if (history.items?.[0])
      selected.value = await api.version(scope(), history.items[0].id);
  } catch (failure) {
    if (seq === loadSeq) {
      error.value = true;
      showError(failure);
    }
  } finally {
    if (seq === loadSeq) loading.value = false;
  }
}
async function loadConfigs() {
  loading.value = true;
  processSearch.value = "";
  workspace.value = null;
  selected.value = null;
  processType.value = "";
  configs.value = [];
  error.value = false;
  try {
    configs.value = await api.configs(module.value);
    processType.value = configs.value[0]?.process_type || "";
  } catch (failure) {
    error.value = true;
    showError(failure);
  } finally {
    loading.value = false;
  }
  await changeProcess();
}
// 手动刷新同时更新流程目录和当前工作区，保留仍存在的流程选择与版本浏览位置。
async function refresh() {
  if (locked.value || memoryEditing.value) return;
  refreshing.value = true;
  error.value = false;
  try {
    configs.value = await api.configs(module.value);
    if (configs.value.some((row) => row.process_type === processType.value)) {
      await load();
    } else {
      processType.value = configs.value[0]?.process_type || "";
      await changeProcess();
    }
    if (!error.value) message.success(t("feedbackMemory.refreshed"));
  } catch (failure) {
    error.value = true;
    showError(failure);
  } finally {
    refreshing.value = false;
  }
}
async function selectProcess(value: string) {
  if (locked.value || memoryEditing.value || processType.value === value)
    return;
  processType.value = value;
  await changeProcess();
}
async function changeProcess() {
  selected.value = null;
  workspace.value = null;
  activeSource.value = null;
  memoryEditing.value = false;
  versionPage.value = 1;
  versions.value = [];
  versionTotal.value = 0;
  await load();
}
async function run(
  action: () => Promise<unknown>,
  success = "feedbackMemory.saved",
) {
  if (busy.value) return;
  busy.value = true;
  try {
    await action();
    message.success(t(success));
    await load();
  } catch (failure) {
    showError(failure);
  } finally {
    busy.value = false;
  }
}
function edit(entry?: FeedbackMemoryEntry) {
  editing.value = entry || null;
  Object.assign(
    form,
    entry
      ? {
          content: entry.content,
          condition: entry.condition,
          enabled: entry.enabled,
        }
      : { content: "", condition: "", enabled: true },
  );
  editorOpen.value = true;
}
async function saveEntry() {
  if (!form.content.trim()) {
    message.warning(t("feedbackMemory.required"));
    return;
  }
  await run(async () => {
    await api.saveEntry(scope(), { ...form }, editing.value?.id);
    editorOpen.value = false;
  });
}
async function loadComments(reset = false) {
  if (reset) {
    commentPage.value = 1;
    selectedCommentIDs.value = [];
  }
  commentLoading.value = true;
  try {
    const result = await api.comments(
      scope(),
      commentPage.value,
      commentSize.value,
      commentKeyword.value,
      commentFeedback.value,
      commentDeprecation.value,
    );
    comments.value = result.items || [];
    commentTotal.value = result.total;
    if (
      commentPage.value > 1 &&
      (commentPage.value - 1) * commentSize.value >= result.total
    ) {
      commentPage.value = Math.max(
        1,
        Math.ceil(result.total / commentSize.value),
      );
      await loadComments();
    }
  } catch (failure) {
    showError(failure);
  } finally {
    commentLoading.value = false;
  }
}
async function openComments() {
  selectedCommentIDs.value = [];
  commentsOpen.value = true;
  await loadComments(true);
}
async function importComments() {
  await run(async () => {
    await api.importComments(scope(), selectedCommentIDs.value);
    commentsOpen.value = false;
    selectedCommentIDs.value = [];
  });
}
async function generate() {
  if (locked.value || memoryEditing.value) return;
  generating.value = true;
  try {
    selected.value = await api.generate(scope());
    panel.value = "memory";
    activeSource.value = null;
    versionPage.value = 1;
    message.success(t("feedbackMemory.generated"));
    await load();
  } catch (failure) {
    showError(failure);
  } finally {
    generating.value = false;
  }
}
async function selectVersion(id: string) {
  if (memoryEditing.value) return;
  busy.value = true;
  try {
    selected.value = await api.version(scope(), id);
    activeSource.value = null;
    panel.value = "memory";
  } catch (failure) {
    showError(failure);
  } finally {
    busy.value = false;
  }
}
function showSource(ref: string) {
  activeSource.value =
    selected.value?.sources.find((row) => row.ref === ref) || null;
}
async function changeVersionPage(page: number, size: number) {
  versionPage.value = size === versionSize.value ? page : 1;
  versionSize.value = size;
  busy.value = true;
  try {
    await loadVersions();
  } catch (failure) {
    showError(failure);
  } finally {
    busy.value = false;
  }
}
function changeCommentPage(page: number, size: number) {
  commentPage.value = size === commentSize.value ? page : 1;
  commentSize.value = size;
  loadComments();
}
onMounted(loadConfigs);
</script>

<template>
  <div class="feedback-workspace memory-surface">
    <div class="workspace-layout">
      <aside class="process-library panel">
        <div class="library-top">
          <ApartmentOutlined /><span>{{
            t("feedbackMemory.processLibrary")
          }}</span
          ><span class="library-count">{{ configs.length }}</span>
        </div>
        <div class="library-controls">
          <div
            class="scope-nav"
            :aria-label="t('feedbackMemory.businessScope')"
          >
            <button
              v-for="kind in ['audit', 'archive'] as const"
              :key="kind"
              type="button"
              :class="{ selected: module === kind }"
              :aria-pressed="module === kind"
              :disabled="locked || memoryEditing"
              @click="
                module = kind;
                loadConfigs();
              "
            >
              {{ t(`admin.data.llmTab.${kind}`) }}
            </button>
          </div>
          <a-input
            v-model:value="processSearch"
            allow-clear
            :placeholder="t('feedbackMemory.searchProcess')"
            :aria-label="t('feedbackMemory.searchProcess')"
            ><template #prefix><SearchOutlined /></template
          ></a-input>
        </div>
        <div class="library-list">
          <button
            v-for="config in filteredConfigs"
            :key="config.process_type"
            type="button"
            class="library-item"
            :class="{ selected: processType === config.process_type }"
            :aria-pressed="processType === config.process_type"
            :disabled="locked || memoryEditing"
            @click="selectProcess(config.process_type)"
          >
            <span class="item-name">{{ config.process_type }}</span>
            <span v-if="config.process_type_label" class="item-meta">{{
              config.process_type_label
            }}</span>
          </button>
          <p v-if="!loading && !filteredConfigs.length" class="library-empty">
            {{
              t(
                error
                  ? "feedbackMemory.failed"
                  : configs.length
                    ? "feedbackMemory.noMatchingProcess"
                    : "feedbackMemory.noConfigs",
              )
            }}
          </p>
        </div>
      </aside>
      <main class="feedback-main panel">
        <div class="workspace-header">
          <div class="workspace-title">
            <h3>{{ processType || t("feedbackMemory.selectProcess") }}</h3>
            <p>{{ processCategory || t("feedbackMemory.workspaceHint") }}</p>
          </div>
          <a-space wrap>
            <a-button
              class="memory-icon-button"
              :loading="refreshing"
              :disabled="locked || memoryEditing"
              :aria-label="t('feedbackMemory.refresh')"
              :title="t('feedbackMemory.refreshHint')"
              @click="refresh"
              ><template #icon><ReloadOutlined v-if="!refreshing" /></template
            ></a-button>
            <a-button
              :disabled="!workspace || locked"
              @click="promptsOpen = true"
              ><SafetyCertificateOutlined />{{
                t("feedbackMemory.generationRules")
              }}</a-button
            >
          </a-space>
        </div>
        <a-alert
          v-if="error"
          type="error"
          show-icon
          :message="t('feedbackMemory.failed')"
          class="feedback-alert workspace-error"
        />
        <a-spin :spinning="loading">
          <div v-if="processType && workspace" class="workspace-content">
            <div class="workspace-status">
              <span
                >{{ t("feedbackMemory.materials") }}
                <strong>{{ workspace.entries.length }}</strong> ·
                {{ t("feedbackMemory.enabledCount", enabledCount) }}</span
              >
              <span
                class="status-label"
                :class="{
                  'status-label--active':
                    workspace.published && workspace.memory?.enabled,
                }"
                >{{
                  workspace.published && workspace.memory?.enabled
                    ? `${t("feedbackMemory.activeMemory")} v${workspace.published.version_no}`
                    : workspace.published
                      ? `${t("feedbackMemory.disabledStatus")} v${workspace.published.version_no}`
                      : t("feedbackMemory.unpublished")
                }}</span
              >
              <span v-if="implantedCount">{{
                t("feedbackMemory.implantedCount", implantedCount)
              }}</span>
              <a-button
                v-if="workspace.published && !workspace.memory?.enabled"
                size="small"
                :disabled="locked || memoryEditing"
                @click="
                  run(
                    () => api.enable(scope()),
                    'feedbackMemory.enabledSuccess',
                  )
                "
              >
                <template #icon><UndoOutlined /></template
                >{{ t("feedbackMemory.enable") }}
              </a-button>
              <span v-if="workspace.published"
                >{{ t("feedbackMemory.lastPublished") }}
                {{ formatDate(workspace.published.published_at) }}</span
              >
            </div>
            <div class="feedback-heading">
              <nav class="workspace-tab-nav" :aria-label="t('memory.review')">
                <button
                  v-for="tab in panels"
                  :key="tab.key"
                  type="button"
                  class="workspace-tab-btn"
                  :class="{ selected: panel === tab.key }"
                  :aria-pressed="panel === tab.key"
                  @click="panel = tab.key"
                >
                  <component :is="tab.icon" />{{ tab.label }}
                </button>
              </nav>
              <a-space wrap :size="8">
                <a-button
                  :disabled="locked || !selected || memoryEditing"
                  @click="previewOpen = true"
                >
                  <template #icon><EyeOutlined /></template
                  >{{ t("feedbackMemory.previewMemory") }}
                </a-button>
                <a-button
                  type="primary"
                  :loading="generating"
                  :disabled="
                    busy ||
                    loading ||
                    memoryEditing ||
                    (!enabledCount && !workspace.baseline)
                  "
                  @click="generate"
                  ><template #icon
                    ><RobotOutlined v-if="!generating" /></template
                  >{{ t("feedbackMemory.generate") }}</a-button
                >
              </a-space>
            </div>
            <template v-if="panel === 'materials'">
              <div class="material-actions">
                <p>{{ t("feedbackMemory.materialHint") }}</p>
                <a-space wrap>
                  <a-button
                    :disabled="locked || memoryEditing"
                    @click="openComments"
                    ><CommentOutlined />{{
                      t("feedbackMemory.importComments")
                    }}</a-button
                  >
                  <a-button :disabled="locked || memoryEditing" @click="edit()"
                    ><PlusOutlined />{{
                      t("feedbackMemory.addManual")
                    }}</a-button
                  >
                </a-space>
              </div>
              <a-empty
                v-if="!workspace.entries.length"
                :description="t('feedbackMemory.emptyMaterials')"
                class="feedback-empty"
              />
              <article
                v-for="(entry, index) in workspace.entries"
                :key="entry.id"
                class="memory-entry"
                :class="{ 'memory-entry--disabled': !entry.enabled }"
              >
                <div class="entry-top">
                  <div class="entry-meta">
                    <span class="entry-number">{{ index + 1 }}</span
                    ><a-tag class="source-tag">{{
                      sourceLabel(entry.source_type)
                    }}</a-tag
                    ><span>{{
                      entry.source_snapshot.username || entry.created_by
                    }}</span
                    ><span>{{ formatDate(entry.updated_at) }}</span>
                  </div>
                  <a-space>
                    <a-tag
                      v-if="entry.implanted_version_id"
                      class="source-tag"
                      >{{ t("feedbackMemory.implanted") }}</a-tag
                    >
                    <a-switch
                      v-else
                      :checked="entry.enabled"
                      :disabled="locked || memoryEditing"
                      size="small"
                      :aria-label="t('feedbackMemory.enabled')"
                      @change="
                        (value: boolean | string | number) =>
                          run(() =>
                            api.saveEntry(
                              scope(),
                              {
                                content: entry.content,
                                condition: entry.condition,
                                enabled: !!value,
                              },
                              entry.id,
                            ),
                          )
                      "
                    />
                    <a-button
                      v-if="!entry.implanted_version_id"
                      type="text"
                      size="small"
                      :disabled="locked || memoryEditing"
                      :aria-label="t('feedbackMemory.edit')"
                      @click="edit(entry)"
                      ><EditOutlined
                    /></a-button>
                    <a-button
                      type="text"
                      size="small"
                      danger
                      :disabled="locked || memoryEditing"
                      :aria-label="t('feedbackMemory.delete')"
                      @click="openConfirm('delete', entry.id)"
                      ><DeleteOutlined
                    /></a-button>
                  </a-space>
                </div>
                <div v-if="entry.source_snapshot.title" class="source-process">
                  <LinkOutlined />{{ entry.source_snapshot.title }} ·
                  {{ entry.source_snapshot.process_id }}
                </div>
                <p v-if="entry.condition" class="entry-condition">
                  {{ t("feedbackMemory.condition") }}：{{ entry.condition }}
                </p>
                <p class="entry-content">{{ entry.content }}</p>
              </article>
            </template>
            <template v-else-if="panel === 'memory'">
              <a-empty
                v-if="!selected"
                :description="t('feedbackMemory.emptyMemory')"
                class="feedback-empty"
              />
              <template v-else>
                <div class="version-heading">
                  <div>
                    <h3>
                      v{{ selected.version_no }}
                      <a-tag
                        class="status-label"
                        :class="{ 'status-label--active': current }"
                        >{{ versionStatus(selected) }}</a-tag
                      >
                    </h3>
                    <p>
                      {{ formatDate(selected.created_at) }} ·
                      {{ selected.created_by }}
                    </p>
                  </div>
                  <a-space wrap :size="8">
                    <template v-if="memoryEditing">
                      <a-button
                        :disabled="busy"
                        @click="memoryEditing = false"
                        >{{ t("feedbackMemory.cancel") }}</a-button
                      >
                      <a-button
                        type="primary"
                        :loading="busy"
                        @click="saveMemoryDraft"
                        ><template #icon><SaveOutlined v-if="!busy" /></template
                        >{{ t("feedbackMemory.saveDraft") }}</a-button
                      >
                    </template>
                    <a-button
                      v-else-if="canEditMemory"
                      :disabled="locked"
                      @click="startMemoryEdit"
                      ><template #icon><EditOutlined /></template
                      >{{ t("feedbackMemory.editMemory") }}</a-button
                    >
                    <a-button
                      v-if="canPublish && !memoryEditing"
                      type="primary"
                      :disabled="locked"
                      @click="openConfirm('publish')"
                      >{{ t("feedbackMemory.publish") }}</a-button
                    >
                    <a-button
                      v-if="
                        current && workspace.memory?.enabled && !memoryEditing
                      "
                      danger
                      :disabled="locked"
                      @click="openConfirm('disable')"
                      >{{ t("feedbackMemory.disable") }}</a-button
                    >
                  </a-space>
                </div>
                <div
                  v-if="
                    selected.warnings.length ||
                    (selected.status === 'draft' && stale) ||
                    memoryEditing
                  "
                  class="memory-notice"
                  role="note"
                >
                  <InfoCircleOutlined />
                  <div>
                    <strong>{{ t("feedbackMemory.reviewNotes") }}</strong>
                    <p v-if="memoryEditing">
                      {{ t("feedbackMemory.editHint") }}
                    </p>
                    <p v-if="selected.status === 'draft' && stale">
                      {{ t("feedbackMemory.stale") }}
                    </p>
                    <p
                      v-for="(warning, index) in selected.warnings"
                      :key="index"
                    >
                      {{ warning }}
                    </p>
                  </div>
                </div>
                <article
                  v-for="(item, index) in displayedItems"
                  :key="index"
                  class="memory-entry"
                >
                  <div class="entry-top">
                    <strong>{{
                      t("feedbackMemory.memoryItem", index + 1)
                    }}</strong>
                    <a-button
                      v-if="memoryEditing"
                      type="text"
                      danger
                      size="small"
                      :disabled="busy"
                      :aria-label="t('feedbackMemory.removeItem')"
                      @click="draftItems.splice(index, 1)"
                      ><template #icon><DeleteOutlined /></template
                    ></a-button>
                  </div>
                  <a-form
                    v-if="memoryEditing"
                    layout="vertical"
                    class="memory-item-editor"
                  >
                    <a-form-item :label="t('feedbackMemory.condition')"
                      ><a-textarea
                        v-model:value="item.condition"
                        :maxlength="1000"
                        :auto-size="{ minRows: 2, maxRows: 5 }"
                        :disabled="busy"
                    /></a-form-item>
                    <a-form-item
                      :label="t('feedbackMemory.memoryContent')"
                      required
                      ><a-textarea
                        v-model:value="item.content"
                        :maxlength="2000"
                        :auto-size="{ minRows: 3, maxRows: 10 }"
                        :disabled="busy"
                        show-count
                    /></a-form-item>
                    <a-form-item :label="t('feedbackMemory.basedOn')" required
                      ><a-select
                        v-model:value="item.source_refs"
                        mode="multiple"
                        :disabled="busy"
                        :options="
                          selected.sources.map((source, i) => ({
                            value: source.ref,
                            label: `${t('feedbackMemory.sourceNumber', i + 1)} · ${source.content.slice(0, 60)}`,
                          }))
                        "
                    /></a-form-item>
                  </a-form>
                  <template v-else>
                    <p class="entry-condition">
                      {{ t("feedbackMemory.condition") }}：{{
                        item.condition || t("feedbackMemory.defaultCondition")
                      }}
                    </p>
                    <p class="entry-content">{{ item.content }}</p>
                  </template>
                  <div class="source-links">
                    <span>{{ t("feedbackMemory.basedOn") }}</span
                    ><a-button
                      v-for="ref in item.source_refs"
                      :key="ref"
                      type="link"
                      size="small"
                      @click="showSource(ref)"
                      >{{
                        t(
                          "feedbackMemory.sourceNumber",
                          selected.sources.findIndex((row) => row.ref === ref) +
                            1,
                        )
                      }}</a-button
                    >
                  </div>
                </article>
                <a-button
                  v-if="memoryEditing"
                  :disabled="busy || draftItems.length >= 80"
                  @click="
                    draftItems.push({
                      condition: '',
                      content: '',
                      source_refs: [],
                    })
                  "
                  ><template #icon><PlusOutlined /></template
                  >{{ t("feedbackMemory.addItem") }}</a-button
                >
                <a-empty
                  v-if="!displayedItems.length"
                  :description="t('feedbackMemory.noReliableItems')"
                />
                <section class="final-prompt-preview">
                  <h3><EyeOutlined /> {{ t("feedbackMemory.finalPrompt") }}</h3>
                  <p class="memory-help">
                    {{ t("feedbackMemory.finalPromptHint") }}
                  </p>
                  <pre class="readonly-text">{{
                    finalPrompt || t("feedbackMemory.noReliableItems")
                  }}</pre>
                </section>
                <a-collapse ghost class="readonly-prompts">
                  <a-collapse-panel
                    key="trace"
                    :header="t('feedbackMemory.generationTrace')"
                  >
                    <p class="memory-help">
                      {{ t("feedbackMemory.traceHint") }}
                    </p>
                    <a-button
                      :loading="previewLoading"
                      :disabled="locked || memoryEditing"
                      @click="previewInput"
                      ><template #icon
                        ><EyeOutlined v-if="!previewLoading" /></template
                      >{{ t("feedbackMemory.previewInput") }}</a-button
                    >
                    <a-collapse ghost>
                      <a-collapse-panel
                        key="system"
                        :header="t('admin.data.llmSystemPrompt')"
                      >
                        <pre>{{ selected.system_prompt }}</pre>
                      </a-collapse-panel>
                      <a-collapse-panel
                        key="original"
                        :header="t('feedbackMemory.originalOutput')"
                      >
                        <pre>{{
                          JSON.stringify(selected.generated_items, null, 2)
                        }}</pre>
                      </a-collapse-panel>
                      <a-collapse-panel
                        key="input"
                        :header="t('feedbackMemory.actualInput')"
                      >
                        <pre>{{ selected.user_prompt }}</pre>
                      </a-collapse-panel>
                      <a-collapse-panel
                        key="sources"
                        :header="t('feedbackMemory.allSources')"
                        ><article
                          v-for="(source, index) in selected.sources"
                          :key="source.ref"
                          class="memory-entry"
                        >
                          <strong
                            >{{ t("feedbackMemory.sourceNumber", index + 1) }} ·
                            {{ sourceLabel(source.source_type) }}</strong
                          >
                          <p v-if="source.condition" class="entry-condition">
                            {{ source.condition }}
                          </p>
                          <p class="entry-content">{{ source.content }}</p>
                          <a-button
                            type="link"
                            size="small"
                            @click="activeSource = source"
                            >{{ t("feedbackMemory.sourceDetail") }}</a-button
                          >
                        </article></a-collapse-panel
                      >
                    </a-collapse>
                  </a-collapse-panel>
                </a-collapse>
              </template>
            </template>
            <section v-else class="feedback-history">
              <div class="history-title">
                <HistoryOutlined />
                <h3>{{ t("feedbackMemory.history") }}</h3>
                <a-tag>{{ versionTotal }}</a-tag>
              </div>
              <p>{{ t("feedbackMemory.historyHint") }}</p>
              <a-empty
                v-if="!versions.length"
                :description="t('feedbackMemory.emptyHistory')"
              />
              <button
                v-for="version in versions"
                :key="version.id"
                type="button"
                class="version-button"
                :class="{
                  'version-button--selected': selected?.id === version.id,
                }"
                :disabled="locked || memoryEditing"
                @click="selectVersion(version.id)"
              >
                <span class="version-row"
                  ><strong>v{{ version.version_no }}</strong
                  ><span>{{ versionStatus(version) }}</span></span
                >
                <span>{{ formatDate(version.created_at) }}</span
                ><span>{{ version.created_by }}</span>
              </button>
              <div v-if="versionTotal" class="pagination-wrapper">
                <a-pagination
                  :current="versionPage"
                  :page-size="versionSize"
                  :total="versionTotal"
                  size="small"
                  :show-size-changer="true"
                  :page-size-options="['10', '20', '50']"
                  :disabled="locked"
                  @change="changeVersionPage"
                />
              </div>
            </section>
          </div>
          <div v-else class="workspace-empty">
            <a-empty
              v-if="!loading"
              :description="
                t(error ? 'feedbackMemory.failed' : 'feedbackMemory.noConfigs')
              "
            />
          </div>
        </a-spin>
      </main>
    </div>

    <a-modal
      v-model:open="editorOpen"
      :title="t(editing ? 'feedbackMemory.edit' : 'feedbackMemory.addManual')"
      :confirm-loading="busy"
      :ok-text="t('feedbackMemory.save')"
      :cancel-text="t('feedbackMemory.cancel')"
      :closable="!busy"
      :mask-closable="false"
      @ok="saveEntry"
    >
      <a-form layout="vertical">
        <a-form-item :label="t('feedbackMemory.condition')"
          ><a-textarea
            v-model:value="form.condition"
            :maxlength="1000"
            :rows="3"
            :placeholder="t('feedbackMemory.conditionPlaceholder')"
        /></a-form-item>
        <a-form-item :label="t('feedbackMemory.content')" required
          ><a-textarea
            v-model:value="form.content"
            :readonly="editing?.source_type === 'comment'"
            :maxlength="2000"
            :rows="6"
            show-count
            :placeholder="t('feedbackMemory.contentPlaceholder')"
        /></a-form-item>
        <a-alert
          v-if="editing?.source_type === 'comment'"
          type="info"
          :message="t('feedbackMemory.commentReadonly')"
        />
        <a-form-item :label="t('feedbackMemory.enabled')"
          ><a-switch v-model:checked="form.enabled"
        /></a-form-item>
      </a-form>
    </a-modal>

    <a-drawer
      v-model:open="commentsOpen"
      root-class-name="memory-surface feedback-comments-drawer"
      :title="t('feedbackMemory.importComments')"
      width="min(850px, 96vw)"
      :mask-closable="!busy"
      :closable="!busy"
    >
      <a-alert
        type="info"
        show-icon
        :message="t('feedbackMemory.commentScope')"
        class="feedback-alert"
      />
      <div class="comment-filters">
        <a-input
          v-model:value="commentKeyword"
          class="comment-search"
          allow-clear
          :placeholder="t('feedbackMemory.searchComments')"
          :disabled="commentLoading || busy"
          @press-enter="loadComments(true)"
          ><template #prefix><SearchOutlined /></template
        ></a-input>
        <a-button
          :loading="commentLoading"
          :disabled="busy"
          :aria-label="t('feedbackMemory.searchComments')"
          @click="loadComments(true)"
          ><template #icon><SearchOutlined v-if="!commentLoading" /></template
        ></a-button>
        <a-select
          v-model:value="commentFeedback"
          allow-clear
          :placeholder="t('feedbackMemory.satisfaction')"
          :disabled="commentLoading || busy"
          @change="loadComments(true)"
          ><a-select-option value="like">{{
            t("feedbackMemory.like")
          }}</a-select-option
          ><a-select-option value="dislike">{{
            t("feedbackMemory.dislike")
          }}</a-select-option></a-select
        >
        <a-select
          v-model:value="commentDeprecation"
          :disabled="commentLoading || busy"
          :aria-label="t('feedbackMemory.deprecationFilter')"
          @change="loadComments(true)"
        >
          <a-select-option value="active">{{
            t("feedbackMemory.notDeprecated")
          }}</a-select-option>
          <a-select-option value="deprecated">{{
            t("feedbackMemory.deprecated")
          }}</a-select-option>
          <a-select-option value="all">{{
            t("feedbackMemory.allComments")
          }}</a-select-option>
        </a-select>
      </div>
      <a-spin :spinning="commentLoading">
        <a-empty
          v-if="!comments.length"
          :description="t('feedbackMemory.noComments')"
        />
        <a-checkbox-group
          v-model:value="selectedCommentIDs"
          class="comment-list"
        >
          <article
            v-for="comment in comments"
            :key="comment.id"
            class="memory-entry"
          >
            <div class="comment-heading">
              <a-checkbox
                :value="comment.id"
                :disabled="comment.imported || comment.deprecated || busy"
                ><strong>{{ comment.title }}</strong
                ><a-tag v-if="comment.imported">{{
                  t("feedbackMemory.imported")
                }}</a-tag></a-checkbox
              >
              <a-button
                size="small"
                type="text"
                :disabled="busy || commentLoading"
                @click="toggleCommentDeprecated(comment)"
              >
                <template #icon
                  ><UndoOutlined v-if="comment.deprecated" /><StopOutlined
                    v-else /></template
                >{{
                  t(
                    comment.deprecated
                      ? "feedbackMemory.restore"
                      : "feedbackMemory.deprecate",
                  )
                }}
              </a-button>
            </div>
            <a-tag v-if="comment.deprecated">{{
              t("feedbackMemory.deprecated")
            }}</a-tag>
            <div class="entry-meta">
              {{ comment.username }} · {{ formatDate(comment.updated_at) }} ·
              {{ comment.process_id
              }}<a-tag
                v-if="comment.feedback"
                :color="comment.feedback === 'like' ? 'green' : 'orange'"
                >{{ t(`feedbackMemory.${comment.feedback}`) }}</a-tag
              >
            </div>
            <p class="entry-content">{{ comment.content }}</p>
          </article>
        </a-checkbox-group>
      </a-spin>
      <div class="pagination-wrapper">
        <a-pagination
          :current="commentPage"
          :page-size="commentSize"
          :total="commentTotal"
          show-size-changer
          size="small"
          :page-size-options="['10', '20', '50']"
          :disabled="commentLoading || busy"
          @change="changeCommentPage"
        />
      </div>
      <template #footer
        ><div class="drawer-footer">
          <span>{{
            t("feedbackMemory.selectedCount", selectedCommentIDs.length)
          }}</span
          ><a-button
            type="primary"
            :loading="busy"
            :disabled="
              !selectedCommentIDs.length || selectedCommentIDs.length > 100
            "
            @click="importComments"
            >{{ t("feedbackMemory.importSelected") }}</a-button
          >
        </div></template
      >
    </a-drawer>

    <a-drawer
      v-model:open="previewOpen"
      :title="t('feedbackMemory.previewMemory')"
      width="min(900px, 96vw)"
      root-class-name="memory-surface"
    >
      <div class="runtime-preview">
        <p class="memory-help">{{ t("feedbackMemory.runtimePreviewHint") }}</p>
        <template v-if="selected">
          <div class="memory-notice">
            <InfoCircleOutlined />
            <div>
              <strong
                >v{{ selected.version_no }} ·
                {{ versionStatus(selected) }}</strong
              >
              <p>
                {{
                  t(
                    current && workspace?.memory?.enabled
                      ? "feedbackMemory.previewActive"
                      : "feedbackMemory.previewPending",
                  )
                }}
              </p>
              <p v-if="!current || !workspace?.memory?.enabled">
                {{
                  workspace?.memory?.enabled && workspace.published
                    ? t(
                        "feedbackMemory.currentRuntime",
                        workspace.published.version_no,
                      )
                    : t("feedbackMemory.runtimeDisabled")
                }}
              </p>
            </div>
          </div>
          <template
            v-if="!current && workspace?.memory?.enabled && workspace.published"
          >
            <h3>
              {{
                t(
                  "feedbackMemory.currentRuntime",
                  workspace.published.version_no,
                )
              }}
            </h3>
            <pre class="readonly-text">{{
              workspace.published.runtime_user_prompt
            }}</pre>
          </template>
          <h3>{{ t("feedbackMemory.runtimeMemoryBlock") }}</h3>
          <pre class="readonly-text">{{
            selected.runtime_user_prompt || t("feedbackMemory.runtimeNoMemory")
          }}</pre>
          <h3>{{ t("feedbackMemory.runtimeConstraint") }}</h3>
          <pre class="readonly-text">{{
            selected.runtime_system_prompt ||
            t("feedbackMemory.runtimeNoConstraint")
          }}</pre>
        </template>
      </div>
    </a-drawer>

    <a-drawer
      v-model:open="integrationPreviewOpen"
      :title="t('feedbackMemory.previewInput')"
      width="min(900px, 96vw)"
      root-class-name="memory-surface"
    >
      <p class="memory-help">{{ t("feedbackMemory.inputPreviewHint") }}</p>
      <h3>{{ t("admin.data.llmSystemPrompt") }}</h3>
      <pre class="readonly-text">{{ inputPreview?.system_prompt }}</pre>
      <h3>{{ t("feedbackMemory.actualInput") }}</h3>
      <pre class="readonly-text">{{ inputPreview?.user_prompt }}</pre>
    </a-drawer>

    <a-drawer
      v-model:open="promptsOpen"
      :title="t('feedbackMemory.generationRules')"
      width="min(800px, 96vw)"
    >
      <a-alert
        type="info"
        show-icon
        :message="t('feedbackMemory.readonlyHint')"
        class="feedback-alert"
      />
      <h3>{{ t("admin.data.llmSystemPrompt") }}</h3>
      <pre class="readonly-text">{{ workspace?.system_prompt }}</pre>
      <h3>{{ t("feedbackMemory.userTemplate") }}</h3>
      <pre class="readonly-text">{{ workspace?.user_prompt_template }}</pre>
    </a-drawer>
    <a-drawer
      :open="!!activeSource"
      :title="t('feedbackMemory.sourceDetail')"
      width="min(680px, 96vw)"
      @close="activeSource = null"
    >
      <template v-if="activeSource">
        <a-tag>{{ sourceLabel(activeSource.source_type) }}</a-tag>
        <p v-if="activeSource.condition" class="entry-condition">
          {{ t("feedbackMemory.condition") }}：{{ activeSource.condition }}
        </p>
        <p class="entry-content">{{ activeSource.content }}</p>
        <template v-if="sourceComment(activeSource)"
          ><a-descriptions :column="1" bordered size="small"
            ><a-descriptions-item :label="t('feedbackMemory.process')"
              >{{ sourceComment(activeSource)?.title }} ·
              {{ sourceComment(activeSource)?.process_id }}</a-descriptions-item
            ><a-descriptions-item :label="t('feedbackMemory.author')">{{
              sourceComment(activeSource)?.username
            }}</a-descriptions-item
            ><a-descriptions-item :label="t('feedbackMemory.sourceTime')">{{
              formatDate(sourceComment(activeSource)?.updated_at)
            }}</a-descriptions-item></a-descriptions
          ></template
        >
        <a-collapse ghost
          ><a-collapse-panel
            key="snapshot"
            :header="t('feedbackMemory.rawSnapshot')"
          >
            <pre class="readonly-text">{{
              JSON.stringify(activeSource.snapshot, null, 2)
            }}</pre>
          </a-collapse-panel></a-collapse
        >
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.feedback-workspace {
  color: var(--color-text-primary);
}
.workspace-layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}
.panel {
  min-width: 0;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
}
.process-library {
  position: sticky;
  top: 20px;
  overflow: hidden;
}
.library-top {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--color-border-light);
  font-size: 14px;
  font-weight: 600;
}
.library-count {
  margin-left: auto;
  font-size: 12px;
  font-weight: 400;
  color: var(--color-text-tertiary);
}
.library-controls {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border-light);
}
.scope-nav {
  display: flex;
  padding: 3px;
  gap: 4px;
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
}
.scope-nav button {
  flex: 1;
  border: 0;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 12px;
  cursor: pointer;
}
.scope-nav button.selected {
  background: var(--color-bg-card);
  color: var(--color-primary);
  box-shadow: var(--shadow-xs);
  font-weight: 600;
}
.library-list {
  max-height: calc(100vh - 400px);
  overflow-y: auto;
}
.library-item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  padding: 12px 16px 12px 13px;
  text-align: left;
  border: 0;
  border-left: 3px solid transparent;
  border-bottom: 1px solid var(--color-border-light);
  background: transparent;
  color: var(--color-text-primary);
  cursor: pointer;
  transition: background var(--transition-fast);
}
.library-item:last-child {
  border-bottom: 0;
}
.library-item:hover {
  background: var(--color-bg-hover);
}
.library-item.selected {
  border-left-color: var(--color-primary);
  background: var(--color-primary-bg);
}
.library-item.selected .item-name {
  color: var(--color-primary);
}
.library-item:disabled,
.scope-nav button:disabled {
  cursor: not-allowed;
}
.library-item:focus-visible,
.scope-nav button:focus-visible,
.workspace-tab-btn:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: -2px;
}
.item-name {
  font-size: 14px;
  font-weight: 500;
  overflow-wrap: anywhere;
}
.item-meta {
  font-size: 12px;
  line-height: 1.7;
  color: var(--color-text-tertiary);
  overflow-wrap: anywhere;
}
.library-empty {
  padding: 20px 16px;
  margin: 0;
  color: var(--color-text-tertiary);
  font-size: 12px;
  line-height: 1.7;
}
.workspace-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 16px;
  padding: 20px 20px 0;
}
.workspace-title {
  min-width: 0;
  flex: 1 1 220px;
}
.workspace-title h3 {
  font-size: 16px;
  margin: 0 0 6px;
  overflow-wrap: anywhere;
}
.workspace-title p {
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-text-tertiary);
  margin: 0;
}
.workspace-content {
  padding: 16px 20px 32px;
  min-height: 420px;
}
.workspace-status {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px 16px;
  margin-bottom: 20px;
  color: var(--color-text-tertiary);
  font-size: 12px;
}
.workspace-status strong {
  color: var(--color-text-secondary);
  font-weight: 500;
}
.status-label {
  display: inline-flex;
  align-items: center;
  border: 1px solid var(--color-border-light);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  border-radius: var(--radius-sm);
  padding: 1px 8px;
  font-size: 12px;
}
.status-label--active,
.source-tag {
  color: var(--color-primary);
  background: var(--color-primary-bg);
  border-color: var(--color-border-light);
}
.feedback-heading,
.entry-top,
.entry-meta,
.version-heading,
.version-row,
.history-title,
.drawer-footer {
  display: flex;
  align-items: center;
  gap: 12px;
}
.feedback-heading,
.entry-top,
.version-heading,
.version-row,
.drawer-footer {
  justify-content: space-between;
}
.feedback-heading {
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 20px;
}
.workspace-tab-nav {
  display: flex;
  width: fit-content;
  max-width: 100%;
  box-sizing: border-box;
  gap: 4px;
  padding: 4px;
  overflow-x: auto;
  scrollbar-width: none;
  background: var(--color-bg-hover);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
}
.workspace-tab-nav::-webkit-scrollbar {
  display: none;
}
.workspace-tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  padding: 7px 14px;
  border: 0;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-secondary);
  font-size: 13.5px;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
}
.workspace-tab-btn:hover {
  color: var(--color-text-primary);
}
.workspace-tab-btn.selected {
  color: var(--color-primary);
  background: var(--color-bg-card);
  box-shadow: var(--shadow-xs);
  font-weight: 600;
}
.feedback-alert {
  margin-bottom: 16px;
}
.workspace-error {
  margin: 20px 20px 0;
}
.workspace-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 420px;
}
.material-actions {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin: 20px 0;
  flex-wrap: wrap;
}
.material-actions p,
.feedback-history > p,
.version-heading p {
  color: var(--color-text-tertiary);
  font-size: 13px;
  margin: 0;
  line-height: 1.6;
}
.memory-entry {
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 16px;
  margin: 12px 0;
  background: var(--color-bg-card);
}
.memory-entry--disabled {
  background: var(--color-bg-hover);
  color: var(--color-text-tertiary);
}
.entry-top {
  align-items: flex-start;
  flex-wrap: wrap;
}
.entry-meta {
  color: var(--color-text-tertiary);
  font-size: 12px;
  flex-wrap: wrap;
  gap: 8px;
}
.entry-number {
  display: inline-flex;
  justify-content: center;
  align-items: center;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  border-radius: var(--radius-sm);
  width: 24px;
  height: 24px;
  font-weight: 600;
}
.entry-condition {
  background: var(--color-bg-hover);
  border-radius: var(--radius-sm);
  padding: 10px 12px;
  margin: 12px 0;
  font-size: 13px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.entry-content {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  line-height: 1.8;
  margin: 12px 0 4px;
  font-size: 14px;
}
.source-process {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin: 12px 0 0;
  overflow-wrap: anywhere;
}
.source-links {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  color: var(--color-text-tertiary);
  font-size: 12px;
  margin-top: 12px;
}
.version-heading {
  margin: 20px 0;
}
h3 {
  font-size: 15px;
  margin: 0 0 8px;
}
.history-title {
  margin-bottom: 12px;
}
.history-title h3 {
  flex: 1;
  margin: 0;
}
.version-button {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  text-align: left;
  padding: 14px;
  margin-top: 12px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  color: var(--color-text-secondary);
  cursor: pointer;
  font-size: 12px;
}
.version-row {
  width: 100%;
}
.version-row strong {
  color: var(--color-text-primary);
  font-size: 14px;
}
.version-button:hover,
.version-button--selected {
  border-color: var(--color-primary);
  background: var(--color-primary-bg);
}
.version-button:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
.version-button:disabled {
  cursor: wait;
}
.readonly-prompts {
  margin-top: 20px;
}
.readonly-prompts pre,
.readonly-text {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  font: 12px/1.8 var(--font-mono, monospace);
  max-height: 480px;
  overflow: auto;
  background: var(--color-bg-hover);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 16px;
}
.feedback-empty {
  margin: 64px 0;
}
.comment-list {
  display: block;
  width: 100%;
}
.comment-list .entry-meta {
  margin-top: 10px;
}
.comment-filters {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
.comment-search {
  flex: 1 1 240px;
  min-width: 160px;
}
.comment-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}
.comment-heading .ant-checkbox-wrapper {
  min-width: 0;
  overflow-wrap: anywhere;
}
.comment-heading .ant-btn {
  flex-shrink: 0;
}
.memory-notice {
  display: flex;
  gap: 10px;
  margin: 12px 0;
  padding: 12px 14px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.7;
}
.memory-notice > .anticon {
  color: var(--color-primary);
  margin-top: 4px;
}
.memory-notice p {
  margin: 4px 0 0;
}
.memory-item-editor {
  margin-top: 12px;
}
.memory-item-editor .ant-form-item {
  margin-bottom: 12px;
}
.final-prompt-preview {
  margin-top: 20px;
}
.runtime-preview {
  display: flex;
  flex-direction: column;
}
.runtime-preview h3 {
  margin: 28px 0 12px;
  line-height: 1.6;
}
.runtime-preview .readonly-text {
  margin: 0;
  padding: 18px 20px;
  line-height: 1.9;
}
.runtime-preview .memory-notice {
  margin: 12px 0 0;
}
.comment-filters .ant-select {
  min-width: 110px;
}
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
  flex-wrap: wrap;
}
@media (min-width: 1700px) {
  .workspace-layout {
    grid-template-columns: 260px minmax(0, 1fr);
  }
}
@media (max-width: 760px) {
  .workspace-layout {
    grid-template-columns: minmax(0, 1fr);
  }
  .process-library {
    position: static;
  }
  .library-list {
    min-height: 0;
    max-height: 200px;
  }
  .workspace-header {
    padding: 16px 16px 0;
  }
  .workspace-content {
    padding: 16px 16px 24px;
  }
  .workspace-error {
    margin: 16px 16px 0;
  }
  .comment-filters {
    flex-wrap: wrap;
  }
}
</style>
