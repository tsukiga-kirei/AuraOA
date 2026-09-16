package service

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"auraoa/go-service/internal/pkg/oa"
)

type embedOperatorAdapterStub struct {
	oa.OAAdapter
	identity     *oa.UserIdentity
	identityErr  error
	fallbackName string
	fallbackErr  error
}

func (s *embedOperatorAdapterStub) ResolveUserIdentityByOAUserID(context.Context, string) (*oa.UserIdentity, error) {
	return s.identity, s.identityErr
}

func (s *embedOperatorAdapterStub) ResolveUserDisplayNameByOAUserID(context.Context, string) (string, error) {
	return s.fallbackName, s.fallbackErr
}

func TestEmbedOAOperatorIDOnlyUsesCurrentUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "/?oa_current_user_id=28&oa_belong_user_id=99", nil)

	if got := embedOAOperatorID(c, ""); got != "28" {
		t.Fatalf("应只读取 OA 当前操作人，got=%q", got)
	}
	c.Request.Header.Set("X-Embed-OA-User-ID", " 29 ")
	if got := embedOAOperatorID(c, ""); got != "29" {
		t.Fatalf("请求头应优先于查询参数，got=%q", got)
	}
	if got := embedOAOperatorID(c, " 30 "); got != "30" {
		t.Fatalf("显式参数应具有最高优先级，got=%q", got)
	}
}

func TestResolveEmbedOAOperatorUsesIdentitySnapshot(t *testing.T) {
	adapter := &embedOperatorAdapterStub{identity: &oa.UserIdentity{
		UserID:         "28",
		DisplayName:    " 杨乾 ",
		DepartmentName: " 信息部 ",
	}}
	id, name, department := resolveEmbedOAOperator(context.Background(), adapter, " 28 ")
	if id != "28" || name != "杨乾" || department != "信息部" {
		t.Fatalf("人员快照解析错误: id=%q name=%q department=%q", id, name, department)
	}
}

func TestResolveEmbedOAOperatorFallsBackWithoutBlocking(t *testing.T) {
	adapter := &embedOperatorAdapterStub{
		identityErr:  errors.New("OA 部门查询失败"),
		fallbackName: "杨乾",
	}
	id, name, department := resolveEmbedOAOperator(context.Background(), adapter, "28")
	if id != "28" || name != "杨乾" || department != "" {
		t.Fatalf("姓名降级解析错误: id=%q name=%q department=%q", id, name, department)
	}

	adapter.fallbackErr = errors.New("OA 人员查询失败")
	id, name, department = resolveEmbedOAOperator(context.Background(), adapter, "28")
	if id != "28" || name != "" || department != "" {
		t.Fatalf("解析失败不应伪造人员信息: id=%q name=%q department=%q", id, name, department)
	}
}
