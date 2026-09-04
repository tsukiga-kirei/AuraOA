/**
 * 泛微 Ecology9 — AuraOA 嵌入页：传递打开上下文，并在点击保存、提交时安排后台检查
 *
 * 使用步骤：
 * 1. 在 AuraOA「系统管理 → 租户管理 → OA 嵌入」为租户生成嵌入密钥
 * 2. 修改下方 AURA_EMBED_ORIGIN、EMBED_ACCESS_TOKEN、IFRAME_IDS
 * 3. 上传到 OA 静态目录（如 /oa-front/workflow/xxx/aura-embed-notify.js）
 * 4. 流程 → 基础设置 → 自定义页面 → 填入 js 路径并启用
 * 5. 表单/门户 HTML 中 iframe 的 id 与 IFRAME_IDS 一致
 */
(function () {
  console.log('[aura-embed] 脚本已加载');

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

  function captureOperationContext(action) {
    var occurredAtMs = Date.now();
    var base = WfForm.getBaseInfo() || {};
    var store = WfForm.getGlobalStore();
    var currentUserId = store && store.commonParam && store.commonParam.currentUserid != null
      ? String(store.commonParam.currentUserid).trim()
      : '';
    return {
      action: action,
      event_id: createEventId(),
      occurred_at_ms: occurredAtMs,
      requestid: getRequestId(),
      workflow_id: base.workflowid != null ? String(base.workflowid).trim() : '',
      oa_belong_user_id: base.f_weaver_belongto_userid != null
        ? String(base.f_weaver_belongto_userid).trim()
        : '',
      oa_current_user_id: currentUserId
    };
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

  function getCurrentUserId() {
    try {
      if (typeof WfForm !== 'undefined' && WfForm.getGlobalStore) {
        var store = WfForm.getGlobalStore();
        if (store && store.commonParam && store.commonParam.currentUserid != null) {
          return String(store.commonParam.currentUserid).trim();
        }
      }
    } catch (e) {}
    return '';
  }

  function buildPayload(requestid) {
    return {
      type: MSG_REQUESTID,
      requestid: requestid,
      embed_token: EMBED_ACCESS_TOKEN,
      oa_current_user_id: getCurrentUserId()
    };
  }

  function postContextToAura(win) {
    if (!win) return;
    var requestid = getRequestId();
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

  function createEventId() {
    return 'oa-' + Date.now() + '-' + Math.random().toString(16).slice(2);
  }

  function buildEventBody(context) {
    return [
      ['embed_token', EMBED_ACCESS_TOKEN],
      ['process_id', context.requestid],
      ['workflow_id', context.workflow_id],
      ['oa_belong_user_id', context.oa_belong_user_id],
      ['oa_current_user_id', context.oa_current_user_id],
      ['occurred_at_ms', String(context.occurred_at_ms)],
      ['action', context.action],
      ['event_id', context.event_id]
    ].map(function (item) {
      return encodeURIComponent(item[0]) + '=' + encodeURIComponent(item[1] || '');
    }).join('&');
  }

  function notifyBeforeRelease(action, callback) {
    var released = false;
    var timeoutId = null;
    var release = function () {
      if (released) return;
      released = true;
      callback();
    };
    var context = captureOperationContext(action);
    console.log('[aura-embed] OA 操作事件已触发', {
      action: context.action,
      requestid: context.requestid || '(待解析)',
      workflow_id: context.workflow_id,
      event_id: context.event_id
    });
    timeoutId = setTimeout(function () {
      console.warn('[aura-embed] OA 操作事件提交超时，已放行 OA', {
        action: context.action,
        event_id: context.event_id
      });
      release();
    }, 800);
    try {
      var request = fetch(AURA_EMBED_ORIGIN + '/api/embed/events', {
        method: 'POST',
        mode: 'no-cors',
        credentials: 'omit',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8'
        },
        body: buildEventBody(context)
      });
      Promise.resolve(request).then(function () {
        clearTimeout(timeoutId);
        console.log('[aura-embed] OA 操作事件已提交', {
          action: context.action,
          requestid: context.requestid || '(待解析)',
          workflow_id: context.workflow_id,
          event_id: context.event_id
        });
        release();
      }, function (error) {
        clearTimeout(timeoutId);
        console.warn('[aura-embed] OA 操作事件提交失败，已放行 OA', error);
        release();
      });
    } catch (e) {
      clearTimeout(timeoutId);
      console.warn('[aura-embed] OA 操作事件提交失败，已放行 OA', e);
      release();
    }
  }

  function registerOAEvents() {
    WfForm.registerCheckEvent(WfForm.OPER_SAVE, function (callback) {
      notifyBeforeRelease('save_requested', callback);
    });
    WfForm.registerCheckEvent(WfForm.OPER_SUBMIT, function (callback) {
      notifyBeforeRelease('submit_requested', callback);
    });
    console.log('[aura-embed] 已注册 OA 保存/提交事件');
  }

  function initMessageListener() {
    window.addEventListener('message', function (event) {
      if (event.origin !== AURA_EMBED_ORIGIN) return;
      if (!event.data) return;
      if (event.data.type !== MSG_REQUEST) return;

      var requestid = getRequestId();
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
    console.log('[aura-embed] 脚本已初始化');
    initMessageListener();

    getIframes().forEach(function (iframe) {
      iframe.addEventListener('load', notifyAuraIframes);
    });

    var contextTries = 0;
    var contextTimer = setInterval(function () {
      contextTries++;
      notifyAuraIframes();
      if (getRequestId() || contextTries >= 200) {
        clearInterval(contextTimer);
      }
    }, 300);
    registerOAEvents();

    window.addEventListener('hashchange', function () {
      notifyAuraIframes();
    });
    window.addEventListener('popstate', function () {
      notifyAuraIframes();
    });
  }

  jQuery().ready(function () {
    init();
  });
})();
