package oa

import (
	"fmt"

	"oa-smart-audit/go-service/internal/model"
)

// supportedDrivers 记录每种 OA 类型支持的数据库驱动。
var supportedDrivers = map[string][]string{
	"weaver_e9": {"mysql", "oracle", "dm"},
}

// NewOAAdapter 根据 oa_type 和 conn.Driver 创建对应的 OA 适配器实例。
// 当前支持: "weaver_e9"（泛微 E9）— MySQL / Oracle
func NewOAAdapter(oaType string, conn *model.OADatabaseConnection) (OAAdapter, error) {
	drivers, ok := supportedDrivers[oaType]
	if !ok {
		return nil, fmt.Errorf("不支持的 OA 类型: %s", oaType)
	}

	if !contains(drivers, conn.Driver) {
		return nil, fmt.Errorf("OA 类型 %s 不支持数据库驱动 %s（支持: %v）", oaType, conn.Driver, drivers)
	}

	switch oaType {
	case "weaver_e9":
		return NewEcology9Adapter(conn)
	default:
		return nil, fmt.Errorf("不支持的 OA 类型: %s", oaType)
	}
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
