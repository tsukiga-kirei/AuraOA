package service

import (
	"strings"
	"testing"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/crypto"
)

func TestPrepareAgentModelConfig(t *testing.T) {
	if err := crypto.SetKey(strings.Repeat("a", 32)); err != nil {
		t.Fatal(err)
	}
	const secret = "test-agent-api-key"
	encrypted, err := crypto.Encrypt(secret)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"primary", "fallback"} {
		t.Run(name, func(t *testing.T) {
			stored := &model.AIModelConfig{APIKey: encrypted, ModelName: name}
			prepared, err := prepareAgentModelConfig(stored)
			if err != nil || prepared == nil || prepared.APIKey != secret || prepared.ModelName != name {
				t.Fatal("调用配置未正确解密或丢失模型信息")
			}
			if stored.APIKey != encrypted {
				t.Fatal("解密修改了原始存储配置")
			}
		})
	}
	t.Run("local", func(t *testing.T) {
		cfg, err := prepareAgentModelConfig(&model.AIModelConfig{ModelName: "local"})
		if err != nil || cfg == nil || cfg.APIKey != "" {
			t.Fatal("无密钥的本地模型应正常通过")
		}
	})
	t.Run("invalid", func(t *testing.T) {
		cfg, err := prepareAgentModelConfig(&model.AIModelConfig{APIKey: secret})
		if err == nil || cfg != nil {
			t.Fatal("解密失败时不得返回可调用配置")
		}
		if strings.Contains(err.Error(), secret) {
			t.Fatal("错误信息不得包含密钥")
		}
	})
	t.Run("absent", func(t *testing.T) {
		cfg, err := prepareAgentModelConfig(nil)
		if err != nil || cfg != nil {
			t.Fatal("未配置备用模型应正常通过")
		}
	})
}
