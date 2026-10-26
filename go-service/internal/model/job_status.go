package model

// 公共任务异步状态常量（模块无关，audit 和 archive 共用）
const (
	JobStatusPending    = "pending"
	JobStatusAssembling = "assembling"
	JobStatusReasoning  = "reasoning"
	JobStatusExtracting = "extracting"
	JobStatusRechecking = "rechecking"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
	JobStatusCancelled  = "cancelled"
)

// ActiveReviewJobStatuses 返回审核与归档仍可推进或取消的状态，每次返回独立切片。
func ActiveReviewJobStatuses() []string {
	return []string{JobStatusPending, JobStatusAssembling, JobStatusReasoning, JobStatusExtracting, JobStatusRechecking}
}
