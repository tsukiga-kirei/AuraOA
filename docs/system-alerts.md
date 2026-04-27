# 系统告警模块文档

**更新日期**: 2026-04-27

---

## 1. 概述

系统告警模块基于系统运行监控数据自动生成告警，帮助系统管理员及时发现资源瓶颈和服务异常。

告警在仪表盘的「系统运行监控」组件中展示，仅系统管理员可见。

---

## 2. 架构

```
┌─────────────────────────────────────────────────────────────────┐
│  SystemMonitorWidget (前端)                                      │
│  ├── 资源仪表盘 (CPU / 内存 / 磁盘)                              │
│  ├── 服务状态列表                                                │
│  └── 告警列表 ← 本次新增                                         │
└────────────────────────────┬────────────────────────────────────┘
                             │ GET /api/admin/system-monitor
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  SystemMonitorService (后端)                                     │
│  ├── getCPUUsage()                                              │
│  ├── getMemoryUsage()                                           │
│  ├── getDiskUsage()                                             │
│  ├── checkServices()                                            │
│  └── generateAlerts() ← 本次新增                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 告警规则

| 来源 | 级别 | 触发条件 | i18n Key |
|------|------|----------|----------|
| CPU | warning | 使用率 > 80% | `alert.cpu.warning` |
| CPU | critical | 使用率 > 95% | `alert.cpu.critical` |
| 内存 | warning | 使用率 > 85% | `alert.memory.warning` |
| 内存 | critical | 使用率 > 95% | `alert.memory.critical` |
| 磁盘 | warning | 使用率 > 90% | `alert.disk.warning` |
| 磁盘 | critical | 使用率 > 95% | `alert.disk.critical` |
| 服务 | warning | 状态为 degraded | `alert.service.degraded` |
| 服务 | critical | 状态为 offline | `alert.service.offline` |

阈值常量定义位置：
- 后端: `go-service/internal/service/system_monitor_service.go`
- 前端: `frontend/composables/useThemeColors.ts` (`MONITOR_THRESHOLDS`)

---

## 4. 数据结构

### 4.1 后端 DTO

```go
// SystemAlertDTO 系统告警条目
type SystemAlertDTO struct {
    Level   string `json:"level"`   // "warning" | "critical"
    Source  string `json:"source"`  // "cpu" | "memory" | "disk" | "service"
    Message string `json:"message"` // i18n key（前端翻译）
    Value   string `json:"value"`   // 当前值（用于前端格式化）
}
```

### 4.2 前端类型

```typescript
interface SystemAlert {
  level: 'warning' | 'critical'
  source: 'cpu' | 'memory' | 'disk' | 'service'
  message: string  // i18n key
  value: string    // 当前值
}
```

### 4.3 API 响应示例

```json
{
  "cpu_usage": 87.5,
  "memory_usage": 72.3,
  "disk_usage": 45.0,
  "services": [...],
  "uptime_seconds": 86400,
  "alerts": [
    {
      "level": "warning",
      "source": "cpu",
      "message": "alert.cpu.warning",
      "value": "87.5"
    }
  ]
}
```

---

## 5. 国际化

告警消息通过 i18n key 传递，前端负责翻译。

| Key | 中文 | English |
|-----|------|---------|
| `overview.monitor.recentAlerts` | 最近告警 | Recent Alerts |
| `overview.monitor.noAlerts` | 暂无告警 | No alerts |
| `overview.monitor.alert.cpu.critical` | CPU 使用率严重过高：{0}% | CPU usage critically high: {0}% |
| `overview.monitor.alert.cpu.warning` | CPU 使用率偏高：{0}% | CPU usage elevated: {0}% |
| `overview.monitor.alert.memory.critical` | 内存使用率严重过高：{0}% | Memory usage critically high: {0}% |
| `overview.monitor.alert.memory.warning` | 内存使用率偏高：{0}% | Memory usage elevated: {0}% |
| `overview.monitor.alert.disk.critical` | 磁盘使用率严重过高：{0}% | Disk usage critically high: {0}% |
| `overview.monitor.alert.disk.warning` | 磁盘使用率偏高：{0}% | Disk usage elevated: {0}% |
| `overview.monitor.alert.service.offline` | 服务离线：{0} | Service offline: {0} |
| `overview.monitor.alert.service.degraded` | 服务降级：{0} | Service degraded: {0} |

---

## 6. 相关文件

| 文件 | 说明 |
|------|------|
| `go-service/internal/dto/system_monitor_dto.go` | 告警 DTO 定义 |
| `go-service/internal/service/system_monitor_service.go` | 告警生成逻辑 |
| `frontend/types/dashboard-overview.ts` | 前端类型定义 |
| `frontend/components/SystemMonitorWidget.vue` | 告警 UI 展示 |
| `frontend/composables/useThemeColors.ts` | 阈值常量与颜色函数 |
| `frontend/locales/zh-CN.ts` | 中文翻译 |
| `frontend/locales/en-US.ts` | 英文翻译 |
