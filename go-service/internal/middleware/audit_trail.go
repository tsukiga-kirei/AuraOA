package middleware

import (
	"context"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/systemflags"
	"auraoa/go-service/internal/repository"
)

// AuditTrail 在请求结束后异步写入 operation_audit_logs（需 system.enable_audit_trail 为 true）。
// 仅记录已登录用户（存在 user_id）且路径以 /api/ 开头的请求；轮询类路径跳过。
func AuditTrail(flags *systemflags.Resolver, opRepo *repository.OperationAuditLogRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if flags == nil || opRepo == nil || !flags.AuditTrailEnabled() {
			return
		}
		uidVal, ok := c.Get("user_id")
		if !ok {
			return
		}
		uidStr, _ := uidVal.(string)
		if strings.TrimSpace(uidStr) == "" {
			return
		}
		userUUID, err := uuid.Parse(uidStr)
		if err != nil {
			return
		}
		path := c.Request.URL.Path
		if !strings.HasPrefix(path, "/api/") {
			return
		}
		if IsLowValuePollingPath(path) {
			return
		}
		latencyMs := int(time.Since(start).Milliseconds())
		status := c.Writer.Status()
		method := c.Request.Method

		var tenantUUID *uuid.UUID
		if tid, ok := c.Get("tenant_id"); ok {
			if s, ok2 := tid.(string); ok2 && s != "" {
				if tidParsed, err := uuid.Parse(s); err == nil {
					tenantUUID = &tidParsed
				}
			}
		}

		entry := &model.OperationAuditLog{
			ID:         uuid.New(),
			UserID:     userUUID,
			TenantID:   tenantUUID,
			Method:     method,
			Path:       path,
			StatusCode: status,
			LatencyMs:  latencyMs,
			ClientIP:   c.ClientIP(),
		}
		go func(logEntry *model.OperationAuditLog) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = opRepo.Create(ctx, logEntry)
		}(entry)
	}
}
