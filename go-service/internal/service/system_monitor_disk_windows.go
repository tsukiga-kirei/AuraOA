//go:build windows

package service

import (
	"golang.org/x/sys/windows"
	"math"
	"os"
)

// getDiskUsage 返回当前工作目录所在磁盘的使用率。
func (s *SystemMonitorService) getDiskUsage() float64 {
	dir, err := os.Getwd()
	if err != nil {
		return 0
	}
	path, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return 0
	}
	var available, total, free uint64
	if windows.GetDiskFreeSpaceEx(path, &available, &total, &free) != nil || total == 0 {
		return 0
	}
	return math.Round(float64(total-free)/float64(total)*10000) / 100
}
