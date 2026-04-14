package logger

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// CleanupGlobalLogs 清理全局日志目录下超过 retentionDays 天的轮转备份文件。
// 仅删除带时间戳的备份文件（如 app-2025-01-15T02-00-00.000.log），
// 不删除当前正在写入的 app.log。
// 单个文件删除失败时记录 WARN 并继续，不中断整体清理任务。
// 返回删除的文件数量、释放的字节数以及遍历目录时遇到的错误。
func CleanupGlobalLogs(retentionDays int) (deletedCount int, freedBytes int64, err error) {
	// 全局日志所在目录即 globalCfg.Dir
	dir := globalCfg.Dir

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0, err
	}

	// 计算过期截止时间：当前时间减去保留天数
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	for _, entry := range entries {
		// 跳过子目录，只处理普通文件
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// 跳过当前写入的 app.log，只清理轮转备份文件
		if name == "app.log" {
			continue
		}

		// 获取文件详细信息以读取修改时间
		fullPath := filepath.Join(dir, name)
		info, statErr := os.Stat(fullPath)
		if statErr != nil {
			Global().Warn("获取文件信息失败，跳过该文件",
				zap.String("file", fullPath),
				zap.Error(statErr),
			)
			continue
		}

		// 判断文件修改时间是否早于截止时间
		if info.ModTime().Before(cutoff) {
			size := info.Size()
			if removeErr := os.Remove(fullPath); removeErr != nil {
				Global().Warn("删除过期全局日志备份文件失败",
					zap.String("file", fullPath),
					zap.Error(removeErr),
				)
				continue
			}
			deletedCount++
			freedBytes += size
		}
	}

	return deletedCount, freedBytes, nil
}

// CleanupTenantLogs 清理各租户日志目录下超过指定天数的轮转备份文件。
// retentionMap 为 tenantCode → retentionDays 的映射，
// 每个租户可配置不同的保留天数。
// 仅删除带时间戳的备份文件（如 tenant-2025-01-15T02-00-00.000.log），
// 不删除当前正在写入的 tenant.log。
// 单个文件删除失败时记录 WARN 并继续，不中断整体清理任务。
// 返回删除的文件总数、释放的总字节数以及遍历目录时遇到的错误。
func CleanupTenantLogs(retentionMap map[string]int) (deletedCount int, freedBytes int64, err error) {
	// 租户日志根目录：{globalCfg.Dir}/tenants/
	tenantsRoot := filepath.Join(globalCfg.Dir, "tenants")

	for tenantCode, retentionDays := range retentionMap {
		// 当前租户的日志目录
		tenantDir := filepath.Join(tenantsRoot, tenantCode)

		entries, readErr := os.ReadDir(tenantDir)
		if readErr != nil {
			// 目录不存在或无法读取时记录 WARN 并继续处理其他租户
			Global().Warn("读取租户日志目录失败，跳过该租户",
				zap.String("tenantCode", tenantCode),
				zap.String("dir", tenantDir),
				zap.Error(readErr),
			)
			continue
		}

		// 计算当前租户的过期截止时间
		cutoff := time.Now().AddDate(0, 0, -retentionDays)

		for _, entry := range entries {
			// 跳过子目录，只处理普通文件
			if entry.IsDir() {
				continue
			}

			name := entry.Name()

			// 跳过当前写入的 tenant.log，只清理轮转备份文件
			if name == "tenant.log" {
				continue
			}

			// 获取文件详细信息以读取修改时间
			fullPath := filepath.Join(tenantDir, name)
			info, statErr := os.Stat(fullPath)
			if statErr != nil {
				Global().Warn("获取租户日志文件信息失败，跳过该文件",
					zap.String("tenantCode", tenantCode),
					zap.String("file", fullPath),
					zap.Error(statErr),
				)
				continue
			}

			// 判断文件修改时间是否早于截止时间
			if info.ModTime().Before(cutoff) {
				size := info.Size()
				if removeErr := os.Remove(fullPath); removeErr != nil {
					Global().Warn("删除过期租户日志备份文件失败",
						zap.String("tenantCode", tenantCode),
						zap.String("file", fullPath),
						zap.Error(removeErr),
					)
					continue
				}
				deletedCount++
				freedBytes += size
			}
		}
	}

	return deletedCount, freedBytes, nil
}
