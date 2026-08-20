package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"auraoa/go-service/internal/cache"
	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/crypto"
	"auraoa/go-service/internal/pkg/errcode"
	pkglogger "auraoa/go-service/internal/pkg/logger"
	"auraoa/go-service/internal/pkg/oa"
	"auraoa/go-service/internal/repository"
)

// OAConnectionService 处理 OA 数据库连接的业务逻辑。
type OAConnectionService struct {
	repo          *repository.OAConnectionRepo
	tenantRepo    *repository.TenantRepo
	invalidator   *cache.InvalidationManager
	oaConnections *oa.ConnectionManager
}

// NewOAConnectionService 创建 OAConnectionService，注入 OA 连接仓储。
func NewOAConnectionService(
	repo *repository.OAConnectionRepo,
	tenantRepo *repository.TenantRepo,
	invalidator *cache.InvalidationManager,
	oaConnections *oa.ConnectionManager,
) *OAConnectionService {
	return &OAConnectionService{
		repo:          repo,
		tenantRepo:    tenantRepo,
		invalidator:   invalidator,
		oaConnections: oaConnections,
	}
}

// List 返回所有 OA 连接。
func (s *OAConnectionService) List() ([]dto.OAConnectionResponse, error) {
	items, err := s.repo.List()
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.OAConnectionResponse, len(items))
	for i := range items {
		result[i] = toOAConnectionResponse(&items[i])
	}
	return result, nil
}

// Create 创建新的 OA 连接。
func (s *OAConnectionService) Create(req *dto.CreateOAConnectionRequest) (*dto.OAConnectionResponse, error) {
	conn := &model.OADatabaseConnection{
		ID:                uuid.New(),
		Name:              req.Name,
		OAType:            req.OAType,
		OATypeLabel:       req.OATypeLabel,
		Driver:            req.Driver,
		Host:              req.Host,
		Port:              req.Port,
		DatabaseName:      req.DatabaseName,
		Username:          req.Username,
		Password:          req.Password,
		PoolSize:          req.PoolSize,
		ConnectionTimeout: req.ConnectionTimeout,
		TestOnBorrow:      req.TestOnBorrow,
		SyncInterval:      req.SyncInterval,
		Enabled:           req.Enabled,
		Description:       req.Description,
		WeaverAPIURL:      req.WeaverAPIURL,
		WeaverAppID:       req.WeaverAppID,
		WeaverDefaultUser: req.WeaverDefaultUser,
	}

	// 加密密码
	if req.Password != "" {
		encrypted, err := crypto.Encrypt(req.Password)
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "加密失败")
		}
		conn.Password = encrypted
	}

	// 应用默认值
	if conn.Port == 0 {
		conn.Port = 3306
	}
	if conn.PoolSize == 0 {
		conn.PoolSize = 10
	}
	if conn.ConnectionTimeout == 0 {
		conn.ConnectionTimeout = 30
	}
	if conn.SyncInterval == 0 {
		conn.SyncInterval = 30
	}
	if conn.Status == "" {
		conn.Status = "disconnected"
	}

	if err := validateWeaverAttachmentKeys(conn.OAType, conn.WeaverAPIURL, conn.WeaverAppID, conn.WeaverDefaultUser); err != nil {
		return nil, err
	}

	if err := s.repo.Create(conn); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	pkglogger.Global().Info("OA连接创建成功", zap.String("connName", conn.Name), zap.String("oaType", conn.OAType))
	resp := toOAConnectionResponse(conn)
	return &resp, nil
}

// invalidateAffectedTenantCaches 查找引用此 OA 连接的所有租户并清除其缓存。
func (s *OAConnectionService) invalidateAffectedTenantCaches(connID uuid.UUID) {
	if s.invalidator == nil || s.tenantRepo == nil {
		return
	}
	tenants, err := s.tenantRepo.List()
	if err != nil {
		return
	}
	for _, t := range tenants {
		if t.OADBConnectionID != nil && *t.OADBConnectionID == connID {
			if err := s.invalidator.InvalidateTenantCache(context.Background(), t.ID); err != nil {
				pkglogger.Global().Warn("OA连接变更后清除租户缓存失败",
					zap.String("tenantID", t.ID.String()),
					zap.Error(err),
				)
			}
		}
	}
}

// Update 更新 OA 连接。
func (s *OAConnectionService) Update(id uuid.UUID, req *dto.UpdateOAConnectionRequest) (*dto.OAConnectionResponse, error) {
	existing, err := s.repo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "OA连接不存在")
	}

	fields := make(map[string]interface{})
	poolConfigChanged := false
	if req.Name != "" {
		fields["name"] = req.Name
	}
	if req.OAType != "" {
		fields["oa_type"] = req.OAType
		poolConfigChanged = poolConfigChanged || req.OAType != existing.OAType
	}
	if req.OATypeLabel != "" {
		fields["oa_type_label"] = req.OATypeLabel
	}
	if req.Driver != "" {
		fields["driver"] = req.Driver
		poolConfigChanged = poolConfigChanged || req.Driver != existing.Driver
	}
	if req.Host != "" {
		fields["host"] = req.Host
		poolConfigChanged = poolConfigChanged || req.Host != existing.Host
	}
	if req.Port != 0 {
		fields["port"] = req.Port
		poolConfigChanged = poolConfigChanged || req.Port != existing.Port
	}
	if req.DatabaseName != "" {
		fields["database_name"] = req.DatabaseName
		poolConfigChanged = poolConfigChanged || req.DatabaseName != existing.DatabaseName
	}
	if req.Username != "" {
		fields["username"] = req.Username
		poolConfigChanged = poolConfigChanged || req.Username != existing.Username
	}
	if req.Password != "" {
		encrypted, err := crypto.Encrypt(req.Password)
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "加密失败")
		}
		fields["password"] = encrypted
		poolConfigChanged = true
	}
	if req.PoolSize != 0 {
		fields["pool_size"] = req.PoolSize
		poolConfigChanged = poolConfigChanged || req.PoolSize != existing.PoolSize
	}
	if req.ConnectionTimeout != 0 {
		fields["connection_timeout"] = req.ConnectionTimeout
		poolConfigChanged = poolConfigChanged || req.ConnectionTimeout != existing.ConnectionTimeout
	}
	if req.TestOnBorrow != nil {
		fields["test_on_borrow"] = *req.TestOnBorrow
	}
	if req.SyncInterval != 0 {
		fields["sync_interval"] = req.SyncInterval
	}
	if req.Enabled != nil {
		fields["enabled"] = *req.Enabled
		poolConfigChanged = poolConfigChanged || (!*req.Enabled && existing.Enabled)
	}
	if req.Description != "" {
		fields["description"] = req.Description
	}
	// 泛微 E9 密钥（按字段独立更新；空字符串视为不修改）
	if req.WeaverAPIURL != "" {
		fields["weaver_api_url"] = req.WeaverAPIURL
	}
	if req.WeaverAppID != "" {
		fields["weaver_appid"] = req.WeaverAppID
	}
	if req.WeaverDefaultUser != "" {
		fields["weaver_default_user"] = req.WeaverDefaultUser
	}

	oaType := existing.OAType
	if req.OAType != "" {
		oaType = req.OAType
	}
	apiURL := firstNonEmpty(req.WeaverAPIURL, existing.WeaverAPIURL)
	appID := firstNonEmpty(req.WeaverAppID, existing.WeaverAppID)
	loginID := firstNonEmpty(req.WeaverDefaultUser, existing.WeaverDefaultUser)
	if err := validateWeaverAttachmentKeys(oaType, apiURL, appID, loginID); err != nil {
		return nil, err
	}

	if len(fields) > 0 {
		if err := s.repo.Update(id, fields); err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	conn, err := s.repo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	pkglogger.Global().Info("OA连接更新成功", zap.String("connID", id.String()))

	if poolConfigChanged {
		// 建连参数变更或连接被停用后立即关闭旧共享池；
		// 其他实例也会在下次取用时通过配置指纹自动替换。
		s.oaConnections.Invalidate(id)
	}

	// 清除引用此 OA 连接的所有租户缓存
	s.invalidateAffectedTenantCaches(id)

	resp := toOAConnectionResponse(conn)
	return &resp, nil
}

// Delete 删除 OA 连接。
func (s *OAConnectionService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "OA连接不存在")
	}

	// 先清除引用此 OA 连接的所有租户缓存（删除前执行，因为删除后无法查找引用）
	s.invalidateAffectedTenantCaches(id)

	if err := s.repo.Delete(id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	s.oaConnections.Invalidate(id)
	pkglogger.Global().Info("OA连接删除成功", zap.String("connID", id.String()))
	return nil
}

func toOAConnectionResponse(c *model.OADatabaseConnection) dto.OAConnectionResponse {
	return dto.OAConnectionResponse{
		ID:                    c.ID.String(),
		Name:                  c.Name,
		OAType:                c.OAType,
		OATypeLabel:           c.OATypeLabel,
		Driver:                c.Driver,
		Host:                  c.Host,
		Port:                  c.Port,
		DatabaseName:          c.DatabaseName,
		Username:              c.Username,
		PoolSize:              c.PoolSize,
		ConnectionTimeout:     c.ConnectionTimeout,
		TestOnBorrow:          c.TestOnBorrow,
		Status:                c.Status,
		SyncInterval:          c.SyncInterval,
		Enabled:               c.Enabled,
		Description:           c.Description,
		WeaverAPIURL:          c.WeaverAPIURL,
		WeaverAppIDConfigured: strings.TrimSpace(c.WeaverAppID) != "",
		WeaverDefaultUser:     c.WeaverDefaultUser,
		CreatedAt:             c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:             c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// TestConnection 根据已保存的 OA 连接 ID 测试数据库连通性，并将结果持久化到数据库。
func (s *OAConnectionService) TestConnection(id uuid.UUID) error {
	conn, err := s.repo.FindByID(id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "OA连接不存在")
	}

	// 解密密码
	password, err := crypto.Decrypt(conn.Password)
	if err != nil {
		return newServiceError(errcode.ErrInternalServer, "密码解密失败")
	}
	conn.Password = password

	testErr := s.testOAConnection(conn)

	// 持久化连接状态
	newStatus := "connected"
	if testErr != nil {
		newStatus = "disconnected"
	}
	_ = s.repo.Update(id, map[string]interface{}{"status": newStatus})

	if testErr == nil {
		pkglogger.Global().Info("OA连接测试成功", zap.String("connID", id.String()), zap.String("status", newStatus))
	}
	return testErr
}

// TestConnectionByParams 根据传入参数直接测试数据库连通性（用于新建/编辑时的测试按钮）。
func (s *OAConnectionService) TestConnectionByParams(req *dto.CreateOAConnectionRequest) error {
	conn := &model.OADatabaseConnection{
		OAType:            req.OAType,
		Driver:            req.Driver,
		Host:              req.Host,
		Port:              req.Port,
		DatabaseName:      req.DatabaseName,
		Username:          req.Username,
		Password:          req.Password, // 前端传入的是明文
		PoolSize:          req.PoolSize,
		ConnectionTimeout: req.ConnectionTimeout,
	}
	if conn.PoolSize == 0 {
		conn.PoolSize = 10
	}

	return s.testOAConnection(conn)
}

// testOAConnection 实际执行 OA 数据库连接测试。
func (s *OAConnectionService) testOAConnection(conn *model.OADatabaseConnection) error {
	// 测试连接使用独立短生命周期连接，不进入业务共享池。
	timeout := time.Duration(conn.ConnectionTimeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	if timeout < 5*time.Second {
		timeout = 5 * time.Second
	}
	if timeout > 5*time.Minute {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	adapter, closeFn, err := s.oaConnections.OpenTransientAdapter(ctx, conn.OAType, conn)
	if err != nil {
		return newServiceError(errcode.ErrOATypeUnsupported, err.Error())
	}
	defer func() {
		if closeErr := closeFn(); closeErr != nil {
			pkglogger.Global().Warn("关闭 OA 测试连接失败", zap.Error(closeErr))
		}
	}()

	// 用配置的连接超时做一次简单查询验证连通性。
	// ValidateProcess 用一个不存在的流程名测试，只要不报连接错误就算通
	_, err = adapter.ValidateProcess(ctx, "__connection_test__")
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			return nil
		}
		svcErr := newServiceError(errcode.ErrOAConnectionFailed, fmt.Sprintf("连接失败: %s", err.Error()))
		pkglogger.Global().Warn("OA连接测试失败", zap.Error(svcErr))
		return svcErr
	}
	return nil
}

// validateWeaverAttachmentKeys：填写附件接口 URL 后，appid 与 loginid 必须齐全。
func validateWeaverAttachmentKeys(oaType, apiURL, appID, loginID string) error {
	if oaType != "weaver_e9" || strings.TrimSpace(apiURL) == "" {
		return nil
	}
	if strings.TrimSpace(appID) == "" {
		return newServiceError(errcode.ErrParamValidation, "配置附件接口 URL 时必须填写泛微 appid")
	}
	if strings.TrimSpace(loginID) == "" {
		return newServiceError(errcode.ErrParamValidation, "配置附件接口 URL 时必须填写默认调用用户 loginid")
	}
	return nil
}
