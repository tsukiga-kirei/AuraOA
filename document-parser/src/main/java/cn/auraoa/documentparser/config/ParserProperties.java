package cn.auraoa.documentparser.config;

import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Positive;
import org.springframework.boot.context.properties.ConfigurationProperties;
import org.springframework.validation.annotation.Validated;
import org.springframework.util.unit.DataSize;

/**
 * 文档解析服务的安全限制与运行参数。
 */
@Validated
@ConfigurationProperties(prefix = "parser")
public record ParserProperties(
        String apiKey,
        @NotNull
        DataSize maxUploadSize,
        @Positive
        int maxOutputChars,
        @NotNull
        DataSize maxConvertedPdfSize,
        @NotNull
        DataSize maxPoiAllocationSize,
        @NotNull
        DataSize maxOfdUncompressedSize,
        @Positive
        int maxOfdEntries,
        @Positive
        double maxOfdCompressionRatio,
        @Positive
        int maxXlsSheets,
        @Positive
        int maxXlsRowsPerSheet,
        @Positive
        int maxXlsCellsPerRow,
        @Positive
        int maxPptSlides
) {
    public ParserProperties {
        requirePositive(maxUploadSize, "parser.max-upload-size");
        requirePositive(maxConvertedPdfSize, "parser.max-converted-pdf-size");
        requirePositive(maxPoiAllocationSize, "parser.max-poi-allocation-size");
        requirePositive(maxOfdUncompressedSize, "parser.max-ofd-uncompressed-size");
    }

    private static void requirePositive(DataSize value, String propertyName) {
        if (value == null || value.toBytes() <= 0) {
            throw new IllegalArgumentException(propertyName + " 必须大于 0");
        }
    }
}
