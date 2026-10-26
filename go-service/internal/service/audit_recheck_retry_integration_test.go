package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/repository"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// TestAuditRecheckRetryPipeline 使用隔离 schema、Redis DB 和本地模型替身验证真正的入队、Worker、持久化及权限链路。
func TestAuditRecheckRetryPipeline(t *testing.T) {
	dsn, redisAddr := os.Getenv("EXECUTION_CONFIG_TEST_DSN"), os.Getenv("RECHECK_TEST_REDIS_ADDR")
	if dsn == "" || redisAddr == "" {
		t.Skip("未配置隔离复核测试库及 Redis")
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	schema := "recheck_test_" + uuid.New().String()[:8]
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(db.Exec(`CREATE SCHEMA "` + schema + `"`).Error)
	defer sqlDB.Close()
	defer db.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)
	must(db.Exec(`SET search_path TO "` + schema + `"`).Error)
	must(db.AutoMigrate(&model.AuditLog{}, &model.AuditProcessSnapshot{}, &model.ProcessAuditConfig{}, &model.ExecutionConfigVersion{}, &model.TenantConfigVersion{}, &model.Tenant{}, &model.AIModelConfig{}, &model.TenantLLMMessageLog{}, &model.TenantLLMMessagePayload{}, &model.SystemConfig{}))
	must(db.Create(&model.SystemConfig{Key: "system.ai_retry_count", Value: "1"}).Error)
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr, DB: 14})
	defer rdb.Close()
	must(rdb.Ping(context.Background()).Err())
	var output atomic.Value
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var request map[string]interface{}
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("模型请求必须为 JSON")
		}
		if request["enable_thinking"] == true {
			t.Error("复核不能意外开启深度思考")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		chunk, _ := json.Marshal(map[string]interface{}{"choices": []interface{}{map[string]interface{}{"delta": map[string]string{"content": output.Load().(string)}}}})
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", chunk)
	}))
	defer server.Close()
	tid, uid, configID, versionID, primaryID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	must(db.Create(&model.AIModelConfig{ID: primaryID, Provider: "vllm", ModelName: "recheck-test", Endpoint: server.URL, MaxTokens: 2048}).Error)
	must(db.Create(&model.Tenant{ID: tid, Code: "recheck-test", Name: "复核测试", PrimaryModelID: &primaryID, TokenQuota: -1, MaxTokensPerRequest: 2048}).Error)
	must(db.Create(&model.ProcessAuditConfig{ID: configID, TenantID: tid, ProcessType: "project", Status: "active", EmbedEnabled: true}).Error)
	cfg := model.AIConfigData{Recheck: &model.AIRecheckConfig{Enabled: true, SystemPrompt: defaultRecheckSystemPrompt, UserPrompt: defaultRecheckUserPrompt}}
	aiRaw, _ := json.Marshal(cfg)
	snapshot := AuditExecutionConfigSnapshot{AIConfig: aiRaw, MergedRules: "检查金额", EffectiveRules: []model.AuditRule{{RuleContent: "检查金额"}}}
	raw, _ := json.Marshal(snapshot)
	must(db.Create(&model.ExecutionConfigVersion{ID: versionID, TenantID: tid, Module: "audit", SourceConfigID: configID, VersionNo: 3, ConfigSnapshot: raw, Fingerprint: "bound-original"}).Error)
	logRepo := repository.NewAuditLogRepo(db)
	svc := &AuditExecuteService{db: db, rdb: rdb, auditLogRepo: logRepo, auditSnapshotRepo: repository.NewAuditProcessSnapshotRepo(db), configRepo: repository.NewProcessAuditConfigRepo(db), tenantRepo: repository.NewTenantRepo(db), aiModelRepo: repository.NewAIModelRepo(db), executionVersions: repository.NewExecutionConfigVersionRepo(db)}
	svc.aiCaller = NewAIModelCallerService(svc.tenantRepo, repository.NewLLMMessageLogRepo(db), db, nil)
	c := svc.workerGinContext(context.Background(), tid, uid)
	c.Set("embed_mode", true)
	other := svc.workerGinContext(context.Background(), uuid.New(), uid)
	other.Set("embed_mode", true)
	for _, entry := range []struct {
		name, response  string
		changed, failed bool
	}{
		{"沿用", `{"reuse_original":true}`, false, false},
		{"调整", `{"reuse_original":false,"reason":"修正金额矛盾","final_result":` + finalAuditRecheck + `}`, true, false},
		{"格式错误", `not-json`, false, true},
		{"空响应", ``, false, true},
	} {
		t.Run(entry.name, func(t *testing.T) {
			output.Store(entry.response)
			source := &model.AuditLog{ID: uuid.New(), TenantID: tid, UserID: uid, ProcessID: uuid.NewString(), ProcessType: "project", Title: "立项", Status: model.JobStatusCompleted, TriggerSource: model.AuditTriggerEmbedManual, ConfigVersionID: &versionID, CreatedAt: apptime.Now(), Recommendation: "approve", Score: 95, Confidence: 95, AuditResult: datatypes.JSON(initialAuditRecheck), AIReasoning: "原分析"}
			source.AIRecheck = datatypes.JSON(persistRecheckJSON(&model.AIRecheckRecord{Status: "failed", OriginalResult: json.RawMessage(initialAuditRecheck), ReuseOriginal: true, Error: "旧调用失败"}))
			must(logRepo.Create(source))
			if _, err := svc.RetryRecheck(other, source.ID); err == nil {
				t.Fatal("跨租户不能复核")
			}
			response, err := svc.RetryRecheck(c, source.ID)
			must(err)
			jobID := uuid.MustParse(response.ID)
			if response.Status != model.JobStatusPending || jobID == source.ID {
				t.Fatal("应新建任务")
			}
			if _, err := svc.RetryRecheck(c, source.ID); err == nil {
				t.Fatal("重复提交必须阻止")
			}
			progress, err := svc.GetAuditJobStatus(c, jobID)
			must(err)
			steps := progress["progress_steps"].([]map[string]interface{})
			if steps[len(steps)-1]["key"] != "rechecking" || steps[len(steps)-1]["current"] != true {
				t.Fatal("排队期间也应直接显示复核步骤")
			}
			before := calls.Load()
			must(svc.processAuditJob(context.Background(), jobID, tid, uid, model.JobQueueKindInteractive))
			if calls.Load() != before+1 {
				t.Fatal("只能调用一次复核，不能重跑初审/结构化")
			}
			job, err := logRepo.GetByID(c, jobID)
			must(err)
			var record model.AIRecheckRecord
			must(json.Unmarshal(job.AIRecheck, &record))
			if job.Status != model.JobStatusCompleted || record.Changed != entry.changed || (record.Status == "failed") != entry.failed || !record.RetryOnly || record.SourceLogID != source.ID.String() {
				t.Fatalf("复核状态不符：%+v", record)
			}
			if !entry.changed && (job.Score != source.Score || job.Recommendation != source.Recommendation) {
				t.Fatal("失败或沿用时必须保留初审")
			}
			shared, err := svc.auditSnapshotRepo.GetByProcessIDAndChannel(c, source.ProcessID, model.AuditSnapshotChannelEmbed)
			must(err)
			if shared == nil || shared.LatestValidLogID != jobID {
				t.Fatal("应更新对应嵌入渠道快照")
			}
			unchanged, err := logRepo.GetByID(c, source.ID)
			must(err)
			if string(unchanged.AIRecheck) != string(source.AIRecheck) {
				t.Fatal("原审核记录不能被覆盖")
			}
			// 等待统一调用入口的异步日志，确认仅记录 recheck 并关联新任务。
			deadline := time.Now().Add(3 * time.Second)
			var count int64
			for time.Now().Before(deadline) {
				must(db.Model(&model.TenantLLMMessageLog{}).Where("business_log_id = ? AND call_type = ?", jobID, "recheck").Count(&count).Error)
				if count == 1 {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if count != 1 {
				t.Fatal("复核调用必须落库")
			}
		})
	}
}
