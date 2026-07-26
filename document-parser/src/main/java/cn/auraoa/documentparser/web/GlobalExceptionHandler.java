package cn.auraoa.documentparser.web;

import cn.auraoa.documentparser.dto.ErrorResponse;
import cn.auraoa.documentparser.exception.DocumentLimitExceededException;
import cn.auraoa.documentparser.exception.DocumentParserException;
import cn.auraoa.documentparser.exception.InvalidDocumentRequestException;
import cn.auraoa.documentparser.exception.UnsupportedDocumentTypeException;
import jakarta.servlet.http.HttpServletRequest;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.HttpMediaTypeNotSupportedException;
import org.springframework.web.bind.annotation.ExceptionHandler;
import org.springframework.web.bind.annotation.RestControllerAdvice;
import org.springframework.web.multipart.MaxUploadSizeExceededException;
import org.springframework.web.multipart.MultipartException;
import org.springframework.web.multipart.support.MissingServletRequestPartException;

/**
 * 将内部异常转换为稳定响应，日志只记录类型和路由，不记录文件正文或凭据。
 */
@RestControllerAdvice
public class GlobalExceptionHandler {

    private static final Logger LOGGER = LoggerFactory.getLogger(GlobalExceptionHandler.class);

    @ExceptionHandler(InvalidDocumentRequestException.class)
    public ResponseEntity<ErrorResponse> handleInvalidRequest(InvalidDocumentRequestException exception) {
        return error(HttpStatus.BAD_REQUEST, "invalid_request", exception.getMessage());
    }

    @ExceptionHandler({
            MissingServletRequestPartException.class,
            MultipartException.class
    })
    public ResponseEntity<ErrorResponse> handleMultipart(Exception exception) {
        return error(HttpStatus.BAD_REQUEST, "invalid_multipart", "必须使用 multipart/form-data 上传 file");
    }

    @ExceptionHandler({
            UnsupportedDocumentTypeException.class,
            HttpMediaTypeNotSupportedException.class
    })
    public ResponseEntity<ErrorResponse> handleUnsupportedType(Exception exception) {
        String message = exception instanceof UnsupportedDocumentTypeException
                ? exception.getMessage()
                : "请求媒体类型不受支持";
        return error(HttpStatus.UNSUPPORTED_MEDIA_TYPE, "unsupported_file_type", message);
    }

    @ExceptionHandler({
            DocumentLimitExceededException.class,
            MaxUploadSizeExceededException.class
    })
    public ResponseEntity<ErrorResponse> handleLimit(Exception exception) {
        String message = exception instanceof DocumentLimitExceededException
                ? exception.getMessage()
                : "上传文件超过大小限制";
        return error(HttpStatus.PAYLOAD_TOO_LARGE, "document_limit_exceeded", message);
    }

    @ExceptionHandler(DocumentParserException.class)
    public ResponseEntity<ErrorResponse> handleParseFailure(
            DocumentParserException exception,
            HttpServletRequest request
    ) {
        LOGGER.warn(
                "文档处理失败，路由={}，错误类型={}，原因类型={}",
                request.getRequestURI(),
                exception.getClass().getSimpleName(),
                exception.getCause() == null ? "none" : exception.getCause().getClass().getSimpleName()
        );
        return error(HttpStatus.UNPROCESSABLE_ENTITY, "parse_failed", exception.getMessage());
    }

    @ExceptionHandler(Exception.class)
    public ResponseEntity<ErrorResponse> handleUnexpected(Exception exception, HttpServletRequest request) {
        LOGGER.error(
                "文档解析服务发生未处理异常，路由={}，错误类型={}",
                request.getRequestURI(),
                exception.getClass().getSimpleName()
        );
        return error(HttpStatus.INTERNAL_SERVER_ERROR, "internal_error", "文档解析服务内部错误");
    }

    private ResponseEntity<ErrorResponse> error(HttpStatus status, String code, String message) {
        return ResponseEntity.status(status).body(new ErrorResponse(code, message));
    }
}
