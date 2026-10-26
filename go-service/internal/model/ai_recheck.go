package model

import "encoding/json"

// AIRecheckConfig 是独立于审核尺度、随执行配置版本保存的可选第三阶段设置。
type AIRecheckConfig struct {
	Enabled      bool   `json:"enabled"`
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
}

// AIRecheckRecord 保留初审和最终结果，调用失败不等于业务不通过。
type AIRecheckRecord struct {
	Status         string          `json:"status"`
	Changed        bool            `json:"changed"`
	Reason         string          `json:"reason"`
	Error          string          `json:"error,omitempty"`
	DurationMs     int64           `json:"duration_ms"`
	OriginalResult json.RawMessage `json:"original_result"`
	FinalResult    json.RawMessage `json:"final_result,omitempty"`
}
