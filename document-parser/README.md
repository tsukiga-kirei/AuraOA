# AuraOA 文档内容解析服务

该服务通过 PDFBox、Apache POI 和 OFDRW 直接提取电子文档内容，不依赖 OCR。管理员可在
AuraOA 中按扩展名选择是否使用本服务；未选择的 PDF 与新版 Office 仍可交给 MinerU。
服务不调用大语言模型，也不会在日志中记录附件正文。

## 支持范围

| 格式 | 解析器 | `/parse` | `/convert/pdf` |
|---|---|---:|---:|
| `.pdf` | Apache PDFBox | 是 | 否 |
| `.doc` | Apache POI HWPF | 是 | 否 |
| `.docx` | Apache POI XWPF | 是 | 否 |
| `.xls` | Apache POI HSSF | 是 | 否 |
| `.xlsx` | Apache POI XSSF | 是 | 否 |
| `.ppt` | Apache POI HSLF | 是 | 否 |
| `.pptx` | Apache POI XSLF | 是 | 否 |
| `.ofd` | OFDRW | 是 | 是 |

PDF 没有文字层时，`/parse` 会提示调用方把原文件交给 MinerU；OFD 没有可提取文字时，
调用方可再请求 `/convert/pdf` 后将 PDF 交给 MinerU。DOC、DOCX、PPT、PPTX
没有文字时会返回警告，但第一阶段没有可执行的 Office 转 PDF 回退，因此
`fallback_required` 仍为 `false`。

## 构建与运行

需要 Java 21 和 Maven 3.9：

```bash
mvn clean test
mvn clean package
java -jar target/document-parser-1.0.0-SNAPSHOT.jar
```

也可以直接构建容器；多阶段构建会执行完整测试：

```bash
docker build -t auraoa-document-parser:latest .
docker run --rm -p 8090:8090 \
  -e PARSER_API_KEY=replace-with-a-secret \
  auraoa-document-parser:latest
```

容器运行层基于 Ubuntu Jammy JRE 21，包含 Noto CJK 字体，以非 root 用户运行。默认端口为
`8090`，可以通过 `SERVER_PORT` 覆盖；容器健康检查会读取同一环境变量。

## 接口

### 健康检查

`GET /health` 始终免鉴权：

```json
{
  "status": "ok",
  "capabilities": {
    "pdf": true,
    "doc": true,
    "docx": true,
    "xls": true,
    "xlsx": true,
    "ppt": true,
    "pptx": true,
    "ofd": true,
    "ofd_to_pdf": true
  }
}
```

`GET /ready` 返回相同能力信息，但在配置 `PARSER_API_KEY` 后需要有效 Bearer 凭据，供 AuraOA
后台的“测试连接”操作同时验证连通性和鉴权配置。

### 解析

```bash
curl -X POST http://localhost:8090/parse \
  -H "Authorization: Bearer ${PARSER_API_KEY}" \
  -F "file=@sample.xls"
```

响应示例：

```json
{
  "parser": "apache-poi-hssf",
  "file_type": "xls",
  "content": "## 工作表：Sheet1\n\n| 姓名 | 金额 |",
  "has_text_layer": true,
  "fallback_required": false,
  "fallback_format": null,
  "warnings": []
}
```

### OFD 转 PDF

```bash
curl -X POST http://localhost:8090/convert/pdf \
  -H "Authorization: Bearer ${PARSER_API_KEY}" \
  -F "file=@sample.ofd" \
  --output sample.pdf
```

## 配置

| 环境变量 | 默认值 | 说明 |
|---|---:|---|
| `SERVER_PORT` | `8090` | HTTP 端口 |
| `PARSER_API_KEY` | 空 | 非空时 `/parse`、`/convert/pdf` 必须使用 Bearer 凭据 |
| `PARSER_MAX_FILE_SIZE` | `50MB` | 单文件大小上限，同时用于 Spring multipart 和业务校验 |
| `PARSER_MAX_REQUEST_SIZE` | `51MB` | multipart 请求大小上限 |
| `PARSER_MAX_OUTPUT_CHARS` | `5000000` | 文本输出最大字符数 |
| `PARSER_MAX_CONVERTED_PDF_SIZE` | `64MB` | 转换后 PDF 大小上限，与 Go 调用方接收上限一致 |
| `PARSER_MAX_POI_ALLOCATION_SIZE` | `100MB` | POI 单次内部字节数组分配上限 |
| `PARSER_MAX_OFD_UNCOMPRESSED_SIZE` | `200MB` | OFD 解压后总大小上限 |
| `PARSER_MAX_OFD_ENTRIES` | `10000` | OFD ZIP 条目数量上限 |
| `PARSER_MAX_OFD_COMPRESSION_RATIO` | `100` | OFD 条目最大压缩比 |
| `PARSER_MAX_XLS_SHEETS` | `100` | XLS / XLSX 最大工作表数 |
| `PARSER_MAX_XLS_ROWS_PER_SHEET` | `10000` | 每个工作表最大行数 |
| `PARSER_MAX_XLS_CELLS_PER_ROW` | `256` | 每行最大单元格数 |
| `PARSER_MAX_PPT_SLIDES` | `1000` | PPT / PPTX 最大幻灯片数 |
| `PARSER_MAX_PDF_PAGES` | `1000` | PDF 最大文字层解析页数 |

生产环境应设置高强度 `PARSER_API_KEY`，只通过 Docker 内网向 Go 服务开放本服务，不要直接暴露公网。

## 安全与边界

- 每次请求使用独立临时目录，并在成功或失败后递归清理。
- OFD 在交给 OFDRW 前检查条目路径、重复条目、条目数、实际解压大小和压缩比。
- POI 设置全局单次字节数组分配上限；Excel、PowerPoint、PDF 和输出文本另有数量上限。
- 服务按扩展名路由，底层解析器仍会验证实际容器；伪造或损坏文件返回 HTTP 422。
- 加密、带密码、厂商私有扩展或严重损坏的文件可能无法解析。
- 解析结果用于信息提取，不改变原始附件；带电子签章的 OFD 应保留原件作为法律凭据。

## 依赖与许可证

- Spring Boot：Apache-2.0
- Apache POI：Apache-2.0
- Apache PDFBox：Apache-2.0
- OFDRW：Apache-2.0

依赖版本均固定在 `pom.xml`。本服务随 AuraOA 主项目使用 MIT 许可证。
