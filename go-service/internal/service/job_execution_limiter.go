package service

import (
	"context"

	"auraoa/go-service/internal/model"
)

// jobExecutionLimiter 同时限制模块总并发，并为交互任务预留一个执行名额。
// 当总并发大于 1 时，非交互任务最多占用 total-1 个名额，避免后台任务挤满全部容量。
type jobExecutionLimiter struct {
	total          chan struct{}
	nonInteractive chan struct{}
}

func newJobExecutionLimiter(total int) *jobExecutionLimiter {
	if total < 1 {
		total = 1
	}
	nonInteractive := total - 1
	if nonInteractive < 1 {
		nonInteractive = 1
	}
	return &jobExecutionLimiter{
		total:          make(chan struct{}, total),
		nonInteractive: make(chan struct{}, nonInteractive),
	}
}

// Acquire 等待执行名额；返回的 release 必须在任务结束时调用。
func (l *jobExecutionLimiter) Acquire(
	ctx context.Context,
	queueKind string,
) (release func(), acquired bool) {
	if l == nil {
		return func() {}, true
	}
	interactive := queueKind == model.JobQueueKindInteractive
	if !interactive {
		select {
		case l.nonInteractive <- struct{}{}:
		case <-ctx.Done():
			return nil, false
		}
	}
	select {
	case l.total <- struct{}{}:
	case <-ctx.Done():
		if !interactive {
			<-l.nonInteractive
		}
		return nil, false
	}
	return func() {
		<-l.total
		if !interactive {
			<-l.nonInteractive
		}
	}, true
}
