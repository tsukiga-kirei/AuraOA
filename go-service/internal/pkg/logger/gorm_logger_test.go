package logger

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestCompactDBError(t *testing.T) {
	if got := CompactDBError(nil); got != "" {
		t.Fatalf("expected empty for nil error, got %q", got)
	}
	if got := CompactDBError(errors.New("plain sql error")); got != "plain sql error" {
		t.Fatalf("unexpected plain error: %q", got)
	}
	dmErr := errors.New("Error -6128: 无效的数字\nstack info:\n   1). dm_build_424\n   2). dm_build_413")
	if got := CompactDBError(dmErr); got != "Error -6128: 无效的数字" {
		t.Fatalf("expected compact DM error, got %q", got)
	}
	dmCRLF := errors.New("Error -6128: 无效的数字\r\nstack info:\r\n   1). dm_build_424")
	if got := CompactDBError(dmCRLF); got != "Error -6128: 无效的数字" {
		t.Fatalf("expected compact CRLF DM error, got %q", got)
	}
}

func TestGormLoggerTraceStripsDMStackInfo(t *testing.T) {
	core, logs := observer.New(zapcore.DebugLevel)
	prev := globalLogger
	globalLogger = zap.New(core)
	defer func() { globalLogger = prev }()

	l := NewGormLogger(200*time.Millisecond, true)
	dmErr := errors.New("Error -6128: 无效的数字\nstack info:\n   1). github.com/Rulessly/dm-driver-gorm")
	l.Trace(context.Background(), time.Now().Add(-5*time.Millisecond), func() (string, int64) {
		return "SELECT 1 FROM DOCIMAGEFILE", 0
	}, dmErr)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}
	gotErr := entries[0].ContextMap()["error"]
	if gotErr != "Error -6128: 无效的数字" {
		t.Fatalf("expected compacted error field, got %v", gotErr)
	}
	if strings.Contains(gotErr.(string), "stack info:") {
		t.Fatalf("stack info should be stripped, got %v", gotErr)
	}
}
