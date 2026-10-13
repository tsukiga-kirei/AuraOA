package handler

import (
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/service"
	"net/http"
	"testing"
)

// TestFeedbackMemoryHTTPStatus 验证反馈库校验失败不会误报服务器错误。
func TestFeedbackMemoryHTTPStatus(t *testing.T) {
	for code, want := range map[int]int{errcode.ErrFeedbackMemoryLimit: http.StatusBadRequest, errcode.ErrFeedbackMemoryEmpty: http.StatusBadRequest, errcode.ErrFeedbackMemoryStale: http.StatusConflict} {
		if got := mapServiceErrorToHTTP(&service.ServiceError{Code: code}); got != want {
			t.Fatalf("错误码 %d 映射为 %d，期望 %d", code, got, want)
		}
	}
}
