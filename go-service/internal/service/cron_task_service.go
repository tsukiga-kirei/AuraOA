package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/repository"
)

// CronTaskService 处理定时任务实例的业务逻辑。
type CronTaskService struct {
	taskRepo   *repository.CronTaskRepo
	logRepo    *repository.CronLogRepo
	presetRepo *repository.CronTaskTypePresetRepo
	configRepo *repository.CronTaskTypeConfigRepo
	auditSvc   *AuditExecuteService
	archiveSvc *ArchiveReviewService
	scheduler  *CronScheduler // 延迟注入，避免循环依赖
}

// NewCronTaskService 创建一个新的 CronTaskService 实例。
func NewCronTaskService(
	taskRepo *repository.CronTaskRepo,
	logRepo *repository.CronLogRepo,
	presetRepo *repository.CronTaskTypePresetRepo,
	configRepo *repository.CronTaskTypeConfigRepo,
	auditSvc *AuditExecuteService,
	archiveSvc *ArchiveReviewService,
) *CronTaskService {
	return &CronTaskService{
		taskRepo:   taskRepo,
		logRepo:    logRepo,
		presetRepo: presetRepo,
		configRepo: configRepo,
		auditSvc:   auditSvc,
		archiveSvc: archiveSvc,
	}
}

// SetScheduler 延迟注入调度器（避免循环引用）。
func (s *CronTaskService) SetScheduler(sch *CronScheduler) {
	s.scheduler = sch
}

// ============================================================
// CRUD 操作
// ============================================================

// ListTasks 获取当前租户的所有任务实例。
func (s *CronTaskService) ListTasks(c *gin.Context) ([]dto.CronTaskResponse, error) {
	tasks, err := s.taskRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	presetMap := s.loadPresetMap()
	result := make([]dto.CronTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, taskToResponse(t, presetMap))
	}
	return result, nil
}

// CreateTask 为当前租户创建一个新任务实例。
func (s *CronTaskService) CreateTask(c *gin.Context, req *dto.CreateCronTaskRequest) (*dto.CronTaskResponse, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 校验任务类型是否在系统预设中存在
	preset, err := s.presetRepo.GetByTaskType(req.TaskType)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, fmt.Sprintf("任务类型 %s 不存在", req.TaskType))
	}

	// 校验租户是否已启用该任务类型
	_, err = s.configRepo.GetByTaskType(c, req.TaskType)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound,
			fmt.Sprintf("任务类型 %s 尚未启用，请先由管理员在「定时任务配置」中启用", req.TaskType))
	}

	label := req.TaskLabel
	if label == "" {
		label = preset.LabelZh
	}

	task := &model.CronTask{
		ID:             uuid.New(),
		TenantID:       tenantID,
		TaskType:       req.TaskType,
		TaskLabel:      label,
		CronExpression: req.CronExpression,
		IsActive:       true,
		IsBuiltin:      false,
		PushEmail:      req.PushEmail,
		NextRunAt:      ParseNextRun(req.CronExpression),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "创建任务失败")
	}

	if s.scheduler != nil {
		s.scheduler.AddOrUpdate(*task)
	}

	resp := taskToResponse(*task, s.loadPresetMap())
	return &resp, nil
}

// UpdateTask 更新任务的 cron 表达式、标签、推送邮箱。
func (s *CronTaskService) UpdateTask(c *gin.Context, id uuid.UUID, req *dto.UpdateCronTaskRequest) (*dto.CronTaskResponse, error) {
	task, err := s.taskRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}

	fields := map[string]interface{}{"updated_at": time.Now()}

	if req.TaskLabel != "" {
		fields["task_label"] = req.TaskLabel
		task.TaskLabel = req.TaskLabel
	}
	if req.CronExpression != "" {
		fields["cron_expression"] = req.CronExpression
		task.CronExpression = req.CronExpression
		if next := ParseNextRun(req.CronExpression); next != nil {
			fields["next_run_at"] = next
			task.NextRunAt = next
		}
	}
	// PushEmail 为指针：nil=不修改，其他均更新（包括清空""）
	if req.PushEmail != nil {
		fields["push_email"] = *req.PushEmail
		task.PushEmail = *req.PushEmail
	}

	if err := s.taskRepo.Update(c, id, fields); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "更新任务失败")
	}

	if s.scheduler != nil {
		s.scheduler.AddOrUpdate(*task)
	}

	resp := taskToResponse(*task, s.loadPresetMap())
	return &resp, nil
}

// DeleteTask 删除任务（内置任务不可删除）。
func (s *CronTaskService) DeleteTask(c *gin.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}
	if task.IsBuiltin {
		return newServiceError(errcode.ErrParamValidation, "内置任务不可删除")
	}
	if s.scheduler != nil {
		s.scheduler.Remove(task.ID)
	}
	if err := s.taskRepo.Delete(c, id); err != nil {
		return newServiceError(errcode.ErrDatabase, "删除任务失败")
	}
	return nil
}

// ToggleTask 切换任务启用/禁用状态。
func (s *CronTaskService) ToggleTask(c *gin.Context, id uuid.UUID) (*dto.CronTaskResponse, error) {
	task, err := s.taskRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}
	newActive := !task.IsActive
	if err := s.taskRepo.Update(c, id, map[string]interface{}{
		"is_active":  newActive,
		"updated_at": time.Now(),
	}); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "更新任务状态失败")
	}
	task.IsActive = newActive

	if s.scheduler != nil {
		if newActive {
			s.scheduler.AddOrUpdate(*task)
		} else {
			s.scheduler.Remove(task.ID)
		}
	}

	resp := taskToResponse(*task, s.loadPresetMap())
	return &resp, nil
}

// ExecuteNow 立即触发任务执行（手动触发，不影响调度时间）。
func (s *CronTaskService) ExecuteNow(c *gin.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}

	// 取手动触发人姓名
	createdBy := "unknown"
	if claims, ok := c.Get("jwt_claims"); ok {
		if jc, ok := claims.(*jwtpkg.JWTClaims); ok {
			if jc.Username != "" {
				createdBy = jc.Username
			}
		}
	}

	logEntry := &model.CronLog{
		ID:          uuid.New(),
		TenantID:    task.TenantID,
		TaskID:      task.ID,
		TaskType:    task.TaskType,
		TaskLabel:   task.TaskLabel,
		TriggerType: "manual",
		CreatedBy:   createdBy,
		Status:      "running",
		StartedAt:   time.Now(),
	}
	_ = s.logRepo.Create(logEntry)

	go func() {
		ctx := context.Background()
		execErr := s.runTaskByType(ctx, task)
		status := "success"
		msg := fmt.Sprintf("%s 手动触发执行成功", time.Now().Format("2006-01-02 15:04:05"))
		if execErr != nil {
			status = "failed"
			msg = execErr.Error()
		}
		_ = s.logRepo.Finish(logEntry.ID, status, msg)
		_ = s.taskRepo.UpdateRunStats(task.ID, time.Now(), nil, execErr == nil)
	}()
	return nil
}

// TriggerScheduled 由调度器调用——执行任务并更新统计。
func (s *CronTaskService) TriggerScheduled(ctx context.Context, taskID uuid.UUID) {
	var task model.CronTask
	if err := s.taskRepo.DB().WithContext(ctx).Where("id = ?", taskID).First(&task).Error; err != nil {
		return
	}
	if !task.IsActive {
		return
	}

	logEntry := &model.CronLog{
		ID:          uuid.New(),
		TenantID:    task.TenantID,
		TaskID:      task.ID,
		TaskType:    task.TaskType,
		TaskLabel:   task.TaskLabel,
		TriggerType: "scheduled",
		CreatedBy:   "system",
		Status:      "running",
		StartedAt:   time.Now(),
	}
	_ = s.logRepo.Create(logEntry)

	execErr := s.runTaskByType(ctx, &task)

	status := "success"
	msg := fmt.Sprintf("%s 定时触发执行成功", time.Now().Format("2006-01-02 15:04:05"))
	if execErr != nil {
		status = "failed"
		msg = execErr.Error()
	}
	_ = s.logRepo.Finish(logEntry.ID, status, msg)
	_ = s.taskRepo.UpdateRunStats(task.ID, time.Now(), nil, execErr == nil)
}

// ListLogs 获取指定任务的执行日志。
func (s *CronTaskService) ListLogs(c *gin.Context, taskID uuid.UUID) ([]model.CronLog, error) {
	if _, err := s.taskRepo.GetByID(c, taskID); err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}
	logs, err := s.logRepo.ListByTask(taskID, 50)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询日志失败")
	}
	return logs, nil
}

// ListAllLogs 数据管理页：分页查询当前租户所有任务日志。
func (s *CronTaskService) ListAllLogs(c *gin.Context, filter repository.CronLogFilter, page, pageSize int) ([]model.CronLog, int64, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, 0, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	items, total, err := s.logRepo.ListPagedByTenant(tenantID, filter, page, pageSize)
	if err != nil {
		return nil, 0, newServiceError(errcode.ErrDatabase, "查询日志失败")
	}
	return items, total, nil
}

// GetCronLogStats 数据管理页：获取当前租户任务日志统计。
func (s *CronTaskService) GetCronLogStats(c *gin.Context) (*repository.CronLogStats, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	stats, err := s.logRepo.CountStatsByTenant(tenantID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "统计查询失败")
	}
	return stats, nil
}

// ============================================================
// 任务执行分发
// ============================================================

func (s *CronTaskService) runTaskByType(ctx context.Context, task *model.CronTask) error {
	gc := buildWorkerContext(ctx, task.TenantID, uuid.Nil)

	switch task.TaskType {
	case "audit_batch":
		return s.runAuditBatch(gc, task)
	case "archive_batch":
		return s.runArchiveBatch(gc, task)
	case "audit_daily", "audit_weekly", "archive_daily", "archive_weekly":
		return s.runReportTask(task)
	default:
		return fmt.Errorf("未知任务类型: %s", task.TaskType)
	}
}

func (s *CronTaskService) runAuditBatch(c *gin.Context, task *model.CronTask) error {
	if s.auditSvc == nil {
		return fmt.Errorf("审核服务未初始化")
	}
	limit := 50
	if cfg, err := s.configRepo.GetByTaskType(c, task.TaskType); err == nil && cfg.BatchLimit != nil && *cfg.BatchLimit > 0 {
		limit = *cfg.BatchLimit
	}

	items, err := s.auditSvc.ListPendingForBatch(c, limit)
	if err != nil || len(items) == 0 {
		return nil // 无待处理项，正常
	}
	_, err = s.auditSvc.BatchExecute(c, items)
	return err
}

func (s *CronTaskService) runArchiveBatch(c *gin.Context, task *model.CronTask) error {
	if s.archiveSvc == nil {
		return fmt.Errorf("归档服务未初始化")
	}
	limit := 50
	if cfg, err := s.configRepo.GetByTaskType(c, task.TaskType); err == nil && cfg.BatchLimit != nil && *cfg.BatchLimit > 0 {
		limit = *cfg.BatchLimit
	}

	items, err := s.archiveSvc.ListPendingForBatch(c, limit)
	if err != nil || len(items) == 0 {
		return nil
	}
	_, err = s.archiveSvc.BatchExecute(c, items)
	return err
}

// runReportTask 报告推送类任务占位（接入邮件服务后实现）。
func (s *CronTaskService) runReportTask(task *model.CronTask) error {
	// TODO: 接入邮件服务后，查询统计数据、渲染模板、SMTP 推送至 task.PushEmail
	_ = task
	return nil
}

// ============================================================
// 辅助函数
// ============================================================

func (s *CronTaskService) loadPresetMap() map[string]model.CronTaskTypePreset {
	presets, _ := s.presetRepo.ListAll()
	m := make(map[string]model.CronTaskTypePreset, len(presets))
	for _, p := range presets {
		m[p.TaskType] = p
	}
	return m
}

// buildWorkerContext 构造调度器使用的伪 gin.Context。
func buildWorkerContext(ctx context.Context, tenantID, userID uuid.UUID) *gin.Context {
	rec := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	gc.Request = req
	gc.Set("tenant_id", tenantID.String())
	gc.Set("jwt_claims", &jwtpkg.JWTClaims{Sub: userID.String(), Username: "scheduler"})
	gc.Set("is_system_admin", false)
	return gc
}

// taskToResponse 将模型转换为响应 DTO。
func taskToResponse(t model.CronTask, presetMap map[string]model.CronTaskTypePreset) dto.CronTaskResponse {
	module := ""
	if p, ok := presetMap[t.TaskType]; ok {
		module = p.Module
	}
	return dto.CronTaskResponse{
		ID:             t.ID.String(),
		TenantID:       t.TenantID.String(),
		TaskType:       t.TaskType,
		TaskLabel:      t.TaskLabel,
		Module:         module,
		CronExpression: t.CronExpression,
		IsActive:       t.IsActive,
		IsBuiltin:      t.IsBuiltin,
		PushEmail:      t.PushEmail,
		LastRunAt:      t.LastRunAt,
		NextRunAt:      t.NextRunAt,
		SuccessCount:   t.SuccessCount,
		FailCount:      t.FailCount,
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}
