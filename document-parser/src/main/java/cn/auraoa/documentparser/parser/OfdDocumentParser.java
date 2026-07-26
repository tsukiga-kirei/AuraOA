package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentLimitExceededException;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.ofdrw.converter.export.TextExporter;
import org.springframework.stereotype.Component;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/**
 * 使用 OFDRW 提取 OFD 文字层；无文字层时指示调用方转 PDF 后交给视觉解析器。
 */
@Component
public class OfdDocumentParser implements DocumentFormatParser {

    private final ParserProperties properties;
    private final OfdArchiveValidator archiveValidator;

    public OfdDocumentParser(ParserProperties properties, OfdArchiveValidator archiveValidator) {
        this.properties = properties;
        this.archiveValidator = archiveValidator;
    }

    @Override
    public String fileType() {
        return "ofd";
    }

    @Override
    public ParseResult parse(Path input) {
        archiveValidator.validate(input);
        long maximumOutputBytes = Math.max(4096L, (long) properties.maxOutputChars() * 4L);
        try (InputStream source = Files.newInputStream(input);
             ByteArrayOutputStream buffer = new ByteArrayOutputStream();
             OutputStream limited = new LimitedOutputStream(buffer, maximumOutputBytes);
             TextExporter exporter = new TextExporter(source, limited)) {
            exporter.export();
            return buildResult(buffer.toString(StandardCharsets.UTF_8));
        } catch (DocumentLimitExceededException exception) {
            throw exception;
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("OFD 文件文字层解析失败", exception);
        }
    }

    ParseResult buildResult(String extractedText) {
        List<String> warnings = new ArrayList<>();
        String content = TextSupport.limit(extractedText, properties.maxOutputChars(), warnings);
        boolean hasText = !content.isBlank();
        if (!hasText) {
            warnings.add("OFD 未检测到可用文字层，需要转为 PDF 后进行版面识别");
        }
        return new ParseResult(
                "ofdrw",
                fileType(),
                content,
                hasText,
                !hasText,
                hasText ? null : "pdf",
                List.copyOf(warnings)
        );
    }
}
