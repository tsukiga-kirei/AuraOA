package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
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

// TestFeedbackMemoryPrompt 两阶段、系统与用户模板均支持变量，旧模板自动追加，未发布不注入。
func TestFeedbackMemoryPrompt(t *testing.T) {
	for _, kind := range []string{"reasoning", "structured"} {
		for _, pair := range [][2]string{{"规则", "业务数据"}, {"规则 {{feedback_memory}}", "业务数据"}, {"规则", "业务 {{feedback_memory}}"}} {
			req := &ai.ChatRequest{SystemPrompt: pair[0], UserPrompt: pair[1], CallType: kind}
			applyFeedbackMemory(req, "反馈记忆 v2：只对材料不足时说明缺项")
			if strings.Count(req.SystemPrompt+req.UserPrompt, "反馈记忆 v2") != 1 || strings.Contains(req.SystemPrompt+req.UserPrompt, "{{feedback_memory}}") || !strings.Contains(req.SystemPrompt, "不得覆盖正式强制规则") {
				t.Fatalf("反馈记忆注入错误: %+v", req)
			}
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
