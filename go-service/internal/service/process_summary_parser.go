package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"auraoa/go-service/internal/model"
)

type summaryBlockPayload struct {
	BlockID string   `json:"block_id"`
	Title   string   `json:"title"`
	Content string   `json:"content"`
	Summary string   `json:"summary"`
	Text    string   `json:"text"`
	Points  []string `json:"points"`
}

type summaryBlocksPayload struct {
	Blocks []summaryBlockPayload `json:"blocks"`
}

// ParseSummaryBlockResult 宽松解析单块总结。解析失败时返回 raw 兜底内容和错误信息。
func ParseSummaryBlockResult(raw string, block model.SummaryBlockConfig) (model.ProcessSummaryBlockResult, error) {
	result := model.ProcessSummaryBlockResult{
		BlockID: block.ID,
		Title:   block.Title,
		Points:  []string{},
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		result.Content = "（模型未返回总结内容）"
		return result, fmt.Errorf("模型未返回内容")
	}

	cleaned := cleanJSONResponse(trimmed)
	var single summaryBlockPayload
	if err := json.Unmarshal([]byte(cleaned), &single); err == nil {
		result.Content = firstNonEmpty(single.Content, single.Summary, single.Text)
		if strings.TrimSpace(single.BlockID) != "" {
			result.BlockID = single.BlockID
		}
		if strings.TrimSpace(single.Title) != "" {
			result.Title = single.Title
		}
		result.Points = coalesceStringSlice(single.Points)
		if strings.TrimSpace(result.Content) != "" {
			return result, nil
		}
	}

	var multi summaryBlocksPayload
	if err := json.Unmarshal([]byte(cleaned), &multi); err == nil && len(multi.Blocks) > 0 {
		first := multi.Blocks[0]
		result.Content = firstNonEmpty(first.Content, first.Summary, first.Text)
		if strings.TrimSpace(first.BlockID) != "" {
			result.BlockID = first.BlockID
		}
		if strings.TrimSpace(first.Title) != "" {
			result.Title = first.Title
		}
		result.Points = coalesceStringSlice(first.Points)
		if strings.TrimSpace(result.Content) != "" {
			return result, nil
		}
	}

	result.Content = trimmed
	return result, fmt.Errorf("总结块 JSON 解析失败，已使用原始模型输出兜底")
}
