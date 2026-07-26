package cn.auraoa.documentparser.dto;

import java.util.List;

/**
 * 文档解析的统一响应。
 */
public record ParseResponse(
        String parser,
        String fileType,
        String content,
        boolean hasTextLayer,
        boolean fallbackRequired,
        String fallbackFormat,
        List<String> warnings
) {
}
