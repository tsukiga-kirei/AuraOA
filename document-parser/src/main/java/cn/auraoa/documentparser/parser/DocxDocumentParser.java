package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.apache.poi.xwpf.extractor.XWPFWordExtractor;
import org.apache.poi.xwpf.usermodel.XWPFDocument;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/**
 * 使用 Apache POI XWPF 提取 DOCX 的段落与表格文本。
 */
@Component
public class DocxDocumentParser implements DocumentFormatParser {

    private final ParserProperties properties;

    public DocxDocumentParser(ParserProperties properties) {
        this.properties = properties;
    }

    @Override
    public String fileType() {
        return "docx";
    }

    @Override
    public ParseResult parse(Path input) {
        List<String> warnings = new ArrayList<>();
        try (InputStream stream = Files.newInputStream(input);
             XWPFDocument document = new XWPFDocument(stream);
             XWPFWordExtractor extractor = new XWPFWordExtractor(document)) {
            String content = TextSupport.limit(extractor.getText(), properties.maxOutputChars(), warnings);
            boolean hasText = !content.isBlank();
            if (!hasText) {
                warnings.add("未从 DOCX 中提取到可用文本");
            }
            return new ParseResult("apache-poi-xwpf", fileType(), content, hasText, false, null, List.copyOf(warnings));
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("DOCX 文件解析失败", exception);
        }
    }
}
