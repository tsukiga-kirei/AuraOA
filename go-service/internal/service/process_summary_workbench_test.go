package service

import (
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
