package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentParserException;
import org.apache.poi.hslf.usermodel.HSLFNotes;
import org.apache.poi.hslf.usermodel.HSLFShape;
import org.apache.poi.hslf.usermodel.HSLFSlide;
import org.apache.poi.hslf.usermodel.HSLFSlideShow;
import org.apache.poi.hslf.usermodel.HSLFTextShape;
import org.springframework.stereotype.Component;

import java.io.IOException;
import java.io.InputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/**
 * 使用 Apache POI HSLF 按幻灯片提取 PowerPoint 97-2003 文本与备注。
 */
@Component
public class PptDocumentParser implements DocumentFormatParser {

    private final ParserProperties properties;

    public PptDocumentParser(ParserProperties properties) {
        this.properties = properties;
    }

    @Override
    public String fileType() {
        return "ppt";
    }

    @Override
    public ParseResult parse(Path input) {
        List<String> warnings = new ArrayList<>();
        try (InputStream stream = Files.newInputStream(input);
             HSLFSlideShow slideShow = new HSLFSlideShow(stream)) {
            String content = extractSlides(slideShow, warnings);
            boolean hasText = !content.isBlank();
            if (!hasText) {
                warnings.add("未从 PPT 中提取到可用文本，建议转换为 PDF 后进行版面识别");
            }
            return new ParseResult(
                    "apache-poi-hslf",
                    fileType(),
                    content,
                    hasText,
                    false,
                    null,
                    List.copyOf(warnings)
            );
        } catch (IOException | RuntimeException exception) {
            throw new DocumentParserException("PPT 文件解析失败", exception);
        }
    }

    private String extractSlides(HSLFSlideShow slideShow, List<String> warnings) {
        StringBuilder output = new StringBuilder();
        List<HSLFSlide> slides = slideShow.getSlides();
        int slideCount = Math.min(slides.size(), properties.maxPptSlides());
        if (slides.size() > slideCount) {
            warnings.add("幻灯片数量超过上限，结果仅包含前 " + slideCount + " 页");
        }

        for (int index = 0; index < slideCount; index++) {
            HSLFSlide slide = slides.get(index);
            String slideText = extractTextShapes(slide.getShapes());
            HSLFNotes notes = slide.getNotes();
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

        String content = output.toString();
        if (content.length() > properties.maxOutputChars()) {
            content = content.substring(0, properties.maxOutputChars());
        }
        return TextSupport.normalize(content);
    }

    private String extractTextShapes(List<HSLFShape> shapes) {
        StringBuilder text = new StringBuilder();
        for (HSLFShape shape : shapes) {
            if (shape instanceof HSLFTextShape textShape) {
                String value = TextSupport.normalize(textShape.getText());
                if (!value.isBlank()) {
                    text.append(value).append('\n');
                }
            }
        }
        return text.toString().strip();
    }
}
