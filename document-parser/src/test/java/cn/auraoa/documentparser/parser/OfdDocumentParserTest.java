package cn.auraoa.documentparser.parser;

import cn.auraoa.documentparser.config.ParserProperties;
import org.junit.jupiter.api.Test;
import org.springframework.util.unit.DataSize;

import static org.assertj.core.api.Assertions.assertThat;

class OfdDocumentParserTest {

    @Test
    void emptyTextLayerRequiresPdfFallback() {
        ParserProperties properties = properties();
        OfdDocumentParser parser = new OfdDocumentParser(properties, new OfdArchiveValidator(properties));

        ParseResult result = parser.buildResult(" \r\n ");

        assertThat(result.hasTextLayer()).isFalse();
        assertThat(result.fallbackRequired()).isTrue();
        assertThat(result.fallbackFormat()).isEqualTo("pdf");
        assertThat(result.warnings()).isNotEmpty();
    }

    @Test
    void extractedTextDoesNotRequireFallback() {
        ParserProperties properties = properties();
        OfdDocumentParser parser = new OfdDocumentParser(properties, new OfdArchiveValidator(properties));

        ParseResult result = parser.buildResult("电子发票\n金额：100.00");

        assertThat(result.hasTextLayer()).isTrue();
        assertThat(result.fallbackRequired()).isFalse();
        assertThat(result.fallbackFormat()).isNull();
    }

    private ParserProperties properties() {
        return new ParserProperties(
                "",
                DataSize.ofMegabytes(50),
                5_000_000,
                DataSize.ofMegabytes(64),
                DataSize.ofMegabytes(100),
                DataSize.ofMegabytes(200),
                10_000,
                100,
                100,
                10_000,
                256,
                1_000,
                1_000
        );
    }
}
