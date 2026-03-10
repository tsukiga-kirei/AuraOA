// Package dm 提供达梦（DM）数据库的 GORM 驱动封装。
// 基于 github.com/Rulessly/dm-driver-gorm 驱动。
// 达梦使用 Oracle 兼容模式，SQL 语法与 Oracle 保持一致。
package dm

import (
	"fmt"

	dmdriver "github.com/Rulessly/dm-driver-gorm"
	"gorm.io/gorm"
)

// Open 返回达梦数据库的 GORM Dialector。
func Open(dsn string) gorm.Dialector {
	return dmdriver.Open(dsn)
}

// BuildDSN 构建达梦数据库连接字符串。
// 格式: dm://user:pass@host:port?ignoreCase=false
func BuildDSN(username, password, host string, port int, dbName string) string {
	return fmt.Sprintf("dm://%s:%s@%s:%d?schema=%s&ignoreCase=false",
		username, password, host, port, dbName)
}
