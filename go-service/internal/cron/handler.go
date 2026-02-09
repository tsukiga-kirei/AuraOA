package cron

import (
	"log"
	"net/http"
	"time"

	"oa-smart-audit/internal/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Handler exposes cron task management endpoints.
type Handler struct {
	// In-memory store for demo; replace with database
	tasks map[string]CronTask
}

func NewHandler() *Handler {
	return &Handler{tasks: make(map[string]CronTask)}
}

// RegisterRoutes registers cron routes.
func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/cron", h.ListTasks)
	api.POST("/cron", h.CreateTask)
	api.DELETE("/cron/:id", h.DeleteTask)
	api.POST("/cron/:id/execute", h.ExecuteTask)
}

func (h *Handler) ListTasks(c *gin.Context) {
	claims := auth.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var userTasks []CronTask
	for _, t := range h.tasks {
		if t.UserID == claims.UserID {
			userTasks = append(userTasks, t)
		}
	}
	if userTasks == nil {
		userTasks = []CronTask{}
	}
	c.JSON(http.StatusOK, gin.H{"tasks": userTasks})
}

func (h *Handler) CreateTask(c *gin.Context) {
	claims := auth.GetClaims(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	var input CronTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	now := time.Now()
	task := CronTask{
		ID:             uuid.New().String(),
		UserID:         claims.UserID,
		CronExpression: input.CronExpression,
		TaskType:       input.TaskType,
		IsActive:       true,
		CreatedAt:      now,
	}

	h.tasks[task.ID] = task
	// TODO: register with robfig/cron scheduler

	c.JSON(http.StatusCreated, task)
}

func (h *Handler) DeleteTask(c *gin.Context) {
	taskID := c.Param("id")
	if _, ok := h.tasks[taskID]; !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	delete(h.tasks, taskID)
	c.JSON(http.StatusOK, gin.H{"message": "task deleted"})
}

func (h *Handler) ExecuteTask(c *gin.Context) {
	taskID := c.Param("id")
	task, ok := h.tasks[taskID]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	// TODO: batch audit execution logic
	now := time.Now()
	task.LastRunAt = &now
	h.tasks[taskID] = task

	log.Printf("Cron task %s executed for user %s", taskID, task.UserID)

	c.JSON(http.StatusOK, TaskResult{
		TaskID:    taskID,
		Success:   true,
		Message:   "batch audit completed",
		ItemCount: 0,
	})
}
