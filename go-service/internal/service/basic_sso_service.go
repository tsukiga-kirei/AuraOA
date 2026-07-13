package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/pkg/errcode"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/repository"
)

const (
	basicSSOHandoffPrefix = "auth:sso:basic:handoff:"
	basicSSOHandoffTTL    = 60 * time.Second
)

type basicSSOHandoff struct {
	TenantID string `json:"tenant_id"`
	Username string `json:"username"`
	Portal   string `json:"portal"`
}

// BasicSSOService 处理租户 Basic 共享凭据校验和一次性浏览器交接码。
type BasicSSOService struct {
	tenantRepo  *repository.TenantRepo
	userRepo    *repository.UserRepo
	authService *AuthService
	rdb         *redis.Client
}

// NewBasicSSOService 创建 Basic 单点登录服务。
func NewBasicSSOService(tenantRepo *repository.TenantRepo, userRepo *repository.UserRepo, authService *AuthService, rdb *redis.Client) *BasicSSOService {
	return &BasicSSOService{tenantRepo: tenantRepo, userRepo: userRepo, authService: authService, rdb: rdb}
}

// PrepareHandoff 校验 OA 服务端提交的 Basic 凭据、来源白名单和本地角色，并创建一次性交接码。
func (s *BasicSSOService) PrepareHandoff(authorization, portal, clientIP, origin, referer string) (string, error) {
	if portal == "" {
		portal = "business"
	}
	if portal != "business" && portal != "tenant_admin" {
		return "", newServiceError(errcode.ErrParamValidation, "单点登录入口仅支持 business 或 tenant_admin")
	}

	tenantCode, username, password, err := parseBasicSSOCredential(authorization)
	if err != nil {
		return "", err
	}
	tenant, findErr := s.tenantRepo.FindByCode(tenantCode)
	if findErr != nil || tenant.Status != "active" || !tenant.SSOBasicEnabled {
		return "", newServiceError(errcode.ErrSSOCredentialInvalid, "单点登录凭据无效")
	}

	plainPassword, decryptErr := crypto.Decrypt(tenant.SSOBasicPassword)
	if decryptErr != nil || !constantTimeStringEqual(plainPassword, password) {
		pkglogger.GetTenantLogger(tenant.Code).Warn("Basic 单点登录失败：共享密码不匹配")
		return "", newServiceError(errcode.ErrSSOCredentialInvalid, "单点登录凭据无效")
	}
	if !matchesAllowedIP(tenant.SSOBasicAllowedIPs, clientIP) || !matchesAllowedDomain(tenant.SSOBasicAllowedDomains, origin, referer) {
		pkglogger.GetTenantLogger(tenant.Code).Warn("Basic 单点登录失败：请求来源不在白名单", zap.String("clientIP", clientIP))
		return "", newServiceError(errcode.ErrSSOSourceDenied, "当前请求来源不允许使用单点登录")
	}

	user, userErr := s.userRepo.FindByUsername(username)
	if userErr != nil || user.Status != "active" {
		return "", newServiceError(errcode.ErrSSOUserUnavailable, "单点登录对应的本地用户不存在或已停用")
	}
	assignments, assignmentErr := s.userRepo.FindRoleAssignments(user.ID)
	if assignmentErr != nil {
		return "", newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	hasRole := false
	for _, assignment := range assignments {
		if assignment.TenantID != nil && *assignment.TenantID == tenant.ID && assignment.Role == portal {
			hasRole = true
			break
		}
	}
	if !hasRole {
		return "", newServiceError(errcode.ErrNoRoleInTenant, "用户在该租户没有所选入口角色")
	}

	code, randomErr := randomHandoffCode()
	if randomErr != nil {
		return "", newServiceError(errcode.ErrInternalServer, "创建单点登录地址失败")
	}
	payload, _ := json.Marshal(basicSSOHandoff{TenantID: tenant.ID.String(), Username: user.Username, Portal: portal})
	if redisErr := s.rdb.Set(context.Background(), basicSSOHandoffPrefix+code, payload, basicSSOHandoffTTL).Err(); redisErr != nil {
		return "", newServiceError(errcode.ErrRedisConn, "创建单点登录地址失败")
	}

	pkglogger.GetTenantLogger(tenant.Code).Info("Basic 单点登录交接码已签发", zap.String("username", user.Username), zap.String("portal", portal))
	return code, nil
}

// ConsumeHandoff 原子消费一次性交接码，并签发 AuraOA 自身的访问令牌和刷新令牌。
func (s *BasicSSOService) ConsumeHandoff(code, clientIP, userAgent string) (*dto.LoginResponse, error) {
	decodedCode, decodeErr := base64.RawURLEncoding.DecodeString(code)
	if decodeErr != nil || len(decodedCode) != 32 {
		return nil, newServiceError(errcode.ErrSSOHandoffInvalid, "单点登录地址无效或已过期，请从 OA 重新进入")
	}
	payload, err := s.rdb.GetDel(context.Background(), basicSSOHandoffPrefix+code).Bytes()
	if err != nil {
		return nil, newServiceError(errcode.ErrSSOHandoffInvalid, "单点登录地址无效或已过期，请从 OA 重新进入")
	}
	var handoff basicSSOHandoff
	if jsonErr := json.Unmarshal(payload, &handoff); jsonErr != nil {
		return nil, newServiceError(errcode.ErrSSOHandoffInvalid, "单点登录地址无效或已过期，请从 OA 重新进入")
	}
	if _, parseErr := uuid.Parse(handoff.TenantID); parseErr != nil {
		return nil, newServiceError(errcode.ErrSSOHandoffInvalid, "单点登录地址无效或已过期，请从 OA 重新进入")
	}
	tenantID, _ := uuid.Parse(handoff.TenantID)
	tenant, tenantErr := s.tenantRepo.FindByID(tenantID)
	if tenantErr != nil || tenant.Status != "active" || !tenant.SSOBasicEnabled {
		return nil, newServiceError(errcode.ErrSSOHandoffInvalid, "单点登录地址无效或已过期，请从 OA 重新进入")
	}
	return s.authService.LoginWithBasicSSO(handoff.Username, handoff.TenantID, handoff.Portal, clientIP, userAgent)
}

func parseBasicSSOCredential(authorization string) (string, string, string, error) {
	if !strings.HasPrefix(strings.ToLower(authorization), "basic ") {
		return "", "", "", newServiceError(errcode.ErrSSOCredentialInvalid, "缺少单点登录 Basic 凭据")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(authorization[6:]))
	if err != nil {
		return "", "", "", newServiceError(errcode.ErrSSOCredentialInvalid, "单点登录 Basic 凭据格式不正确")
	}
	credential := string(decoded)
	colon := strings.IndexByte(credential, ':')
	if colon <= 0 {
		return "", "", "", newServiceError(errcode.ErrSSOCredentialInvalid, "单点登录 Basic 凭据格式不正确")
	}
	principal, password := credential[:colon], credential[colon+1:]
	slash := strings.IndexByte(principal, '/')
	if slash <= 0 || slash == len(principal)-1 {
		return "", "", "", newServiceError(errcode.ErrSSOCredentialInvalid, "Basic 用户名格式应为 tenantCode/username")
	}
	return principal[:slash], principal[slash+1:], password, nil
}

func constantTimeStringEqual(expected, actual string) bool {
	if len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func matchesAllowedIP(csv, clientIP string) bool {
	if strings.TrimSpace(csv) == "" {
		return true
	}
	ip := net.ParseIP(strings.TrimSpace(clientIP))
	if ip == nil {
		return false
	}
	for _, item := range strings.Split(csv, ",") {
		candidate := strings.TrimSpace(item)
		if candidate == "" {
			continue
		}
		if allowedIP := net.ParseIP(candidate); allowedIP != nil && allowedIP.Equal(ip) {
			return true
		}
		if _, network, err := net.ParseCIDR(candidate); err == nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func matchesAllowedDomain(csv, origin, referer string) bool {
	if strings.TrimSpace(csv) == "" {
		return true
	}
	host := sourceHost(origin)
	if host == "" {
		host = sourceHost(referer)
	}
	if host == "" {
		return false
	}
	for _, item := range strings.Split(csv, ",") {
		allowed := strings.ToLower(strings.TrimSpace(item))
		if parsed := sourceHost(allowed); parsed != "" {
			allowed = parsed
		}
		if allowed != "" && (host == allowed || strings.HasSuffix(host, "."+allowed)) {
			return true
		}
	}
	return false
}

func sourceHost(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	return strings.ToLower(parsed.Hostname())
}

func randomHandoffCode() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}
