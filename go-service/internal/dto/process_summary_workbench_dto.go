package dto

import "time"

// SummaryWorkbenchListParams 流程总结工作台列表查询参数。
type SummaryWorkbenchListParams struct {
	Keyword                string
	Applicant              string
	Department             string
	ProcessType            string
	SummaryStatus          string
	Page                   int
	PageSize               int
	SubmitDateStart        *time.Time
	SubmitDateEndExclusive *time.Time
}

// SummaryWorkbenchProcessItem 流程总结工作台列表项。
type SummaryWorkbenchProcessItem struct {
	ProcessID        string      `json:"process_id"`
	Title            string      `json:"title"`
	Applicant        string      `json:"applicant"`
	Department       string      `json:"department"`
	ProcessType      string      `json:"process_type"`
	ProcessTypeLabel string      `json:"process_type_label"`
	CurrentNode      string      `json:"current_node"`
	SubmitTime       string      `json:"submit_time"`
	Source           string      `json:"source"`
	HasSummary       bool        `json:"has_summary"`
	SummaryStatus    string      `json:"summary_status"`
	SummaryResult    interface{} `json:"summary_result,omitempty"`
	SummaryUpdatedAt string      `json:"summary_updated_at,omitempty"`
	RunningJobID     string      `json:"running_job_id,omitempty"`
	VisibleBlockIDs  []string    `json:"visible_block_ids,omitempty"`
}

// SummaryWorkbenchListResponse 流程总结工作台分页响应。
type SummaryWorkbenchListResponse struct {
	Items    []SummaryWorkbenchProcessItem `json:"items"`
	Total    int                           `json:"total"`
	Page     int                           `json:"page"`
	PageSize int                           `json:"page_size"`
}

// SummaryWorkbenchStats 流程总结工作台统计。
type SummaryWorkbenchStats struct {
	TotalCount      int `json:"total_count"`
	SummarizedCount int `json:"summarized_count"`
	PendingCount    int `json:"pending_count"`
	RunningCount    int `json:"running_count"`
	FailedCount     int `json:"failed_count"`
}
