package ai

import (
	"fmt"

	"oa-smart-audit/go-service/internal/model"
)

// NewAIModelCaller 根据 deploy_type 创建对应的 AI 模型调用器实例。
// 当前支持: "local"（Xinference 本地部署）、"cloud"（阿里云百炼）
func NewAIModelCaller(cfg *model.AIModelConfig) (AIModelCaller, error) {
	switch cfg.DeployType {
	case "local":
		return NewXinferenceCaller(cfg)
	case "cloud":
		return NewAliyunBailianCaller(cfg)
	default:
		return nil, fmt.Errorf("不支持的 AI 部署类型: %s", cfg.DeployType)
	}
}
