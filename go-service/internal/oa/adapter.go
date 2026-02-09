package oa

import "context"

type OAProcess struct {
	ProcessID   string `json:"process_id"`
	Title       string `json:"title"`
	Applicant   string `json:"applicant"`
	SubmitTime  string `json:"submit_time"`
	ProcessType string `json:"process_type"`
	Status      string `json:"status"`
}

type FormField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProcessFormData struct {
	ProcessID  string      `json:"process_id"`
	FormFields []FormField `json:"form_fields"`
	Applicant  string      `json:"applicant"`
	SubmitTime string      `json:"submit_time"`
}

// OAAdapter connects to OA database and fetches process data.
type OAAdapter interface {
	FetchTodoProcesses(ctx context.Context, userID string) ([]OAProcess, error)
	FetchProcessDetail(ctx context.Context, processID string) (ProcessFormData, error)
	HealthCheck(ctx context.Context) error
}

// AdapterRegistry manages OA adapters by type and version.
type AdapterRegistry interface {
	GetAdapter(oaType string, version string) (OAAdapter, error)
	RegisterAdapter(oaType string, version string, adapter OAAdapter) error
}
