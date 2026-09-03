package cn.auraoa.documentparser.web;

import cn.auraoa.documentparser.dto.HealthResponse;
import cn.auraoa.documentparser.dto.ParseResponse;
import cn.auraoa.documentparser.service.DocumentParseService;
import cn.auraoa.documentparser.service.PdfConversionService;
import org.springframework.http.ContentDisposition;
import org.springframework.http.HttpHeaders;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestPart;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

import java.nio.charset.StandardCharsets;
import java.util.LinkedHashMap;
import java.util.Map;

/**
 * 文档内容解析服务接口。
 */
@RestController
@RequestMapping
public class DocumentParserController {

    private final DocumentParseService parseService;
    private final PdfConversionService conversionService;

    public DocumentParserController(
            DocumentParseService parseService,
            PdfConversionService conversionService
    ) {
        this.parseService = parseService;
        this.conversionService = conversionService;
    }

    /**
     * GET /health
     * 返回服务状态和格式能力；该接口始终免鉴权，供容器健康检查使用。
     */
    @GetMapping(path = "/health", produces = MediaType.APPLICATION_JSON_VALUE)
    public HealthResponse health() {
        return status();
    }

    /**
     * GET /ready
     * 返回服务就绪状态；当配置 API Key 时，该接口同样需要 Bearer 鉴权。
     */
    @GetMapping(path = "/ready", produces = MediaType.APPLICATION_JSON_VALUE)
    public HealthResponse ready() {
        return status();
    }

    private HealthResponse status() {
        Map<String, Boolean> capabilities = new LinkedHashMap<>();
        capabilities.put("pdf", true);
        capabilities.put("doc", true);
        capabilities.put("docx", true);
        capabilities.put("xls", true);
        capabilities.put("xlsx", true);
        capabilities.put("ppt", true);
        capabilities.put("pptx", true);
        capabilities.put("ofd", true);
        capabilities.put("ofd_to_pdf", true);
        return new HealthResponse("ok", capabilities);
    }

    /**
     * POST /parse
     * multipart/form-data 参数 file，返回统一的文本解析结果和回退提示。
     */
    @PostMapping(
            path = "/parse",
            consumes = MediaType.MULTIPART_FORM_DATA_VALUE,
            produces = MediaType.APPLICATION_JSON_VALUE
    )
    public ParseResponse parse(@RequestPart("file") MultipartFile file) {
        return parseService.parse(file);
    }

    /**
     * POST /convert/pdf
     * multipart/form-data 参数 file，第一阶段支持 OFD 转 PDF。
     */
    @PostMapping(
            path = "/convert/pdf",
            consumes = MediaType.MULTIPART_FORM_DATA_VALUE,
            produces = MediaType.APPLICATION_PDF_VALUE
    )
    public ResponseEntity<byte[]> convertToPdf(@RequestPart("file") MultipartFile file) {
        byte[] pdf = conversionService.convert(file);
        ContentDisposition disposition = ContentDisposition.attachment()
                .filename("converted.pdf", StandardCharsets.UTF_8)
                .build();
        return ResponseEntity.ok()
                .header(HttpHeaders.CONTENT_DISPOSITION, disposition.toString())
                .contentLength(pdf.length)
                .contentType(MediaType.APPLICATION_PDF)
                .body(pdf);
    }
}
