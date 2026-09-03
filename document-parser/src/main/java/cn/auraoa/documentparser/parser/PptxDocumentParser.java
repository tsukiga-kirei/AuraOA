package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.apache.poi.xslf.usermodel.XMLSlideShow;
import org.apache.poi.xslf.usermodel.XSLFNotes;
import org.apache.poi.xslf.usermodel.XSLFShape;
import org.apache.poi.xslf.usermodel.XSLFSlide;
import org.apache.poi.xslf.usermodel.XSLFTextShape;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/**
 * 使用 Apache POI XSLF 按幻灯片提取 PPTX 文本与备注。
 */
@Component
public class PptxDocumentParser implements DocumentFormatParser {

    private final ParserProperties properties;

    public PptxDocumentParser(ParserProperties properties) {
        this.properties = properties;
    }

    @Override
    public String fileType() {
        return "pptx";
    }

    @Override
    public ParseResult parse(Path input) {
        List<String> warnings = new ArrayList<>();
        try (InputStream stream = Files.newInputStream(input);
             XMLSlideShow slideShow = new XMLSlideShow(stream)) {
            String content = extractSlides(slideShow, warnings);
            boolean hasText = !content.isBlank();
            if (!hasText) {
                warnings.add("未从 PPTX 中提取到可用文本");
            }
            return new ParseResult("apache-poi-xslf", fileType(), content, hasText, false, null, List.copyOf(warnings));
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("PPTX 文件解析失败", exception);
        }
    }

    private String extractSlides(XMLSlideShow slideShow, List<String> warnings) {
        StringBuilder output = new StringBuilder();
        List<XSLFSlide> slides = slideShow.getSlides();
        int slideCount = Math.min(slides.size(), properties.maxPptSlides());
        if (slides.size() > slideCount) {
            warnings.add("幻灯片数量超过上限，结果仅包含前 " + slideCount + " 页");
        }
        for (int index = 0; index < slideCount; index++) {
            XSLFSlide slide = slides.get(index);
            String slideText = extractTextShapes(slide.getShapes());
            XSLFNotes notes = slide.getNotes();
            String notesText = notes == null ? "" : extractTextShapes(notes.getShapes());
            if (slideText.isBlank() && notesText.isBlank()) {
                continue;
            }
            output.append("## 幻灯片 ").append(index + 1).append("\n\n");
            if (!slideText.isBlank()) {
                output.append(slideText).append("\n\n");
            }
            if (!notesText.isBlank()) {
                output.append("### 备注\n\n").append(notesText).append("\n\n");
            }
            if (output.length() >= properties.maxOutputChars()) {
                warnings.add("解析文本超过长度上限，结果已截断");
                break;
            }
        }
        return TextSupport.limit(output.toString(), properties.maxOutputChars(), warnings);
    }

    private String extractTextShapes(List<? extends XSLFShape> shapes) {
        StringBuilder text = new StringBuilder();
        for (XSLFShape shape : shapes) {
            if (shape instanceof XSLFTextShape textShape) {
                String value = TextSupport.normalize(textShape.getText());
                if (!value.isBlank()) {
                    text.append(value).append('\n');
                }
            }
        }
        return text.toString().strip();
    }
}
