package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import org.apache.poi.ss.usermodel.Cell;
import org.apache.poi.ss.usermodel.DataFormatter;
import org.apache.poi.ss.usermodel.Row;
import org.apache.poi.ss.usermodel.Sheet;
import org.apache.poi.ss.usermodel.Workbook;

import java.util.List;
import java.util.Locale;

/**
 * 将新旧 Excel 工作簿统一转换为 Markdown 表格。
 */
final class WorkbookTextExtractor {

    private WorkbookTextExtractor() {
    }

    static String extract(Workbook workbook, ParserProperties properties, List<String> warnings) {
        StringBuilder output = new StringBuilder();
        DataFormatter formatter = new DataFormatter(Locale.SIMPLIFIED_CHINESE);
        int sheetCount = Math.min(workbook.getNumberOfSheets(), properties.maxXlsSheets());
        if (workbook.getNumberOfSheets() > sheetCount) {
            warnings.add("工作表数量超过上限，结果仅包含前 " + sheetCount + " 个工作表");
        }

        boolean outputLimitReached = false;
        for (int sheetIndex = 0; sheetIndex < sheetCount && !outputLimitReached; sheetIndex++) {
            Sheet sheet = workbook.getSheetAt(sheetIndex);
            append(output, "## 工作表：" + TextSupport.normalize(sheet.getSheetName()) + "\n\n", properties);

            int firstRow = Math.max(sheet.getFirstRowNum(), 0);
            int lastRow = Math.min(sheet.getLastRowNum(), firstRow + properties.maxXlsRowsPerSheet() - 1);
            if (sheet.getLastRowNum() > lastRow) {
                warnings.add("工作表“" + TextSupport.normalize(sheet.getSheetName()) + "”行数超过上限，结果已截断");
            }

            for (int rowIndex = firstRow; rowIndex <= lastRow; rowIndex++) {
                Row row = sheet.getRow(rowIndex);
                int cellCount = determineCellCount(row, properties);
                append(output, "|", properties);
                for (int cellIndex = 0; cellIndex < cellCount; cellIndex++) {
                    Cell cell = row == null ? null : row.getCell(cellIndex, Row.MissingCellPolicy.RETURN_BLANK_AS_NULL);
                    String value = cell == null ? "" : formatter.formatCellValue(cell);
                    append(output, " " + TextSupport.markdownCell(value) + " |", properties);
                }
                append(output, "\n", properties);

                if (rowIndex == firstRow && cellCount > 0) {
                    append(output, "|", properties);
                    for (int cellIndex = 0; cellIndex < cellCount; cellIndex++) {
                        append(output, " --- |", properties);
                    }
                    append(output, "\n", properties);
                }

                if (output.length() >= properties.maxOutputChars()) {
                    outputLimitReached = true;
                    warnings.add("解析文本超过长度上限，结果已截断");
                    break;
                }
            }
            append(output, "\n", properties);
        }
        return output.toString().strip();
    }

    private static int determineCellCount(Row row, ParserProperties properties) {
        if (row == null || row.getLastCellNum() < 0) {
            return 0;
        }
        return Math.min(row.getLastCellNum(), properties.maxXlsCellsPerRow());
    }

    private static void append(StringBuilder output, String value, ParserProperties properties) {
        int remaining = properties.maxOutputChars() - output.length();
        if (remaining > 0) {
            output.append(value, 0, Math.min(value.length(), remaining));
        }
    }
}
