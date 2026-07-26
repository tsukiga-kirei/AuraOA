package cn.auraoa.documentparser.dto;

import java.util.Map;

/**
 * 健康检查与解析能力响应。
 */
public record HealthResponse(
        String status,
        Map<String, Boolean> capabilities
) {
}
