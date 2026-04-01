package service

import (
	"context"
	"encoding/json"
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
	userRepo   *repository.UserRepo
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
	userRepo *repository.UserRepo,
	auditSvc *AuditExecuteService,
	archiveSvc *ArchiveReviewService,
) *CronTaskService {
	return &CronTaskService{
		taskRepo:   taskRepo,
		logRepo:    logRepo,
		presetRepo: presetRepo,
		configRepo: configRepo,
		userRepo:   userRepo,
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

// ListTasks 获取当前登录用户在当前租户下的任务实例。
func (s *CronTaskService) ListTasks(c *gin.Context) ([]dto.CronTaskResponse, error) {
	ownerID, err := getUserUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "用户ID无效")
	}
	tasks, err := s.taskRepo.ListByOwner(c, ownerID)
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

// CreateTask 为当前登录用户创建一个新任务实例（按用户隔离）。
func (s *CronTaskService) CreateTask(c *gin.Context, req *dto.CreateCronTaskRequest) (*dto.CronTaskResponse, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	ownerID, err := getUserUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "用户ID无效")
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
		OwnerUserID:    ownerID,
		TaskType:       req.TaskType,
		TaskLabel:      label,
		CronExpression: req.CronExpression,
		IsActive:       true,
		IsBuiltin:      false,
		PushEmail:      req.PushEmail,
		WorkflowIds:    req.WorkflowIds,
		DateRange:      req.DateRange,
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
	ownerID, err := getUserUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "用户ID无效")
	}
	task, err := s.taskRepo.GetByIDForOwner(c, id, ownerID)
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
	if req.WorkflowIds != nil {
		fields["workflow_ids"] = *req.WorkflowIds
		task.WorkflowIds = *req.WorkflowIds
	}
	if req.DateRange != nil {
		fields["date_range"] = *req.DateRange
		task.DateRange = *req.DateRange
	}

	if err := s.taskRepo.Update(c, id, ownerID, fields); err != nil {
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
	ownerID, err := getUserUUID(c)
	if err != nil {
		return newServiceError(errcode.ErrParamValidation, "用户ID无效")
	}
	task, err := s.taskRepo.GetByIDForOwner(c, id, ownerID)
	if err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}
	if task.IsBuiltin {
		return newServiceError(errcode.ErrParamValidation, "内置任务不可删除")
	}
	if s.scheduler != nil {
		s.scheduler.Remove(task.ID)
	}
	if err := s.taskRepo.Delete(c, id, ownerID); err != nil {
		return newServiceError(errcode.ErrDatabase, "删除任务失败")
	}
	return nil
}

// ToggleTask 切换任务启用/禁用状态。
func (s *CronTaskService) ToggleTask(c *gin.Context, id uuid.UUID) (*dto.CronTaskResponse, error) {
	ownerID, err := getUserUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "用户ID无效")
	}
	task, err := s.taskRepo.GetByIDForOwner(c, id, ownerID)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}
	newActive := !task.IsActive
	if err := s.taskRepo.Update(c, id, ownerID, map[string]interface{}{
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
	ownerID, err := getUserUUID(c)
	if err != nil {
		return newServiceError(errcode.ErrParamValidation, "用户ID无效")
	}
	task, err := s.taskRepo.GetByIDForOwner(c, id, ownerID)
	if err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}

	// 取手动触发人展示名（优先 display_name）
	createdBy := "unknown"
	if claims, ok := c.Get("jwt_claims"); ok {
		if jc, ok := claims.(*jwtpkg.JWTClaims); ok {
			if jc.DisplayName != "" {
				createdBy = jc.DisplayName
			} else if jc.Username != "" {
				createdBy = jc.Username
			}
		}
	}

	ouid := task.OwnerUserID
	logEntry := &model.CronLog{
		ID:              uuid.New(),
		TenantID:        task.TenantID,
		TaskID:          task.ID,
		TaskType:        task.TaskType,
		TaskLabel:       task.TaskLabel,
		TriggerType:     "manual",
		CreatedBy:       createdBy,
		TaskOwnerUserID: &ouid,
		Status:          "running",
		StartedAt:       time.Now(),
	}
	_ = s.logRepo.Create(logEntry)

	// 更新任务当前运行状态
	_ = s.taskRepo.UpdateFields(c, task.ID, ownerID, map[string]interface{}{
		"current_log_id": logEntry.ID,
	})

	go func() {
		ctx := context.Background()
		tcopy := *task
		tcopy.CurrentLogID = &logEntry.ID
		execErr := s.runTaskByType(ctx, &tcopy)
		
		status := "success"
		msg := fmt.Sprintf("%s 手动触发执行成功", time.Now().Format("2006-01-02 15:04:05"))
		if execErr != nil {
			if execErr.Error() == "job_aborted" {
				status = "failed"
				msg = "用户主动中止"
			} else {
				status = "failed"
				msg = execErr.Error()
			}
		}
		_ = s.logRepo.Finish(logEntry.ID, status, msg)
		_ = s.taskRepo.UpdateRunStats(tcopy.ID, time.Now(), nil, execErr == nil)
		// 执行完毕清除 CurrentLogID
		_ = s.taskRepo.UpdateFields(context.Background(), tcopy.ID, ownerID, map[string]interface{}{"current_log_id": nil})
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

	ouid := task.OwnerUserID
	logEntry := &model.CronLog{
		ID:              uuid.New(),
		TenantID:        task.TenantID,
		TaskID:          task.ID,
		TaskType:        task.TaskType,
		TaskLabel:       task.TaskLabel,
		TriggerType:     "scheduled",
		CreatedBy:       "system",
		TaskOwnerUserID: &ouid,
		Status:          "running",
		StartedAt:       time.Now(),
	}
	_ = s.logRepo.Create(logEntry)
	_ = s.taskRepo.DB().Model(&model.CronTask{}).Where("id = ?", task.ID).Update("current_log_id", logEntry.ID)

	task.CurrentLogID = &logEntry.ID
	execErr := s.runTaskByType(ctx, &task)

	status := "success"
	msg := fmt.Sprintf("%s 定时触发执行成功", time.Now().Format("2006-01-02 15:04:05"))
	if execErr != nil {
		if execErr.Error() == "job_aborted" {
			status = "failed"
			msg = "任务在执行周期中由于中止指令被终止"
		} else {
			status = "failed"
			msg = execErr.Error()
		}
	}
	_ = s.logRepo.Finish(logEntry.ID, status, msg)
	_ = s.taskRepo.UpdateRunStats(task.ID, time.Now(), nil, execErr == nil)
	_ = s.taskRepo.DB().Model(&model.CronTask{}).Where("id = ?", task.ID).Update("current_log_id", nil)
}

// AbortTask 发送中止信号。
func (s *CronTaskService) AbortTask(c *gin.Context, id uuid.UUID) error {
	ownerID, err := getUserUUID(c)
	if err != nil {
		return err
	}
	task, err := s.taskRepo.GetByIDForOwner(c, id, ownerID)
	if err != nil {
		return err
	}
	if task.CurrentLogID == nil {
		return nil
	}
	// 在 Redis 中设置中止标记，有效期 10 分钟（应对批量循环）
	key := fmt.Sprintf("cron:abort:%s", id.String())
	if s.auditSvc.BatchRdb() != nil {
		s.auditSvc.BatchRdb().Set(c.Request.Context(), key, "1", 10*time.Minute)
	}
	return nil
}

// ListLogs 获取指定任务的执行日志。
func (s *CronTaskService) ListLogs(c *gin.Context, taskID uuid.UUID) ([]model.CronLog, error) {
	ownerID, err := getUserUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "用户ID无效")
	}
	if _, err := s.taskRepo.GetByIDForOwner(c, taskID, ownerID); err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "任务不存在")
	}
	logs, err := s.logRepo.ListByTask(taskID, 50)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询日志失败")
	}
	return logs, nil
}

// ListAllLogs 数据管理页：分页查询当前租户所有任务日志。
func (s *CronTaskService) ListAllLogs(c *gin.Context, filter repository.CronLogFilter, page, pageSize int) ([]repository.CronLogListRow, int64, error) {
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
	oaUsername := ""
	if s.userRepo != nil {
		if owner, err := s.userRepo.FindByID(task.OwnerUserID); err == nil && owner != nil {
			oaUsername = owner.Username
		}
	}
	gc := buildWorkerContext(ctx, task.TenantID, task.OwnerUserID, oaUsername)

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
	limit := 10 // 默认单次执行上限 10
	if cfg, err := s.configRepo.GetByTaskType(c, task.TaskType); err == nil && cfg.BatchLimit != nil && *cfg.BatchLimit > 0 {
		limit = *cfg.BatchLimit
	}

	workflowIds := []string{}
	_ = json.Unmarshal(task.WorkflowIds, &workflowIds)

	items, err := s.auditSvc.ListPendingForBatch(c, workflowIds, task.DateRange, limit)
	if err != nil || len(items) == 0 {
		return nil // 无待处理项，正常
	}

	for _, item := range items {
		// 检查中止信号
		if s.checkAbort(c, task.ID) {
			return fmt.Errorf("job_aborted")
		}
		// 逐条执行以保障事务与状态更新准确
		_, _ = s.auditSvc.BatchExecute(c, []AuditExecuteRequest{item})
	}
	return nil
}

func (s *CronTaskService) runArchiveBatch(c *gin.Context, task *model.CronTask) error {
	if s.archiveSvc == nil {
		return fmt.Errorf("归档服务未初始化")
	}
	limit := 10
	if cfg, err := s.configRepo.GetByTaskType(c, task.TaskType); err == nil && cfg.BatchLimit != nil && *cfg.BatchLimit > 0 {
		limit = *cfg.BatchLimit
		if limit > 10 {
			limit = 10
		}
	}

	workflowIds := []string{}
	_ = json.Unmarshal(task.WorkflowIds, &workflowIds)

	items, err := s.archiveSvc.ListPendingForBatch(c, workflowIds, task.DateRange, limit)
	if err != nil || len(items) == 0 {
		return nil
	}

	for _, item := range items {
		if s.checkAbort(c, task.ID) {
			return fmt.Errorf("job_aborted")
		}
		_, _ = s.archiveSvc.BatchExecute(c, []dto.ArchiveReviewExecuteRequest{item})
	}
	return nil
}

func (s *CronTaskService) checkAbort(c *gin.Context, taskID uuid.UUID) bool {
	key := fmt.Sprintf("cron:abort:%s", taskID.String())
	if s.auditSvc.BatchRdb() != nil {
		val, _ := s.auditSvc.BatchRdb().Get(c.Request.Context(), key).Result()
		if val == "1" {
			s.auditSvc.BatchRdb().Del(c.Request.Context(), key)
			return true
		}
	}
	return false
}

// runReportTask 报告推送类任务占位（获取变量、发送邮件）。
func (s *CronTaskService) runReportTask(task *model.CronTask) error {
	// TODO: 使用 ReportCalculatorService 计算变量
	// TODO: 使用 MailService 发送邮件（读取 system_configs 里的 SMTP 配置）
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

// buildWorkerContext 构造调度器使用的伪 gin.Context（Sub 为归属用户，Username 为 OA 登录名）。
func buildWorkerContext(ctx context.Context, tenantID, ownerUserID uuid.UUID, oaUsername string) *gin.Context {
	rec := httptest.NewRecorder()
	gc, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	gc.Request = req
	gc.Set("tenant_id", tenantID.String())
	gc.Set("user_id", ownerUserID.String())
	if oaUsername == "" {
		oaUsername = "scheduler"
	}
	gc.Set("jwt_claims", &jwtpkg.JWTClaims{Sub: ownerUserID.String(), Username: oaUsername})
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
		OwnerUserID:    t.OwnerUserID.String(),
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
		WorkflowIds:    t.WorkflowIds,
		DateRange:      t.DateRange,
		CurrentLogID:   func() *string { if t.CurrentLogID != nil { s := t.CurrentLogID.String(); return &s }; return nil }(),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
	}
}
