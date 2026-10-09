package handler

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/repository"
)

// snapshotExportTestDB 用固定查询结果验证实际生成的工作簿及错误响应，不连接业务数据库。
func snapshotExportTestDB(t *testing.T, historyError error) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test"}), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal(err)
	}
	err = db.Callback().Query().After("gorm:query").Register("test:export_rows", func(tx *gorm.DB) {
		switch dest := tx.Statement.Dest.(type) {
		case *[]repository.AuditSnapshotListRow:
			for i := 0; i < 25; i++ {
				*dest = append(*dest, repository.AuditSnapshotListRow{AuditProcessSnapshot: model.AuditProcessSnapshot{
					ProcessID: fmt.Sprintf("request-%d", i), Title: "审核流程",
				}})
			}
		case *[]repository.ArchiveSnapshotListRow:
			for i := 0; i < 25; i++ {
				*dest = append(*dest, repository.ArchiveSnapshotListRow{ArchiveProcessSnapshot: model.ArchiveProcessSnapshot{
					ProcessID: fmt.Sprintf("request-%d", i), Title: "归档流程",
				}})
			}
		case *[]repository.AuditLogWithUser:
			if historyError != nil {
				tx.AddError(historyError)
				return
			}
			*dest = []repository.AuditLogWithUser{{AuditLog: model.AuditLog{
				ProcessID: "request-0", CreatedAt: time.Date(2026, 10, 9, 3, 0, 0, 0, time.UTC),
				Recommendation: "approve", Score: 87, Confidence: 93, TriggerSource: model.AuditTriggerWorkbenchManual,
			}, UserName: "张三"}}
		case *[]repository.ArchiveLogWithUser:
			if historyError != nil {
				tx.AddError(historyError)
				return
			}
			*dest = []repository.ArchiveLogWithUser{{ArchiveLog: model.ArchiveLog{
				ProcessID: "request-0", CreatedAt: time.Date(2026, 10, 9, 3, 0, 0, 0, time.UTC),
				Compliance: "compliant", ComplianceScore: 87, Confidence: 93,
			}, UserName: "张三"}}
		default:
			t.Errorf("导出走入非预期查询（可能回到了分页查询）: %T", dest)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func runSnapshotExport(t *testing.T, module, locale string, historyError error) *httptest.ResponseRecorder {
	t.Helper()
	db := snapshotExportTestDB(t, historyError)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("tenant_id", "11111111-1111-4111-8111-111111111111")
	c.Request = httptest.NewRequest(http.MethodGet, "/?page=2&page_size=20&columns=process_id,title&include_history=true", nil)
	c.Request.Header.Set("Accept-Language", locale)
	if module == "audit" {
		NewAuditHandler(nil, repository.NewAuditProcessSnapshotRepo(db), repository.NewAuditLogRepo(db)).ExportSnapshots(c)
	} else {
		NewArchiveReviewHandler(nil, repository.NewArchiveProcessSnapshotRepo(db), repository.NewArchiveLogRepo(db)).ExportSnapshots(c)
	}
	return w
}

func TestSnapshotExportWorkbookIncludesAllRowsAndHistory(t *testing.T) {
	for _, module := range []string{"audit", "archive"} {
		for _, locale := range []string{"zh-CN", "en-US"} {
			t.Run(module+"/"+locale, func(t *testing.T) {
				w := runSnapshotExport(t, module, locale, nil)
				if w.Code != http.StatusOK {
					t.Fatalf("导出失败: %d %s", w.Code, w.Body.String())
				}
				book, err := excelize.OpenReader(bytes.NewReader(w.Body.Bytes()))
				if err != nil {
					t.Fatal(err)
				}
				defer book.Close()
				rows, err := book.GetRows("Sheet1")
				if err != nil {
					t.Fatal(err)
				}
				if len(rows) != 26 || rows[25][0] != "request-24" {
					t.Fatalf("导出应包含表头和全部 25 条数据，实际行数 %d", len(rows))
				}
				if len(rows[0]) != 3 || len(rows[1]) != 3 {
					t.Fatalf("应包含请求 ID、标题、历史三列: %v", rows[:2])
				}
				for _, fragment := range []string{"张三", "87", "93%"} {
					if !strings.Contains(rows[1][2], fragment) {
						t.Errorf("历史列缺少 %q: %s", fragment, rows[1][2])
					}
				}
				if locale == "en-US" && !strings.Contains(rows[1][2], "Operator: ") {
					t.Errorf("英文历史仍使用中文标签: %s", rows[1][2])
				}
			})
		}
	}
}

func TestSnapshotExportHistoryFailureReturnsError(t *testing.T) {
	for _, module := range []string{"audit", "archive"} {
		t.Run(module, func(t *testing.T) {
			w := runSnapshotExport(t, module, "zh-CN", errors.New("历史查询失败"))
			if w.Code != http.StatusInternalServerError || !strings.Contains(w.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("历史查询失败应返回错误，不能生成空白历史文件: %d %s", w.Code, w.Header().Get("Content-Type"))
			}
		})
	}
}
