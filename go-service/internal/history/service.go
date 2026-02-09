package history

import (
	"context"
	"io"
	"oa-smart-audit/internal/oa"
	"oa-smart-audit/internal/rule"
	"time"
)

type AuditResult struct {
	Recommendation string       `json:"recommendation"` // approve | reject | revise
	Details        []RuleResult `json:"details"`
}

type RuleResult struct {
	RuleID    string `json:"rule_id"`
	Passed    bool   `json:"passed"`
	Reasoning string `json:"reasoning"`
}

type UserFeedback struct {
	Adopted     bool      `json:"adopted"`
	ActionTaken string    `json:"action_taken"`
	FeedbackAt  time.Time `json:"feedback_at"`
}

type AuditSnapshot struct {
	ID           string              `json:"snapshot_id"`
	TenantID     string              `json:"tenant_id"`
	UserID       string              `json:"user_id"`
	ProcessID    string              `json:"process_id"`
	FormInput    oa.ProcessFormData  `json:"form_input"`
	ActiveRules  []rule.MergedRule   `json:"active_rules"`
	AIReasoning  string              `json:"ai_reasoning"`
	AuditResult  AuditResult         `json:"audit_result"`
	UserFeedback *UserFeedback       `json:"user_feedback,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	OperatorID   string              `json:"operator_id"`
}

type SearchQuery struct {
	TenantID    string     `json:"tenant_id"`
	TimeFrom    *time.Time `json:"time_from,omitempty"`
	TimeTo      *time.Time `json:"time_to,omitempty"`
	Department  string     `json:"department,omitempty"`
	ProcessType string     `json:"process_type,omitempty"`
	Page        int        `json:"page"`
	PageSize    int        `json:"page_size"`
}

type SearchResult struct {
	Total     int64           `json:"total"`
	Snapshots []AuditSnapshot `json:"snapshots"`
}

type ExportFormat string

const (
	ExportJSON ExportFormat = "json"
	ExportCSV  ExportFormat = "csv"
)

// HistoryService manages audit snapshots (append-only, immutable).
type HistoryService interface {
	SaveAuditSnapshot(ctx context.Context, snapshot AuditSnapshot) error
	SearchSnapshots(ctx context.Context, query SearchQuery) (SearchResult, error)
	ExportSnapshots(ctx context.Context, query SearchQuery, format ExportFormat) (io.Reader, error)
}
