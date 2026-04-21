package dto

// ServiceStatusDTO 单个服务的健康状态。
type ServiceStatusDTO struct {
	Name           string `json:"name"`
	Status         string `json:"status"`           // "online" | "offline" | "degraded"
	ResponseTimeMs int64  `json:"response_time_ms"`
}

// SystemMonitorResponse 系统运行监控数据（GET /api/admin/system-monitor）。
type SystemMonitorResponse struct {
	CPUUsage      float64            `json:"cpu_usage"`       // CPU 使用率 (0-100)
	MemoryUsage   float64            `json:"memory_usage"`    // 内存使用率 (0-100)
	DiskUsage     float64            `json:"disk_usage"`      // 磁盘使用率 (0-100)
	Services      []ServiceStatusDTO `json:"services"`        // 关键服务状态列表
	UptimeSeconds float64            `json:"uptime_seconds"`  // 系统运行时间（秒）
}
