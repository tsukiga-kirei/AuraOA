package service

import (
	jwtpkg "auraoa/go-service/internal/pkg/jwt"
	"auraoa/go-service/internal/pkg/oa"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/repository"
)

func TestSummaryWorkbenchTextMatches(t *testing.T) {
	tests := []struct {
		name   string
		params dto.SummaryWorkbenchListParams
		want   bool
	}{
		{name: "空筛选", want: true},
		{name: "标题忽略大小写", params: dto.SummaryWorkbenchListParams{Keyword: "CONTRACT"}, want: true},
		{name: "申请人模糊匹配", params: dto.SummaryWorkbenchListParams{Applicant: "张"}, want: true},
		{name: "部门精确匹配", params: dto.SummaryWorkbenchListParams{Department: "财务部"}, want: true},
		{name: "多流程类型命中", params: dto.SummaryWorkbenchListParams{ProcessType: "expense, contract"}, want: true},
		{name: "流程类型未命中", params: dto.SummaryWorkbenchListParams{ProcessType: "purchase"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summaryWorkbenchTextMatches(tt.params, "Contract Review", "张三", "财务部", "contract")
			if got != tt.want {
				t.Fatalf("summaryWorkbenchTextMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummaryWorkbenchStatusMatches(t *testing.T) {
	tests := []struct {
		name   string
		filter string
		item   dto.SummaryWorkbenchProcessItem
		want   bool
	}{
		{name: "已有总结", filter: "summarized", item: dto.SummaryWorkbenchProcessItem{HasSummary: true}, want: true},
		{name: "待生成", filter: "pending", item: dto.SummaryWorkbenchProcessItem{SummaryStatus: "pending"}, want: true},
		{name: "生成中", filter: "running", item: dto.SummaryWorkbenchProcessItem{SummaryStatus: model.JobStatusReasoning, RunningJobID: "job-1"}, want: true},
		{name: "失败", filter: "failed", item: dto.SummaryWorkbenchProcessItem{SummaryStatus: model.JobStatusFailed}, want: true},
		{name: "失败不算待生成", filter: "pending", item: dto.SummaryWorkbenchProcessItem{SummaryStatus: model.JobStatusFailed}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := summaryWorkbenchStatusMatches(tt.filter, tt.item); got != tt.want {
				t.Fatalf("summaryWorkbenchStatusMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeWeeklyTrendIncludesSummary(t *testing.T) {
	got := mergeWeeklyTrend(
		[]repository.DayCount{{Date: "09-01", Count: 1}},
		[]repository.DayCount{{Date: "09-01", Count: 2}},
		[]repository.DayCount{{Date: "09-01", Count: 3}},
		[]repository.DayCount{{Date: "09-01", Count: 4}},
		[]repository.DayCount{{Date: "09-01", Count: 5}},
	)
	if len(got) != 1 || got[0].AuditCount+got[0].CronCount+got[0].ArchiveCount+got[0].SummaryCount+got[0].ChatCount != 15 || got[0].SummaryCount != 4 || got[0].ChatCount != 5 {
		t.Fatalf("mergeWeeklyTrend() = %#v", got)
	}
}

func TestSummaryTodoCountIncludesExistingEmbedResults(t *testing.T) {
	items := []dto.SummaryWorkbenchProcessItem{
		{Source: "todo", HasSummary: true},
		{Source: "todo", SummaryStatus: model.JobStatusPending},
		{Source: "archived", HasSummary: true},
	}
	stats := summaryWorkbenchStatsFromItems(items)
	if stats.TotalCount != 3 || stats.TodoCount != 2 || stats.SummarizedCount != 2 || stats.PendingCount != 1 {
		t.Fatalf("待办应包含已有通用总结与未生成流程，且不包含归档：%+v", stats)
	}
}

type summaryVisibilityAdapter struct {
	oa.OAAdapter
	visible bool
	err     error
}

func (a summaryVisibilityAdapter) CheckProcessVisibility(context.Context, string, string) (bool, error) {
	return a.visible, a.err
}

func TestSummaryVisibilityIsPersonalForEveryRole(t *testing.T) {
	for _, role := range []string{"business", "tenant_admin"} {
		for _, visible := range []bool{false, true} {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("GET", "/api/summary/history/42", nil)
			c.Set("jwt_claims", &jwtpkg.JWTClaims{ActiveRole: jwtpkg.ActiveRoleClaim{Role: role}})
			svc := &ProcessSummaryService{}
			got, err := svc.userCanAccessSummaryProcess(c, summaryVisibilityAdapter{visible: visible}, "user", "42")
			if err != nil || got != visible {
				t.Fatalf("角色 %s 的访问结果应只由 OA 参与权限决定：%v %v", role, got, err)
			}
			got, err = svc.userCanAccessSummaryProcess(c, summaryVisibilityAdapter{err: errors.New("OA unavailable")}, "user", "42")
			if err == nil || got {
				t.Fatal("OA 校验失败不得放行")
			}
		}
	}
}
