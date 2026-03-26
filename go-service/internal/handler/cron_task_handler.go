package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
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
