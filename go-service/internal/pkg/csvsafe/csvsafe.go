// Package csvsafe 提供 CSV 导出的公式注入防护。
package csvsafe

// Cell 将可能被电子表格软件解释为公式的文本转为纯文本。
// 覆盖 Excel / WPS 常见的公式起始字符：=、+、-、@，以及制表符和回车。
// 单引号是电子表格的文本标记，不会改变展示值。
func Cell(value string) string {
	if value == "" {
		return value
	}
	switch value[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + value
	default:
		return value
	}
}
