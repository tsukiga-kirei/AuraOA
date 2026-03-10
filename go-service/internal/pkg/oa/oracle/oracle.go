// Package oracle 提供 Oracle 数据库的 GORM 驱动封装。
// 基于 github.com/godoes/gorm-oracle（纯 Go 实现，无需 Oracle Instant Client）。
package oracle

import (
	"fmt"

	goracle "github.com/godoes/gorm-oracle"
	"gorm.io/gorm"
)

// Open 返回 Oracle 的 GORM Dialector。
func Open(dsn string) gorm.Dialector {
	return goracle.New(goracle.Config{
		DSN:                     dsn,
		IgnoreCase:              false,
		NamingCaseSensitive:     true,
		VarcharSizeIsCharLength: true,
	})
}

// BuildDSN 构建 Oracle 连接字符串。
// 格式: oracle://user:pass@host:port/service_name
func BuildDSN(username, password, host string, port int, serviceName string) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		username, password, host, port, serviceName)
}
