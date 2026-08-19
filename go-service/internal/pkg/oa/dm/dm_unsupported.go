//go:build !linux && !windows

// Package dm 提供达梦（DM）数据库的跨平台构建占位实现。
package dm

import (
	"fmt"
	"net/url"

	"gorm.io/gorm"
)

// OpenWithConn 在达梦上游驱动不支持的平台返回空 Dialector。
// 调用方会先通过 sql.Open("dm", ...) 得到“未知驱动”错误，不会进入 GORM 初始化。
func OpenWithConn(_ string, _ gorm.ConnPool) gorm.Dialector {
	return nil
}

// BuildDSN 构建达梦连接字符串，保持配置预览和单元测试跨平台一致。
func BuildDSN(username, password, host string, port int, dbName string, timeoutSeconds int) string {
	return fmt.Sprintf("dm://%s:%s@%s:%d?schema=%s&ignoreCase=true&socketTimeout=%d",
		url.QueryEscape(username), url.QueryEscape(password), host, port, dbName, timeoutSeconds)
}
