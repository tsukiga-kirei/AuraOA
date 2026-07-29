package service

import (
	"testing"

	"github.com/google/uuid"
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
	if got := normalizeEmbedRefreshAction("submit"); got != "submit" {
		t.Fatalf("合法事件动作不应改变: %s", got)
	}
	if got := normalizeEmbedRefreshAction("unknown"); got != "save_or_submit" {
		t.Fatalf("未知事件动作应回落 save_or_submit: %s", got)
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
