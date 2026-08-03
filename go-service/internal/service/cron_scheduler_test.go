package service

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/ai"
	"auraoa/go-service/internal/pkg/oa"
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
	detail, queueKind := normalizeSummaryTriggerDetail(
		model.SummaryTriggerEmbedAuto,
		model.SummaryTriggerDetailScheduled,
	)
	if detail != model.SummaryTriggerDetailScheduled || queueKind != model.JobQueueKindScheduled {
		t.Fatalf("定时总结来源或队列类型错误: detail=%s queue_kind=%s", detail, queueKind)
	}
	detail, queueKind = normalizeSummaryTriggerDetail(
		model.SummaryTriggerEmbedAuto,
		model.SummaryTriggerDetailSaveRequested,
	)
	if detail != model.SummaryTriggerDetailSaveRequested || queueKind != model.JobQueueKindBackground {
		t.Fatalf("保存请求总结来源或队列类型错误: detail=%s queue_kind=%s", detail, queueKind)
	}
	detail, queueKind = normalizeSummaryTriggerDetail(
		model.SummaryTriggerEmbedManual,
		model.SummaryTriggerDetailScheduled,
	)
	if detail != model.SummaryTriggerDetailManual || queueKind != model.JobQueueKindInteractive {
		t.Fatalf("手动总结必须覆盖来源并进入交互队列: detail=%s queue_kind=%s", detail, queueKind)
	}
}

func TestNormalizeAuditTriggerDetail(t *testing.T) {
	detail, queueKind := normalizeAuditTriggerDetail(
		model.AuditTriggerEmbedAuto,
		model.SummaryTriggerDetailScheduled,
	)
	if detail != model.SummaryTriggerDetailScheduled || queueKind != model.JobQueueKindScheduled {
		t.Fatalf("定时嵌入审核来源或队列类型错误: detail=%s queue_kind=%s", detail, queueKind)
	}
	detail, queueKind = normalizeAuditTriggerDetail(
		model.AuditTriggerEmbedAuto,
		model.SummaryTriggerDetailSubmitRequested,
	)
	if detail != model.SummaryTriggerDetailSubmitRequested || queueKind != model.JobQueueKindBackground {
		t.Fatalf("提交请求审核来源或队列类型错误: detail=%s queue_kind=%s", detail, queueKind)
	}
	detail, queueKind = normalizeAuditTriggerDetail(
		model.AuditTriggerEmbedManual,
		model.SummaryTriggerDetailScheduled,
	)
	if detail != model.SummaryTriggerDetailManual || queueKind != model.JobQueueKindInteractive {
		t.Fatalf("手动嵌入审核必须覆盖来源并进入交互队列: detail=%s queue_kind=%s", detail, queueKind)
	}
	_, queueKind = normalizeAuditTriggerDetail(model.AuditTriggerWorkbenchManual, "")
	if queueKind != model.JobQueueKindWorkbench {
		t.Fatalf("审核工作台必须进入独立工作台队列: queue_kind=%s", queueKind)
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

func TestEmbedRefreshActionVersionBoundary(t *testing.T) {
	for _, action := range []string{"page_open", "save", "submit", "save_or_submit", "save_complete", "unknown"} {
		if !isObsoleteEmbedRefreshAction(action) {
			t.Fatalf("旧动作应被识别并清理: %s", action)
		}
	}
	if isObsoleteEmbedRefreshAction(model.SummaryTriggerDetailSaveRequested) {
		t.Fatal("save_requested 不应被识别为旧动作")
	}
	if isObsoleteEmbedRefreshAction(model.SummaryTriggerDetailSubmitRequested) {
		t.Fatal("submit_requested 不应被识别为旧动作")
	}
	if isObsoleteEmbedRefreshAction(model.SummaryTriggerDetailScheduled) {
		t.Fatal("scheduled_scan 不应被识别为旧动作")
	}
	if !shouldRetryEmbedEvent(model.SummaryTriggerDetailSaveRequested) ||
		!shouldRetryEmbedEvent(model.SummaryTriggerDetailSubmitRequested) {
		t.Fatal("保存和提交请求都应支持延迟重试")
	}
	if shouldRetryEmbedEvent("save_or_submit") {
		t.Fatal("旧动作 save_or_submit 不应继续重试")
	}
}

func TestEmbedRefreshResultName(t *testing.T) {
	tests := []struct {
		result embedRefreshResult
		want   string
	}{
		{result: embedRefreshDone, want: "done"},
		{result: embedRefreshRetry, want: "retry"},
		{result: embedRefreshRunning, want: "running"},
	}
	for _, tt := range tests {
		if got := embedRefreshResultName(tt.result); got != tt.want {
			t.Fatalf("刷新检查状态名称错误: result=%d got=%s want=%s", tt.result, got, tt.want)
		}
	}
}

func TestSelectResolvedProcessCandidate(t *testing.T) {
	sole := []oa.ProcessRequestCandidate{{ProcessID: "617100", CreatorID: "198"}}
	selected, matchCount := selectResolvedProcessCandidate(sole, "28")
	if selected == nil || selected.ProcessID != "617100" || matchCount != 0 {
		t.Fatalf("唯一候选不应被人员标识排除: selected=%+v matches=%d", selected, matchCount)
	}

	multiple := []oa.ProcessRequestCandidate{
		{ProcessID: "617101", CreatorID: "198"},
		{ProcessID: "617102", CreatorID: "28"},
	}
	selected, matchCount = selectResolvedProcessCandidate(multiple, "28", "29")
	if selected == nil || selected.ProcessID != "617102" || matchCount != 1 {
		t.Fatalf("多候选时应使用人员标识辅助消歧: selected=%+v matches=%d", selected, matchCount)
	}

	selected, matchCount = selectResolvedProcessCandidate(multiple, "99")
	if selected != nil || matchCount != 0 {
		t.Fatalf("无法消歧时不应猜测候选: selected=%+v matches=%d", selected, matchCount)
	}

	sameCreator := []oa.ProcessRequestCandidate{
		{ProcessID: "617103", CreatorID: "28"},
		{ProcessID: "617104", CreatorID: "28"},
	}
	selected, matchCount = selectResolvedProcessCandidate(sameCreator, "28")
	if selected != nil || matchCount != 2 {
		t.Fatalf("人员标识仍匹配多个候选时不应猜测: selected=%+v matches=%d", selected, matchCount)
	}
}

func TestEmbedRefreshClientDelay(t *testing.T) {
	received := time.UnixMilli(10_000)
	if got := embedRefreshClientDelay(8_500, received); got != 1_500 {
		t.Fatalf("客户端延迟计算错误: got=%d", got)
	}
	if got := embedRefreshClientDelay(11_000, received); got != 0 {
		t.Fatalf("客户端时钟超前时延迟应归零: got=%d", got)
	}
	if got := embedRefreshClientDelay(0, received); got != 0 {
		t.Fatalf("缺少客户端时间时延迟应为零: got=%d", got)
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

	inactive := buildEmbedRefreshSchedule(
		embedRefreshModuleAudit,
		uuid.New(),
		uuid.New(),
		"项目立项申请流程",
		false,
		3,
		5,
	)
	if inactive.IsActive {
		t.Fatal("默认关闭的流程配置不应生成活跃调度")
	}
}

func TestScheduleSourceConfigMustBeExplicitlyEnabled(t *testing.T) {
	audit := &model.ProcessAuditConfig{
		Status:       "active",
		EmbedEnabled: true,
		EmbedConfig:  datatypes.JSON([]byte(`{"scheduled_refresh_enabled":true}`)),
	}
	if !isAuditScheduleConfigActive(audit) {
		t.Fatal("满足全部条件的审核配置应允许定时检查")
	}
	audit.EmbedEnabled = false
	if isAuditScheduleConfigActive(audit) {
		t.Fatal("未启用 OA 嵌入审核时不应允许定时检查")
	}

	summary := &model.ProcessSummaryConfig{
		Status:       "active",
		EmbedEnabled: true,
		EmbedConfig:  datatypes.JSON([]byte(`{"scheduled_refresh_enabled":false}`)),
	}
	if isSummaryScheduleConfigActive(summary) {
		t.Fatal("未明确开启定时检查的总结配置不应允许执行")
	}
}
