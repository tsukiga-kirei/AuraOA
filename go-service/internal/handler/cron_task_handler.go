// 定时任务实例处理器，负责任务实例的增删改查、触发执行及日志管理。
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

// NewCronTaskHandler 创建定时任务实例处理器实例。
func NewCronTaskHandler(svc *service.CronTaskService) *CronTaskHandler {
	return &CronTaskHandler{svc: svc}
}

// ListTasks 获取当前用户在当前租户下的所有定时任务实例列表。
// GET /api/tenant/cron/tasks
// 返回：任务实例数组，包含 cron 表达式、启用状态、最近执行时间等。
func (h *CronTaskHandler) ListTasks(c *gin.Context) {
	tasks, err := h.svc.ListTasks(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tasks)
}

// CreateTask 创建新的定时任务实例。
// POST /api/tenant/cron/tasks
// 请求体：CreateCronTaskRequest（任务类型、cron 表达式、流程范围等）
// 返回：新建的任务实例对象。
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

// UpdateTask 更新指定定时任务实例的配置。
// PUT /api/tenant/cron/tasks/:id
// 路径参数：id（任务 UUID）
// 请求体：UpdateCronTaskRequest
// 返回：更新后的任务实例对象。
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

// DeleteTask 删除指定定时任务实例。
// DELETE /api/tenant/cron/tasks/:id
// 路径参数：id（任务 UUID）
// 返回：null（成功时）。
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

// ToggleTask 切换指定定时任务的启用/禁用状态。
// POST /api/tenant/cron/tasks/:id/toggle
// 路径参数：id（任务 UUID）
// 返回：更新后的任务实例对象（含最新 is_active 状态）。
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

// ExecuteNow 立即触发指定定时任务执行一次（手动触发）。
// POST /api/tenant/cron/tasks/:id/execute
// 路径参数：id（任务 UUID）
// 返回：{"status": "triggered"}。
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

// ListLogs 获取指定定时任务的执行日志列表。
// GET /api/tenant/cron/tasks/:id/logs
// 路径参数：id（任务 UUID）
// 返回：该任务的执行日志数组，按时间倒序排列。
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

// ListAllLogs 分页查询当前租户所有定时任务的执行日志（租户管理员数据管理页）。
// GET /api/tenant/cron/logs
// 查询参数：keyword、status、task_type、trigger_type、created_by、department、start_date、end_date、page、page_size
// 返回：分页日志列表（items + total + page + page_size）。
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

// GetAllLogsStats 获取当前租户定时任务执行日志的统计汇总（租户管理员数据管理页）。
// GET /api/tenant/cron/logs/stats
// 返回：按状态分类的执行次数统计。
func (h *CronTaskHandler) GetAllLogsStats(c *gin.Context) {
	stats, err := h.svc.GetCronLogStats(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// ExportAllLogs 导出定时任务执行日志为 CSV 文件（最多 5000 条）。
// GET /api/tenant/cron/logs/export
// 查询参数：同 ListAllLogs（不分页）
// 返回：text/csv 格式文件下载，含 UTF-8 BOM 以兼容 Excel。
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
	// 写入 UTF-8 BOM，确保 Excel 正确识别中文
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

// parseCronLogQuery 解析定时任务日志列表的过滤参数及分页参数。
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
			// 结束日期扩展到当天末尾（23:59:59）
			end := t.Add(24*time.Hour - time.Second)
			filter.EndDate = &end
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 20)
	return filter, page, pageSize
}
