package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/**
 * 使用 Apache POI XSSF 将 Excel 工作表转为 Markdown。
 */
@Component
public class XlsxDocumentParser implements DocumentFormatParser {

    private final ParserProperties properties;

    public XlsxDocumentParser(ParserProperties properties) {
        this.properties = properties;
    }

    @Override
    public String fileType() {
        return "xlsx";
    }

    @Override
    public ParseResult parse(Path input) {
        List<String> warnings = new ArrayList<>();
        try (InputStream stream = Files.newInputStream(input);
             XSSFWorkbook workbook = new XSSFWorkbook(stream)) {
            String content = WorkbookTextExtractor.extract(workbook, properties, warnings);
            boolean hasText = !content.isBlank();
            if (!hasText) {
                warnings.add("XLSX 中没有可输出的单元格内容");
            }
            return new ParseResult("apache-poi-xssf", fileType(), content, hasText, false, null, List.copyOf(warnings));
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("XLSX 文件解析失败", exception);
        }
    }
}
