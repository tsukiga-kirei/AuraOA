package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
)

func TestCronParserSupportsFiveAndSixFields(t *testing.T) {
	tests := []string{
		"0 9 * * 1-5",
		"0 0 9 * * 1-5",
	}
	for _, expr := range tests {
		if _, err := newCronParser().Parse(expr); err != nil {
			t.Fatalf("表达式 %q 应可解析: %v", expr, err)
		}
		if ParseNextRun(expr) == nil {
			t.Fatalf("表达式 %q 应能计算下次执行时间", expr)
		}
	}
}

func TestNormalizeSummaryTriggerDetail(t *testing.T) {
	detail, priority := normalizeSummaryTriggerDetail(
		model.SummaryTriggerEmbedAuto,
		model.SummaryTriggerDetailScheduled,
	)
	if detail != model.SummaryTriggerDetailScheduled || priority != model.SummaryPriorityScheduled {
		t.Fatalf("定时总结来源或优先级错误: detail=%s priority=%d", detail, priority)
	}
	detail, priority = normalizeSummaryTriggerDetail(
		model.SummaryTriggerEmbedManual,
		model.SummaryTriggerDetailScheduled,
	)
	if detail != model.SummaryTriggerDetailManual || priority != model.SummaryPriorityManual {
		t.Fatalf("手动总结必须覆盖来源并使用最高优先级: detail=%s priority=%d", detail, priority)
	}
}

func TestAIErrorRetryClassification(t *testing.T) {
	contextErr := errors.New("返回错误 (状态码 400): maximum context length; input_tokens")
	if isRetryableAIError(contextErr) {
		t.Fatal("上下文超限属于确定性 400 错误，不应重复调用")
	}
	if !isRetryableAIError(errors.New("connection reset by peer")) {
		t.Fatal("临时网络错误应允许重试")
	}
}

func TestAdjustMaxTokensForContextError(t *testing.T) {
	req := &ai.ChatRequest{MaxTokens: 8192}
	err := errors.New(
		"maximum context length is 131072 tokens. However, you requested 8192 output tokens " +
			"and your prompt contains at least 122881 input tokens",
	)
	adjusted, maxTokens := adjustMaxTokensForContextError(req, err)
	if !adjusted || maxTokens != 7679 || req.MaxTokens != 7679 {
		t.Fatalf("上下文超限应自动收缩输出预算，adjusted=%v max=%d req=%d", adjusted, maxTokens, req.MaxTokens)
	}
	adjusted, _ = adjustMaxTokensForContextError(req, err)
	if adjusted {
		t.Fatal("同一错误已调整到安全预算后不应再次调整")
	}
}

func TestNormalizeCronExpression(t *testing.T) {
	got := normalizeCronExpression("  0   9  * *  1-5 ")
	if got != "0 9 * * 1-5" {
		t.Fatalf("表达式清理结果错误: %q", got)
	}
}

func TestValidateCronExpression(t *testing.T) {
	if err := validateCronExpression("*/5 * * * *"); err != nil {
		t.Fatalf("五段式表达式应通过校验: %v", err)
	}
	if err := validateCronExpression("0 */10 * * * *"); err != nil {
		t.Fatalf("六段式表达式应通过校验: %v", err)
	}
	if err := validateCronExpression("not-a-cron"); err == nil {
		t.Fatal("非法表达式不应通过校验")
	}
}

func TestNormalizeScheduledRefreshConfig(t *testing.T) {
	lookbackDays, intervalMinutes := 0, 7
	normalizeScheduledRefreshConfig(&lookbackDays, &intervalMinutes)
	if lookbackDays != 3 || intervalMinutes != 5 {
		t.Fatalf("非法定时检查配置应回落默认值，得到 days=%d interval=%d", lookbackDays, intervalMinutes)
	}

	lookbackDays, intervalMinutes = 14, 10
	normalizeScheduledRefreshConfig(&lookbackDays, &intervalMinutes)
	if lookbackDays != 14 || intervalMinutes != 10 {
		t.Fatalf("合法定时检查配置不应改变，得到 days=%d interval=%d", lookbackDays, intervalMinutes)
	}
}

func TestNormalizeEmbedRefreshAction(t *testing.T) {
	if got := normalizeEmbedRefreshAction("submit"); got != "save_or_submit" {
		t.Fatalf("保存和提交应统一为 save_or_submit: %s", got)
	}
	if got := normalizeEmbedRefreshAction("unknown"); got != "save_or_submit" {
		t.Fatalf("未知事件动作应回落 save_or_submit: %s", got)
	}
	if got := normalizeEmbedRefreshAction("page_open"); got != "page_open" {
		t.Fatalf("旧版 page_open 应保留为兼容的忽略事件: %s", got)
	}
}

func TestBuildEmbedRefreshSchedule(t *testing.T) {
	schedule := buildEmbedRefreshSchedule(
		embedRefreshModuleAudit,
		uuid.New(),
		uuid.New(),
		"费用报销",
		true,
		3,
		10,
	)
	if !schedule.IsActive {
		t.Fatal("启用配置应生成活跃调度")
	}
	if schedule.CronExpression != "0 */10 * * * *" {
		t.Fatalf("Cron 表达式错误: %s", schedule.CronExpression)
	}
	if schedule.LookbackDays != 3 || schedule.IntervalMinutes != 10 {
		t.Fatalf("调度参数错误: days=%d interval=%d", schedule.LookbackDays, schedule.IntervalMinutes)
	}
}
