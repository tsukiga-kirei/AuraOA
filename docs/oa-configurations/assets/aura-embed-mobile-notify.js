/**
 * 泛微 Ecology9 移动端 — AuraOA 嵌入脚本（方案B：状态胶囊按钮 + 弹窗查看详情 + 变更感知）
 *
 * 适用环境：泛微 E9 移动端审批、EMobile、企业微信移动审批工作流
 *
 * 使用步骤：
 * 1. 在 AuraOA「系统管理 → 租户管理 → OA 嵌入」为租户生成嵌入密钥并导出移动端脚本
 * 2. 将本文件上传至 OA 静态目录（如 /oa-front/workflow/AuraOA/aura-embed-mobile-notify.js）
 * 3. 流程 → 基础设置 → 自定义页面（或移动端页面设置）填入 js 路径并启用
 * 4. 表单设计中可添加自定义 HTML 块 <div id="getMyBt"></div> 作为按钮挂载位（如无则自动以浮动方式挂载）
 */
(function () {
  console.log('[aura-embed-mobile] 移动端脚本已加载');

  // ========== 按需配置 ==========
  var AURA_EMBED_ORIGIN = 'https://aura.example.com'; // AuraOA 访问地址（末尾不加斜杠）
  var EMBED_ACCESS_TOKEN = 'aura_emb_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'; // 租户嵌入访问密钥
  var SHOW_ON_DESKTOP = false; // 是否在电脑端启用本脚本的按钮和弹窗；移动端不受影响
  var BUTTON_CONTAINER_ID = 'getMyBt'; // 表单设计器中预留的挂载容器 ID（与 oa-front 习惯一致）
  var EMBED_TYPE = 'audit'; // 嵌入类型：'audit'（AI 审核）或 'summary'（流程总结）
  // ==============================

  var MSG_STATUS = 'aura-oa-embed-status';
  var STATUS_POLL_MS = 3000;
  var STATUS_WATCH_MS = 180000;

  // 不按窗口宽度判断，避免电脑端窄侧栏被识别为移动端；兼容 iPad 桌面 UA。
  var mobileClient = /Android|iPhone|iPad|iPod|Windows Phone/i.test(navigator.userAgent || '')
    || (/Macintosh/i.test(navigator.userAgent || '') && navigator.maxTouchPoints > 1);
  if (!mobileClient && !SHOW_ON_DESKTOP) return;

  var statusWatchTimer = null;
  var statusWatchUntil = 0;
  var lastStatusSignature = '';
  var resumeRefreshTimer = null;

  function openDesktopDialog(targetUrl) {
    var existing = document.getElementById('auraDesktopEmbedDialog');
    if (existing) return;
    var previousFocus = document.activeElement;
    var overlay = document.createElement('div');
    overlay.id = 'auraDesktopEmbedDialog';
    overlay.style.cssText = 'position:fixed;inset:0;z-index:2147483646;background:rgba(0,0,0,.45);display:flex;align-items:center;justify-content:center;padding:12px;box-sizing:border-box;';
    var dialog = document.createElement('div');
    dialog.setAttribute('role', 'dialog');
    dialog.setAttribute('aria-modal', 'true');
    dialog.setAttribute('aria-labelledby', 'auraDesktopEmbedTitle');
    dialog.style.cssText = 'width:760px;max-width:100%;height:85vh;max-height:100%;background:#fff;border-radius:12px;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 16px 48px rgba(0,0,0,.2);';
    var header = document.createElement('div');
    header.style.cssText = 'display:flex;align-items:center;justify-content:space-between;gap:16px;padding:12px 16px;border-bottom:1px solid #eee;flex-shrink:0;';
    var title = document.createElement('span');
    title.id = 'auraDesktopEmbedTitle';
    title.textContent = EMBED_TYPE === 'summary' ? 'AI 流程总结' : 'AI 流程审核';
    var close = document.createElement('button');
    close.type = 'button';
    close.textContent = '关闭';
    close.style.cssText = 'cursor:pointer;padding:6px 12px;background:#fff;border:1px solid #ddd;border-radius:6px;';
    var frame = document.createElement('iframe');
    frame.title = title.textContent;
    frame.src = targetUrl;
    frame.style.cssText = 'width:100%;flex:1;min-height:0;border:0;display:block;';
    function dismiss() {
      document.removeEventListener('keydown', onKeyDown);
      overlay.remove();
      if (previousFocus && previousFocus.focus) previousFocus.focus();
      refreshCurrentStatus(true);
    }
    function onKeyDown(event) {
      if (event.key === 'Escape') dismiss();
    }
    close.onclick = dismiss;
    overlay.onclick = function (event) { if (event.target === overlay) dismiss(); };
    header.appendChild(title);
    header.appendChild(close);
    dialog.appendChild(header);
    dialog.appendChild(frame);
    overlay.appendChild(dialog);
    document.body.appendChild(overlay);
    document.addEventListener('keydown', onKeyDown);
    close.focus();
  }

  var statusConfig = {
    gray: { color: '#909399', text: 'AI审核详情', bg: '#f4f4f5', bgHover: '#e9e9eb' },
    green: { color: '#67C23A', text: '审核通过', bg: '#f0f9ff', bgHover: '#e1f3f8' },
    yellow: { color: '#E6A23C', text: '建议关注', bg: '#fdf6ec', bgHover: '#faecd8' },
    red: { color: '#F56C6C', text: '建议退回', bg: '#fef0f0', bgHover: '#fde2e2' },
    disabled: { color: '#c0c4cc', text: '暂不可用', bg: '#f5f7fa', bgHover: '#f5f7fa' },
    error: { color: '#f56c6c', text: '加载失败', bg: '#fef0f0', bgHover: '#fef0f0' },
    loading: { color: '#409eff', text: '分析中...', bg: '#ecf5ff', bgHover: '#ecf5ff' }
  };

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
      console.warn('[aura-embed-mobile] WfForm.getBaseInfo 失败', e);
    }
    return '';
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

  function captureOperationContext(action) {
    var occurredAtMs = Date.now();
    var base = (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) ? WfForm.getBaseInfo() || {} : {};
    return {
      action: action,
      event_id: 'oa-' + Date.now() + '-' + Math.random().toString(16).slice(2),
      occurred_at_ms: occurredAtMs,
      requestid: getRequestId(),
      workflow_id: base.workflowid != null ? String(base.workflowid).trim() : '',
      oa_current_user_id: getCurrentUserId()
    };
  }

  function buildEventBody(context) {
    return [
      ['embed_token', EMBED_ACCESS_TOKEN],
      ['process_id', context.requestid],
      ['workflow_id', context.workflow_id],
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
    var release = function () {
      if (released) return;
      released = true;
      startStatusWatch(STATUS_WATCH_MS);
      callback();
    };
    var context = captureOperationContext(action);
    var timeoutId = setTimeout(function () {
      console.warn('[aura-embed-mobile] OA 操作事件提交超时，已放行 OA', { action: context.action });
      release();
    }, 800);

    try {
      var request = fetch(AURA_EMBED_ORIGIN + '/api/embed/events', {
        method: 'POST',
        mode: 'no-cors',
        credentials: 'omit',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8' },
        body: buildEventBody(context)
      });
      Promise.resolve(request).then(function () {
        clearTimeout(timeoutId);
        release();
      }, function (err) {
        clearTimeout(timeoutId);
        console.warn('[aura-embed-mobile] OA 操作事件提交失败，已放行 OA', err);
        release();
      });
    } catch (e) {
      clearTimeout(timeoutId);
      release();
    }
  }

  function registerOAEvents() {
    if (typeof WfForm === 'undefined' || !WfForm.registerCheckEvent) return;
    WfForm.registerCheckEvent(WfForm.OPER_SAVE, function (callback) {
      notifyBeforeRelease('save_requested', callback);
    });
    WfForm.registerCheckEvent(WfForm.OPER_SUBMIT, function (callback) {
      notifyBeforeRelease('submit_requested', callback);
    });
    console.log('[aura-embed-mobile] 已注册 OA 保存/提交感知事件');
  }

  function openEmbedDialog(requestId, userId) {
    var path = EMBED_TYPE === 'summary' ? '/embed/summary' : '/embed/audit';
    var targetUrl = AURA_EMBED_ORIGIN + path
      + '?requestid=' + encodeURIComponent(requestId)
      + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);

    console.log('[aura-embed-mobile] 调起嵌入弹窗');
    startStatusWatch(STATUS_WATCH_MS);

    if (!mobileClient) {
      openDesktopDialog(targetUrl);
    } else if (window.weaJs && typeof window.weaJs.showDialog === 'function') {
      window.weaJs.showDialog(targetUrl, {
        title: EMBED_TYPE === 'summary' ? 'AI 流程总结' : 'AI 流程审核',
        moduleName: 'workflow',
        style: { width: '100%', height: '100%' },
        callback: function () { refreshCurrentStatus(true); }
      });
    } else {
      window.open(targetUrl, '_blank');
    }
  }

  function renderButton(status, textOverride, errorMessage, isLoading, onClick) {
    var currentStatus = isLoading ? statusConfig.loading : (statusConfig[status] || statusConfig.gray);
    var displayText = textOverride || currentStatus.text;
    var isDisabled = status === 'disabled' || status === 'error';
    var cursorStyle = isDisabled ? 'not-allowed' : 'pointer';
    var opacityValue = isDisabled ? '0.65' : '1';

    var btnHtml =
      '<div style="display: inline-flex; align-items: center; padding: 4px 0;">' +
        '<button id="auraMobileEmbedBtn" type="button" style="' +
          'display: inline-flex; align-items: center; justify-content: center;' +
          'white-space: nowrap; height: 28px; padding: 0 14px;' +
          'background-color: ' + currentStatus.bg + '; border: 1.5px solid ' + currentStatus.color + ';' +
          'border-radius: 16px; font-size: 13px; font-weight: 600;' +
          'color: ' + currentStatus.color + '; cursor: ' + cursorStyle + '; opacity: ' + opacityValue + ';' +
          'box-shadow: 0 1px 3px rgba(0,0,0,0.06); transition: all 0.2s ease;' +
        '">' +
          '<span style="' +
            'display: inline-block; width: 8px; height: 8px; border-radius: 50%;' +
            'background-color: ' + currentStatus.color + '; margin-right: 6px;' +
            (isLoading ? 'animation: auraBtnPulse 1.2s infinite ease-in-out;' : '') +
          '"></span>' +
          '<span class="aura-mobile-label"></span>' +
        '</button>' +
      '</div>';

    var $container = jQuery('#' + BUTTON_CONTAINER_ID);
    if (!$container.length) {
      if (!jQuery('#auraMobileEmbedFloatContainer').length) {
        jQuery('body').append('<div id="auraMobileEmbedFloatContainer" style="position:fixed;bottom:16px;left:16px;z-index:9999;"></div>');
      }
      $container = jQuery('#auraMobileEmbedFloatContainer');
    }

    $container.html(btnHtml);
    $container.find('.aura-mobile-label').text(displayText);

    jQuery('#auraMobileEmbedBtn').off('click').on('click', function () {
      if (isLoading) {
        if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
          WfForm.showMessage('数据加载中，请稍候...', 2, 2);
        }
        return;
      }
      if (isDisabled) {
        if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
          WfForm.showMessage(errorMessage || '当前节点暂不可用', 2, 3);
        }
        return;
      }
      if (typeof onClick === 'function') {
        onClick();
      }
    });
  }

  function featureName() {
    return EMBED_TYPE === 'summary' ? '总结' : '审核';
  }

  function openCurrentDetails() {
    var requestId = getRequestId();
    if (!requestId) return;
    openEmbedDialog(requestId, getCurrentUserId());
  }

  function statusSignature(status, text) {
    return status + '|' + (text || '');
  }

  function renderStatusButton(status, textOverride, errorMessage, isLoading, onClick) {
    var signature = statusSignature(status, textOverride || '') + '|' + (isLoading ? '1' : '0');
    if (signature === lastStatusSignature) return;
    lastStatusSignature = signature;
    renderButton(status, textOverride, errorMessage, isLoading, onClick);
  }

  function applyResultToButton(isSummary, result, hasResult, shouldAutoRun, runningJobId) {
    var name = featureName();
    var openDetails = openCurrentDetails;
    if (runningJobId) {
      renderStatusButton('gray', name + '分析中，查看进度', '', false, openDetails);
      return 'running';
    }
    if (hasResult && result) {
      if (result.status === 'failed' || result.status === 'cancelled' || result.parse_error) {
        renderStatusButton('red', name + '异常', 'AI ' + name + '分析出现异常', false, openDetails);
        return 'failed';
      }
      if (isSummary) {
        renderStatusButton('green', '查看流程总结', '', false, openDetails);
        return 'completed';
      }
      var rec = result.recommendation || 'review';
      var score = result.overall_score != null ? Math.round(Number(result.overall_score)) : null;
      var scoreText = score != null ? ' (' + score + '分)' : '';
      if (rec === 'approve') {
        renderStatusButton('green', '审核通过' + scoreText, '', false, openDetails);
      } else if (rec === 'return') {
        renderStatusButton('red', '建议退回' + scoreText, '', false, openDetails);
      } else {
        renderStatusButton('yellow', '建议关注' + scoreText, '', false, openDetails);
      }
      return 'completed';
    }
    renderStatusButton('gray', shouldAutoRun ? '查看并生成' + name : 'AI' + name + '详情', '', false, openDetails);
    return 'pending';
  }

  function applyContextData(data) {
    var isSummary = EMBED_TYPE === 'summary';
    var name = featureName();
    if (!data || typeof data.supported !== 'boolean') {
      renderStatusButton('error', '加载异常', '获取 AI ' + name + '状态失败', false);
      return 'error';
    }
    if (!data.supported) {
      renderStatusButton('disabled', '未开启' + name, data.message || '当前流程未配置 AI ' + name, false);
      return 'unsupported';
    }
    var result = isSummary ? data.summary_result : data.audit_result;
    var hasResult = isSummary ? data.has_summary : data.has_audit;
    var shouldAutoRun = isSummary ? data.should_auto_summary : data.should_auto_audit;
    return applyResultToButton(isSummary, result, hasResult, shouldAutoRun, data.running_job_id);
  }

  function stopStatusWatch() {
    if (statusWatchTimer) {
      clearInterval(statusWatchTimer);
      statusWatchTimer = null;
    }
  }

  function startStatusWatch(durationMs) {
    statusWatchUntil = Date.now() + (durationMs || STATUS_WATCH_MS);
    if (statusWatchTimer) return;
    statusWatchTimer = setInterval(function () {
      if (Date.now() > statusWatchUntil) {
        stopStatusWatch();
        return;
      }
      refreshCurrentStatus(true);
    }, STATUS_POLL_MS);
  }

  function refreshCurrentStatus(preferCached) {
    var requestId = getRequestId();
    if (!requestId) return;
    queryEmbedStatus(requestId, getCurrentUserId(), !!preferCached);
  }

  function queryEmbedStatus(requestId, userId, preferCached) {
    var isSummary = EMBED_TYPE === 'summary';
    var name = featureName();
    var apiPath = isSummary ? '/api/embed/summary/context' : '/api/embed/context';
    var apiUrl = AURA_EMBED_ORIGIN + apiPath
      + '?requestid=' + encodeURIComponent(requestId)
      + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);
    if (preferCached) apiUrl += '&prefer_cached=true';

    fetch(apiUrl, { method: 'GET', credentials: 'omit' })
      .then(function (res) {
        if (!res.ok) throw new Error('HTTP ' + res.status);
        return res.json();
      })
      .then(function (res) {
        // Nuxt 代理直接返回业务数据；同时兼容 Go 接口的统一响应包装。
        var wrapped = res && Object.prototype.hasOwnProperty.call(res, 'code');
        var data = wrapped ? res.data : res;
        if ((wrapped && res.code !== 0) || !data || typeof data.supported !== 'boolean') {
          renderStatusButton('error', '加载异常', '获取 AI ' + name + '状态失败', false);
          return;
        }
        var kind = applyContextData(data);
        if (kind === 'running') startStatusWatch(STATUS_WATCH_MS);
        else if (kind === 'completed' || kind === 'failed' || kind === 'unsupported') stopStatusWatch();
      })
      .catch(function (err) {
        console.warn('[aura-embed-mobile] 请求状态失败:', err);
        renderStatusButton('gray', '查看 AI ' + name, '', false, openCurrentDetails);
      });
  }

  function applyStatusMessage(payload) {
    var isSummary = EMBED_TYPE === 'summary';
    if (payload.running) {
      applyResultToButton(isSummary, null, false, false, 'running');
      startStatusWatch(STATUS_WATCH_MS);
      return;
    }
    var result = {
      status: payload.status || 'completed',
      parse_error: !!payload.parse_error,
      recommendation: payload.recommendation,
      overall_score: payload.overall_score
    };
    var kind = applyResultToButton(isSummary, result, !!payload.has_result, false, '');
    if (kind === 'completed' || kind === 'failed') stopStatusWatch();
  }

  function scheduleResumeRefresh() {
    if (document.visibilityState && document.visibilityState === 'hidden') return;
    if (resumeRefreshTimer) clearTimeout(resumeRefreshTimer);
    resumeRefreshTimer = setTimeout(function () {
      resumeRefreshTimer = null;
      refreshCurrentStatus(true);
    }, 300);
  }

  function bindStatusRefreshListeners() {
    window.addEventListener('message', function (event) {
      if (event.origin !== AURA_EMBED_ORIGIN) return;
      if (!event.data || event.data.type !== MSG_STATUS) return;
      if (event.data.embed_type && event.data.embed_type !== EMBED_TYPE) return;
      var requestId = getRequestId();
      if (event.data.requestid && requestId && String(event.data.requestid) !== String(requestId)) return;
      applyStatusMessage(event.data);
    });
    document.addEventListener('visibilitychange', function () {
      if (document.visibilityState === 'visible') scheduleResumeRefresh();
    });
    window.addEventListener('pageshow', scheduleResumeRefresh);
    window.addEventListener('focus', scheduleResumeRefresh);
  }

  function init() {
    bindStatusRefreshListeners();
    var requestId = getRequestId();
    var userId = getCurrentUserId();

    if (!requestId) {
      renderStatusButton('disabled', '待保存流程', '流程保存并生成编号后即可查看 AI ' + featureName(), false);
      registerOAEvents();
      return;
    }

    renderStatusButton('loading', '加载中...', '', true);
    queryEmbedStatus(requestId, userId, false);
    registerOAEvents();
  }

  jQuery().ready(function () {
    init();
  });
})();
