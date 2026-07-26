package cn.auraoa.documentparser.dto;

/**
 * 不暴露内部异常细节的错误响应。
 */
public record ErrorResponse(
        String code,
        String message
) {
}
