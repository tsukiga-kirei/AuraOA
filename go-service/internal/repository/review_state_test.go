package repository

import (
	"net/http/httptest"
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestArchiveUpdatesProtectTerminalJobs(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	var statement string
	var args []interface{}
	db.Callback().Update().After("gorm:update").Register("test:capture", func(tx *gorm.DB) {
		statement = tx.Statement.SQL.String()
		args = append([]interface{}{}, tx.Statement.Vars...)
	})
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil)
	tenant, id := uuid.New(), uuid.New()
	c.Set("tenant_id", tenant.String())
	_, err = NewArchiveLogRepo(db).UpdateActiveFields(c, id, map[string]interface{}{"status": model.JobStatusFailed})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(statement, "status IN") || !strings.Contains(statement, "tenant_id") || !strings.Contains(statement, "id =") {
		t.Fatalf("更新缺少原子状态或归属保护：%s", statement)
	}
	for _, active := range model.ActiveReviewJobStatuses() {
		found := false
		for _, arg := range args {
			if arg == active {
				found = true
			}
		}
		if !found {
			t.Errorf("遗漏活跃状态 %s", active)
		}
	}
}

func TestAgentHistoryZeroIsWrittenAndReadAsZero(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test"}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, turns := range []int{0, 3, 10} {
		agent := model.AgentDefinition{AgentCode: "test", Name: "测试", ToolHistoryTurns: turns}
		result := db.Create(&agent)
		if result.Error != nil {
			t.Fatal(result.Error)
		}
		if agent.ToolHistoryTurns != turns || agent.EffectiveToolHistoryTurns() != turns {
			t.Errorf("创建或回显覆盖了轮数：%d → %d", turns, agent.ToolHistoryTurns)
		}
		if !strings.Contains(result.Statement.SQL.String(), "tool_history_turns") {
			t.Fatal("写入遗漏历史轮数")
		}
	}
}
