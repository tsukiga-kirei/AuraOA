package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.apache.poi.hssf.usermodel.HSSFWorkbook;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

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
            String content = WorkbookTextExtractor.extract(workbook, properties, warnings);
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

}
