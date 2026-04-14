// Package logger 提供结构化日志系统，支持全局日志和租户隔离日志，
// 基于 go.uber.org/zap 实现，集成 lumberjack 文件轮转。
package logger

// LogConfig 日志系统配置，对应 config.yaml 中的 log 配置节。
// 所有字段均有默认值，缺失时由初始化逻辑自动填充。
type LogConfig struct {
	// Level 日志输出等级，可选值：debug / info / warn / error，默认 info。
	// 生产环境建议设置为 info，调试时可临时改为 debug。
	Level string `mapstructure:"level"`

	// Dir 日志文件根目录，默认 logs。
	// 全局日志写入 {Dir}/app.log，租户日志写入 {Dir}/tenants/{code}/tenant.log。
	Dir string `mapstructure:"dir"`

	// MaxSizeMB 单个日志文件的最大体积（MB），超出后触发轮转，默认 100。
	MaxSizeMB int `mapstructure:"max_size_mb"`

	// MaxBackups 轮转后最多保留的备份文件数量，默认 5。
	// 超出数量的旧备份文件将被自动删除。
	MaxBackups int `mapstructure:"max_backups"`

	// Compress 是否对轮转备份文件进行 gzip 压缩，默认 true。
	// 开启后可显著减少备份文件的磁盘占用。
	Compress bool `mapstructure:"compress"`

	// GlobalRetentionDays 全局日志文件的保留天数，默认 30。
	// 作为 system_configs 表中 system.global_log_retention_days 的兜底默认值，
	// 若数据库中已配置该键，则运行时优先使用数据库中的值。
	GlobalRetentionDays int `mapstructure:"global_retention_days"`
}
