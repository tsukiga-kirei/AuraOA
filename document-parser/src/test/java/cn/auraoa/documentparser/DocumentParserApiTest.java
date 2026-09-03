package cn.auraoa.documentparser;

import org.apache.poi.hslf.usermodel.HSLFSlide;
import org.apache.poi.hslf.usermodel.HSLFSlideShow;
import org.apache.poi.hslf.usermodel.HSLFTextBox;
import org.apache.poi.hssf.usermodel.HSSFWorkbook;
import org.apache.poi.xslf.usermodel.XMLSlideShow;
import org.apache.poi.xslf.usermodel.XSLFSlide;
import org.apache.poi.xslf.usermodel.XSLFTextBox;
import org.apache.poi.xssf.usermodel.XSSFWorkbook;
import org.apache.poi.xwpf.usermodel.XWPFDocument;
import org.apache.pdfbox.pdmodel.PDDocument;
import org.apache.pdfbox.pdmodel.PDPage;
import org.apache.pdfbox.pdmodel.PDPageContentStream;
import org.apache.pdfbox.pdmodel.font.PDType1Font;
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
                .andExpect(jsonPath("$.capabilities.pdf").value(true))
                .andExpect(jsonPath("$.capabilities.docx").value(true))
                .andExpect(jsonPath("$.capabilities.xlsx").value(true))
                .andExpect(jsonPath("$.capabilities.pptx").value(true))
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
    void parsesGeneratedPdfTextLayer() throws Exception {
        byte[] pdfBytes;
        try (PDDocument document = new PDDocument();
             ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            PDPage page = new PDPage();
            document.addPage(page);
            try (PDPageContentStream contentStream = new PDPageContentStream(document, page)) {
                contentStream.beginText();
                contentStream.setFont(PDType1Font.HELVETICA, 12);
                contentStream.newLineAtOffset(72, 720);
                contentStream.showText("AuraOA PDF contract content");
                contentStream.endText();
            }
            document.save(output);
            pdfBytes = output.toByteArray();
        }
        MockMultipartFile file = new MockMultipartFile("file", "contract.pdf", "application/pdf", pdfBytes);

        mockMvc.perform(multipart("/parse").file(file)
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.parser").value("apache-pdfbox"))
                .andExpect(jsonPath("$.has_text_layer").value(true))
                .andExpect(jsonPath("$.fallback_required").value(false))
                .andExpect(jsonPath("$.content", containsString("AuraOA PDF contract content")));
    }

    @Test
    void parsesGeneratedModernOfficeDocuments() throws Exception {
        byte[] docxBytes;
        try (XWPFDocument document = new XWPFDocument();
             ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            document.createParagraph().createRun().setText("DOCX 合同正文");
            document.write(output);
            docxBytes = output.toByteArray();
        }
        mockMvc.perform(multipart("/parse").file(new MockMultipartFile(
                                "file", "合同.docx",
                                "application/vnd.openxmlformats-officedocument.wordprocessingml.document", docxBytes))
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.parser").value("apache-poi-xwpf"))
                .andExpect(jsonPath("$.content", containsString("DOCX 合同正文")));

        byte[] xlsxBytes;
        try (XSSFWorkbook workbook = new XSSFWorkbook();
             ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            workbook.createSheet("预算").createRow(0).createCell(0).setCellValue("预算金额");
            workbook.write(output);
            xlsxBytes = output.toByteArray();
        }
        mockMvc.perform(multipart("/parse").file(new MockMultipartFile(
                                "file", "预算.xlsx",
                                "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxBytes))
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.parser").value("apache-poi-xssf"))
                .andExpect(jsonPath("$.content", containsString("预算金额")));

        byte[] pptxBytes;
        try (XMLSlideShow slideShow = new XMLSlideShow();
             ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            XSLFSlide slide = slideShow.createSlide();
            XSLFTextBox textBox = slide.createTextBox();
            textBox.setText("PPTX 项目汇报");
            slideShow.write(output);
            pptxBytes = output.toByteArray();
        }
        mockMvc.perform(multipart("/parse").file(new MockMultipartFile(
                                "file", "汇报.pptx",
                                "application/vnd.openxmlformats-officedocument.presentationml.presentation", pptxBytes))
                        .header(HttpHeaders.AUTHORIZATION, "Bearer test-secret"))
                .andExpect(status().isOk())
                .andExpect(jsonPath("$.parser").value("apache-poi-xslf"))
                .andExpect(jsonPath("$.content", containsString("PPTX 项目汇报")));
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
