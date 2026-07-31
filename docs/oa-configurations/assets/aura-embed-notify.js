/**
 * 泛微 Ecology9 — AuraOA 嵌入页：传递打开上下文，并在 OA 操作完成后安排后台检查
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
  var MSG_REFRESH_EVENT = 'aura-oa-refresh-event';
  var MSG_RUNNER_READY = 'aura-runner-ready';
  var MSG_RUNNER_EVENT_ACK = 'aura-runner-event-ack';
  var RUNNER_IFRAME_ID = 'aura-embed-runner';
  var runnerReady = false;
  var pendingRunnerActions = [];
  var pendingRunnerAcks = {};
  var oaEventsRegistered = false;

  function getRequestId() {
    try {
      if (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) {
        var base = WfForm.getBaseInfo();
        var requestid = base && base.requestid != null ? String(base.requestid).trim() : '';
        if (requestid &&
            requestid !== '-1' &&
            requestid !== '0' &&
            requestid.toLowerCase() !== 'null' &&
            requestid.toLowerCase() !== 'undefined') {
          return requestid;
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

  function getRunnerIframe() {
    return document.getElementById(RUNNER_IFRAME_ID);
  }

  function ensureRunnerIframe() {
    var existing = getRunnerIframe();
    if (existing) return existing;
    var iframe = document.createElement('iframe');
    iframe.id = RUNNER_IFRAME_ID;
    iframe.src = AURA_EMBED_ORIGIN + '/embed/runner';
    iframe.setAttribute('aria-hidden', 'true');
    iframe.setAttribute('tabindex', '-1');
    iframe.style.cssText = 'position:fixed;width:1px;height:1px;opacity:0;pointer-events:none;border:0;left:-9999px;top:-9999px;';
    document.body.appendChild(iframe);
    return iframe;
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
    var runner = getRunnerIframe();
    if (runner && runner.contentWindow) {
      postContextToAura(runner.contentWindow);
    }
  }

  function createEventId() {
    return 'oa-' + Date.now() + '-' + Math.random().toString(16).slice(2);
  }

  function postRunnerAction(action, eventId) {
    var requestid = getRequestId();
    var runner = ensureRunnerIframe();
    if (!requestid || !runner || !runner.contentWindow) return false;
    runner.contentWindow.postMessage({
      type: MSG_REFRESH_EVENT,
      requestid: requestid,
      action: action || 'save_complete',
      event_id: eventId || createEventId()
    }, AURA_EMBED_ORIGIN);
    return true;
  }

  function notifyAuraRunner(action, eventId) {
    if (!getRequestId()) return false;
    if (!runnerReady) {
      pendingRunnerActions.push({
        action: action || 'save_complete',
        eventId: eventId || createEventId()
      });
      ensureRunnerIframe();
      return true;
    }
    return postRunnerAction(action, eventId);
  }

  function flushRunnerActions() {
    var actions = pendingRunnerActions.splice(0);
    actions.forEach(function (item) {
      postRunnerAction(item.action, item.eventId);
    });
  }

  function notifyBeforeRelease(action, eventId, callback) {
    var released = false;
    var release = function () {
      if (released) return;
      released = true;
      delete pendingRunnerAcks[eventId];
      callback();
    };
    if (!getRequestId()) {
      release();
      return;
    }
    pendingRunnerAcks[eventId] = release;
    setTimeout(release, 400);
    try {
      if (!notifyAuraRunner(action, eventId)) release();
    } catch (e) {
      release();
    }
  }

  function registerOAEvents() {
    if (oaEventsRegistered) return true;
    try {
      if (typeof WfForm === 'undefined' || !WfForm.registerCheckEvent) return false;
      if (typeof WfForm.OPER_SAVECOMPLETE === 'undefined') return false;
      WfForm.registerCheckEvent(WfForm.OPER_SAVECOMPLETE, function (callback) {
        notifyBeforeRelease('save_complete', createEventId(), callback);
      });
      oaEventsRegistered = true;
      return true;
    } catch (e) {
      console.warn('[aura-embed] 注册 OA 操作完成事件失败', e);
      return false;
    }
  }

  function initMessageListener() {
    window.addEventListener('message', function (event) {
      if (event.origin !== AURA_EMBED_ORIGIN) return;
      if (!event.data) return;
      if (event.data.type === MSG_RUNNER_EVENT_ACK) {
        var release = pendingRunnerAcks[String(event.data.event_id || '')];
        if (release) release();
        return;
      }
      if (event.data.type === MSG_RUNNER_READY) {
        runnerReady = true;
        if (event.source) postContextToAura(event.source);
        flushRunnerActions();
        return;
      }
      if (event.data.type !== MSG_REQUEST) return;

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
    ensureRunnerIframe();

    getIframes().forEach(function (iframe) {
      iframe.addEventListener('load', notifyAuraIframes);
    });

    var tries = 0;
    var timer = setInterval(function () {
      tries++;
      notifyAuraIframes();
      registerOAEvents();
      if (oaEventsRegistered || tries >= 200) {
        clearInterval(timer);
      }
    }, 300);

    window.addEventListener('hashchange', function () {
      notifyAuraIframes();
    });
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
