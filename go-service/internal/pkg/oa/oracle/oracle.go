// Package oracle 提供 Oracle 数据库的 GORM 驱动封装。
// 基于 github.com/godoes/gorm-oracle（纯 Go 实现，无需 Oracle Instant Client）。
package oracle

import (
	"fmt"

	goracle "github.com/godoes/gorm-oracle"
	"gorm.io/gorm"
)

// Open 返回 Oracle 的 GORM Dialector。
// IgnoreCase=true + NamingCaseSensitive=false 使驱动不给标识符加双引号，
// Oracle 会自动将不带引号的标识符转为大写匹配，兼容泛微 E9 的大写表名。
func Open(dsn string) gorm.Dialector {
	return goracle.New(goracle.Config{
		DSN:                     dsn,
		IgnoreCase:              true,
		NamingCaseSensitive:     false,
		VarcharSizeIsCharLength: true,
	})
}

// BuildDSN 构建 Oracle 连接字符串。
// 格式: oracle://user:pass@host:port/service_name
func BuildDSN(username, password, host string, port int, serviceName string) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		username, password, host, port, serviceName)
}
