package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestQueryRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	g := r.Group("/api/tenant/settings")

	// 统一 Query 路由
	g.GET("/archive-configs/full", func(c *gin.Context) {
		c.String(http.StatusOK, "query:"+c.Query("process_type"))
	})

	// 测试请求 Query 路由（包含斜杠与特殊字符）
	reqQuery := httptest.NewRequest(http.MethodGet, "/api/tenant/settings/archive-configs/full?process_type=%E5%85%AC%E5%85%B1%E8%B4%B9%E7%94%A8%2F%E8%B5%84%E4%BA%A7%E6%8A%A5%E9%94%80%E5%8D%95", nil)
	wQuery := httptest.NewRecorder()
	r.ServeHTTP(wQuery, reqQuery)
	if wQuery.Code != http.StatusOK {
		t.Fatalf("expected 200 for query route, got %d", wQuery.Code)
	}
	if wQuery.Body.String() != "query:公共费用/资产报销单" {
		t.Fatalf("expected 'query:公共费用/资产报销单', got '%s'", wQuery.Body.String())
	}
}
