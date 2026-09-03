package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.text.PDFTextStripper;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/**
 * 使用 PDFBox 提取 PDF 文字层；扫描件无文字时提示调用方回退 MinerU。
 */
@Component
public class PdfDocumentParser implements DocumentFormatParser {

    private final ParserProperties properties;

    public PdfDocumentParser(ParserProperties properties) {
        this.properties = properties;
    }

    @Override
    public String fileType() {
        return "pdf";
    }

    @Override
    public ParseResult parse(Path input) {
        List<String> warnings = new ArrayList<>();
        try (PDDocument document = PDDocument.load(input.toFile())) {
            if (document.isEncrypted()) {
                throw new DocumentParserException("PDF 已加密，无法提取文字层");
            }
            int pageCount = document.getNumberOfPages();
            int endPage = Math.min(pageCount, properties.maxPdfPages());
            if (pageCount > endPage) {
                warnings.add("PDF 页数超过上限，结果仅包含前 " + endPage + " 页");
            }
            PDFTextStripper stripper = new PDFTextStripper();
            stripper.setStartPage(1);
            stripper.setEndPage(endPage);
            stripper.setSortByPosition(true);
            String content = TextSupport.limit(stripper.getText(document), properties.maxOutputChars(), warnings);
            boolean hasText = !content.isBlank();
            if (!hasText) {
                warnings.add("PDF 没有可提取的文字层，将回退到 MinerU 进行版面或 OCR 识别");
            }
            return new ParseResult("apache-pdfbox", fileType(), content, hasText, !hasText,
                    hasText ? null : "pdf", List.copyOf(warnings));
        } catch (DocumentParserException exception) {
            throw exception;
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("PDF 文件解析失败", exception);
        }
    }
}
