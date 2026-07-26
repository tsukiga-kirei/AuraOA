package cn.auraoa.documentparser.exception;

/**
 * 文档超过安全解析限制。
 */
public class DocumentLimitExceededException extends RuntimeException {

    public DocumentLimitExceededException(String message) {
        super(message);
    }
}
