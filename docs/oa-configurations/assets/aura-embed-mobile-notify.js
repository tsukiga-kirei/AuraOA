/**
 * 泛微 Ecology9 移动端 — AuraOA 嵌入脚本（状态胶囊按钮 + 弹窗查看详情 + 变更感知）
 *
 * 适用环境：泛微 E9 移动端审批、EMobile、企业微信移动审批工作流
 *
 * 使用步骤：
 * 1. 在 AuraOA「系统管理 → 租户管理 → OA 嵌入」为租户生成嵌入密钥并导出移动端脚本
 * 2. 将本文件上传至 OA 静态目录（如 /oa-front/workflow/AuraOA/aura-embed-mobile-notify.js）
 * 3. 流程 → 基础设置 → 自定义页面（或移动端页面设置）填入 js 路径并启用
 * 4. 表单设计中可添加自定义 HTML 块 <div id="getMyBt"></div> 作为按钮挂载位（如无则自动以左下角浮动方式挂载）
 */
(function () {
  console.log('[aura-embed-mobile] 移动端脚本已加载');

  // ========== 按需配置 ==========
  var AURA_EMBED_ORIGIN = 'https://aura.example.com'; // AuraOA 访问地址（末尾不加斜杠）
  var EMBED_ACCESS_TOKEN = 'aura_emb_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'; // 租户嵌入访问密钥
  var SHOW_ON_DESKTOP = false; // 是否在电脑端启用本脚本的按钮和弹窗；移动端不受影响
  var AUTO_RUN_BEFORE_OPEN = false; // true=进入表单即自动审/总结（仍受「打开即审」等配置约束）；false=点开详情才审（默认）
  var BUTTON_CONTAINER_ID = 'getMyBt'; // 表单设计器中预留的挂载容器 ID（与 oa-front 习惯一致）
  var EMBED_TYPE = 'audit'; // 嵌入类型：'all'（全部功能双按钮）、'audit'（AI 审核）或 'summary'（流程总结）
  // ==============================

  var MSG_STATUS = 'aura-oa-embed-status';
  var STATUS_POLL_MS = 3000;
  var STATUS_WATCH_MS = 180000;

  // 不按窗口宽度判断，避免电脑端窄侧栏被识别为移动端；兼容 iPad 桌面 UA。
  var mobileClient = /Android|iPhone|iPad|iPod|Windows Phone/i.test(navigator.userAgent || '')
    || (/Macintosh/i.test(navigator.userAgent || '') && navigator.maxTouchPoints > 1);
  if (!mobileClient && !SHOW_ON_DESKTOP) return;

  // 动画样式仅在真实浏览器拥有 document.head 时注入
  try {
    if (typeof document !== 'undefined' && document.head && document.head.appendChild && (!document.getElementById || !document.getElementById('aura-btn-pulse-style'))) {
      var style = document.createElement('style');
      style.id = 'aura-btn-pulse-style';
      style.textContent =
        '@keyframes auraBtnSpin{to{transform:rotate(360deg)}}' +
        '.aura-status-button{box-sizing:border-box;appearance:none;-webkit-appearance:none;display:inline-flex;align-items:center;gap:9px;min-height:40px;max-width:100%;padding:6px 11px 6px 7px;margin:0;border:1px solid rgba(148,163,184,.22);border-radius:14px;background:#fff;color:#263247;box-shadow:0 4px 16px rgba(15,23,42,.08),0 1px 3px rgba(15,23,42,.04);font:600 13px/1.4 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;letter-spacing:.1px;text-align:left;cursor:pointer;touch-action:manipulation;-webkit-tap-highlight-color:transparent;transition:box-shadow .18s ease,transform .18s ease,border-color .18s ease;}' +
        '.aura-status-icon{display:flex;align-items:center;justify-content:center;width:28px;height:28px;flex:0 0 28px;border-radius:9px;background:var(--aura-status-tint);color:var(--aura-status-color);}' +
        '.aura-status-icon svg{display:block;width:18px;height:18px;}' +
        '.aura-status-text{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;}' +
        '.aura-status-score{display:inline-flex;align-items:baseline;gap:2px;flex:none;padding-left:10px;border-left:1px solid #e8edf3;color:var(--aura-status-color);font-variant-numeric:tabular-nums;}' +
        '.aura-status-score b{font-size:17px;font-weight:700;line-height:1;}' +
        '.aura-status-score small{font-size:10px;font-weight:500;}' +
        '.aura-status-arrow{display:block;width:14px;height:14px;flex:0 0 14px;color:#94a3b8;}' +
        '.aura-status-button:focus-visible{outline:2px solid var(--aura-status-color);outline-offset:3px;}' +
        '.aura-status-button:active{transform:scale(.98);}' +
        '.aura-status-button[aria-disabled="true"]{cursor:default;}' +
        '.aura-status-button[aria-disabled="true"]:not(.aura-status-button--loading){color:#64748b;box-shadow:0 1px 4px rgba(15,23,42,.05);}' +
        '.aura-status-button--loading .aura-status-icon svg{animation:auraBtnSpin 1s linear infinite;}' +
        '.aura-status-sr{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0;}' +
        '.aura-status-group{display:inline-flex;max-width:100%;align-items:center;gap:8px;flex-wrap:wrap;padding:4px 0;}' +
        '#auraMobileEmbedFloatContainer{position:fixed;bottom:16px;bottom:calc(16px + env(safe-area-inset-bottom,0px));left:16px;left:calc(16px + env(safe-area-inset-left,0px));max-width:calc(100vw - 32px);z-index:9999;}' +
        '@media(hover:hover){.aura-status-button:not([aria-disabled="true"]):hover{transform:translateY(-2px);border-color:var(--aura-status-color);box-shadow:0 7px 22px rgba(15,23,42,.12);}}' +
        '@media(pointer:coarse){.aura-status-button{min-height:44px;}}' +
        '@media(prefers-reduced-motion:reduce){.aura-status-button{transition:none;}' +
        '.aura-status-button--loading .aura-status-icon svg{animation:none;}}';
      document.head.appendChild(style);
    }
  } catch (e) {}

  var statusConfig = {
    gray: { color: '#475569', text: 'AI审核详情', bg: '#ffffff', border: '#cbd5e1', dot: '#94a3b8', shadow: '0 1px 3px rgba(0,0,0,0.06)' },
    green: { color: '#15803d', text: '审核通过', bg: '#f0fdf4', border: '#86efac', dot: '#22c55e', shadow: '0 2px 6px rgba(34,197,94,0.12)' },
    yellow: { color: '#b45309', text: '建议关注', bg: '#fffbeb', border: '#fde68a', dot: '#f59e0b', shadow: '0 2px 6px rgba(245,158,11,0.12)' },
    red: { color: '#b91c1c', text: '建议退回', bg: '#fef2f2', border: '#fca5a5', dot: '#ef4444', shadow: '0 2px 6px rgba(239,68,68,0.12)' },
    disabled: { color: '#94a3b8', text: '暂不可用', bg: '#f8fafc', border: '#e2e8f0', dot: '#cbd5e1', shadow: 'none' },
    error: { color: '#dc2626', text: '加载失败', bg: '#fef2f2', border: '#fca5a5', dot: '#ef4444', shadow: 'none' },
    loading: { color: '#1d4ed8', text: '分析中...', bg: '#eff6ff', border: '#93c5fd', dot: '#3b82f6', shadow: '0 2px 6px rgba(59,130,246,0.12)' }
  };

  // 单功能、双功能及电脑端共用按钮结构，分数独立排版，完整文案保留给辅助技术。
  function escapeButtonText(value) {
    return String(value || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  function buildStatusButton(btnId, feature, status, displayText, isLoading, labelClass) {
    var theme = isLoading ? statusConfig.loading : (statusConfig[status] || statusConfig.gray);
    var unavailable = status === 'disabled' || status === 'error';
    var scoreMatch = displayText.match(/ [(]([0-9]+)([^()]*)[)]$/);
    var label = scoreMatch ? displayText.slice(0, scoreMatch.index) : displayText;
    var iconPath = '<path d="m12 3 2.4 6.6L21 12l-6.6 2.4L12 21l-2.4-6.6L3 12l6.6-2.4Z"/>';
    if (feature === 'summary') iconPath = '<path d="M14 3H6v18h12V7Z M14 3v5h4 M9 12h6 M9 16h4"/>';
    if (status === 'green' && feature === 'audit') iconPath = '<path d="m6 12 4 4 8-8"/>';
    if (status === 'red' || status === 'error') iconPath = '<path d="m8 8 8 8 M16 8l-8 8"/>';
    if (status === 'yellow') iconPath = '<path d="M12 6v7 M12 17h.01"/>';
    if (status === 'disabled') iconPath = '<rect x="5" y="10" width="14" height="11" rx="3"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/>';
    if (isLoading) iconPath = '<path d="M20 12a8 8 0 1 1-8-8"/>';
    var icon = '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' + iconPath + '</svg>';
    return '<button id="' + btnId + '" type="button" class="aura-status-button' + (isLoading ? ' aura-status-button--loading' : '') + '" aria-disabled="' + (unavailable || isLoading ? 'true' : 'false') + '" style="--aura-status-color:' + theme.color + ';--aura-status-tint:' + theme.bg + ';">' +
      '<span class="aura-status-icon" aria-hidden="true">' + icon + '</span>' +
      '<span class="aura-status-text" aria-hidden="true">' + escapeButtonText(label) + '</span>' +
      (scoreMatch ? '<span class="aura-status-score" aria-hidden="true"><b>' + escapeButtonText(scoreMatch[1]) + '</b><small>' + escapeButtonText(scoreMatch[2]) + '</small></span>' : '') +
      (!unavailable && !isLoading ? '<svg class="aura-status-arrow" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 4 4 4-4 4"/></svg>' : '') +
      '<span class="aura-status-sr ' + labelClass + '">' + escapeButtonText(displayText) + '</span>' +
    '</button>';
  }

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
      if (typeof WfForm !== 'undefined' && WfForm.getBaseInfo) {
        var base = WfForm.getBaseInfo() || {};
        if (base.f_weaver_belongto_userid != null) {
          return String(base.f_weaver_belongto_userid).trim();
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

  var formOpenPrefetchState = {};

  // 进入表单后按需后台预审：仅当 AUTO_RUN_BEFORE_OPEN 且后端判定 should_auto_* 时发起。
  function startFormOpenPrefetch(featType, shouldAutoRun, runningJobId, onFail) {
    if (!AUTO_RUN_BEFORE_OPEN || !shouldAutoRun || runningJobId) return false;
    var requestId = getRequestId();
    if (!requestId) return false;
    var key = featType + ':' + requestId;
    var state = formOpenPrefetchState[key];
    if (state === 'inflight' || state === 'done') return true;
    if (state === 'failed') return false;
    formOpenPrefetchState[key] = 'inflight';
    var isSummary = featType === 'summary';
    var userId = getCurrentUserId();
    var apiPath = isSummary ? '/api/embed/summary/execute' : '/api/embed/execute';
    var apiUrl = AURA_EMBED_ORIGIN + apiPath
      + '?embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);
    fetch(apiUrl, {
      method: 'POST',
      credentials: 'omit',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        process_id: requestId,
        trigger_source: isSummary ? 'summary_embed_auto' : 'embed_auto',
        trigger_detail: 'form_open',
        oa_user_id: userId
      })
    }).then(function (res) {
      if (!res.ok) throw new Error('HTTP ' + res.status);
      return res.json();
    }).then(function () {
      formOpenPrefetchState[key] = 'done';
    }).catch(function (err) {
      formOpenPrefetchState[key] = 'failed';
      console.warn('[aura-embed-mobile] 进入表单预审发起失败', err);
      if (typeof onFail === 'function') onFail();
    });
    return true;
  }

  // =========================================================================
  // 单模式（audit 或 summary）
  // =========================================================================
  if (EMBED_TYPE !== 'all') {
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
      var btnHtml = '<div class="aura-status-group">' +
        buildStatusButton('auraMobileEmbedBtn', EMBED_TYPE, status, displayText, isLoading, 'aura-mobile-label') +
        '</div>';

      var $container = jQuery('#' + BUTTON_CONTAINER_ID);
      if (!$container.length) {
        if (!jQuery('#auraMobileEmbedFloatContainer').length) {
          jQuery('body').append('<div id="auraMobileEmbedFloatContainer"></div>');
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
      if (shouldAutoRun && startFormOpenPrefetch(isSummary ? 'summary' : 'audit', true, '', function () {
        lastStatusSignature = '';
        renderStatusButton('gray', '查看并生成' + name, '', false, openDetails);
        stopStatusWatch();
      })) {
        renderStatusButton('gray', name + '分析中，查看进度', '', false, openDetails);
        return 'running';
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
    return;
  }

  // =========================================================================
  // 全部功能模式（all: 挂载审核 + 总结两个状态胶囊按钮）
  // =========================================================================
  var allStatusWatchTimer = null;
  var allStatusWatchUntil = 0;
  var allResumeRefreshTimer = null;

  var dualState = {
    audit: { status: 'loading', text: '加载中...', errorMessage: '', isLoading: true, lastSig: '' },
    summary: { status: 'loading', text: '加载中...', errorMessage: '', isLoading: true, lastSig: '' }
  };

  function openDualDesktopDialog(type, targetUrl) {
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
    title.textContent = type === 'summary' ? 'AI 流程总结' : 'AI 流程审核';
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
      refreshDualStatus(type, true);
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

  function openDualEmbedDialog(type, requestId, userId) {
    var path = type === 'summary' ? '/embed/summary' : '/embed/audit';
    var targetUrl = AURA_EMBED_ORIGIN + path
      + '?requestid=' + encodeURIComponent(requestId)
      + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);

    console.log('[aura-embed-mobile] 调起嵌入弹窗: ' + type);
    startDualStatusWatch(STATUS_WATCH_MS);

    var title = type === 'summary' ? 'AI 流程总结' : 'AI 流程审核';
    if (!mobileClient) {
      openDualDesktopDialog(type, targetUrl);
    } else if (window.weaJs && typeof window.weaJs.showDialog === 'function') {
      window.weaJs.showDialog(targetUrl, {
        title: title,
        moduleName: 'workflow',
        style: { width: '100%', height: '100%' },
        callback: function () { refreshDualStatus(type, true); }
      });
    } else {
      window.open(targetUrl, '_blank');
    }
  }

  function renderDualButtons() {
    function buildBtn(featType) {
      var item = dualState[featType];
      var currentStatus = item.isLoading ? statusConfig.loading : (statusConfig[item.status] || statusConfig.gray);
      var displayText = item.text || currentStatus.text;
      var isDisabled = item.status === 'disabled' || item.status === 'error';
      var btnId = featType === 'summary' ? 'auraMobileEmbedSummaryBtn' : 'auraMobileEmbedAuditBtn';
      return buildStatusButton(btnId, featType, item.status, displayText, item.isLoading, 'aura-mobile-label-' + featType);
    }

    var dualHtml =
      '<div class="aura-status-group">' +
        buildBtn('audit') +
        buildBtn('summary') +
      '</div>';

    var $container = jQuery('#' + BUTTON_CONTAINER_ID);
    if (!$container.length) {
      if (!jQuery('#auraMobileEmbedFloatContainer').length) {
        jQuery('body').append('<div id="auraMobileEmbedFloatContainer"></div>');
      }
      $container = jQuery('#auraMobileEmbedFloatContainer');
    }

    $container.html(dualHtml);

    function bindBtn(featType) {
      var btnId = featType === 'summary' ? '#auraMobileEmbedSummaryBtn' : '#auraMobileEmbedAuditBtn';
      var item = dualState[featType];
      jQuery(btnId).off('click').on('click', function () {
        if (item.isLoading) {
          if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
            WfForm.showMessage('数据加载中，请稍候...', 2, 2);
          }
          return;
        }
        if (item.status === 'disabled' || item.status === 'error') {
          if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
            WfForm.showMessage(item.errorMessage || '当前流程未开启 AI ' + (featType === 'summary' ? '总结' : '审核'), 2, 3);
          }
          return;
        }
        var reqId = getRequestId();
        if (reqId) openDualEmbedDialog(featType, reqId, getCurrentUserId());
      });
    }

    bindBtn('audit');
    bindBtn('summary');
  }

  function setDualFeatureState(featType, status, text, errorMessage, isLoading) {
    var sig = status + '|' + text + '|' + (isLoading ? '1' : '0');
    if (dualState[featType].lastSig === sig) return;
    dualState[featType].lastSig = sig;
    dualState[featType].status = status;
    dualState[featType].text = text;
    dualState[featType].errorMessage = errorMessage || '';
    dualState[featType].isLoading = !!isLoading;
    renderDualButtons();
  }

  function applyDualData(featType, data) {
    var isSummary = featType === 'summary';
    var name = isSummary ? '总结' : '审核';
    if (!data || typeof data.supported !== 'boolean') {
      setDualFeatureState(featType, 'error', '加载异常', '获取 AI ' + name + '状态失败', false);
      return 'error';
    }
    if (!data.supported) {
      setDualFeatureState(featType, 'disabled', '未开启' + name, data.message || '当前流程未配置 AI ' + name, false);
      return 'unsupported';
    }
    var result = isSummary ? data.summary_result : data.audit_result;
    var hasResult = isSummary ? data.has_summary : data.has_audit;
    var shouldAutoRun = isSummary ? data.should_auto_summary : data.should_auto_audit;
    if (data.running_job_id) {
      setDualFeatureState(featType, 'loading', name + '分析中...', '', true);
      return 'running';
    }
    if (hasResult && result) {
      if (result.status === 'failed' || result.status === 'cancelled' || result.parse_error) {
        setDualFeatureState(featType, 'red', name + '异常', 'AI ' + name + '分析出现异常', false);
        return 'failed';
      }
      if (isSummary) {
        setDualFeatureState(featType, 'green', '查看流程总结', '', false);
        return 'completed';
      }
      var rec = result.recommendation || 'review';
      var score = result.overall_score != null ? Math.round(Number(result.overall_score)) : null;
      var scoreText = score != null ? ' (' + score + '分)' : '';
      if (rec === 'approve') {
        setDualFeatureState(featType, 'green', '审核通过' + scoreText, '', false);
      } else if (rec === 'return') {
        setDualFeatureState(featType, 'red', '建议退回' + scoreText, '', false);
      } else {
        setDualFeatureState(featType, 'yellow', '建议关注' + scoreText, '', false);
      }
      return 'completed';
    }
    if (shouldAutoRun && startFormOpenPrefetch(featType, true, '', function () {
      dualState[featType].lastSig = '';
      setDualFeatureState(featType, 'gray', '生成' + name, '', false);
    })) {
      setDualFeatureState(featType, 'loading', name + '分析中...', '', true);
      return 'running';
    }
    setDualFeatureState(featType, 'gray', shouldAutoRun ? '生成' + name : 'AI' + name + '详情', '', false);
    return 'pending';
  }

  function refreshDualStatus(featType, preferCached) {
    var reqId = getRequestId();
    if (!reqId) return;
    var userId = getCurrentUserId();
    var apiPath = featType === 'summary' ? '/api/embed/summary/context' : '/api/embed/context';
    var apiUrl = AURA_EMBED_ORIGIN + apiPath
      + '?requestid=' + encodeURIComponent(reqId)
      + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);
    if (preferCached) apiUrl += '&prefer_cached=true';

    fetch(apiUrl, { method: 'GET', credentials: 'omit' })
      .then(function (res) {
        if (!res.ok) throw new Error('HTTP ' + res.status);
        return res.json();
      })
      .then(function (res) {
        var wrapped = res && Object.prototype.hasOwnProperty.call(res, 'code');
        var data = wrapped ? res.data : res;
        var kind = applyDualData(featType, data);
        if (kind === 'running') startDualStatusWatch(STATUS_WATCH_MS);
      })
      .catch(function (err) {
        console.warn('[aura-embed-mobile] 请求状态失败:', featType, err);
        setDualFeatureState(featType, 'gray', 'AI' + (featType === 'summary' ? '总结' : '审核') + '详情', '', false);
      });
  }

  function refreshAllDualStatus(preferCached) {
    refreshDualStatus('audit', preferCached);
    refreshDualStatus('summary', preferCached);
  }

  function startDualStatusWatch(durationMs) {
    allStatusWatchUntil = Date.now() + (durationMs || STATUS_WATCH_MS);
    if (allStatusWatchTimer) return;
    allStatusWatchTimer = setInterval(function () {
      if (Date.now() > allStatusWatchUntil) {
        clearInterval(allStatusWatchTimer);
        allStatusWatchTimer = null;
        return;
      }
      refreshAllDualStatus(true);
    }, STATUS_POLL_MS);
  }

  function notifyBeforeReleaseDual(action, callback) {
    var released = false;
    var release = function () {
      if (released) return;
      released = true;
      startDualStatusWatch(STATUS_WATCH_MS);
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

  function registerDualOAEvents() {
    if (typeof WfForm === 'undefined' || !WfForm.registerCheckEvent) return;
    WfForm.registerCheckEvent(WfForm.OPER_SAVE, function (callback) {
      notifyBeforeReleaseDual('save_requested', callback);
    });
    WfForm.registerCheckEvent(WfForm.OPER_SUBMIT, function (callback) {
      notifyBeforeReleaseDual('submit_requested', callback);
    });
  }

  function scheduleDualResumeRefresh() {
    if (document.visibilityState && document.visibilityState === 'hidden') return;
    if (allResumeRefreshTimer) clearTimeout(allResumeRefreshTimer);
    allResumeRefreshTimer = setTimeout(function () {
      allResumeRefreshTimer = null;
      refreshAllDualStatus(true);
    }, 300);
  }

  function bindDualStatusRefreshListeners() {
    window.addEventListener('message', function (event) {
      if (event.origin !== AURA_EMBED_ORIGIN) return;
      if (!event.data || event.data.type !== MSG_STATUS) return;
      var requestId = getRequestId();
      if (event.data.requestid && requestId && String(event.data.requestid) !== String(requestId)) return;
      var featType = event.data.embed_type === 'summary' ? 'summary' : 'audit';
      if (event.data.running) {
        setDualFeatureState(featType, 'loading', (featType === 'summary' ? '总结' : '审核') + '分析中...', '', true);
        startDualStatusWatch(STATUS_WATCH_MS);
        return;
      }
      var res = {
        status: event.data.status || 'completed',
        parse_error: !!event.data.parse_error,
        recommendation: event.data.recommendation,
        overall_score: event.data.overall_score
      };
      applyDualData(featType, {
        supported: true,
        [featType === 'summary' ? 'summary_result' : 'audit_result']: res,
        [featType === 'summary' ? 'has_summary' : 'has_audit']: !!event.data.has_result
      });
    });
    document.addEventListener('visibilitychange', function () {
      if (document.visibilityState === 'visible') scheduleDualResumeRefresh();
    });
    window.addEventListener('pageshow', scheduleDualResumeRefresh);
    window.addEventListener('focus', scheduleDualResumeRefresh);
  }

  function initDual() {
    bindDualStatusRefreshListeners();
    var reqId = getRequestId();
    if (!reqId) {
      setDualFeatureState('audit', 'disabled', '待保存流程', '流程保存并生成编号后即可查看 AI 审核', false);
      setDualFeatureState('summary', 'disabled', '待保存流程', '流程保存并生成编号后即可查看 AI 总结', false);
      registerDualOAEvents();
      return;
    }
    setDualFeatureState('audit', 'loading', '审核加载中...', '', true);
    setDualFeatureState('summary', 'loading', '总结加载中...', '', true);
    refreshAllDualStatus(false);
    registerDualOAEvents();
  }

  jQuery().ready(function () {
    initDual();
  });
})();
