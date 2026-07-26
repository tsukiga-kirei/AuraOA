package cn.auraoa.documentparser.service;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.dto.ParseResponse;
import cn.auraoa.documentparser.exception.DocumentLimitExceededException;
import cn.auraoa.documentparser.exception.InvalidDocumentRequestException;
import cn.auraoa.documentparser.exception.UnsupportedDocumentTypeException;
import cn.auraoa.documentparser.parser.DocumentFormatParser;
import cn.auraoa.documentparser.parser.ParseResult;
import cn.auraoa.documentparser.util.TempWorkspace;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.nio.file.Path;
import java.util.HashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

/**
 * 校验上传并将文件路由到对应格式解析器。
 */
@Service
public class DocumentParseService {

    private final ParserProperties properties;
    private final Map<String, DocumentFormatParser> parsers;

    public DocumentParseService(ParserProperties properties, List<DocumentFormatParser> parserList) {
        this.properties = properties;
        this.parsers = new HashMap<>();
        for (DocumentFormatParser parser : parserList) {
            DocumentFormatParser previous = parsers.put(parser.fileType(), parser);
            if (previous != null) {
                throw new IllegalStateException("文档格式解析器重复：" + parser.fileType());
            }
        }
    }

    public ParseResponse parse(MultipartFile file) {
        String fileType = validateAndResolveFileType(file);
        DocumentFormatParser parser = parsers.get(fileType);
        if (parser == null) {
            throw new UnsupportedDocumentTypeException("不支持的文件类型：" + fileType);
        }

        try (TempWorkspace workspace = TempWorkspace.create()) {
            Path input = workspace.resolve("input." + fileType);
            file.transferTo(input);
            ParseResult result = parser.parse(input);
            return new ParseResponse(
                    result.parser(),
                    result.fileType(),
                    result.content(),
                    result.hasTextLayer(),
                    result.fallbackRequired(),
                    result.fallbackFormat(),
                    result.warnings()
            );
        } catch (IOException exception) {
            throw new InvalidDocumentRequestException("读取上传文件失败");
        }
    }

    public String validateAndResolveFileType(MultipartFile file) {
        if (file == null || file.isEmpty()) {
            throw new InvalidDocumentRequestException("必须上传非空文件");
        }
        if (file.getSize() > properties.maxUploadSize().toBytes()) {
            throw new DocumentLimitExceededException("上传文件超过大小限制");
        }
        String originalFilename = file.getOriginalFilename();
        if (originalFilename == null || originalFilename.isBlank()) {
            throw new InvalidDocumentRequestException("上传文件缺少文件名");
        }
        String normalizedFilename = originalFilename.replace('\\', '/');
        String filename = normalizedFilename.substring(normalizedFilename.lastIndexOf('/') + 1);
        if (filename.isBlank() || filename.indexOf('\0') >= 0) {
            throw new InvalidDocumentRequestException("上传文件名无效");
        }
        int extensionSeparator = filename.lastIndexOf('.');
        if (extensionSeparator < 0 || extensionSeparator == filename.length() - 1) {
            throw new UnsupportedDocumentTypeException("无法识别文件类型");
        }
        String fileType = filename.substring(extensionSeparator + 1).toLowerCase(Locale.ROOT);
        if (!parsers.containsKey(fileType)) {
            throw new UnsupportedDocumentTypeException("不支持的文件类型：" + fileType);
        }
        return fileType;
    }
}
