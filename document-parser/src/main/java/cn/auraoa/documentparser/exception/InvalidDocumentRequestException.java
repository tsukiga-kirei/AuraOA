package cn.auraoa.documentparser.exception;

/**
 * 上传请求不符合解析要求。
 */
public class InvalidDocumentRequestException extends RuntimeException {

    public InvalidDocumentRequestException(String message) {
        super(message);
    }
}
