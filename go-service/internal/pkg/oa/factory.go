package oa

import (
	"fmt"

	"oa-smart-audit/go-service/internal/model"
)

// NewOAAdapter 根据 oa_type 创建对应的 OA 适配器实例。
// 当前支持: "weaver_e9"（泛微 E9）
func NewOAAdapter(oaType string, conn *model.OADatabaseConnection) (OAAdapter, error) {
	switch oaType {
	case "weaver_e9":
		return NewEcology9Adapter(conn)
	default:
		return nil, fmt.Errorf("不支持的 OA 类型: %s", oaType)
	}
}
