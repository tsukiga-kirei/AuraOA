package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	jwtpkg "auraoa/go-service/internal/pkg/jwt"
)

type pagePermissionCheckerStub struct {
	allowed bool
	err     error
	page    string
}

func (s *pagePermissionCheckerStub) UserHasPagePermission(_ *gin.Context, _ uuid.UUID, page string) (bool, error) {
	s.page = page
	return s.allowed, s.err
}

func TestRequirePagePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	tests := []struct {
		name       string
		checker    *pagePermissionCheckerStub
		wantStatus int
		wantCalled bool
	}{
		{name: "允许已分配页面", checker: &pagePermissionCheckerStub{allowed: true}, wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "拒绝未分配页面", checker: &pagePermissionCheckerStub{}, wantStatus: http.StatusForbidden, wantCalled: true},
		{name: "权限查询失败", checker: &pagePermissionCheckerStub{err: errors.New("db error")}, wantStatus: http.StatusInternalServerError, wantCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("jwt_claims", &jwtpkg.JWTClaims{Sub: userID.String()})
				c.Set("tenant_id", uuid.New().String())
				c.Next()
			})
			router.Use(RequirePagePermission(tt.checker, "/summary"))
			router.GET("/api/summary/processes", func(c *gin.Context) { c.Status(http.StatusNoContent) })

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/api/summary/processes", nil)
			router.ServeHTTP(w, req)
			if w.Code != tt.wantStatus {
				t.Fatalf("状态码 = %d, want %d; body=%s", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantCalled && tt.checker.page != "/summary" {
				t.Fatalf("页面权限检查参数 = %q, want /summary", tt.checker.page)
			}
		})
	}
}
