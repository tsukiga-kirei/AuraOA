package service

import (
	"auraoa/go-service/internal/model"
	"github.com/google/uuid"
	"testing"
)

func TestAuthorizeEmbedAuditTask(t *testing.T) {
	tenant, otherTenant, owner, otherUser := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	tests := []struct {
		name, trigger, detail           string
		logTenant, configTenant, viewer uuid.UUID
		enabled                         bool
		status                          string
		wantAllowed, wantResolve        bool
	}{
		{"标准自动审核无前台登录身份", model.AuditTriggerEmbedAuto, "", tenant, tenant, uuid.Nil, true, "active", true, false},
		{"标准手动审核无前台登录身份", model.AuditTriggerEmbedManual, "", tenant, tenant, uuid.Nil, true, "active", true, false},
		{"跨租户任务", model.AuditTriggerEmbedAuto, "", otherTenant, tenant, owner, true, "active", false, false},
		{"跨租户配置", model.AuditTriggerEmbedAuto, "", tenant, otherTenant, owner, true, "active", false, false},
		{"嵌入已禁用", model.AuditTriggerEmbedAuto, "", tenant, tenant, owner, false, "active", false, false},
		{"流程配置已停用", model.AuditTriggerEmbedAuto, "", tenant, tenant, owner, true, "inactive", false, false},
		{"本人个人定制审核", model.AuditTriggerWorkbenchManual, "personal_embed_manual", tenant, tenant, owner, true, "active", true, true},
		{"他人个人定制审核", model.AuditTriggerWorkbenchManual, "personal_embed_manual", tenant, tenant, otherUser, true, "active", false, true},
		{"个人任务缺少OA身份不能借用执行账号", model.AuditTriggerWorkbenchManual, "personal_embed_manual", tenant, tenant, uuid.Nil, true, "active", false, true},
		{"历史嵌入来源个人记录仍检查所有者", model.AuditTriggerEmbedManual, "personal_embed_manual", tenant, tenant, otherUser, true, "active", false, true},
		{"可读取本人工作台复用任务", model.AuditTriggerWorkbenchManual, "", tenant, tenant, owner, true, "active", true, true},
		{"不能读取其他人系统内任务", model.AuditTriggerWorkbenchManual, "", tenant, tenant, otherUser, true, "active", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			log := &model.AuditLog{TenantID: tt.logTenant, UserID: owner, TriggerSource: tt.trigger, TriggerDetail: tt.detail}
			config := &model.ProcessAuditConfig{TenantID: tt.configTenant, Status: tt.status, EmbedEnabled: tt.enabled}
			err := authorizeEmbedAuditTask(tenant, log, config, func() (uuid.UUID, error) { called = true; return tt.viewer, nil })
			if (err == nil) != tt.wantAllowed || called != tt.wantResolve {
				t.Fatalf("授权错误：err=%v, resolve=%v", err, called)
			}
		})
	}
}
