package service

import (
	"bufio"
	"context"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/dto"
)

// SystemMonitorService 提供系统资源监控和服务健康检查。
type SystemMonitorService struct {
	db        *gorm.DB
	rdb       *redis.Client
	startTime time.Time
}

// NewSystemMonitorService 创建 SystemMonitorService 实例，startTime 记录服务启动时间。
func NewSystemMonitorService(db *gorm.DB, rdb *redis.Client) *SystemMonitorService {
	return &SystemMonitorService{
		db:        db,
		rdb:       rdb,
		startTime: time.Now(),
	}
}

// GetSystemMonitorData 采集系统资源使用率和关键服务健康状态。
func (s *SystemMonitorService) GetSystemMonitorData(ctx context.Context) (*dto.SystemMonitorResponse, error) {
	resp := &dto.SystemMonitorResponse{
		CPUUsage:      s.getCPUUsage(),
		MemoryUsage:   s.getMemoryUsage(),
		DiskUsage:     s.getDiskUsage(),
		Services:      s.checkServices(ctx),
		UptimeSeconds: math.Round(time.Since(s.startTime).Seconds()*100) / 100,
	}
	return resp, nil
}

// ── CPU 使用率（Linux /proc/stat） ──────────────────────────────────────────

// cpuTimes 从 /proc/stat 的 cpu 行解析出 idle 和 total 时间。
func cpuTimes() (idle, total uint64, ok bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu ") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			return 0, 0, false
		}
		// fields: cpu user nice system idle iowait irq softirq steal guest guest_nice
		var sum uint64
		for i := 1; i < len(fields); i++ {
			v, err := strconv.ParseUint(fields[i], 10, 64)
			if err != nil {
				return 0, 0, false
			}
			sum += v
			if i == 4 { // idle 字段
				idle = v
			}
		}
		return idle, sum, true
	}
	return 0, 0, false
}

// getCPUUsage 读取 /proc/stat 两次（间隔 100ms）计算 CPU 使用率。
// 非 Linux 环境返回 0。
func (s *SystemMonitorService) getCPUUsage() float64 {
	idle1, total1, ok := cpuTimes()
	if !ok {
		return 0
	}
	time.Sleep(100 * time.Millisecond)
	idle2, total2, ok := cpuTimes()
	if !ok {
		return 0
	}

	idleDelta := float64(idle2 - idle1)
	totalDelta := float64(total2 - total1)
	if totalDelta == 0 {
		return 0
	}
	usage := (1.0 - idleDelta/totalDelta) * 100.0
	return math.Round(usage*100) / 100
}

// ── 内存使用率 ──────────────────────────────────────────────────────────────

// getMemoryUsage 从 /proc/meminfo 读取 MemTotal 和 MemAvailable 计算使用率。
// 非 Linux 环境回退到 runtime.MemStats。
func (s *SystemMonitorService) getMemoryUsage() float64 {
	if usage, ok := s.memoryFromProc(); ok {
		return usage
	}
	// 回退：使用 Go runtime 内存统计（仅反映本进程）
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Sys == 0 {
		return 0
	}
	usage := float64(m.Alloc) / float64(m.Sys) * 100.0
	return math.Round(usage*100) / 100
}

func (s *SystemMonitorService) memoryFromProc() (float64, bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, false
	}
	defer f.Close()

	var memTotal, memAvailable uint64
	var foundTotal, foundAvailable bool

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			memTotal = parseMemInfoValue(line)
			foundTotal = true
		} else if strings.HasPrefix(line, "MemAvailable:") {
			memAvailable = parseMemInfoValue(line)
			foundAvailable = true
		}
		if foundTotal && foundAvailable {
			break
		}
	}
	if !foundTotal || !foundAvailable || memTotal == 0 {
		return 0, false
	}
	usage := float64(memTotal-memAvailable) / float64(memTotal) * 100.0
	return math.Round(usage*100) / 100, true
}

// parseMemInfoValue 解析 /proc/meminfo 行中的数值（单位 kB）。
func parseMemInfoValue(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	v, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return v
}

// ── 磁盘使用率 ──────────────────────────────────────────────────────────────

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

// ── 服务健康检查 ────────────────────────────────────────────────────────────

const healthCheckTimeout = 2 * time.Second

// checkServices 检查各关键服务的健康状态。
func (s *SystemMonitorService) checkServices(ctx context.Context) []dto.ServiceStatusDTO {
	services := make([]dto.ServiceStatusDTO, 0, 4)

	// 1. API 服务（自身）— 能响应即在线
	services = append(services, dto.ServiceStatusDTO{
		Name:           "API 服务",
		Status:         "online",
		ResponseTimeMs: 0,
	})

	// 2. 数据库
	services = append(services, s.checkDatabase(ctx))

	// 3. Redis
	services = append(services, s.checkRedis(ctx))

	// 4. AI 模型服务（通过数据库查询已启用的模型数量判断）
	services = append(services, s.checkAIModelService(ctx))

	return services
}

// checkDatabase 检查数据库连接健康状态。
func (s *SystemMonitorService) checkDatabase(ctx context.Context) dto.ServiceStatusDTO {
	svc := dto.ServiceStatusDTO{Name: "数据库"}

	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	sqlDB, err := s.db.DB()
	if err != nil {
		svc.Status = "offline"
		svc.ResponseTimeMs = time.Since(start).Milliseconds()
		return svc
	}
	err = sqlDB.PingContext(checkCtx)
	svc.ResponseTimeMs = time.Since(start).Milliseconds()
	if err != nil {
		svc.Status = "offline"
	} else {
		svc.Status = "online"
	}
	return svc
}

// checkRedis 检查 Redis 连接健康状态。
func (s *SystemMonitorService) checkRedis(ctx context.Context) dto.ServiceStatusDTO {
	svc := dto.ServiceStatusDTO{Name: "Redis"}

	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	err := s.rdb.Ping(checkCtx).Err()
	svc.ResponseTimeMs = time.Since(start).Milliseconds()
	if err != nil {
		svc.Status = "offline"
	} else {
		svc.Status = "online"
	}
	return svc
}

// checkAIModelService 通过查询已启用的 AI 模型数量判断 AI 模型服务状态。
// 有已启用模型 → online；无已启用模型 → degraded；查询失败 → offline。
func (s *SystemMonitorService) checkAIModelService(ctx context.Context) dto.ServiceStatusDTO {
	svc := dto.ServiceStatusDTO{Name: "AI 模型服务"}

	checkCtx, cancel := context.WithTimeout(ctx, healthCheckTimeout)
	defer cancel()

	start := time.Now()
	var count int64
	err := s.db.WithContext(checkCtx).
		Table("ai_model_configs").
		Where("enabled = ?", true).
		Count(&count).Error
	svc.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		svc.Status = "offline"
	} else if count == 0 {
		svc.Status = "degraded"
	} else {
		svc.Status = "online"
	}
	return svc
}
