# OA Basic 单点登录接入说明

AuraOA 的 Basic 单点登录采用 HTTP Basic Authentication 作为受信任系统间的换址凭据。OA 已经完成用户认证后，由 OA 服务端取得当前 `loginid`，换取 AuraOA 的一次性浏览器登录地址。共享密码、Access Token 和 Refresh Token 不进入 OA 前端或跳转 URL。

## 1. 登录流程

```text
用户已登录 OA
  → OA 服务端取得当前 loginid
  → OA 服务端携带 Basic 凭据调用 AuraOA basic-redirection
  → AuraOA 校验共享密码、来源白名单、租户、用户和入口角色
  → AuraOA 在 Redis 写入 60 秒、仅可消费一次的交接码
  → AuraOA 返回 302 Location
  → OA 服务端禁止自动跟随 302，只读取 Location
  → OA 将用户浏览器重定向到 Location
  → 浏览器消费交接码并进入 AuraOA 工作台
```

Basic 认证只证明外部用户身份。AuraOA 不自动创建用户，也不会接受外部系统传入业务权限；用户必须已存在，并在对应租户具有 `business` 或 `tenant_admin` 角色。

## 2. AuraOA 租户配置

系统管理员进入“系统管理 → 租户管理 → 租户配置 → 单点登录”，填写：

| 配置 | 说明 |
|------|------|
| 启用 Basic 单点登录 | 开启该租户的换址入口 |
| 共享密码 | OA 服务端与 AuraOA 共同持有，AuraOA 使用 AES-GCM 加密保存且不回显 |
| 允许 IP / CIDR | OA 服务端或网关在 AuraOA 看见的来源地址；多个值用英文逗号分隔 |
| 允许来源域名 | OA 请求的 `Origin` / `Referer` 主机；多个值用英文逗号分隔 |

首次联调可暂时留空白名单。生产环境应使用 HTTPS，至少配置来源 IP，并设置 `SSO_PUBLIC_BASE_URL=https://auraoa.example.com`，保证生成的 `Location` 是浏览器可访问的正式地址。

## 3. 换址协议

```http
GET /api/auth/sso/basic-redirection?portal=business
Authorization: Basic base64(tenantCode/username:sharedPassword)
Origin: https://oa.example.com
```

例如租户编码为 `GJCW`，OA 用户为 `zhangsan`，则参与 Base64 编码的原始凭据是：

```text
GJCW/zhangsan:租户共享密码
```

成功响应：

```http
HTTP/1.1 302 Found
Cache-Control: no-store
Location: https://auraoa.example.com/api/auth/sso/basic-consume?code=一次性交接码
```

`portal` 支持 `business`（默认）和 `tenant_admin`，不支持 `system_admin`。

## 4. 泛微 E9 `com.api.ai` 示例

以下代码结构与 `sinomach_-zs/src/main/java/com/api/ai/AgentPlatformAPI.java` 保持一致。生产配置应放在 OA 安全配置中，不要将真实共享密码提交到仓库。

```java
package com.api.ai;

import com.engine.workflow.util.CommonUtil;
import weaver.hrm.User;

import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.MediaType;
import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.URL;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.util.Base64;

@Path("AuraOAAPI")
public class AuraOAAPI {
    private static String AURAOA_BASIC_REDIRECTION_URL =
        "https://auraoa.example.com/api/auth/sso/basic-redirection";
    private static String AURAOA_TENANT_CODE = "GJCW";
    private static String AURAOA_SHARED_PASSWORD = "请从安全配置读取";
    private static String AURAOA_PORTAL = "business";
    private static String AURAOA_SOURCE_ORIGIN = "https://oa.example.com";

    @GET
    @Path("getToken")
    @Produces(MediaType.TEXT_HTML)
    public void getToken(
        @Context HttpServletRequest request,
        @Context HttpServletResponse response
    ) throws IOException {
        User user = CommonUtil.getUserByRequest(request, response);
        String loginid = user == null ? "" : user.getLoginid();
        if (loginid == null || loginid.trim().isEmpty()) {
            response.sendError(HttpServletResponse.SC_UNAUTHORIZED, "无法识别当前 OA 用户，请重新登录 OA");
            return;
        }
        try {
            response.sendRedirect(requestBrowserLoginUrl(loginid.trim()));
        } catch (IOException exception) {
            // 日志禁止输出 Authorization、共享密码、交接码或 AuraOA Token。
            response.sendError(HttpServletResponse.SC_BAD_GATEWAY, "暂时无法进入 AuraOA，请联系管理员");
        }
    }

    private static String requestBrowserLoginUrl(String username) throws IOException {
        String separator = AURAOA_BASIC_REDIRECTION_URL.contains("?") ? "&" : "?";
        String requestUrl = AURAOA_BASIC_REDIRECTION_URL + separator + "portal="
            + URLEncoder.encode(AURAOA_PORTAL, "UTF-8");
        HttpURLConnection connection = null;
        try {
            connection = (HttpURLConnection) new URL(requestUrl).openConnection();
            connection.setRequestMethod("GET");
            connection.setConnectTimeout(3000);
            connection.setReadTimeout(5000);
            // 必须禁止自动跟随，否则 OA 服务端会先消费一次性交接码。
            connection.setInstanceFollowRedirects(false);
            connection.setRequestProperty("Authorization", basicAuthorization(username));
            connection.setRequestProperty("Accept", "text/html");
            if (!AURAOA_SOURCE_ORIGIN.trim().isEmpty()) {
                connection.setRequestProperty("Origin", AURAOA_SOURCE_ORIGIN.trim());
            }
            int status = connection.getResponseCode();
            String location = connection.getHeaderField("Location");
            if (status == HttpURLConnection.HTTP_MOVED_TEMP
                && location != null && !location.trim().isEmpty()) {
                return location;
            }
            throw new IOException("AuraOA Basic 单点换址失败，HTTP 状态：" + status);
        } finally {
            if (connection != null) connection.disconnect();
        }
    }

    private static String basicAuthorization(String username) {
        String credential = AURAOA_TENANT_CODE.trim() + "/" + username.trim()
            + ":" + AURAOA_SHARED_PASSWORD;
        return "Basic " + Base64.getEncoder().encodeToString(
            credential.getBytes(StandardCharsets.UTF_8)
        );
    }
}
```

## 5. 联调

先使用服务端命令确认 AuraOA 返回 302，不要自动跟随：

```bash
curl -i --max-redirs 0 \
  -u 'GJCW/zhangsan:租户共享密码' \
  -H 'Origin: https://oa.example.com' \
  'https://auraoa.example.com/api/auth/sso/basic-redirection?portal=business'
```

将响应 `Location` 在 60 秒内复制到浏览器打开。该地址只能使用一次。

常见失败：

| 状态 | 原因 |
|------|------|
| `401` | Basic 头缺失、共享密码错误、租户未启用或交接码失效 |
| `403` | 来源不在白名单、用户不存在/停用或没有请求的租户入口角色 |
| 返回 302 后地址失效 | OA HTTP 客户端自动跟随了重定向，或地址已超过 60 秒/被打开过 |
| 登录后仍回到登录页 | `SSO_PUBLIC_BASE_URL` 与实际浏览器访问 Origin 不一致 |

Redis 是一次性交接码的必要依赖。日志中禁止记录共享密码、完整 Authorization、交接码、Access Token 或 Refresh Token。
