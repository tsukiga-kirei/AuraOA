package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

//配置保存应用程序的所有配置。
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	Redis      RedisConfig      `mapstructure:"redis"`
	JWT        JWTConfig        `mapstructure:"jwt"`
	CORS       CORSConfig       `mapstructure:"cors"`
	Encryption EncryptionConfig `mapstructure:"encryption"`
}

//ServerConfig 保存 HTTP 服务器设置。
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

//DatabaseConfig 保存 PostgreSQL 连接设置。
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

//DSN 返回 PostgreSQL 连接字符串。
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

//RedisConfig 保存 Redis 连接设置。
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

//Addr 以主机:端口格式返回 Redis 地址。
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

//JWTConfig 保存 JWT 签名和 TTL 设置。
type JWTConfig struct {
	Secret          string        `mapstructure:"secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

//CORSConfig 保存 CORS 允许的来源。
type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

// EncryptionConfig 保存 AES 对称加密密钥。
type EncryptionConfig struct {
	Key string `mapstructure:"key"`
}

//Load 通过 Viper 读取 config.yaml 并将其解组到 Config 结构中。
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

	//如果未设置，则应用连接池的默认值
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 50
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 10
	}

	//如果未设置，则应用 JWT TTL 的默认值
	if cfg.JWT.AccessTokenTTL == 0 {
		cfg.JWT.AccessTokenTTL = 2 * time.Hour
	}
	if cfg.JWT.RefreshTokenTTL == 0 {
		cfg.JWT.RefreshTokenTTL = 7 * 24 * time.Hour
	}

	return &cfg, nil
}
