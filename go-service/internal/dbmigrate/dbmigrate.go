// Package dbmigrate 封装数据库迁移逻辑，基于 golang-migrate 库对 PostgreSQL 执行版本化迁移。
package dbmigrate

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Up 执行所有尚未应用的 Up 迁移脚本，迁移版本记录在 schema_migrations 表中。
// migrationsDir 必须是包含 *.up.sql 文件的目录的绝对路径或相对路径。
func Up(migrationsDir string, host string, port int, user, password, dbname, sslmode string) error {
	if migrationsDir == "" {
		return fmt.Errorf("迁移目录不能为空")
	}

	// 将迁移目录解析为绝对路径，确保 file:// 协议可正确定位文件
	abs, err := filepath.Abs(migrationsDir)
	if err != nil {
		return fmt.Errorf("解析迁移目录路径失败: %w", err)
	}

	// 构造 DSN，对用户名、密码等特殊字符进行 URL 编码，避免解析错误
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(user),
		url.QueryEscape(password),
		host,
		port,
		url.PathEscape(dbname),
		url.QueryEscape(sslmode),
	)

	// 使用 pgx 驱动建立数据库连接（仅用于迁移，与主业务连接池相互独立）
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("打开迁移数据库连接失败: %w", err)
	}
	defer db.Close()

	// 验证数据库连通性，确保迁移前网络和认证均正常
	if err := db.Ping(); err != nil {
		return fmt.Errorf("迁移数据库连通性检查失败: %w", err)
	}

	// 初始化 golang-migrate 的 PostgreSQL 驱动实例
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("初始化迁移驱动失败: %w", err)
	}

	// 将目录路径转换为 file:// 格式的 URL，兼容 Windows 路径分隔符
	sourceURL := "file://" + filepath.ToSlash(abs)

	// 创建迁移实例，关联迁移脚本目录与目标数据库
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return fmt.Errorf("创建迁移实例失败: %w", err)
	}
	defer m.Close()

	// 执行所有待应用的 Up 迁移；若已是最新版本则忽略 ErrNoChange，视为正常
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("执行迁移失败: %w", err)
	}
	return nil
}
