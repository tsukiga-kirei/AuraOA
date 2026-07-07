/**
 * 泛微 Ecology9 — AuraOA 嵌入页：向 iframe 传递 requestid + embed_token
 *
 * 使用步骤：
 * 1. 在 AuraOA「系统管理 → 租户管理 → OA 嵌入」为租户生成嵌入密钥
 * 2. 修改下方 AURA_EMBED_ORIGIN、EMBED_ACCESS_TOKEN、IFRAME_IDS
 * 3. 上传到 OA 静态目录（如 /oa-front/workflow/xxx/aura-embed-notify.js）
 * 4. 流程 → 基础设置 → 自定义页面 → 填入 js 路径并启用
 * 5. 表单/门户 HTML 中 iframe 的 id 与 IFRAME_IDS 一致
 */
(function () {
  // ========== 按需修改 ==========
  var AURA_EMBED_ORIGIN = 'https://aura.example.com'; // AuraOA 前端地址（无末尾斜杠）

  // 租户管理 → OA 嵌入 → 生成/重置密钥后复制（仅显示一次）
  var EMBED_ACCESS_TOKEN = 'aura_emb_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx';

  // OA 页面中「需要接收 requestid」的 AuraOA iframe id 列表。
  var IFRAME_IDS = ['aura-embed-audit', 'aura-embed-summary'];
  // ==============================

  var MSG_REQUEST = 'aura-oa-request-requestid';
  var MSG_REQUESTID = 'aura-oa-requestid';

  function getRequestId() {
    try {
      if (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) {
        var base = WfForm.getBaseInfo();
        if (base && base.requestid) {
          return String(base.requestid);
        }
      }
    } catch (e) {
      console.warn('[aura-embed] WfForm.getBaseInfo 失败', e);
    }
    return '';
  }

  function getIframes() {
    var seen = {};
    var ids = Array.isArray(IFRAME_IDS) && IFRAME_IDS.length ? IFRAME_IDS : [];
    return ids.map(function (id) {
      if (!id || seen[id]) return null;
      seen[id] = true;
      return document.getElementById(id);
    }).filter(Boolean);
  }

  function buildPayload(requestid) {
    return {
      type: MSG_REQUESTID,
      requestid: requestid,
      embed_token: EMBED_ACCESS_TOKEN
    };
  }

  function postContextToAura(win) {
    if (!win) return;
    var requestid = getRequestId();
    if (!requestid) return;
    if (!EMBED_ACCESS_TOKEN) {
      console.warn('[aura-embed] 未配置 EMBED_ACCESS_TOKEN');
      return;
    }
    win.postMessage(buildPayload(requestid), AURA_EMBED_ORIGIN);
  }

  function notifyAuraIframes() {
    getIframes().forEach(function (iframe) {
      if (iframe && iframe.contentWindow) {
        postContextToAura(iframe.contentWindow);
      }
    });
  }

  function initMessageListener() {
    window.addEventListener('message', function (event) {
      if (event.origin !== AURA_EMBED_ORIGIN) return;
      if (!event.data || event.data.type !== MSG_REQUEST) return;

      var requestid = getRequestId();
      if (!requestid) {
        console.warn('[aura-embed] 无 requestid，请确认流程表单已加载 WfForm');
        return;
      }
      if (!EMBED_ACCESS_TOKEN) {
        console.warn('[aura-embed] 未配置 EMBED_ACCESS_TOKEN');
        return;
      }
      if (event.source) {
        event.source.postMessage(buildPayload(requestid), event.origin);
      }
    });
  }

  function init() {
    initMessageListener();

    getIframes().forEach(function (iframe) {
      iframe.addEventListener('load', notifyAuraIframes);
    });

    var tries = 0;
    var timer = setInterval(function () {
      tries++;
      notifyAuraIframes();
      if (getRequestId() || tries >= 30) {
        clearInterval(timer);
      }
    }, 300);

    window.addEventListener('hashchange', notifyAuraIframes);
  }

  if (typeof jQuery !== 'undefined') {
    jQuery(function () {
      init();
    });
  } else if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
