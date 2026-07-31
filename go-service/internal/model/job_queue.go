package model

const (
	// JobQueueKindInteractive 用于用户可见页面和手动重新执行任务。
	JobQueueKindInteractive = "interactive"
	// JobQueueKindBackground 用于 OA 保存完成检查。
	JobQueueKindBackground = "background"
	// JobQueueKindScheduled 用于流程级定时扫描任务。
	JobQueueKindScheduled = "scheduled"
	// JobQueueKindWorkbench 用于系统内审核工作台任务，仅审核模块使用。
	JobQueueKindWorkbench = "workbench"
)

// NormalizeAuditJobQueueKind 将未知审核队列类型收敛为审核工作台队列。
func NormalizeAuditJobQueueKind(kind string) string {
	switch kind {
	case JobQueueKindInteractive, JobQueueKindBackground, JobQueueKindScheduled, JobQueueKindWorkbench:
		return kind
	default:
		return JobQueueKindWorkbench
	}
}

// NormalizeSummaryJobQueueKind 将未知总结队列类型收敛为普通后台队列。
// 总结模块没有审核工作台，不能接收 workbench 队列类型。
func NormalizeSummaryJobQueueKind(kind string) string {
	switch kind {
	case JobQueueKindInteractive, JobQueueKindScheduled:
		return kind
	default:
		return JobQueueKindBackground
	}
}
