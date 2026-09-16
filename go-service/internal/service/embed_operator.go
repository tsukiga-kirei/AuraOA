package service

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"auraoa/go-service/internal/pkg/oa"
)

// embedOAOperatorID 读取本次 OA 嵌入操作的当前人员标识，不使用归属人员标识代替实际操作者。
func embedOAOperatorID(c *gin.Context, explicit string) string {
	if value := strings.TrimSpace(explicit); value != "" {
		return value
	}
	if value := strings.TrimSpace(c.GetHeader("X-Embed-OA-User-ID")); value != "" {
		return value
	}
	if value := strings.TrimSpace(c.Query("oa_user_id")); value != "" {
		return value
	}
	return strings.TrimSpace(c.Query("oa_current_user_id"))
}

// resolveEmbedOAOperator 将 OA 当前人员解析为姓名和部门快照；解析失败不阻断审核或总结。
func resolveEmbedOAOperator(ctx context.Context, adapter oa.OAAdapter, oaUserID string) (string, string, string) {
	oaUserID = strings.TrimSpace(oaUserID)
	if oaUserID == "" || adapter == nil {
		return oaUserID, "", ""
	}
	if resolver, ok := adapter.(oa.UserIdentityResolver); ok {
		identity, err := resolver.ResolveUserIdentityByOAUserID(ctx, oaUserID)
		if err == nil && identity != nil {
			return oaUserID, strings.TrimSpace(identity.DisplayName), strings.TrimSpace(identity.DepartmentName)
		}
	}
	displayName, err := adapter.ResolveUserDisplayNameByOAUserID(ctx, oaUserID)
	if err != nil {
		return oaUserID, "", ""
	}
	return oaUserID, strings.TrimSpace(displayName), ""
}
