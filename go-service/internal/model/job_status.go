package model

// 公共任务异步状态常量（模块无关，audit 和 archive 共用）
const (
	JobStatusPending    = "pending"
	JobStatusAssembling = "assembling"
	JobStatusReasoning  = "reasoning"
	JobStatusExtracting = "extracting"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)
