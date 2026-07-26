package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.apache.poi.hssf.usermodel.HSSFWorkbook;
import org.apache.poi.ss.usermodel.Cell;
import org.apache.poi.ss.usermodel.DataFormatter;
import org.apache.poi.ss.usermodel.Row;
import org.apache.poi.ss.usermodel.Sheet;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

/**
 * 使用 Apache POI HSSF 将 Excel 97-2003 工作表转为 Markdown。
 */
@Component
public class XlsDocumentParser implements DocumentFormatParser {

    private final ParserProperties properties;

    public XlsDocumentParser(ParserProperties properties) {
        this.properties = properties;
    }

    @Override
    public String fileType() {
        return "xls";
    }

    @Override
    public ParseResult parse(Path input) {
        List<String> warnings = new ArrayList<>();
        try (InputStream stream = Files.newInputStream(input);
             HSSFWorkbook workbook = new HSSFWorkbook(stream)) {
            String content = extractWorkbook(workbook, warnings);
            boolean hasText = !content.isBlank();
            if (!hasText) {
                warnings.add("XLS 中没有可输出的单元格内容");
            }
            return new ParseResult(
                    "apache-poi-hssf",
                    fileType(),
                    content,
                    hasText,
                    false,
                    null,
                    List.copyOf(warnings)
            );
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("XLS 文件解析失败", exception);
        }
    }

    private String extractWorkbook(HSSFWorkbook workbook, List<String> warnings) {
        StringBuilder output = new StringBuilder();
        DataFormatter formatter = new DataFormatter(Locale.SIMPLIFIED_CHINESE);
        int sheetCount = Math.min(workbook.getNumberOfSheets(), properties.maxXlsSheets());
        if (workbook.getNumberOfSheets() > sheetCount) {
            warnings.add("工作表数量超过上限，结果仅包含前 " + sheetCount + " 个工作表");
        }

        boolean outputLimitReached = false;
        for (int sheetIndex = 0; sheetIndex < sheetCount && !outputLimitReached; sheetIndex++) {
            Sheet sheet = workbook.getSheetAt(sheetIndex);
            append(output, "## 工作表：" + TextSupport.normalize(sheet.getSheetName()) + "\n\n");

            int firstRow = Math.max(sheet.getFirstRowNum(), 0);
            int lastRow = Math.min(sheet.getLastRowNum(),
                    firstRow + properties.maxXlsRowsPerSheet() - 1);
            if (sheet.getLastRowNum() > lastRow) {
                warnings.add("工作表“" + TextSupport.normalize(sheet.getSheetName()) + "”行数超过上限，结果已截断");
            }

            for (int rowIndex = firstRow; rowIndex <= lastRow; rowIndex++) {
                Row row = sheet.getRow(rowIndex);
                int cellCount = determineCellCount(row);
                append(output, "|");
                for (int cellIndex = 0; cellIndex < cellCount; cellIndex++) {
                    Cell cell = row == null ? null : row.getCell(cellIndex, Row.MissingCellPolicy.RETURN_BLANK_AS_NULL);
                    String value = cell == null ? "" : formatter.formatCellValue(cell);
                    append(output, " " + TextSupport.markdownCell(value) + " |");
                }
                append(output, "\n");

                if (rowIndex == firstRow && cellCount > 0) {
                    append(output, "|");
                    for (int cellIndex = 0; cellIndex < cellCount; cellIndex++) {
                        append(output, " --- |");
                    }
                    append(output, "\n");
                }

                if (output.length() >= properties.maxOutputChars()) {
                    outputLimitReached = true;
                    warnings.add("解析文本超过长度上限，结果已截断");
                    break;
                }
            }
            append(output, "\n");
        }

        String content = output.toString();
        if (content.length() > properties.maxOutputChars()) {
            content = content.substring(0, properties.maxOutputChars());
        }
        return content.strip();
    }

    private int determineCellCount(Row row) {
        if (row == null || row.getLastCellNum() < 0) {
            return 0;
        }
        return Math.min(row.getLastCellNum(), properties.maxXlsCellsPerRow());
    }

    private void append(StringBuilder output, String value) {
        if (output.length() < properties.maxOutputChars()) {
            output.append(value);
        }
    }
}
