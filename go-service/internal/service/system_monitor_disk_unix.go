//go:build linux || darwin

package service

import (
	"math"
	"syscall"
)

// getDiskUsage 使用 syscall.Statfs 获取根分区磁盘使用率（Linux/macOS 均可用）。
func (s *SystemMonitorService) getDiskUsage() float64 {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return 0
	}
	total := stat.Blocks * uint64(stat.Bsize)
	free := stat.Bfree * uint64(stat.Bsize)
	if total == 0 {
		return 0
	}
	usage := float64(total-free) / float64(total) * 100.0
	return math.Round(usage*100) / 100
}
