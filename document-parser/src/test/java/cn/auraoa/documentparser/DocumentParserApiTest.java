package cn.auraoa.documentparser;

import org.apache.poi.hslf.usermodel.HSLFSlide;
import org.apache.poi.hslf.usermodel.HSLFSlideShow;
import org.apache.poi.hslf.usermodel.HSLFTextBox;
import org.apache.poi.hssf.usermodel.HSSFWorkbook;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.ofdrw.layout.OFDDoc;
import org.ofdrw.layout.element.Paragraph;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.HttpHeaders;
import org.springframework.mock.web.MockMultipartFile;
import org.springframework.test.web.servlet.MockMvc;

import java.io.ByteArrayOutputStream;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.charset.StandardCharsets;
import java.util.zip.ZipEntry;
import java.util.zip.ZipOutputStream;

import static org.hamcrest.Matchers.containsString;
import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.get;
import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.multipart;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.content;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.jsonPath;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.status;

@SpringBootTest(properties = "parser.api-key=test-secret")
@AutoConfigureMockMvc
class DocumentParserApiTest {

    @Autowired
    private MockMvc mockMvc;

    @TempDir
    private Path temporaryDirectory;

    @Test
    void healthDoesNotRequireAuthentication() throws Exception {
        mockMvc.perform(get("/health"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("ok"))
                .andExpect(jsonPath("$.capabilities.ofd_to_pdf").value(true));
    }

    @Test
    void parseRequiresBearerAuthenticationWhenConfigured() throws Exception {
        MockMultipartFile file = new MockMultipartFile(
                "file", "sample.xls", "application/vnd.ms-excel", new byte[]{1}
        );

        mockMvc.perform(multipart("/parse").file(file))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.code").value("unauthorized"));
    }

    @Test
    void readyChecksCorrectAndIncorrectBearerCredentials() throws Exception {
        mockMvc.perform(get("/ready")
                        .header(HttpHeaders.AUTHORIZATION, "Bearer wrong-secret"))
                .andExpect(status().isUnauthorized())
                .andExpect(jsonPath("$.code").value("unauthorized"));

        mockMvc.perform(get("/ready")
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.status").value("ok"));
    }

    @Test
    void rejectsUnsupportedTxtFile() throws Exception {
        MockMultipartFile file = new MockMultipartFile(
                "file", "sample.txt", "text/plain", "hello".getBytes(StandardCharsets.UTF_8)
        );

        mockMvc.perform(multipart("/parse").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isUnsupportedMediaType())
                .andExpect(jsonPath("$.code").value("unsupported_file_type"));
    }

    @Test
    void parsesGeneratedXlsWorkbook() throws Exception {
        byte[] workbookBytes;
        try (HSSFWorkbook workbook = new HSSFWorkbook();
             ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            var sheet = workbook.createSheet("费用明细");
            var header = sheet.createRow(0);
            header.createCell(0).setCellValue("姓名");
            header.createCell(1).setCellValue("金额");
            var data = sheet.createRow(1);
            data.createCell(0).setCellValue("张三");
            data.createCell(1).setCellValue(1200);
            workbook.write(output);
            workbookBytes = output.toByteArray();
        }
        MockMultipartFile file = new MockMultipartFile(
                "file", "费用.xls", "application/vnd.ms-excel", workbookBytes
        );

        mockMvc.perform(multipart("/parse").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.parser").value("apache-poi-hssf"))
                .andExpect(jsonPath("$.file_type").value("xls"))
                .andExpect(jsonPath("$.has_text_layer").value(true))
                .andExpect(jsonPath("$.fallback_required").value(false))
                .andExpect(jsonPath("$.content", containsString("费用明细")))
                .andExpect(jsonPath("$.content", containsString("张三")));
    }

    @Test
    void parsesGeneratedPptSlideshow() throws Exception {
        byte[] slideshowBytes;
        try (HSLFSlideShow slideShow = new HSLFSlideShow();
             ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            HSLFSlide slide = slideShow.createSlide();
            HSLFTextBox textBox = new HSLFTextBox();
            textBox.setText("季度工作总结");
            slide.addShape(textBox);
            slideShow.write(output);
            slideshowBytes = output.toByteArray();
        }
        MockMultipartFile file = new MockMultipartFile(
                "file", "summary.ppt", "application/vnd.ms-powerpoint", slideshowBytes
        );

        mockMvc.perform(multipart("/parse").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.parser").value("apache-poi-hslf"))
                .andExpect(jsonPath("$.file_type").value("ppt"))
                .andExpect(jsonPath("$.has_text_layer").value(true))
                .andExpect(jsonPath("$.content", containsString("季度工作总结")));
    }

    @Test
    void reportsClearBoundaryForInvalidDoc() throws Exception {
        MockMultipartFile file = new MockMultipartFile(
                "file", "broken.doc", "application/msword",
                "not-an-ole-document".getBytes(StandardCharsets.UTF_8)
        );

        mockMvc.perform(multipart("/parse").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isUnprocessableEntity())
                .andExpect(jsonPath("$.code").value("parse_failed"))
                .andExpect(jsonPath("$.message").value("DOC 文件解析失败"));
    }

    @Test
    void rejectsInvalidOfdContainerForParseAndConversion() throws Exception {
        MockMultipartFile file = new MockMultipartFile(
                "file", "broken.ofd", "application/octet-stream",
                "not-a-zip".getBytes(StandardCharsets.UTF_8)
        );

        mockMvc.perform(multipart("/parse").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isUnprocessableEntity())
                .andExpect(jsonPath("$.code").value("parse_failed"));

        mockMvc.perform(multipart("/convert/pdf").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isUnprocessableEntity())
                .andExpect(jsonPath("$.code").value("parse_failed"));
    }

    @Test
    void parsesGeneratedOfdAndConvertsItToPdf() throws Exception {
        byte[] ofdBytes = createOfd("AuraOA OFD end-to-end test");
        MockMultipartFile file = new MockMultipartFile(
                "file", "sample.ofd", "application/octet-stream", ofdBytes
        );

        mockMvc.perform(multipart("/parse").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.parser").value("ofdrw"))
                .andExpect(jsonPath("$.file_type").value("ofd"))
                .andExpect(jsonPath("$.has_text_layer").value(true))
                .andExpect(jsonPath("$.fallback_required").value(false))
                .andExpect(jsonPath("$.content", containsString("AuraOA OFD end-to-end test")));

        mockMvc.perform(multipart("/convert/pdf").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(content().contentType("application/pdf"))
                .andExpect(result -> assertThat(result.getResponse().getContentAsByteArray())
                        .startsWith("%PDF-".getBytes(StandardCharsets.US_ASCII)));
    }

    @Test
    void blocksHighlyCompressedOfdEntry() throws Exception {
        byte[] ofdBytes;
        try (ByteArrayOutputStream output = new ByteArrayOutputStream();
             ZipOutputStream zip = new ZipOutputStream(output)) {
            zip.putNextEntry(new ZipEntry("OFD.xml"));
            zip.write("<ofd/>".getBytes(StandardCharsets.UTF_8));
            zip.closeEntry();
            zip.putNextEntry(new ZipEntry("Doc_0/Res/bomb.bin"));
            zip.write(new byte[2 * 1024 * 1024]);
            zip.closeEntry();
            zip.finish();
            ofdBytes = output.toByteArray();
        }
        MockMultipartFile file = new MockMultipartFile(
                "file", "bomb.ofd", "application/octet-stream", ofdBytes
        );

        mockMvc.perform(multipart("/parse").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isPayloadTooLarge())
                .andExpect(jsonPath("$.code").value("document_limit_exceeded"));
    }

    @Test
    void convertPdfRejectsLegacyOfficeWithoutStartingConverter() throws Exception {
        byte[] workbookBytes;
        try (HSSFWorkbook workbook = new HSSFWorkbook();
             ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            workbook.createSheet("Sheet1").createRow(0).createCell(0).setCellValue("value");
            workbook.write(output);
            workbookBytes = output.toByteArray();
        }
        MockMultipartFile file = new MockMultipartFile(
                "file", "sample.xls", "application/vnd.ms-excel", workbookBytes
        );

        mockMvc.perform(multipart("/convert/pdf").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isUnsupportedMediaType())
                .andExpect(content().contentTypeCompatibleWith("application/json"))
                .andExpect(jsonPath("$.code").value("unsupported_file_type"));
    }

    private byte[] createOfd(String text) throws Exception {
        Path output = temporaryDirectory.resolve("generated.ofd");
        try (OFDDoc document = new OFDDoc(output)) {
            document.add(new Paragraph(text));
        }
        return Files.readAllBytes(output);
    }
}
