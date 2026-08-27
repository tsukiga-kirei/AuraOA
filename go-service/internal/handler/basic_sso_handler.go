package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"

	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/pkg/response"
	"auraoa/go-service/internal/service"
)

// BasicSSOHandler 处理 OA 服务端 Basic 换址和浏览器一次性交接。
type BasicSSOHandler struct {
	service       *service.BasicSSOService
	publicBaseURL string
}

// NewBasicSSOHandler 创建 Basic 单点登录处理器。
func NewBasicSSOHandler(service *service.BasicSSOService, publicBaseURL string) *BasicSSOHandler {
	return &BasicSSOHandler{service: service, publicBaseURL: normalizePublicBaseURL(publicBaseURL)}
}

// Redirect 为受信任 OA 服务端签发短期、仅可消费一次的浏览器登录地址。
// GET /api/auth/sso/basic-redirection?portal=business
func (h *BasicSSOHandler) Redirect(c *gin.Context) {
	code, err := h.service.PrepareHandoff(
		c.GetHeader("Authorization"),
		c.Query("portal"),
		c.ClientIP(),
		c.GetHeader("Origin"),
		c.GetHeader("Referer"),
	)
	if err != nil {
		writeBasicSSOError(c, err)
		return
	}

	baseURL := h.publicBaseURL
	if baseURL == "" {
		scheme := firstHeaderValue(c.GetHeader("X-Forwarded-Proto"))
		if scheme != "http" && scheme != "https" {
			if c.Request.TLS != nil {
				scheme = "https"
			} else {
				scheme = "http"
			}
		}
		baseURL = scheme + "://" + c.Request.Host
	}
	location := baseURL + "/api/auth/sso/basic-consume?code=" + url.QueryEscape(code)
	c.Header("Cache-Control", "no-store")
	c.Redirect(http.StatusFound, location)
}

// Consume 原子消费交接码，建立 AuraOA 登录态并进入工作台。
// GET /api/auth/sso/basic-consume?code=...
func (h *BasicSSOHandler) Consume(c *gin.Context) {
	login, err := h.service.ConsumeHandoff(c.Query("code"), c.ClientIP(), c.GetHeader("User-Agent"))
	if err != nil {
		writeBasicSSOError(c, err)
		return
	}
	payload, marshalErr := json.Marshal(login)
	if marshalErr != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
		return
	}
	encoded := base64.StdEncoding.EncodeToString(payload)
	c.Header("Cache-Control", "no-store")
	c.Header("Referrer-Policy", "no-referrer")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; script-src 'unsafe-inline'; connect-src 'self'")
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, renderBasicSSOBridge(encoded))
}

func writeBasicSSOError(c *gin.Context, err error) {
	if svcErr, ok := err.(*service.ServiceError); ok {
		response.Error(c, mapServiceErrorToHTTP(err), svcErr.Code, svcErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
}

func firstHeaderValue(value string) string {
	if comma := strings.IndexByte(value, ','); comma >= 0 {
		value = value[:comma]
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizePublicBaseURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host
}

func renderBasicSSOBridge(encodedLogin string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="zh-CN">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>AuraOA 单点登录</title></head>
<body><p>单点登录成功，正在进入 AuraOA…</p>
<script>
(async () => {
  const bytes = Uint8Array.from(atob(%q), c => c.charCodeAt(0));
  const data = JSON.parse(new TextDecoder().decode(bytes));
  const activeRole = data.active_role;
  localStorage.setItem('token', data.access_token);
  localStorage.setItem('refresh_token', data.refresh_token);
  let menus = [];
  try {
    const response = await fetch('/api/auth/menu', {
      headers: { Authorization: 'Bearer ' + data.access_token }
    });
    const result = await response.json();
    if (response.ok && result && result.code === 0 && result.data && Array.isArray(result.data.menus)) {
      menus = result.data.menus;
    }
  } catch (_) {
    // 菜单会在业务工作台再次加载；这里失败不能阻断单点登录。
  }
  localStorage.setItem('auth_state', JSON.stringify({
    user_role: activeRole.role,
    user_permissions: data.permissions && data.permissions.length ? data.permissions : [activeRole.role],
    all_roles: data.roles,
    active_role: activeRole,
    current_user: {
      username: data.user.username,
      display_name: data.user.display_name,
      tenant_id: activeRole.tenant_id || '',
      role_label: activeRole.label,
      email: data.user.email || '',
      phone: data.user.phone || ''
    },
    menus,
    locale: data.user.locale || 'zh-CN'
  }));
  window.location.replace('/dashboard');
})();
</script></body></html>`, encodedLogin)
}
