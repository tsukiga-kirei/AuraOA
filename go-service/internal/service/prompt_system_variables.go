package service

import (
	"strings"
	"time"

	"auraoa/go-service/internal/pkg/apptime"
)

var weekdayNamesCN = []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

// replaceSystemPromptVariables 将用户提示词中的系统变量占位符替换为当前时间信息
// 与智能体卡片协议正文。时间使用应用配置的时区（默认 Asia/Shanghai）。
func replaceSystemPromptVariables(prompt string) string {
	return replaceSystemPromptVariablesAt(prompt, apptime.Now())
}

func replaceSystemPromptVariablesAt(prompt string, now time.Time) string {
	now = now.In(apptime.Location())
	weekday := weekdayNamesCN[normalizeWeekday(now.Weekday())]

	prompt = strings.ReplaceAll(prompt, "{{current_date}}", now.Format("2006-01-02"))
	prompt = strings.ReplaceAll(prompt, "{{current_time}}", now.Format("15:04:05"))
	prompt = strings.ReplaceAll(prompt, "{{current_datetime}}", now.Format("2006-01-02 15:04:05"))
	prompt = strings.ReplaceAll(prompt, "{{weekday}}", weekday)
	prompt = replaceUIDisplayProtocolVariables(prompt)
	return prompt
}

// replaceUIDisplayProtocolVariables 展开智能体模块的 Generative UI 卡片协议变量。
// 协议正文为静态文本，与时间无关，单独抽出便于其它提示词构建链路复用。
func replaceUIDisplayProtocolVariables(prompt string) string {
	prompt = strings.ReplaceAll(prompt, SystemVariableFormConfirmProtocol, uiFormConfirmProtocol)
	prompt = strings.ReplaceAll(prompt, SystemVariableChartProtocol, uiChartProtocol)
	return prompt
}

func normalizeWeekday(day time.Weekday) int {
	if day < 0 || int(day) >= len(weekdayNamesCN) {
		return 0
	}
	return int(day)
}
