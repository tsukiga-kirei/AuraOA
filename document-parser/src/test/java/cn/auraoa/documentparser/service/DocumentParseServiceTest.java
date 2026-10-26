package cn.auraoa.documentparser.service;

import cn.auraoa.documentparser.config.ParserProperties;
import cn.auraoa.documentparser.exception.DocumentLimitExceededException;
import cn.auraoa.documentparser.exception.DocumentParserException;
import cn.auraoa.documentparser.parser.DocumentFormatParser;
import cn.auraoa.documentparser.parser.ParseResult;
import org.junit.jupiter.params.ParameterizedTest;
import org.junit.jupiter.params.provider.ValueSource;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.util.unit.DataSize;

import java.nio.file.Files;
import java.nio.file.Path;
import java.util.List;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

/**
 * 验证 Office 回退使用同一份原始文件，并保持原有失败及安全限制语义。
 */
class DocumentParseServiceTest {

    @ParameterizedTest
    @ValueSource(strings = {"docx", "xlsx", "pptx"})
    void fallsBackToMatchingLegacyParser(String modernType) {
        String legacyType = modernType.substring(0, modernType.length() - 1);
        DocumentFormatParser modern = parser(modernType);
        DocumentFormatParser legacy = parser(legacyType);
        when(modern.parse(any(Path.class))).thenThrow(new DocumentParserException("新版格式解析失败"));
        when(legacy.parse(any(Path.class))).thenAnswer(invocation -> {
            assertThat(Files.readAllBytes(invocation.getArgument(0, Path.class))).containsExactly(1, 2, 3);
            return new ParseResult("legacy-parser", legacyType, "附件正文", true, false, null, List.of("原有警告"));
        });

        var result = service(modern, legacy).parse(file(modernType));

        assertThat(result.parser()).isEqualTo("legacy-parser");
        assertThat(result.fileType()).isEqualTo(legacyType);
        assertThat(result.content()).isEqualTo("附件正文");
        assertThat(result.warnings()).containsExactly("原有警告", modernType.toUpperCase()
                + " 解析失败，已使用 " + legacyType.toUpperCase() + " 兼容解析");
        verify(modern).parse(any(Path.class));
        verify(legacy).parse(any(Path.class));
    }

    @ParameterizedTest
    @ValueSource(strings = {"docx", "xlsx", "pptx"})
    void preservesModernSuccessAndDoesNotRetryLimits(String modernType) {
        DocumentFormatParser modern = parser(modernType);
        DocumentFormatParser legacy = parser(modernType.substring(0, modernType.length() - 1));
        var service = service(modern, legacy);
        when(modern.parse(any(Path.class))).thenReturn(
                new ParseResult("modern-parser", modernType, "", false, false, null, List.of()));
        assertThat(service.parse(file(modernType)).parser()).isEqualTo("modern-parser");
        when(modern.parse(any(Path.class))).thenThrow(new DocumentLimitExceededException("超过限制"));
        assertThatThrownBy(() -> service.parse(file(modernType)))
                .isInstanceOf(DocumentLimitExceededException.class);
        verify(legacy, never()).parse(any(Path.class));
    }

    @ParameterizedTest
    @ValueSource(strings = {"docx", "xlsx", "pptx"})
    void retainsOriginalFailureWhenBothParsersFail(String modernType) {
        DocumentFormatParser modern = parser(modernType);
        DocumentFormatParser legacy = parser(modernType.substring(0, modernType.length() - 1));
        var failure = new DocumentParserException("新版格式解析失败");
        when(modern.parse(any(Path.class))).thenThrow(failure);
        when(legacy.parse(any(Path.class))).thenThrow(new DocumentParserException("旧版格式解析失败"));

        assertThatThrownBy(() -> service(modern, legacy).parse(file(modernType))).isSameAs(failure);
        assertThat(failure.getSuppressed()).hasSize(1);
    }

    private DocumentFormatParser parser(String fileType) {
        var parser = mock(DocumentFormatParser.class);
        when(parser.fileType()).thenReturn(fileType);
        return parser;
    }

    private DocumentParseService service(DocumentFormatParser... parsers) {
        var properties = mock(ParserProperties.class);
        when(properties.maxUploadSize()).thenReturn(DataSize.ofMegabytes(1));
        return new DocumentParseService(properties, List.of(parsers));
    }

    private MockMultipartFile file(String fileType) {
        return new MockMultipartFile("file", "附件." + fileType, "application/octet-stream", new byte[]{1, 2, 3});
    }
}
