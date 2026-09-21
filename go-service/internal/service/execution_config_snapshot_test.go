package service

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/model"
)

func TestExecutionVersionNumberPrefersBaseVersion(t *testing.T) {
	// 1. nil 版本返回 nil
	if got := executionVersionNumber(nil); got != nil {
		t.Fatalf("nil 版本应返回 nil，实际: %v", got)
	}

	// 2. 优先使用已预加载的 BaseVersionNo
	baseNo := 10
	vWithBase := &model.ExecutionConfigVersion{
		ID:            uuid.New(),
		VersionNo:     8,
		BaseVersionNo: &baseNo,
	}
	if got := executionVersionNumber(vWithBase); got == nil || *got != 10 {
		t.Fatalf("应优先使用 BaseVersionNo (10)，实际: %v", got)
	}

	// 3. 次优从 ConfigSnapshot JSON 中解析 base_config_version_no
	vWithSnapshot := &model.ExecutionConfigVersion{
		ID:             uuid.New(),
		VersionNo:      8,
		ConfigSnapshot: datatypes.JSON(`{"base_config_version_no":10,"merged_rules":"test"}`),
	}
	if got := executionVersionNumber(vWithSnapshot); got == nil || *got != 10 {
		t.Fatalf("应从 ConfigSnapshot 中提取 base_config_version_no (10)，实际: %v", got)
	}

	// 4. 老数据或未记录基础版本号时，优雅回退到执行快照自身的 VersionNo
	vLegacy := &model.ExecutionConfigVersion{
		ID:             uuid.New(),
		VersionNo:      8,
		ConfigSnapshot: datatypes.JSON(`{"blocks":[]}`),
	}
	if got := executionVersionNumber(vLegacy); got == nil || *got != 8 {
		t.Fatalf("老数据应回退到自身 VersionNo (8)，实际: %v", got)
	}
}
