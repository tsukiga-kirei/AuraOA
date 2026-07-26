package cn.auraoa.documentparser.service;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentLimitExceededException;
import cn.auraoa.documentparser.exception.DocumentParserException;
import cn.auraoa.documentparser.exception.UnsupportedDocumentTypeException;
import cn.auraoa.documentparser.parser.OfdArchiveValidator;
import cn.auraoa.documentparser.util.TempWorkspace;
import org.ofdrw.converter.export.OFDExporter;
import org.ofdrw.converter.export.PDFExporterPDFBox;
import org.springframework.stereotype.Service;
import org.springframework.web.multipart.MultipartFile;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;

/**
 * 将兼容格式转换为 PDF；第一阶段仅开放 OFD。
 */
@Service
public class PdfConversionService {

    private final ParserProperties properties;
    private final DocumentParseService parseService;
    private final OfdArchiveValidator archiveValidator;

    public PdfConversionService(
            ParserProperties properties,
            DocumentParseService parseService,
            OfdArchiveValidator archiveValidator
    ) {
        this.properties = properties;
        this.parseService = parseService;
        this.archiveValidator = archiveValidator;
    }

    public byte[] convert(MultipartFile file) {
        String fileType = parseService.validateAndResolveFileType(file);
        if (!"ofd".equals(fileType)) {
            throw new UnsupportedDocumentTypeException("当前仅支持 OFD 转 PDF");
        }

        try (TempWorkspace workspace = TempWorkspace.create()) {
            Path input = workspace.resolve("input.ofd");
            Path output = workspace.resolve("output.pdf");
            file.transferTo(input);
            archiveValidator.validate(input);
            try (OFDExporter exporter = new PDFExporterPDFBox(input, output)) {
                exporter.export();
            }
            long size = Files.size(output);
            if (size > properties.maxConvertedPdfSize().toBytes()) {
                throw new DocumentLimitExceededException("转换后的 PDF 超过大小限制");
            }
            return Files.readAllBytes(output);
        } catch (DocumentLimitExceededException | UnsupportedDocumentTypeException exception) {
            throw exception;
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("OFD 转 PDF 失败", exception);
        }
    }
}
