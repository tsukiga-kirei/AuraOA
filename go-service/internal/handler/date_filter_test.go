package handler

import (
	"auraoa/go-service/internal/pkg/apptime"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEndDateIncludesWholeBusinessDay(t *testing.T) {
	gin.SetMode(gin.TestMode)
	originalZone := apptime.Name()
	defer func() {
		if err := apptime.Configure(originalZone); err != nil {
			t.Fatal(err)
		}
	}()
	parsers := []struct {
		name  string
		parse func(*gin.Context) *time.Time
	}{
		{"审核日志", func(c *gin.Context) *time.Time { f, _, _ := parseAuditLogQuery(c); return f.EndDateExclusive }},
		{"审核快照", func(c *gin.Context) *time.Time { f, _, _ := parseAuditSnapshotQuery(c); return f.EndDateExclusive }},
		{"归档日志", func(c *gin.Context) *time.Time { f, _, _ := parseArchiveLogQuery(c); return f.EndDateExclusive }},
		{"归档快照", func(c *gin.Context) *time.Time { f, _, _ := parseArchiveSnapshotQuery(c); return f.EndDateExclusive }},
		{"总结快照", func(c *gin.Context) *time.Time {
			f, _, _ := parseProcessSummarySnapshotQuery(c)
			return f.EndDateExclusive
		}},
		{"定时任务日志", func(c *gin.Context) *time.Time { f, _, _ := parseCronLogQuery(c); return f.EndDateExclusive }},
		{"模型日志", func(c *gin.Context) *time.Time { f, _, _ := parseLLMLogQuery(c); return f.EndDateExclusive }},
	}
	for _, day := range []struct {
		zone, date string
		hours      int
	}{
		{"Asia/Shanghai", "2026-09-11", 24},
		{"America/New_York", "2026-03-08", 23},
		{"America/New_York", "2026-11-01", 25},
	} {
		if err := apptime.Configure(day.zone); err != nil {
			t.Fatal(err)
		}
		start, err := time.ParseInLocation("2006-01-02", day.date, apptime.Location())
		if err != nil {
			t.Fatal(err)
		}
		for _, parser := range parsers {
			t.Run(day.zone+day.date+parser.name, func(t *testing.T) {
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest("GET", "/?end_date="+day.date, nil)
				end := parser.parse(c)
				if end == nil || end.Sub(start) != time.Duration(day.hours)*time.Hour {
					t.Fatalf("自然日结束边界错误：%v", end)
				}
				last := time.Date(start.Year(), start.Month(), start.Day(), 23, 59, 59, 999999000, apptime.Location())
				if !last.Before(*end) || end.Hour() != 0 || end.Minute() != 0 || end.Second() != 0 {
					t.Fatalf("当天最后微秒必须包含，次日零点必须排除：%v", end)
				}
				invalid, _ := gin.CreateTestContext(httptest.NewRecorder())
				invalid.Request = httptest.NewRequest("GET", "/?end_date=invalid", nil)
				if parser.parse(invalid) != nil {
					t.Fatal("非法日期应保持原来的忽略行为")
				}
			})
		}
	}
}
