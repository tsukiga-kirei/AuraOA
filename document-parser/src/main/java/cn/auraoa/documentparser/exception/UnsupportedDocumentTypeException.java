package cn.auraoa.documentparser.exception;

/**
 * 文件类型不在文档内容解析服务支持范围内。
 */
public class UnsupportedDocumentTypeException extends RuntimeException {

    public UnsupportedDocumentTypeException(String message) {
        super(message);
    }
}
