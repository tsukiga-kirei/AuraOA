package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.apache.poi.hwpf.extractor.WordExtractor;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/**
 * 使用 Apache POI HWPF 提取 Word 97-2003 文本。
 */
@Component
public class DocDocumentParser implements DocumentFormatParser {

    private final ParserProperties properties;

    public DocDocumentParser(ParserProperties properties) {
        this.properties = properties;
    }

    @Override
    public String fileType() {
        return "doc";
    }

    @Override
    public ParseResult parse(Path input) {
        List<String> warnings = new ArrayList<>();
        try (InputStream stream = Files.newInputStream(input);
             WordExtractor extractor = new WordExtractor(stream)) {
            String content = TextSupport.limit(extractor.getText(), properties.maxOutputChars(), warnings);
            boolean hasText = !content.isBlank();
            if (!hasText) {
                warnings.add("未从 DOC 中提取到可用文本，建议转换为 PDF 后进行版面识别");
            }
            return new ParseResult(
                    "apache-poi-hwpf",
                    fileType(),
                    content,
                    hasText,
                    false,
                    null,
                    List.copyOf(warnings)
            );
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("DOC 文件解析失败", exception);
        }
    }
}
