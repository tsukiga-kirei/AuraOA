package sanitize

import (
	"regexp"
	"strings"
)

// 正则表达式预编译
var (
	// 身份证号：18位（17位数字+校验码）或15位数字
	idCardRegex = regexp.MustCompile(`\b(\d{6})\d{8,11}(\w{4})\b`)
	// 手机号：1开头的11位数字
	phoneRegex = regexp.MustCompile(`\b(1\d{2})\d{4}(\d{4})\b`)
	// 银行卡号：16-19位数字
	bankCardRegex = regexp.MustCompile(`\b\d{12,15}(\d{4})\b`)
	// 薪资金额：常见格式如 ¥12345.67、12345元、12345.00
	salaryRegex = regexp.MustCompile(`[¥￥]?\s*(\d+(?:\.\d{1,2})?)\s*元?`)
)

// MaskIDCard 身份证号脱敏：保留前3后4，中间用 * 替换。
// 示例：110101199001011234 → 110***********1234
func MaskIDCard(idCard string) string {
	idCard = strings.TrimSpace(idCard)
	if len(idCard) < 8 {
		return idCard
	}
	prefix := idCard[:3]
	suffix := idCard[len(idCard)-4:]
	masked := prefix + strings.Repeat("*", len(idCard)-7) + suffix
	return masked
}

// MaskPhone 手机号脱敏：保留前3后4，中间用 * 替换。
// 示例：13812345678 → 138****5678
func MaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) < 8 {
		return phone
	}
	prefix := phone[:3]
	suffix := phone[len(phone)-4:]
	masked := prefix + strings.Repeat("*", len(phone)-7) + suffix
	return masked
}

// MaskBankCard 银行卡号脱敏：仅保留后4位，其余用 * 替换。
// 示例：6222021234567890 → ************7890
func MaskBankCard(card string) string {
	card = strings.TrimSpace(card)
	if len(card) < 5 {
		return card
	}
	suffix := card[len(card)-4:]
	masked := strings.Repeat("*", len(card)-4) + suffix
	return masked
}

// MaskSalary 薪资金额脱敏：替换为区间描述。
// 示例：15000.00 → [10000-20000]
func MaskSalary(amount float64) string {
	switch {
	case amount < 0:
		return "[金额异常]"
	case amount < 3000:
		return "[0-3000]"
	case amount < 5000:
		return "[3000-5000]"
	case amount < 8000:
		return "[5000-8000]"
	case amount < 10000:
		return "[8000-10000]"
	case amount < 20000:
		return "[10000-20000]"
	case amount < 50000:
		return "[20000-50000]"
	default:
		return "[50000以上]"
	}
}

// SanitizeText 对文本中的敏感信息进行批量脱敏。
// 依次处理身份证号、手机号、银行卡号。
func SanitizeText(text string) string {
	// 身份证号脱敏
	text = idCardRegex.ReplaceAllStringFunc(text, func(match string) string {
		return MaskIDCard(match)
	})
	// 手机号脱敏
	text = phoneRegex.ReplaceAllStringFunc(text, func(match string) string {
		return MaskPhone(match)
	})
	// 银行卡号脱敏
	text = bankCardRegex.ReplaceAllStringFunc(text, func(match string) string {
		return MaskBankCard(match)
	})
	return text
}
