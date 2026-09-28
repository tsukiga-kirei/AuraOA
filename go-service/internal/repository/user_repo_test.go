package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
)

type loginFailSQLRecorder struct {
	logger.Interface
	statements []string
}

func (r *loginFailSQLRecorder) Trace(_ context.Context, _ time.Time, sql func() (string, int64), _ error) {
	query, _ := sql()
	r.statements = append(r.statements, query)
}

// TestUpdateLoginFailAfterLockExpires 校验过期后的失败次数和持久化状态重新开始。
func TestUpdateLoginFailAfterLockExpires(t *testing.T) {
	recorder := &loginFailSQLRecorder{Interface: logger.Default.LogMode(logger.Silent)}
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=127.0.0.1 user=test dbname=test", PreferSimpleProtocol: true}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, SkipDefaultTransaction: true, Logger: recorder})
	if err != nil {
		t.Fatal(err)
	}

	expired := apptime.Now().Add(-time.Minute)
	user := &model.User{ID: uuid.New(), LoginFailCount: 5, LockedUntil: &expired}
	repo := NewUserRepo(db)
	for attempt := 1; attempt <= 5; attempt++ {
		if err := repo.UpdateLoginFail(user); err != nil {
			t.Fatal(err)
		}
		if user.LoginFailCount != attempt {
			t.Fatalf("第 %d 次失败后计数为 %d", attempt, user.LoginFailCount)
		}
		if attempt < 5 && user.LockedUntil != nil {
			t.Fatalf("第 %d 次失败时不应重新锁定", attempt)
		}
	}
	if user.LockedUntil == nil || !user.LockedUntil.After(apptime.Now()) {
		t.Fatal("新一轮第 5 次失败后应重新锁定")
	}
	if len(recorder.statements) != 5 || !strings.Contains(recorder.statements[0], `"locked_until"=NULL`) {
		t.Fatalf("过期后的首次失败应清除数据库锁定时间：%v", recorder.statements)
	}
}
