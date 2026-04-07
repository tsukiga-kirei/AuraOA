package handler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/repository"
	"oa-smart-audit/go-service/internal/service"
)

// CronTaskHandler 处理定时任务实例相关的 HTTP 请求。
type CronTaskHandler struct {
	svc *service.CronTaskService
}

// NewCronTaskHandler 创建一个新的 CronTaskHandler 实例。
func NewCronTaskHandler(svc *service.CronTaskService) *CronTaskHandler {
	return &CronTaskHandler{svc: svc}
}

// ListTasks  GET /api/tenant/cron/tasks
func (h *CronTaskHandler) ListTasks(c *gin.Context) {
	tasks, err := h.svc.ListTasks(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tasks)
}

// CreateTask  POST /api/tenant/cron/tasks
func (h *CronTaskHandler) CreateTask(c *gin.Context) {
	var req dto.CreateCronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	task, err := h.svc.CreateTask(c, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, task)
}

// UpdateTask  PUT /api/tenant/cron/tasks/:id
func (h *CronTaskHandler) UpdateTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	var req dto.UpdateCronTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	task, err := h.svc.UpdateTask(c, id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, task)
}

// DeleteTask  DELETE /api/tenant/cron/tasks/:id
func (h *CronTaskHandler) DeleteTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	if err := h.svc.DeleteTask(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// ToggleTask  POST /api/tenant/cron/tasks/:id/toggle
func (h *CronTaskHandler) ToggleTask(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	task, err := h.svc.ToggleTask(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, task)
}

// ExecuteNow  POST /api/tenant/cron/tasks/:id/execute
func (h *CronTaskHandler) ExecuteNow(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	if err := h.svc.ExecuteNow(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{"status": "triggered"})
}

// ListLogs  GET /api/tenant/cron/tasks/:id/logs
func (h *CronTaskHandler) ListLogs(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "任务 ID 无效")
		return
	}
	logs, err := h.svc.ListLogs(c, id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, logs)
}

// ListAllLogs GET /api/tenant/cron/logs (tenant_admin 数据管理页)
func (h *CronTaskHandler) ListAllLogs(c *gin.Context) {
	filter, page, pageSize := parseCronLogQuery(c)
	items, total, err := h.svc.ListAllLogs(c, filter, page, pageSize)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetAllLogsStats GET /api/tenant/cron/logs/stats (tenant_admin)
func (h *CronTaskHandler) GetAllLogsStats(c *gin.Context) {
	stats, err := h.svc.GetCronLogStats(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// ExportAllLogs GET /api/tenant/cron/logs/export (tenant_admin) — CSV 下载
func (h *CronTaskHandler) ExportAllLogs(c *gin.Context) {
	filter, _, _ := parseCronLogQuery(c)
	items, _, err := h.svc.ListAllLogs(c, filter, 1, 5000)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	filename := fmt.Sprintf("cron_logs_%s.csv", time.Now().Format("20060102150405"))
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Writer.Write([]byte("\xef\xbb\xbf"))

	w := csv.NewWriter(c.Writer)
	_ = w.Write([]string{"记录ID", "任务名称", "任务类型", "触发类型", "触发人", "任务归属人", "状态", "备注", "开始时间", "结束时间"})
	for _, item := range items {
		finishedAt := ""
		if item.FinishedAt != nil {
			finishedAt = item.FinishedAt.Format("2006-01-02 15:04:05")
		}
		ownerName := item.TaskOwnerDisplayName
		if ownerName == "" {
			ownerName = "-"
		}
		_ = w.Write([]string{
			item.ID.String(),
			item.TaskLabel,
			item.TaskType,
			item.TriggerType,
			item.CreatedBy,
			ownerName,
			item.Status,
			item.Message,
			item.StartedAt.Format("2006-01-02 15:04:05"),
			finishedAt,
		})
	}
	w.Flush()
}

func parseCronLogQuery(c *gin.Context) (repository.CronLogFilter, int, int) {
	filter := repository.CronLogFilter{
		Keyword:     c.Query("keyword"),
		Status:      c.Query("status"),
		TaskType:    c.Query("task_type"),
		TriggerType: c.Query("trigger_type"),
		CreatedBy:   c.Query("created_by"),
		Department:  c.Query("department"),
	}
	if s := c.Query("start_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			filter.StartDate = &t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			end := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &end
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	return filter, page, pageSize
}
