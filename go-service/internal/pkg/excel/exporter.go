package excel

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ExportConfig 描述一次导出任务的配置。
type ExportConfig struct {
	// ExportType 导出类型，决定列头（若 CustomHeaders 为空则生效）。
	ExportType ExportType
	// CustomHeaders 自定义表头切片，非空时优先使用。
	CustomHeaders []string
	// Locale 用户语言，用于列头和枚举值的国际化。
	Locale Locale
	// SheetName 工作表名称，可选，默认为 "Sheet1"。
	SheetName string
	// Filename 下载文件名，不含扩展名。
	Filename string
}

// WriteExcel 将 rows 数据按 config 配置写入 Excel 并通过 gin.Context 流式响应。
// rows 为 [][]string，每个内层切片对应一行，顺序与列头一致。
// 特殊样式处理：
//  - 表头：沉稳商务深海蓝背景（#1F4E79），纯白加粗字体，上下居中，设置 30pt 行高；
//  - 数据行：浅灰与纯白斑马纹交替，细边框，上下居中，多行文本自动换行；
//  - 首行冻结：滚动浏览时表头始终置顶；
//  - 自适应列宽：按中英文混合字符宽度动态调整列宽，避免内容截断或过宽。
func WriteExcel(c *gin.Context, config ExportConfig, rows [][]string) error {
	sheetName := config.SheetName
	if sheetName == "" {
		sheetName = "Sheet1"
	}

	f := excelize.NewFile()
	defer f.Close()

	// excelize 默认创建名为 "Sheet1" 的工作表；若需要不同名称则重命名。
	defaultSheet := f.GetSheetName(0)
	if sheetName != defaultSheet {
		_ = f.SetSheetName(defaultSheet, sheetName)
	}

	// 1. 确定表头
	var headers []string
	if len(config.CustomHeaders) > 0 {
		headers = config.CustomHeaders
	} else {
		headers = ColHeaders(config.ExportType, config.Locale)
	}

	// 2. 创建样式
	// 2.1 表头样式：深海蓝背景 #1F4E79，白色粗体，水平垂直居中，细边框
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "PingFang SC",
			Bold:   true,
			Size:   11,
			Color:  "#FFFFFF",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#1F4E79"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
			WrapText:   true,
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#3A608F", Style: 1},
			{Type: "bottom", Color: "#3A608F", Style: 1},
			{Type: "left", Color: "#3A608F", Style: 1},
			{Type: "right", Color: "#3A608F", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("excel: failed to create header style: %w", err)
	}

	// 2.2 数据行偶数样式：纯白底色 #FFFFFF
	evenRowStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "PingFang SC",
			Size:   10,
			Color:  "#1F2937",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#FFFFFF"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: true,
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#E2E8F0", Style: 1},
			{Type: "bottom", Color: "#E2E8F0", Style: 1},
			{Type: "left", Color: "#E2E8F0", Style: 1},
			{Type: "right", Color: "#E2E8F0", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("excel: failed to create even row style: %w", err)
	}

	// 2.3 数据行奇数样式：极淡蓝灰底色 #F8FAFC（斑马纹）
	oddRowStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Family: "PingFang SC",
			Size:   10,
			Color:  "#1F2937",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#F8FAFC"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Vertical: "center",
			WrapText: true,
		},
		Border: []excelize.Border{
			{Type: "top", Color: "#E2E8F0", Style: 1},
			{Type: "bottom", Color: "#E2E8F0", Style: 1},
			{Type: "left", Color: "#E2E8F0", Style: 1},
			{Type: "right", Color: "#E2E8F0", Style: 1},
		},
	})
	if err != nil {
		return fmt.Errorf("excel: failed to create odd row style: %w", err)
	}

	// 3. 写入表头（第 1 行）并设置行高与样式
	_ = f.SetRowHeight(sheetName, 1, 30)
	colMaxWidths := make([]int, len(headers))

	for colIdx, header := range headers {
		cell, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err != nil {
			return fmt.Errorf("excel: failed to compute header cell name: %w", err)
		}
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("excel: failed to set header cell %s: %w", cell, err)
		}
		_ = f.SetCellStyle(sheetName, cell, cell, headerStyle)
		colMaxWidths[colIdx] = cellDisplayWidth(header)
	}

	// 4. 写入数据行（从第 2 行开始）
	for rowIdx, row := range rows {
		excelRowNum := rowIdx + 2
		// 默认行高 22；若单行中有换行（多行文本），稍微调高
		hasNewline := false
		for _, val := range row {
			if strings.Contains(val, "\n") {
				hasNewline = true
				break
			}
		}
		if hasNewline {
			_ = f.SetRowHeight(sheetName, excelRowNum, 45)
		} else {
			_ = f.SetRowHeight(sheetName, excelRowNum, 22)
		}

		currentStyle := evenRowStyle
		if rowIdx%2 == 1 {
			currentStyle = oddRowStyle
		}

		for colIdx, value := range row {
			if colIdx >= len(headers) {
				continue
			}
			cell, err := excelize.CoordinatesToCellName(colIdx+1, excelRowNum)
			if err != nil {
				return fmt.Errorf("excel: failed to compute data cell name: %w", err)
			}
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				return fmt.Errorf("excel: failed to set data cell %s: %w", cell, err)
			}
			_ = f.SetCellStyle(sheetName, cell, cell, currentStyle)

			w := cellDisplayWidth(value)
			if w > colMaxWidths[colIdx] {
				colMaxWidths[colIdx] = w
			}
		}
	}

	// 5. 冻结表头（第 1 行）
	_ = f.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       true,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// 6. 自动调整列宽
	for colIdx := range headers {
		colName, err := excelize.ColumnNumberToName(colIdx + 1)
		if err != nil {
			continue
		}
		width := float64(colMaxWidths[colIdx]) + 4
		if width < 12 {
			width = 12
		}
		if width > 65 {
			width = 65
		}
		_ = f.SetColWidth(sheetName, colName, colName, width)
	}

	// 7. 设置响应头
	encodedFilename := url.PathEscape(config.Filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s.xlsx", encodedFilename))

	// 将 Excel 文件写入响应流
	if err := f.Write(c.Writer); err != nil {
		return fmt.Errorf("excel: failed to write excel to response: %w", err)
	}

	return nil
}

// cellDisplayWidth 计算字符串在 Excel 中的显示字符宽度（ASCII 计 1，中日韩等多字节计 2，遇到换行取最大单行宽度）
func cellDisplayWidth(s string) int {
	if s == "" {
		return 0
	}
	maxW := 0
	lines := strings.Split(s, "\n")
	for _, line := range lines {
		w := 0
		for _, r := range line {
			if r > 127 {
				w += 2
			} else {
				w += 1
			}
		}
		if w > maxW {
			maxW = w
		}
	}
	return maxW
}

