package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/repository"
)

// CronScheduler 基于 robfig/cron/v3 的定时任务调度器。
// 从 DB 加载活跃任务并按 cron 表达式调度；支持运行时增删改任务。
type CronScheduler struct {
	c        *cron.Cron
	taskRepo *repository.CronTaskRepo
	taskSvc  *CronTaskService
	logger   *zap.Logger

	mu       sync.Mutex
	entryMap map[uuid.UUID]cron.EntryID // taskID → cron entryID
}

// NewCronScheduler 创建调度器实例（尚未启动）。
func NewCronScheduler(
	taskRepo *repository.CronTaskRepo,
	taskSvc *CronTaskService,
	logger *zap.Logger,
) *CronScheduler {
	return &CronScheduler{
		c: cron.New(
			cron.WithSeconds(), // 支持 6-field（含秒），标准5-field同样兼容
			cron.WithChain(cron.Recover(cron.DefaultLogger)),
		),
		taskRepo: taskRepo,
		taskSvc:  taskSvc,
		logger:   logger,
		entryMap: make(map[uuid.UUID]cron.EntryID),
	}
}

// Start 从 DB 加载所有活跃任务并启动调度器。
func (s *CronScheduler) Start(ctx context.Context) error {
	tasks, err := s.taskRepo.ListActiveByAllTenants()
	if err != nil {
		return fmt.Errorf("加载定时任务失败: %w", err)
	}

	for _, t := range tasks {
		s.addEntry(t)
	}

	s.c.Start()

	if s.logger != nil {
		s.logger.Info("cron scheduler started", zap.Int("tasks", len(tasks)))
	}

	// 监听 ctx，优雅停止
	go func() {
		<-ctx.Done()
		s.c.Stop()
		if s.logger != nil {
			s.logger.Info("cron scheduler stopped")
		}
	}()

	return nil
}

// AddOrUpdate 在调度器中注册或更新任务（创建/编辑任务时调用）。
func (s *CronScheduler) AddOrUpdate(task model.CronTask) {
	if !task.IsActive {
		s.Remove(task.ID)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	// 若已存在，先移除旧 entry
	if old, ok := s.entryMap[task.ID]; ok {
		s.c.Remove(old)
		delete(s.entryMap, task.ID)
	}

	s.addEntryLocked(task)
}

// Remove 从调度器移除任务（删除/禁用时调用）。
func (s *CronScheduler) Remove(taskID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, ok := s.entryMap[taskID]; ok {
		s.c.Remove(entryID)
		delete(s.entryMap, taskID)
	}
}

// addEntry 线程不安全版本（仅在 Start 初始化时调用）。
func (s *CronScheduler) addEntry(task model.CronTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.addEntryLocked(task)
}

// addEntryLocked 向 cron 引擎注册一个任务 entry（调用前需持有 mu 锁）。
func (s *CronScheduler) addEntryLocked(task model.CronTask) {
	taskID := task.ID
	svc := s.taskSvc
	logger := s.logger

	// 使用指针间接引用，解决闭包内使用 entryID 自引用的问题
	var entryID cron.EntryID
	var err error

	entryID, err = s.c.AddFunc(toFiveFieldCron(task.CronExpression), func() {
		if logger != nil {
			logger.Info("cron task triggered", zap.String("task_id", taskID.String()), zap.String("task_type", task.TaskType))
		}
		svc.TriggerScheduled(context.Background(), taskID)

		// 更新 next_run_at（entryID 此时已赋值）
		if entry := s.c.Entry(entryID); entry.Next != (time.Time{}) {
			next := entry.Next
			_ = svc.taskRepo.UpdateRunStats(taskID, time.Now(), &next, true)
		}
	})
	if err != nil {
		if logger != nil {
			logger.Warn("cron scheduler: invalid cron expression",
				zap.String("task_id", taskID.String()),
				zap.String("expr", task.CronExpression),
				zap.Error(err))
		}
		return
	}
	s.entryMap[taskID] = entryID

	// 初始化 next_run_at
	if entry := s.c.Entry(entryID); entry.Next != (time.Time{}) {
		next := entry.Next
		_ = s.taskRepo.UpdateRunStats(taskID, task.CreatedAt, &next, true)
	}
}

// RegisterCustomJob 向调度器注册一个自定义函数任务（不依赖数据库 CronTask 记录）。
// 适用于系统内置的定时维护任务，如日志清理、数据归档等。
// expr 为标准5字段 cron 表达式，fn 为任务执行函数。
// 若 cron 表达式非法，返回错误。
func (s *CronScheduler) RegisterCustomJob(expr string, fn func()) error {
	_, err := s.c.AddFunc(expr, fn)
	if err != nil {
		return fmt.Errorf("注册自定义定时任务失败，表达式 %q 非法: %w", expr, err)
	}
	return nil
}

// ParseNextRun 解析 cron 表达式并返回下次执行时间（供外部调用）。
func ParseNextRun(expr string) *time.Time {
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := p.Parse(expr)
	if err != nil {
		return nil
	}
	next := schedule.Next(time.Now())
	return &next
}

// toFiveFieldCron 确保使用标准5字段格式（robfig/cron 默认支持5字段）。
func toFiveFieldCron(expr string) string {
	return expr
}
