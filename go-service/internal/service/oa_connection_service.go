package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/crypto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/oa"
	"oa-smart-audit/go-service/internal/repository"
)

// OAConnectionService 处理 OA 数据库连接的业务逻辑。
type OAConnectionService struct {
	repo *repository.OAConnectionRepo
}

func NewOAConnectionService(repo *repository.OAConnectionRepo) *OAConnectionService {
	return &OAConnectionService{repo: repo}
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

	if err := s.repo.Create(conn); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	resp := toOAConnectionResponse(conn)
	return &resp, nil
}

// Update 更新 OA 连接。
func (s *OAConnectionService) Update(id uuid.UUID, req *dto.UpdateOAConnectionRequest) (*dto.OAConnectionResponse, error) {
	_, err := s.repo.FindByID(id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "OA连接不存在")
	}

	fields := make(map[string]interface{})
	if req.Name != "" {
		fields["name"] = req.Name
	}
	if req.OAType != "" {
		fields["oa_type"] = req.OAType
	}
	if req.OATypeLabel != "" {
		fields["oa_type_label"] = req.OATypeLabel
	}
	if req.Driver != "" {
		fields["driver"] = req.Driver
	}
	if req.Host != "" {
		fields["host"] = req.Host
	}
	if req.Port != 0 {
		fields["port"] = req.Port
	}
	if req.DatabaseName != "" {
		fields["database_name"] = req.DatabaseName
	}
	if req.Username != "" {
		fields["username"] = req.Username
	}
	if req.Password != "" {
		encrypted, err := crypto.Encrypt(req.Password)
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "加密失败")
		}
		fields["password"] = encrypted
	}
	if req.PoolSize != 0 {
		fields["pool_size"] = req.PoolSize
	}
	if req.ConnectionTimeout != 0 {
		fields["connection_timeout"] = req.ConnectionTimeout
	}
	if req.TestOnBorrow != nil {
		fields["test_on_borrow"] = *req.TestOnBorrow
	}
	if req.SyncInterval != 0 {
		fields["sync_interval"] = req.SyncInterval
	}
	if req.Enabled != nil {
		fields["enabled"] = *req.Enabled
	}
	if req.Description != "" {
		fields["description"] = req.Description
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

	resp := toOAConnectionResponse(conn)
	return &resp, nil
}

// Delete 删除 OA 连接。
func (s *OAConnectionService) Delete(id uuid.UUID) error {
	_, err := s.repo.FindByID(id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "OA连接不存在")
	}
	if err := s.repo.Delete(id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

func toOAConnectionResponse(c *model.OADatabaseConnection) dto.OAConnectionResponse {
	lastSync := ""
	if c.LastSync != nil {
		lastSync = c.LastSync.Format("2006-01-02T15:04:05Z07:00")
	}
	return dto.OAConnectionResponse{
		ID:                c.ID.String(),
		Name:              c.Name,
		OAType:            c.OAType,
		OATypeLabel:       c.OATypeLabel,
		Driver:            c.Driver,
		Host:              c.Host,
		Port:              c.Port,
		DatabaseName:      c.DatabaseName,
		Username:          c.Username,
		PoolSize:          c.PoolSize,
		ConnectionTimeout: c.ConnectionTimeout,
		TestOnBorrow:      c.TestOnBorrow,
		Status:            c.Status,
		LastSync:          lastSync,
		SyncInterval:      c.SyncInterval,
		Enabled:           c.Enabled,
		Description:       c.Description,
		CreatedAt:         c.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// TestConnection 根据已保存的 OA 连接 ID 测试数据库连通性。
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

	return s.testOAConnection(conn)
}

// TestConnectionByParams 根据传入参数直接测试数据库连通性（用于新建/编辑时的测试按钮）。
func (s *OAConnectionService) TestConnectionByParams(req *dto.CreateOAConnectionRequest) error {
	conn := &model.OADatabaseConnection{
		OAType:       req.OAType,
		Driver:       req.Driver,
		Host:         req.Host,
		Port:         req.Port,
		DatabaseName: req.DatabaseName,
		Username:     req.Username,
		Password:     req.Password, // 前端传入的是明文
		PoolSize:     req.PoolSize,
	}
	if conn.PoolSize == 0 {
		conn.PoolSize = 5
	}

	return s.testOAConnection(conn)
}

// testOAConnection 实际执行 OA 数据库连接测试。
func (s *OAConnectionService) testOAConnection(conn *model.OADatabaseConnection) error {
	adapter, err := oa.NewOAAdapter(conn.OAType, conn)
	if err != nil {
		return newServiceError(errcode.ErrOATypeUnsupported, err.Error())
	}

	// 用 5 秒超时做一次简单查询验证连通性
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ValidateProcess 用一个不存在的流程名测试，只要不报连接错误就算通
	_, err = adapter.ValidateProcess(ctx, "__connection_test__")
	if err != nil {
		if strings.Contains(err.Error(), "不存在") {
			return nil
		}
		return newServiceError(errcode.ErrOAConnectionFailed, fmt.Sprintf("连接失败: %s", err.Error()))
	}
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
