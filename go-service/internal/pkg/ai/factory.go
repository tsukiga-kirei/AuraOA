package ai

import (
	"fmt"

	"oa-smart-audit/go-service/internal/model"
)

// providerEndpoints 云端 provider 的默认 API Endpoint。
// 如果 ai_model_configs.endpoint 有值则优先使用配置值。
var providerEndpoints = map[string]string{
	"aliyun_bailian": "https://dashscope.aliyuncs.com/compatible-mode/v1",
	"deepseek":       "https://api.deepseek.com/v1",
	"zhipu":          "https://open.bigmodel.cn/api/paas/v4",
	"openai":         "https://api.openai.com/v1",
}

// NewAIModelCaller 根据 provider 创建对应的 AI 模型调用器实例。
//
// 本地部署 (deploy_type=local):
//   - xinference, ollama, vllm → OpenAICompatCaller（需要 endpoint）
//
// 云端 API (deploy_type=cloud):
//   - aliyun_bailian, deepseek, zhipu, openai → OpenAICompatCaller（需要 api_key）
//   - azure_openai → OpenAICompatCaller（endpoint 格式特殊）
func NewAIModelCaller(cfg *model.AIModelConfig) (AIModelCaller, error) {
	switch cfg.Provider {
	// ── 本地部署 ──
	case "xinference", "ollama", "vllm":
		if cfg.Endpoint == "" {
			return nil, fmt.Errorf("本地部署 provider '%s' 需要配置 endpoint", cfg.Provider)
		}
		return NewOpenAICompatCaller(cfg)

	// ── 云端 API ──
	case "aliyun_bailian", "deepseek", "zhipu", "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("云端 provider '%s' 需要配置 API Key", cfg.Provider)
		}
		// 如果未配置 endpoint，使用默认值
		if cfg.Endpoint == "" {
			if defaultEP, ok := providerEndpoints[cfg.Provider]; ok {
				cfg.Endpoint = defaultEP
			}
		}
		return NewOpenAICompatCaller(cfg)

	case "azure_openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("Azure OpenAI 需要配置 API Key")
		}
		if cfg.Endpoint == "" {
			return nil, fmt.Errorf("Azure OpenAI 需要配置 endpoint")
		}
		return NewOpenAICompatCaller(cfg)

	default:
		return nil, fmt.Errorf("不支持的 AI provider: %s (deploy_type=%s)", cfg.Provider, cfg.DeployType)
	}
}
