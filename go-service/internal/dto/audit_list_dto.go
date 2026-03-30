package dto

import "time"

// AuditListParams 审核工作台列表查询（与归档列表类似：OA SQL 时间 + 后端分页）。
type AuditListParams struct {
	Tab     string `json:"tab"`
	Keyword string `json:"keyword"`
	// Applicant 申请人模糊匹配
	Applicant string `json:"applicant"`
	// ProcessType 多个流程类型名，逗号分隔，与 OA workflow_name 匹配
	ProcessType string `json:"process_type"`
	Department  string `json:"department"`
	// AuditStatus 审核建议：approve / return / review
	AuditStatus string `json:"audit_status"`
	Page        int    `json:"page"`
	PageSize    int    `json:"page_size"`
	// SubmitDateStart 待办流程按 OA 提交/创建时间下界（含），在 SQL 中作用于 requestbase.createdate
	SubmitDateStart *time.Time `json:"-"`
	// SubmitDateEndExclusive 上界（不含），通常为结束日次日 0 点
	SubmitDateEndExclusive *time.Time `json:"-"`
}

// AuditProcessListResponse 审核工作台分页列表。
type AuditProcessListResponse struct {
	Items    []map[string]interface{} `json:"items"`
	Total    int                        `json:"total"`
	Page     int                        `json:"page"`
	PageSize int                        `json:"page_size"`
}
