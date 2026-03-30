package dto

// ArchiveReviewExecuteRequest 发起归档复盘请求。
type ArchiveReviewExecuteRequest struct {
	ProcessID   string `json:"process_id" binding:"required"`
	ProcessType string `json:"process_type" binding:"required"`
	Title       string `json:"title"`
}

// ArchiveReviewSubmitResponse 归档复盘任务提交响应。
type ArchiveReviewSubmitResponse struct {
	Status    string `json:"status"`
	ID        string `json:"id"`
	TraceID   string `json:"trace_id"`
	ProcessID string `json:"process_id"`
	CreatedAt string `json:"created_at"`
}

// ArchiveBatchExecuteRequest 批量提交归档复盘任务。
type ArchiveBatchExecuteRequest struct {
	Items []ArchiveReviewExecuteRequest `json:"items" binding:"required"`
}

// ArchiveBatchExecuteResponse 批量提交结果。
type ArchiveBatchExecuteResponse struct {
	Results  []ArchiveReviewSubmitResponse `json:"results"`
	Total    int                           `json:"total"`
	Accepted int                           `json:"accepted"`
	Failed   int                           `json:"failed"`
}

// ArchiveReviewStats 归档复盘统计。
type ArchiveReviewStats struct {
	TotalCount        int `json:"total_count"`
	CompliantCount    int `json:"compliant_count"`
	PartialCount      int `json:"partial_count"`
	NonCompliantCount int `json:"non_compliant_count"`
	UnauditedCount    int `json:"unaudited_count"`
	RunningCount      int `json:"running_count"`
}

// ArchiveListParams 归档流程列表查询参数。
type ArchiveListParams struct {
	Keyword     string `json:"keyword"`
	Applicant   string `json:"applicant"`
	ProcessType string `json:"process_type"`
	Department  string `json:"department"`
	AuditStatus string `json:"audit_status"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
}

// ArchiveProcessListResponse 已归档流程列表响应。
type ArchiveProcessListResponse struct {
	Items    []map[string]interface{} `json:"items"`
	Total    int                      `json:"total"`
	Page     int                      `json:"page"`
	PageSize int                      `json:"page_size"`
}
