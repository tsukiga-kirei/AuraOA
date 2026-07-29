package service

import (
	"context"
	"testing"

	"auraoa/go-service/internal/model"
)

func TestJobExecutionLimiterReservesInteractiveSlot(t *testing.T) {
	limiter := newJobExecutionLimiter(3)
	releaseFirst, ok := limiter.Acquire(context.Background(), model.JobQueueKindWorkbench)
	if !ok {
		t.Fatal("第一个非交互任务应获得执行名额")
	}
	defer releaseFirst()
	releaseSecond, ok := limiter.Acquire(context.Background(), model.JobQueueKindBackground)
	if !ok {
		t.Fatal("第二个非交互任务应获得执行名额")
	}
	defer releaseSecond()

	waitCtx, cancelWait := context.WithCancel(context.Background())
	cancelWait()
	if _, acquired := limiter.Acquire(waitCtx, model.JobQueueKindScheduled); acquired {
		t.Fatal("非交互任务不应占用为交互任务预留的最后一个名额")
	}

	releaseInteractive, acquired := limiter.Acquire(context.Background(), model.JobQueueKindInteractive)
	if !acquired {
		t.Fatal("交互任务应能使用预留名额")
	}
	releaseInteractive()
}

func TestJobExecutionLimiterSupportsSingleTotalSlot(t *testing.T) {
	limiter := newJobExecutionLimiter(1)
	release, acquired := limiter.Acquire(context.Background(), model.JobQueueKindBackground)
	if !acquired {
		t.Fatal("总并发为 1 时非交互任务仍应能够执行")
	}
	release()
}
