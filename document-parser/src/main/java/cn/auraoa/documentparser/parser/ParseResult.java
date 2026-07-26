package cn.auraoa.documentparser.parser;

import java.util.List;

/**
 * 解析器内部统一结果。
 */
public record ParseResult(
        String parser,
        String fileType,
        String content,
        boolean hasTextLayer,
        boolean fallbackRequired,
        String fallbackFormat,
        List<String> warnings
) {
}
