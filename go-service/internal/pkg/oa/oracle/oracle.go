// Package oracle 提供 Oracle 数据库的 GORM 驱动封装。
// 基于 github.com/dzwvip/oracle 驱动（GORM 官方推荐的 Oracle 适配）。
package oracle

import (
	"fmt"

	"github.com/dzwvip/oracle"
	"gorm.io/gorm"
)

// Open 返回 Oracle 的 GORM Dialector。
func Open(dsn string) gorm.Dialector {
	return oracle.Open(dsn)
}

// BuildDSN 构建 Oracle 连接字符串。
// 格式: oracle://user:pass@host:port/service_name
func BuildDSN(username, password, host string, port int, serviceName string) string {
	return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
		username, password, host, port, serviceName)
}
