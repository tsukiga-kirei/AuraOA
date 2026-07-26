package cn.auraoa.documentparser.parser;

import java.util.List;

/**
 * 解析文本的清理、截断与 Markdown 转义工具。
 */
final class TextSupport {

    private TextSupport() {
    }

    static String normalize(String input) {
        if (input == null || input.isEmpty()) {
            return "";
        }
        String normalized = input.replace("\r\n", "\n").replace('\r', '\n');
        StringBuilder result = new StringBuilder(normalized.length());
        for (int i = 0; i < normalized.length(); i++) {
            char character = normalized.charAt(i);
            if (character == '\n' || character == '\t'
                    || (!Character.isISOControl(character) && character != '\uFFFE' && character != '\uFFFF')) {
                result.append(character);
            }
        }
        return result.toString().strip();
    }

    static String limit(String input, int maxChars, List<String> warnings) {
        String normalized = normalize(input);
        if (normalized.length() <= maxChars) {
            return normalized;
        }
        warnings.add("解析文本超过长度上限，结果已截断");
        return normalized.substring(0, maxChars).stripTrailing();
    }

    static String markdownCell(String value) {
        String normalized = normalize(value);
        return normalized
                .replace("\\", "\\\\")
                .replace("|", "\\|")
                .replace("\n", "<br>");
    }
}
