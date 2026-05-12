package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	pkglogger "oa-smart-audit/go-service/internal/pkg/logger"
	"oa-smart-audit/go-service/internal/repository"
)

const (
	dbBackupFilePrefix = "aura_pg_"
	dbBackupFileSuffix = ".dump"
)

// DbBackupConfig 整库备份所需的连接与路径（来自 config.yaml）。
type DbBackupConfig struct {
	Host                  string
	Port                  int
	User                  string
	Password              string
	DBName                string
	SSLMode               string
	Dir                   string
	RetentionFallbackDays int
	DumpTimeout           time.Duration
}

// DbBackupService 按 system_configs 中 system.backup_* 执行 PostgreSQL 整库 pg_dump 与过期文件清理。
type DbBackupService struct {
	mu                 sync.Mutex
	lastFiredSlot      time.Time
	lastMissingDumpLog time.Time
	systemConfigRepo   *repository.SystemConfigRepo
	cfg                DbBackupConfig
}

// NewDbBackupService 创建备份服务；dir 建议为绝对路径或可写相对路径。
func NewDbBackupService(repo *repository.SystemConfigRepo, cfg DbBackupConfig) *DbBackupService {
	if cfg.RetentionFallbackDays <= 0 {
		cfg.RetentionFallbackDays = 30
	}
	if cfg.DumpTimeout <= 0 {
		cfg.DumpTimeout = 45 * time.Minute
	}
	return &DbBackupService{
		systemConfigRepo: repo,
		cfg:              cfg,
	}
}

// DumpTimeout 返回单次 pg_dump 命令的超时时间。
func (s *DbBackupService) DumpTimeout() time.Duration {
	return s.cfg.DumpTimeout
}

var cronBackupParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

func (s *DbBackupService) readString(key, def string) string {
	v, err := s.systemConfigRepo.FindByKey(key)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			pkglogger.Global().Warn("读取系统配置失败", zap.String("key", key), zap.Error(err))
		}
		return def
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	return v
}

func (s *DbBackupService) readBool(key string, def bool) bool {
	v := strings.ToLower(strings.TrimSpace(s.readString(key, "")))
	if v == "" {
		return def
	}
	if v == "false" || v == "0" || v == "no" || v == "off" {
		return false
	}
	return v == "true" || v == "1" || v == "yes" || v == "on"
}

func (s *DbBackupService) readInt(key string, def int) int {
	v := strings.TrimSpace(s.readString(key, ""))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return def
	}
	return n
}

// Tick 由定时器每分钟调用一次：按需执行 pg_dump，并按保留天数清理本服务生成的备份文件。
func (s *DbBackupService) Tick(parent context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.cfg.Dir, 0o755); err != nil {
		pkglogger.Global().Warn("创建备份目录失败", zap.String("dir", s.cfg.Dir), zap.Error(err))
		return
	}

	enabled := s.readBool("system.backup_enabled", false)
	cronExpr := s.readString("system.backup_cron", "0 2 * * *")
	retentionDays := s.readInt("system.backup_retention_days", s.cfg.RetentionFallbackDays)
	if retentionDays <= 0 {
		retentionDays = s.cfg.RetentionFallbackDays
	}

	if err := s.pruneOldBackups(retentionDays); err != nil {
		pkglogger.Global().Warn("清理过期数据库备份失败", zap.Error(err))
	}

	if !enabled {
		return
	}

	sched, err := cronBackupParser.Parse(cronExpr)
	if err != nil {
		pkglogger.Global().Warn("system.backup_cron 表达式无效，跳过数据库备份",
			zap.String("expr", cronExpr),
			zap.Error(err),
		)
		return
	}

	now := time.Now()
	lastSlot, ok := lastCronFireOnOrBefore(sched, now)
	if !ok {
		return
	}
	if !lastSlot.After(s.lastFiredSlot) {
		return
	}

	pgDump, err := exec.LookPath("pg_dump")
	if err != nil {
		if time.Since(s.lastMissingDumpLog) > time.Hour {
			s.lastMissingDumpLog = time.Now()
			pkglogger.Global().Warn("未找到 pg_dump 可执行文件，已跳过数据库备份（请在运行环境安装 PostgreSQL 客户端）",
				zap.Error(err),
			)
		}
		return
	}

	outPath := s.buildDumpPath(lastSlot)
	ctx, cancel := context.WithTimeout(parent, s.cfg.DumpTimeout)
	defer cancel()

	if err := s.runPgDump(ctx, pgDump, outPath); err != nil {
		pkglogger.Global().Error("数据库整库备份失败",
			zap.String("path", outPath),
			zap.Error(err),
		)
		return
	}

	s.lastFiredSlot = lastSlot
	pkglogger.Global().Info("数据库整库备份完成",
		zap.String("path", outPath),
		zap.String("cron_slot", lastSlot.UTC().Format(time.RFC3339)),
	)
}

// lastCronFireOnOrBefore 返回不晚于 now 的最近一次 cron 触发时刻（用于与 lastFiredSlot 去重）。
func lastCronFireOnOrBefore(sched cron.Schedule, now time.Time) (time.Time, bool) {
	start := now.Add(-1200 * 24 * time.Hour)
	t := start
	var last time.Time
	const maxIter = 5_000_000
	for range maxIter {
		n := sched.Next(t)
		if !n.After(t) {
			return time.Time{}, false
		}
		if n.After(now) {
			return last, !last.IsZero()
		}
		last = n
		t = n
	}
	return time.Time{}, false
}

var unsafeFileChars = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

func (s *DbBackupService) safeDBName() string {
	base := unsafeFileChars.ReplaceAllString(s.cfg.DBName, "_")
	if base == "" {
		return "db"
	}
	return base
}

func (s *DbBackupService) buildDumpPath(slot time.Time) string {
	ts := slot.UTC().Format("20060102T150405Z")
	name := fmt.Sprintf("%s%s_%s%s", dbBackupFilePrefix, s.safeDBName(), ts, dbBackupFileSuffix)
	return filepath.Join(s.cfg.Dir, name)
}

func (s *DbBackupService) runPgDump(ctx context.Context, pgDumpPath, outPath string) error {
	args := []string{
		"-h", s.cfg.Host,
		"-p", strconv.Itoa(s.cfg.Port),
		"-U", s.cfg.User,
		"-d", s.cfg.DBName,
		"-Fc",
		"-f", outPath,
		"--no-owner",
		"--no-acl",
	}
	cmd := exec.CommandContext(ctx, pgDumpPath, args...)
	cmd.Env = append(os.Environ(),
		"PGCLIENTENCODING=UTF8",
		fmt.Sprintf("PGPASSWORD=%s", s.cfg.Password),
		fmt.Sprintf("PGSSLMODE=%s", s.cfg.SSLMode),
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		_ = os.Remove(outPath)
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func (s *DbBackupService) pruneOldBackups(retentionDays int) error {
	entries, err := os.ReadDir(s.cfg.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour)
	var removed int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, dbBackupFilePrefix) || !strings.HasSuffix(name, dbBackupFileSuffix) {
			continue
		}
		full := filepath.Join(s.cfg.Dir, name)
		st, err := os.Stat(full)
		if err != nil {
			continue
		}
		if st.ModTime().Before(cutoff) {
			if err := os.Remove(full); err == nil {
				removed++
			} else {
				pkglogger.Global().Warn("删除过期备份文件失败", zap.String("path", full), zap.Error(err))
			}
		}
	}
	if removed > 0 {
		pkglogger.Global().Info("已删除过期数据库备份文件", zap.Int("count", removed), zap.Int("retention_days", retentionDays))
	}
	return nil
}
