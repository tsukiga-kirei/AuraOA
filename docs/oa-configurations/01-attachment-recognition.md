# 01 - 附件识别（Attachment Recognition）

> 说明：本能力已在主干落地，原已知问题 `001-流程附件未被AI识别` 已关闭。
>
> 关联代码：
> - 后端服务：[`go-service/internal/service/attachment_recognition_service.go`](../../go-service/internal/service/attachment_recognition_service.go)
> - 泛微适配器：[`go-service/internal/pkg/oa/ecology9.go`](../../go-service/internal/pkg/oa/ecology9.go)（`FetchProcessData` / `recognizeMainAttachments`）
> - 数据库迁移：[`db/migrations/000033_attachment_recognition_configs.up.sql`](../../db/migrations/000033_attachment_recognition_configs.up.sql)、[`db/migrations/000034_attachment_recognition_extended_configs.up.sql`](../../db/migrations/000034_attachment_recognition_extended_configs.up.sql)

## 背景

当流程涉及合同 / 发票 / 报价单 / 扫描件等关键证据存在附件时，仅靠主表字段送给 AI 会漏掉关键信息。AuraOA 需要：

1. 从 OA 数据库里识别出**附件类型字段**（泛微 E9 中 `workflow_billfield.fieldhtmltype = 6`）。
2. 通过 OA 系统暴露的**REST 接口**根据 `docId` 列表取最新版本的附件二进制（base64）。
3. 把 base64 上传到 [MinerU](https://github.com/opendatalab/MinerU)，得到结构化文本。
4. 把附件文本拼到 AI prompt 的 `{{attachments}}` 占位符。

## 整体架构

```
┌─────────────┐  ① 取附件二进制流  ┌──────────────┐  ② 解析正文      ┌──────────────┐
│  OA 系统     │ ─────────────────▶│  AuraOA       │ ───────────────▶│   MinerU     │
│ (Weaver E9) │                    │ 附件识别服务   │                  │ 文档解析服务  │
└─────────────┘                    └──────────────┘                  └──────────────┘
```

| 角色 | 职责 |
|------|------|
| OA 系统 | 提供按 `docIds` 批量取最新版本附件 base64 的 REST 接口 |
| AuraOA | 在 `FetchProcessData` 里识别附件字段，调用 OA 接口取流，再调用 MinerU 解析正文，结果作为 `ProcessData.Attachments` 传给 prompt builder |
| MinerU | 实际做 PDF / 图片 / Office / OCR / 表格 / 公式的提取 |

> **「最新版本」由 OA 系统侧决定**：附件版本表 `docimagefile.versionid` 在 OA 数据库里维护；AuraOA 不直接读它，而是要求 OA 接口在返回时已挑选好最新版本（`SELECT ... ORDER BY versionid DESC LIMIT 1`）。这样选版本逻辑跟 OA 厂商绑定，未来换 OA 不需要改 AuraOA。

## 在 AuraOA 里配置

进入「系统设置 → 附件识别」面板，配置以下项（对应 `system_configs` 表 key）：

### 基础开关

| 字段 | system_configs key | 默认值 | 说明 |
|------|--------------------|--------|------|
| 启用附件识别 | `attachment.recognition_enabled` | `false` | 关闭时所有 OA 表单字段照常送入 AI，但跳过附件识别 |
| 最大文件大小（MB） | `attachment.max_file_size_mb` | `10` | 超出大小的文件以「已跳过」标记进入 prompt，不会下发到 MinerU |
| 支持的文件类型 | `attachment.supported_types` | `pdf,png,jpg,jpeg,bmp,gif,tiff,webp,docx,xlsx,txt` | 逗号分隔；不在白名单的扩展名按「已跳过」处理 |

### MinerU 解析服务

| 字段 | system_configs key | 默认值 | 说明 |
|------|--------------------|--------|------|
| MinerU 端点 | `attachment.mineru_endpoint` | _(空)_ | 自建 MinerU 服务的根地址，**不要带尾部 `/`**；后端会自动拼 `/file_parse`、`/health`，必要时按返回的 `result_url` 继续拉取任务结果 |
| API Key（可选） | `attachment.mineru_api_key` | _(空)_ | MinerU 服务若开启了 Bearer 鉴权才需要配置 |
| Backend | `attachment.mineru_backend` | `pipeline` | 取值：`pipeline` / `vlm-auto-engine` / `vlm-http-client` / `hybrid-auto-engine` / `hybrid-http-client` |
| 公式识别 | `attachment.mineru_enable_formula` | `true` | |
| 表格识别 | `attachment.mineru_enable_table` | `true` | |
| 解析方式 | `attachment.mineru_parse_method` | `ocr` | `auto`（自动）/ `txt`（文本提取）/ `ocr`（OCR 识别）；旧版 `attachment.mineru_enable_ocr` 仍可读作兼容 |
| 解析语言 | `attachment.mineru_language` | `ch` | 与 MinerU 服务支持的语言列表保持一致 |

> **测试连接**仅探测 `GET {mineru_endpoint}/health`，不会真实调用 `/file_parse` 解析文件。当前适配同时兼容两类 MinerU 返回：一类直接在同步响应中返回 Markdown，另一类先返回已完成任务摘要，再通过 `result_url` 拉取最终 Markdown。

### OA 系统附件接口（按 OA 适配器单独配置）

从当前版本开始，OA 附件接口参数不再走 `system_configs` 的通用认证项，而是由每个 OA 适配器单独处理。  
目前仅接入 `ecology9`，需在 OA 连接里配置 `weaver_api_url + weaver_appid + weaver_default_user`。

## OA 系统侧需要做什么

### 通用契约

OA 系统需要暴露一个**只接受 `docIds` 列表、返回每个附件最新版本 base64** 的 REST 接口：

```
GET {weaver_api_url}?docIds=123,124,125&appid=<appid>&loginid=<loginid>
（可附带任意自定义鉴权头）

返回：
{
  "code": 0,
  "msg": "ok",
  "data": [
    { "docId": "123", "fileName": "合同.pdf",  "fileSize": 1024000, "fileData": "<base64>" },
    { "docId": "124", "fileName": "发票.jpg",  "fileSize":  512000, "fileData": "<base64>" }
  ]
}
```

约束：

- `docId` 字符串原样回填
- `fileSize` 单位：字节
- `fileData` 必须是 base64（标准编码，不带前缀）
- **必须挑选最新版本**：`docimagefile` 表里同一个 `docId` 可能有多版本，由 OA 实现侧 `ORDER BY versionid DESC LIMIT 1`

### OA 类型差异 — 泛微 Ecology9

泛微 E9 是目前**唯一**需要在 AuraOA 里维护**原生 API 密钥**的 OA。

#### 在 AuraOA 里配置

进入「系统设置 → OA 数据库 → 编辑/新建」，**OA 类型选择「泛微 E9」时**会显示三个字段（保存到 `oa_database_connections` 表）：

| 字段 | 数据库列 | 说明 |
|------|----------|------|
| 附件接口 URL | `weaver_api_url` | 泛微 E9 附件接口地址，形如 `http://oa.example.com/api/aurabridge/attachments` |
| 应用 ID | `weaver_appid` | 泛微 E9 注册应用时分配的 appid |
| 默认调用用户 | `weaver_default_user` | 调泛微接口默认使用的 `loginid`（一般使用具备附件读权限的运维账号） |

> 这些字段属于「这个 OA 实例的密钥」，跟 `oa_database_connections` 一对一，因此放在该表里而非 `system_configs`。换租户、换 OA 数据库时不会串号。

#### 泛微 E9 自建附件接口示例

参考泛微开放平台「自定义 REST 接口注册」流程：<https://e-cloudstore.com/doc.html?appId=af09c25938714c26b9736f535ca20fc9>

下面是一个最小可用实现，供 OA 实施侧参考。`docId` → 最新 `imagefileid` → `imagefile` 二进制流 → base64：

```java
// 文件：weaver/aurabridge/AttachmentRest.java
package weaver.aurabridge;

import com.alibaba.fastjson.JSONArray;
import com.alibaba.fastjson.JSONObject;
import org.apache.commons.logging.Log;
import org.apache.commons.logging.LogFactory;
import weaver.conn.RecordSet;
import weaver.file.ImageFileManager;

import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.MediaType;
import java.io.ByteArrayOutputStream;
import java.io.InputStream;
import java.util.Base64;

@Path("/aurabridge")
public class AttachmentRest {

    private static final Log logger = LogFactory.getLog(AttachmentRest.class.getName());

    @GET
    @Path("/attachments")
    @Produces(MediaType.APPLICATION_JSON)
    public String getAttachments(@QueryParam("docIds") String docIds) {
        JSONObject result = new JSONObject();
        JSONArray data = new JSONArray();
        logger.info("[aurabridge] getAttachments called, docIds=" + docIds);

        if (docIds == null || docIds.trim().isEmpty()) {
            logger.warn("[aurabridge] docIds is empty");
            result.put("code", 0);
            result.put("data", data);
            return result.toJSONString();
        }

        for (String docId : docIds.split(",")) {
            docId = docId.trim();
            if (docId.isEmpty()) {
                continue;
            }
            try {
                logger.info("[aurabridge] processing docId=" + docId);

                // 1. 选最新版本：versionid 最大的一条
                RecordSet rs = new RecordSet();
                rs.executeQuery(
                    "SELECT imagefileid FROM docimagefile WHERE docid = ? ORDER BY versionid DESC",
                    docId
                );
                if (!rs.next()) {
                    logger.warn("[aurabridge] docimagefile not found, docId=" + docId);
                    continue;
                }
                int imageFileId = rs.getInt("imagefileid");
                logger.info("[aurabridge] docId=" + docId + " -> imagefileid=" + imageFileId);

                // 2. 取文件名 / 大小
                RecordSet rs2 = new RecordSet();
                rs2.executeQuery(
                    // 达梦/Oracle 物理列名为 FILESIZE（勿写成 imagefilesize）
                    "SELECT imagefilename, filesize FROM imagefile WHERE imagefileid = ?",
                    imageFileId
                );
                if (!rs2.next()) {
                    logger.warn("[aurabridge] imagefile not found, docId=" + docId + ", imagefileid=" + imageFileId);
                    continue;
                }
                String fileName = rs2.getString("imagefilename");
                logger.info("[aurabridge] docId=" + docId + ", fileName=" + fileName);

                // 3. 拿文件流
                InputStream is = ImageFileManager.getInputStreamById(imageFileId);
                if (is == null) {
                    logger.warn("[aurabridge] ImageFileManager returned null stream, docId=" + docId + ", imagefileid=" + imageFileId);
                    continue;
                }
                byte[] bytes = readAllBytes(is);
                is.close();
                if (bytes == null || bytes.length == 0) {
                    logger.warn("[aurabridge] empty file bytes, docId=" + docId + ", imagefileid=" + imageFileId);
                    continue;
                }

                JSONObject item = new JSONObject();
                item.put("docId", docId);
                item.put("fileName", fileName);
                item.put("fileSize", bytes.length);
                item.put("fileData", Base64.getEncoder().encodeToString(bytes));
                data.add(item);
                logger.info("[aurabridge] attachment ready, docId=" + docId + ", fileName=" + fileName + ", size=" + bytes.length);
            } catch (Exception e) {
                logger.error("[aurabridge] failed docId=" + docId, e);
            }
        }

        result.put("code", 0);
        result.put("msg", "ok");
        result.put("data", data);
        logger.info("[aurabridge] response data.size=" + data.size() + ", docIds=" + docIds);
        return result.toJSONString();
    }

    private static byte[] readAllBytes(InputStream is) throws Exception {
        ByteArrayOutputStream buf = new ByteArrayOutputStream();
        byte[] tmp = new byte[8192];
        int len;
        while ((len = is.read(tmp)) != -1) {
            buf.write(tmp, 0, len);
        }
        return buf.toByteArray();
    }
}
```

> **排查 `data` 为空**：AuraOA 日志若出现 `泛微附件接口返回成功` 且 `fileCount=0`，说明接口返回了 `code=0` 但 `data=[]`。请在泛微应用日志中搜索 `[aurabridge]`，常见原因是 `docimagefile` 无记录、`imagefile` 无记录，或 `ImageFileManager.getInputStreamById` 返回 `null`（上述分支会打出对应 WARN）。

> 当前版本调用约定：AuraOA 以 `docIds + appid + loginid` 作为请求参数调用 `weaver_api_url`，不使用通用认证配置。

#### 涉及的泛微表 / 字段（仅供参考）

| 表 | 关键字段 | 说明 |
|----|---------|------|
| `workflow_billfield` | `billid` / `fieldname` / `fieldhtmltype` | `fieldhtmltype = 6` 表示「附件上传」字段；用于识别哪些字段需要做附件提取 |
| `htmllabelinfo`     | `indexid` / `labelname` / `languageid` | 字段中文名（`languageid = 7` 是简体中文） |
| 主表 `formtable_main_xxx` | `requestid` + 各字段列 | 附件字段保存的是逗号分隔的 `docId` 列表 |
| `docimagefile`      | `docid` / `imagefileid` / `versionid` | 一个 `docid` 可有多版本，**取 `versionid` 最大的一条** |
| `imagefile`         | `imagefileid` / `imagefilename` / `filesize` | 附件文件名 / 大小；二进制由 `ImageFileManager.getInputStreamById` 取流 |

## 行为约定

- **未启用附件识别**：`Attachments` 字段为空切片，prompt 中 `{{attachments}}` 显示「（本流程未提取到附件内容）」。
- **OA 接口不可达**：附件识别整体失败时**不会**阻断主审核流程，只在日志里 `WARN` 一条；prompt 里 `{{attachments}}` 仍以占位文本呈现。
- **单个文件失败**：保留在 `Attachments` 里，但 `Error` 字段写明原因（不支持的类型 / 超大 / MinerU 报错），便于 AI 上下文里知晓有附件缺失。
- **明细表附件**：当前只识别**主表**附件字段；明细表附件如有需要，后续在 `recognizeMainAttachments` 旁边并行扩展即可。

## 维护与排查

1. 新接入一种 OA 系统时：
   - 在 `oa.NewOAAdapter` 工厂里增加分支，并按需把 `AttachmentRecognitionService` 注入到对应 adapter；
   - 在本目录新增一节，写清楚该 OA 类型自建接口的实现示例。
2. 调试时优先看 `app.log` 里的 `WARN`：
   - `调用 OA 附件接口失败`：检查 OA 连接中的 `weaver_api_url / weaver_appid / weaver_default_user`；
   - `MinerU 服务返回错误`：用「测试连接」按钮先确认 `/health`；再看 backend / language / OCR 配置；
   - `识别附件字段失败，跳过该字段`：通常是某条 `docId` 在 OA 库里被物理清理了，对照 `imagefile` 表确认。
