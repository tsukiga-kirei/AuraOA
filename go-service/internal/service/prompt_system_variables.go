package service

import (
	"strings"
	"time"

	"auraoa/go-service/internal/pkg/apptime"
)

var weekdayNamesCN = []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}

// replaceSystemPromptVariables 将用户提示词中的系统变量占位符替换为当前时间信息。
// 使用应用配置的时区（默认 Asia/Shanghai）。
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
	return prompt
}

func normalizeWeekday(day time.Weekday) int {
	if day < 0 || int(day) >= len(weekdayNamesCN) {
		return 0
	}
	return int(day)
}
