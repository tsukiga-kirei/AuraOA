package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"auraoa/go-service/internal/service"
)

// TestAuditExperienceSettingsBinding 验证开关必填且必须为布尔值，true/false 均能通过参数校验。
func TestAuditExperienceSettingsBinding(t *testing.T) {
	h := NewExperienceHandler(service.NewExperienceService(nil, nil))
	for _, tt := range []struct {
		body string
		want int
	}{
		{`{}`, http.StatusBadRequest},
		{`{"audit_comments_own_only":null}`, http.StatusBadRequest},
		{`{"audit_comments_own_only":"false"}`, http.StatusBadRequest},
		{`{"audit_comments_own_only":1}`, http.StatusBadRequest},
		// 合法布尔值应进入服务层并因缺少租户而被拒绝，而非参数错误。
		{`{"audit_comments_own_only":false}`, http.StatusForbidden},
		{`{"audit_comments_own_only":true}`, http.StatusForbidden},
	} {
		t.Run(tt.body, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodPut, "/api/tenant/experience/audit-settings", strings.NewReader(tt.body))
			c.Request.Header.Set("Content-Type", "application/json")
			h.UpdateAuditSettings(c)
			if w.Code != tt.want {
				t.Fatalf("期望状态 %d，实际 %d: %s", tt.want, w.Code, w.Body.String())
			}
		})
	}
}
