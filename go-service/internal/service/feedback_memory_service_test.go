package service

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/apptime"
)

// TestFeedbackMemoryValidation 验证虚构来源、无来源、重复来源及超长输出不能成为记忆。
func TestFeedbackMemoryValidation(t *testing.T) {
	sources := []FeedbackMemorySource{{Ref: "comment-1"}, {Ref: "previous:v1:1"}}
	valid := []model.FeedbackMemoryItem{{Content: "明确缺失材料", Condition: "资料不足", SourceRefs: []string{"comment-1", "previous:v1:1"}}}
	if err := validateFeedbackMemoryItems(valid, sources); err != nil {
		t.Fatal(err)
	}
	for _, items := range [][]model.FeedbackMemoryItem{
		{{Content: "结论", SourceRefs: []string{"foreign"}}}, {{Content: "结论"}}, {{Content: "  ", SourceRefs: []string{"comment-1"}}},
		{{Content: "结论", SourceRefs: []string{"comment-1", "comment-1"}}}, {{Content: strings.Repeat("字", 2001), SourceRefs: []string{"comment-1"}}},
	} {
		if validateFeedbackMemoryItems(items, sources) == nil {
			t.Fatal("不合法的来源或正文通过校验")
		}
	}
}

// TestFeedbackMemoryIncrementalSources 验证只使用新素材，删除素材不会从旧记忆复活，合并后的其他来源仍保留。
func TestFeedbackMemoryIncrementalSources(t *testing.T) {
	oldID, newID, deletedID := uuid.New(), uuid.New(), uuid.New()
	versionID := uuid.New()
	now := apptime.Now()
	w := &dto.FeedbackMemoryWorkspace{Baseline: &model.FeedbackMemoryVersion{ID: versionID, CreatedAt: now}, Entries: []model.FeedbackMemoryEntry{
		{ID: oldID, Enabled: true, ImplantedVersionID: &versionID, Content: "旧素材"},
		{ID: newID, Enabled: true, Content: "新素材"},
		{ID: uuid.New(), Enabled: false, Content: "未参与素材"},
	}, DeletedEntries: []model.FeedbackMemoryEntry{{ID: deletedID, DeletedAt: &now, Content: "已删除依据"}}}
	inherited := []FeedbackMemorySource{{Ref: oldID.String(), Content: "旧素材"}, {Ref: deletedID.String(), Content: "已删除依据"}}
	items := []model.FeedbackMemoryItem{{Content: "合并记忆", SourceRefs: []string{oldID.String(), deletedID.String()}}, {Content: "应被撤销", SourceRefs: []string{deletedID.String()}}}
	sources, previous, deleted := buildFeedbackMemorySources(w, items, inherited)
	if len(sources) != 2 || !sources[0].Inherited || sources[1].Ref != newID.String() || sources[1].Inherited {
		t.Fatalf("已植入素材重复输入或新素材缺失: %+v", sources)
	}
	if len(previous) != 1 || len(previous[0].SourceRefs) != 1 || previous[0].SourceRefs[0] != oldID.String() {
		t.Fatalf("删除来源未剔除或有效来源丢失: %+v", previous)
	}
	if len(deleted) != 1 || deleted[0].Ref != deletedID.String() {
		t.Fatal("未告知AI删除的素材")
	}
	if validateFeedbackMemoryItems([]model.FeedbackMemoryItem{{Content: "恢复已删除建议", SourceRefs: []string{deletedID.String()}}}, sources) == nil {
		t.Fatal("已删除来源仍可被引用")
	}
	// 人工移除条目后，已植入原文也不能重新作为新素材提炼。
	sources, previous, _ = buildFeedbackMemorySources(w, nil, inherited)
	if len(previous) != 0 || len(sources) != 1 || sources[0].Ref != newID.String() {
		t.Fatal("人工移除的记忆从原素材复活")
	}
}

// TestFeedbackMemoryLegacyReferences 验证旧previous链展开后保留原始评论快照与稳定引用。
func TestFeedbackMemoryLegacyReferences(t *testing.T) {
	oldID, entryID, memoryID := uuid.New(), uuid.New(), uuid.New()
	oldItems, _ := json.Marshal([]model.FeedbackMemoryItem{{Content: "旧记忆", SourceRefs: []string{entryID.String()}}})
	oldSources, _ := json.Marshal([]FeedbackMemorySource{{Ref: entryID.String(), SourceType: "comment", Content: "原始评论"}})
	parent := &model.FeedbackMemoryVersion{ID: oldID, MemoryID: memoryID, Items: oldItems, Sources: oldSources}
	ref := "previous:" + oldID.String() + ":1"
	currentItems, _ := json.Marshal([]model.FeedbackMemoryItem{{Content: "人工调整后的记忆", SourceRefs: []string{ref}}})
	currentSources, _ := json.Marshal([]FeedbackMemorySource{{Ref: ref, SourceType: "previous"}})
	v := &model.FeedbackMemoryVersion{MemoryID: memoryID, BaseVersionID: &oldID, Items: currentItems, Sources: currentSources}
	items, sources, err := normalizeFeedbackMemoryVersion(v, func(id uuid.UUID) (*model.FeedbackMemoryVersion, error) {
		if id != oldID {
			t.Fatal("来源版本错误")
		}
		return parent, nil
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].Content != "人工调整后的记忆" || items[0].SourceRefs[0] != entryID.String() || sources[0].Content != "原始评论" {
		t.Fatal("旧引用展开丢失人工调整或原始依据")
	}
	parent.MemoryID = uuid.New()
	if _, _, err := normalizeFeedbackMemoryVersion(v, func(uuid.UUID) (*model.FeedbackMemoryVersion, error) { return parent, nil }, 0); err == nil {
		t.Fatal("允许引用其他记忆库")
	}
}

// TestFeedbackMemoryMergeReferences 验证相同记忆合并后的多个来源均保留。
func TestFeedbackMemoryMergeReferences(t *testing.T) {
	items := mergeFeedbackMemoryDuplicates([]model.FeedbackMemoryItem{{Content: " 核对历史月份 ", Condition: "月度流程", SourceRefs: []string{"a"}}, {Content: "核对历史月份", Condition: "月度流程", SourceRefs: []string{"a", "b"}}, {Content: "核对历史月份", Condition: "其他流程", SourceRefs: []string{"c"}}})
	if len(items) != 2 || len(items[0].SourceRefs) != 2 || items[0].SourceRefs[1] != "b" {
		t.Fatal("重复条目未合并或来源丢失")
	}
}

// TestFeedbackMemoryRuntimePreview 验证页面预览与实际执行注入使用完全相同的记忆段及约束。
func TestFeedbackMemoryRuntimePreview(t *testing.T) {
	v := &model.FeedbackMemoryVersion{ID: uuid.New(), VersionNo: 3, CreatedAt: apptime.Now(), PromptText: "1. 适用条件：月度费用\n参考记忆：对比历史金额"}
	fillFeedbackMemoryRuntimePreview(v)
	req := &ai.ChatRequest{UserPrompt: "{{feedback_memory}}"}
	applyFeedbackMemory(req, feedbackMemoryRuntimeText(v))
	if v.RuntimeUserPrompt != req.UserPrompt || v.RuntimeSystemPrompt != strings.TrimSpace(req.SystemPrompt) {
		t.Fatal("预览与执行注入不一致")
	}
	if !strings.Contains(v.RuntimeUserPrompt, v.ID.String()) || !strings.Contains(v.RuntimeUserPrompt, "对比历史金额") || strings.Contains(v.RuntimeUserPrompt, "source_refs") {
		t.Fatal("审核预览错误地包含整合输入或遗漏记忆")
	}
	v.PromptText = ""
	fillFeedbackMemoryRuntimePreview(v)
	if v.RuntimeSystemPrompt != "" || v.RuntimeUserPrompt != "" {
		t.Fatal("空记忆预览不一致")
	}
}

// TestFeedbackMemoryPrompt 不论新旧模板均自动追加一次记忆，旧变量不重复注入，未启用时不注入。
func TestFeedbackMemoryPrompt(t *testing.T) {
	for _, pair := range [][2]string{{"规则", "业务数据"}, {"规则 {{feedback_memory}}", "业务数据"}, {"规则", "业务 {{feedback_memory}}"}, {"规则 {{feedback_memory}}", "业务 {{feedback_memory}} {{feedback_memory}}"}} {
		req := &ai.ChatRequest{SystemPrompt: pair[0], UserPrompt: pair[1], CallType: "reasoning"}
		applyFeedbackMemory(req, "反馈记忆 v2：只对材料不足时说明缺项")
		if strings.Count(req.UserPrompt, "反馈记忆 v2") != 1 || strings.Contains(req.SystemPrompt, "反馈记忆 v2") || strings.Contains(req.SystemPrompt+req.UserPrompt, "{{feedback_memory}}") || !strings.Contains(req.SystemPrompt, "不得覆盖正式强制规则") {
			t.Fatalf("反馈记忆注入错误: %+v", req)
		}
	}
	req := &ai.ChatRequest{SystemPrompt: "原系统", UserPrompt: "原用户"}
	applyFeedbackMemory(req, "")
	if req.SystemPrompt != "原系统" || req.UserPrompt != "原用户" {
		t.Fatal("未发布时改变了原提示词")
	}
	req.UserPrompt = "{{feedback_memory}}"
	applyFeedbackMemory(req, "")
	if strings.Contains(req.UserPrompt, "{{") {
		t.Fatal("空记忆留下未替换变量")
	}
}

// TestFeedbackMemoryExtraction 结构化只消费已有分析，不再次传入原记忆或残留旧模板变量。
func TestFeedbackMemoryExtraction(t *testing.T) {
	for _, module := range []string{"audit", "archive"} {
		req := &ai.ChatRequest{SystemPrompt: "结构化要求 {{feedback_memory}}", UserPrompt: "分析结论：已结合历史金额，要求核对异常波动。 {{feedback_memory}}", CallType: "structured", RequestType: module}
		applyFeedbackMemory(req, "反馈记忆 v3：原始记忆正文")
		if strings.Contains(req.SystemPrompt+req.UserPrompt, "原始记忆正文") || strings.Contains(req.SystemPrompt+req.UserPrompt, "{{feedback_memory}}") || !strings.Contains(req.SystemPrompt, "不重新审核") || !strings.Contains(req.UserPrompt, "已结合历史金额") {
			t.Fatalf("结构化阶段重复注入或丢失既有分析: %+v", req)
		}
	}
}

// TestFeedbackMemoryRequiresTenant 在任何数据库访问前拒绝缺少租户的请求。
func TestFeedbackMemoryRequiresTenant(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	s := &FeedbackMemoryService{}
	if _, err := s.Workspace(c, dto.FeedbackMemoryScope{Module: "audit", ProcessType: "100"}); err == nil {
		t.Fatal("缺少租户时必须拒绝")
	}
	if _, err := s.RuntimeText(c, "audit", "100"); err == nil {
		t.Fatal("执行链缺少租户时必须拒绝")
	}
}
