package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/pkg/errcode"
)

// TestAuditCommentValidation 验证评论不能为空、中文长度上限及可选满意度。
func TestAuditCommentValidation(t *testing.T) {
	bad, like := "invalid", "like"
	cases := []struct {
		name string
		req  dto.AuditCommentRequest
	}{
		{"纯空白", dto.AuditCommentRequest{Content: " \n\t"}},
		{"只有满意度没有评论", dto.AuditCommentRequest{Feedback: &like}},
		{"中文超长", dto.AuditCommentRequest{Content: strings.Repeat("评", 2001)}},
		{"无效满意度", dto.AuditCommentRequest{Content: "评论", Feedback: &bad}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			_, err := (&ExperienceService{}).Comment(c, uuid.New(), tt.req)
			se, ok := err.(*ServiceError)
			if !ok || se.Code != errcode.ErrParamValidation {
				t.Fatalf("应在访问数据库前拒绝非法评论，实际 %v", err)
			}
		})
	}
	// 合法正文不要求满意度，但没有租户上下文时必须拒绝，不能执行无租户查询。
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	_, err := (&ExperienceService{}).Comment(c, uuid.New(), dto.AuditCommentRequest{Content: strings.Repeat("评", 2000)})
	se, ok := err.(*ServiceError)
	if !ok || se.Code != errcode.ErrPermissionDenied {
		t.Fatalf("缺少租户时必须拒绝: %v", err)
	}
}
