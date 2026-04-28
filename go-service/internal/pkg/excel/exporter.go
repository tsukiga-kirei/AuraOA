package excel

import (
	"fmt"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

// ExportConfig 描述一次导出任务的配置。
type ExportConfig struct {
	// ExportType 导出类型，决定列头。
	ExportType ExportType
	// Locale 用户语言，用于列头和枚举值的国际化。
	Locale Locale
	// SheetName 工作表名称，可选，默认为 "Sheet1"。
	SheetName string
	// Filename 下载文件名，不含扩展名。
	Filename string
}

// WriteExcel 将 rows 数据按 config 配置写入 Excel 并通过 gin.Context 流式响应。
// rows 为 [][]string，每个内层切片对应一行，顺序与列头一致。
// 自动设置 Content-Type 为 xlsx MIME 类型，
// 并按 RFC 5987 UTF-8 编码设置 Content-Disposition 响应头。
// 若 config.SheetName 为空，默认使用 "Sheet1"。
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
		f.SetSheetName(defaultSheet, sheetName)
	}

	// 写入列头（第一行）
	headers := ColHeaders(config.ExportType, config.Locale)
	for col, header := range headers {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return fmt.Errorf("excel: failed to compute header cell name: %w", err)
		}
		if err := f.SetCellValue(sheetName, cell, header); err != nil {
			return fmt.Errorf("excel: failed to set header cell %s: %w", cell, err)
		}
	}

	// 写入数据行（从第二行开始）
	for rowIdx, row := range rows {
		for col, value := range row {
			cell, err := excelize.CoordinatesToCellName(col+1, rowIdx+2)
			if err != nil {
				return fmt.Errorf("excel: failed to compute data cell name: %w", err)
			}
			if err := f.SetCellValue(sheetName, cell, value); err != nil {
				return fmt.Errorf("excel: failed to set data cell %s: %w", cell, err)
			}
		}
	}

	// 设置响应头
	encodedFilename := url.PathEscape(config.Filename)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s.xlsx", encodedFilename))

	// 将 Excel 文件写入响应流
	if err := f.Write(c.Writer); err != nil {
		return fmt.Errorf("excel: failed to write excel to response: %w", err)
	}

	return nil
}
