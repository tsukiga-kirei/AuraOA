// Package config 负责加载和解析应用程序配置，基于 Viper 读取 config.yaml。
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"

	"oa-smart-audit/go-service/internal/pkg/logger"
)

// Config 应用程序全局配置，对应 config.yaml 的顶层结构。
type Config struct {
	// Server HTTP 服务器配置
	Server ServerConfig `mapstructure:"server"`
	// Database PostgreSQL 数据库连接配置
	Database DatabaseConfig `mapstructure:"database"`
	// Redis 缓存连接配置
	Redis RedisConfig `mapstructure:"redis"`
	// JWT 令牌签发与校验配置
	JWT JWTConfig `mapstructure:"jwt"`
	// CORS 跨域资源共享配置
	CORS CORSConfig `mapstructure:"cors"`
	// Encryption AES 对称加密配置
	Encryption EncryptionConfig `mapstructure:"encryption"`
	// Log 日志系统配置
	Log logger.LogConfig `mapstructure:"log"`
}

// ServerConfig HTTP 服务器配置。
type ServerConfig struct {
	// Port 监听端口号
	Port int `mapstructure:"port"`
}

// DatabaseConfig PostgreSQL 数据库连接配置。
type DatabaseConfig struct {
	// Host 数据库主机地址
	Host string `mapstructure:"host"`
	// Port 数据库端口
	Port int `mapstructure:"port"`
	// User 数据库用户名
	User string `mapstructure:"user"`
	// Password 数据库密码
	Password string `mapstructure:"password"`
	// DBName 数据库名称
	DBName string `mapstructure:"dbname"`
	// SSLMode SSL 连接模式，如 disable / require
	SSLMode string `mapstructure:"sslmode"`
	// MaxOpenConns 连接池最大打开连接数，默认 50
	MaxOpenConns int `mapstructure:"max_open_conns"`
	// MaxIdleConns 连接池最大空闲连接数，默认 10
	MaxIdleConns int `mapstructure:"max_idle_conns"`
}

// DSN 返回 PostgreSQL 标准连接字符串。
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// RedisConfig Redis 缓存连接配置。
type RedisConfig struct {
	// Host Redis 主机地址
	Host string `mapstructure:"host"`
	// Port Redis 端口
	Port int `mapstructure:"port"`
	// Password Redis 认证密码，无密码时留空
	Password string `mapstructure:"password"`
	// DB 使用的 Redis 数据库编号，默认 0
	DB int `mapstructure:"db"`
}

// Addr 返回 host:port 格式的 Redis 连接地址。
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// JWTConfig JWT 令牌签发与校验配置。
type JWTConfig struct {
	// Secret 用于签名的密钥，生产环境应使用强随机字符串
	Secret string `mapstructure:"secret"`
	// AccessTokenTTL 访问令牌有效期，默认 2h
	AccessTokenTTL time.Duration `mapstructure:"access_token_ttl"`
	// RefreshTokenTTL 刷新令牌有效期，默认 168h（7 天）
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

// CORSConfig 跨域资源共享配置。
type CORSConfig struct {
	// AllowedOrigins 允许跨域访问的来源列表
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// EncryptionConfig AES 对称加密配置。
type EncryptionConfig struct {
	// Key AES 加密密钥，长度须为 16、24 或 32 字节
	Key string `mapstructure:"key"`
}

// Load 通过 Viper 读取 config.yaml 并解析为 Config 结构体。
// 配置文件优先从当前目录查找，其次从上两级目录查找。
// 环境变量会自动覆盖同名配置项（点号替换为下划线）。
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("../../")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 数据库连接池参数缺失时使用默认值
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 50
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 10
	}

	// JWT 令牌有效期缺失时使用默认值
	if cfg.JWT.AccessTokenTTL == 0 {
		cfg.JWT.AccessTokenTTL = 2 * time.Hour
	}
	if cfg.JWT.RefreshTokenTTL == 0 {
		cfg.JWT.RefreshTokenTTL = 7 * 24 * time.Hour
	}

	return &cfg, nil
}
