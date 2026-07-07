/**
 * 泛微 Ecology9 — AuraOA 嵌入页：向 iframe 传递 requestid
 *
 * 使用步骤：
 * 1. 修改下方 AURA_EMBED_ORIGIN、IFRAME_IDS
 * 2. 上传到 OA 静态目录（如 /oa-front/workflow/xxx/aura-embed-notify.js）
 * 3. 流程 → 基础设置 → 自定义页面 → 填入 js 路径并启用
 * 4. 表单/门户 HTML 中 iframe 的 id 与 IFRAME_IDS 一致
 */
(function () {
  // ========== 按需修改 ==========
  var AURA_EMBED_ORIGIN = 'https://aura.example.com'; // AuraOA 前端地址（无末尾斜杠）

  // OA 页面中「需要接收 requestid」的 AuraOA iframe id 列表。
  // 脚本会把 WfForm.getBaseInfo().requestid 同时发送给这些 iframe。
  //
  // 只嵌入审核页时：
  //   var IFRAME_IDS = ['aura-embed-audit'];
  // 只嵌入总结页时：
  //   var IFRAME_IDS = ['aura-embed-summary'];
  // 同一页面同时嵌入审核 + 总结时：
  //   var IFRAME_IDS = ['aura-embed-audit', 'aura-embed-summary'];
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

  function postRequestIdToAura(win) {
    if (!win) return;
    var requestid = getRequestId();
    if (!requestid) return;
    win.postMessage({
      type: MSG_REQUESTID,
      requestid: requestid
    }, AURA_EMBED_ORIGIN);
  }

  function notifyAuraIframes() {
    getIframes().forEach(function (iframe) {
      if (iframe && iframe.contentWindow) {
        postRequestIdToAura(iframe.contentWindow);
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
      if (event.source) {
        event.source.postMessage({
          type: MSG_REQUESTID,
          requestid: requestid
        }, event.origin);
      }
    });
  }

  function init() {
    initMessageListener();

    getIframes().forEach(function (iframe) {
      iframe.addEventListener('load', notifyAuraIframes);
    });

    // WfForm 可能晚于 iframe load，轮询几次
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
