package dto

// ServiceStatusDTO 单个服务的健康状态。
type ServiceStatusDTO struct {
	Name           string `json:"name"`
	Status         string `json:"status"`           // "online" | "offline" | "degraded"
	ResponseTimeMs int64  `json:"response_time_ms"`
}

// SystemAlertDTO 系统告警条目。
type SystemAlertDTO struct {
	Level   string `json:"level"`   // "warning" | "critical"
	Source  string `json:"source"`  // 告警来源: "cpu" | "memory" | "disk" | "service"
	Message string `json:"message"` // 告警消息（i18n key，前端翻译）
	Value   string `json:"value"`   // 当前值（用于前端格式化）
}

// SystemMonitorResponse 系统运行监控数据（GET /api/admin/system-monitor）。
type SystemMonitorResponse struct {
	CPUUsage      float64            `json:"cpu_usage"`       // CPU 使用率 (0-100)
	MemoryUsage   float64            `json:"memory_usage"`    // 内存使用率 (0-100)
	DiskUsage     float64            `json:"disk_usage"`      // 磁盘使用率 (0-100)
	Services      []ServiceStatusDTO `json:"services"`        // 关键服务状态列表
	UptimeSeconds float64            `json:"uptime_seconds"`  // 系统运行时间（秒）
	Alerts        []SystemAlertDTO   `json:"alerts"`          // 系统告警列表
}
