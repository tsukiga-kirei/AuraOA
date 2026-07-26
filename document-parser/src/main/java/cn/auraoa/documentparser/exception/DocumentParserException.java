package cn.auraoa.documentparser.exception;

/**
 * 文档解析失败。
 */
public class DocumentParserException extends RuntimeException {

    public DocumentParserException(String message) {
        super(message);
    }

    public DocumentParserException(String message, Throwable cause) {
        super(message, cause);
    }
}
