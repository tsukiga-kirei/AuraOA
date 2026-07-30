package oa

import (
	"context"
	"database/sql"
	"fmt"

	"auraoa/go-service/internal/model"
	"gorm.io/gorm"
)

// supportedDrivers 记录每种 OA 类型支持的数据库驱动。
var supportedDrivers = map[string][]string{
	"weaver_e9": {"mysql", "oracle", "dm"},
}

// openOADatabase 根据 OA 类型创建底层数据库连接池。
// 新增 OA 适配器时必须从此入口接入 ConnectionManager 的统一生命周期。
func openOADatabase(
	ctx context.Context,
	oaType string,
	conn *model.OADatabaseConnection,
	transient bool,
) (*gorm.DB, *sql.DB, error) {
	if err := validateOAAdapterConfig(oaType, conn); err != nil {
		return nil, nil, err
	}
	switch oaType {
	case "weaver_e9":
		return openEcology9Database(ctx, conn, transient)
	default:
		return nil, nil, fmt.Errorf("不支持的 OA 类型: %s", oaType)
	}
}

// newOAAdapterWithDB 使用共享数据库连接池创建轻量 OA 适配器。
func newOAAdapterWithDB(
	oaType string,
	conn *model.OADatabaseConnection,
	db *gorm.DB,
	attachmentSvc ...AttachmentRecognitionService,
) (OAAdapter, error) {
	if err := validateOAAdapterConfig(oaType, conn); err != nil {
		return nil, err
	}
	if db == nil {
		return nil, fmt.Errorf("OA 数据库连接池为空")
	}

	switch oaType {
	case "weaver_e9":
		var svc AttachmentRecognitionService
		if len(attachmentSvc) > 0 {
			svc = attachmentSvc[0]
		}
		return newEcology9AdapterWithDB(conn, db, svc), nil
	default:
		return nil, fmt.Errorf("不支持的 OA 类型: %s", oaType)
	}
}

func validateOAAdapterConfig(oaType string, conn *model.OADatabaseConnection) error {
	if conn == nil {
		return fmt.Errorf("OA 数据库连接配置为空")
	}
	drivers, ok := supportedDrivers[oaType]
	if !ok {
		return fmt.Errorf("不支持的 OA 类型: %s", oaType)
	}
	if !contains(drivers, conn.Driver) {
		return fmt.Errorf("OA 类型 %s 不支持数据库驱动 %s（支持: %v）", oaType, conn.Driver, drivers)
	}
	return nil
}

// contains 判断字符串切片中是否包含指定元素。
func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
