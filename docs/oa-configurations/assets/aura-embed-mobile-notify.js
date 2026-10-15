/**
 * 泛微 Ecology9 移动端 — AuraOA 嵌入脚本（状态胶囊按钮 + 弹窗查看详情 + 变更感知）
 *
 * 适用环境：泛微 E9 移动端审批、EMobile、企业微信移动审批工作流
 *
 * 使用步骤：
 * 1. 在 AuraOA「系统管理 → 租户管理 → OA 嵌入」为租户生成嵌入密钥并导出移动端脚本
 * 2. 将本文件上传至 OA 静态目录（如 /oa-front/workflow/AuraOA/aura-embed-mobile-notify.js）
 * 3. 流程 → 基础设置 → 自定义页面（或移动端页面设置）填入 js 路径并启用
 * 4. 表单设计中可添加自定义 HTML 块 <div id="getMyBt"></div> 作为按钮挂载位（如无则按 FLOAT_POSITION_MOBILE / FLOAT_POSITION_DESKTOP 浮动挂载，默认左下角）
 */
(function () {
  console.log('[aura-embed-mobile] 移动端脚本已加载');

  // ========== 按需配置 ==========
  var AURA_EMBED_ORIGIN = 'https://aura.example.com'; // AuraOA 访问地址（末尾不加斜杠）
  var EMBED_ACCESS_TOKEN = 'aura_emb_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'; // 租户嵌入访问密钥
  var SHOW_ON_DESKTOP = false; // 是否在电脑端启用本脚本的按钮和弹窗；移动端不受影响
  var USE_NATIVE_MOBILE_DIALOG = false; // true=使用 OA 原生弹窗（兼容模式，无卷曲动画）；默认使用联动弹层
  var AUTO_RUN_BEFORE_OPEN = false; // true=进入表单即自动审/总结（仍受「打开即审」等配置约束）；false=点开详情才审（默认）
  var BUTTON_CONTAINER_ID = 'getMyBt'; // 表单设计器中预留的挂载容器 ID（与 oa-front 习惯一致）
  var FLOAT_POSITION_MOBILE = 'bottom-left'; // 移动端浮动位置：'bottom-left'（左下角）或 'top-right'（右上角）
  var FLOAT_POSITION_DESKTOP = 'bottom-left'; // 电脑端浮动位置：'bottom-left'（左下角）或 'top-right'（右上角，提交/保存下方）；需 SHOW_ON_DESKTOP = true
  var EMBED_TYPE = 'audit'; // 嵌入类型：'all'（全部功能双按钮）、'audit'（AI 审核）或 'summary'（流程总结）
  // ==============================

  var MSG_STATUS = 'aura-oa-embed-status';
  var STATUS_POLL_MS = 3000;
  var STATUS_WATCH_MS = 180000;
  var BUTTON_BUSY_MIN_MS = 480; // 忙碌态至少停留，避免「加载中」未看清就跳到结果
  var STATUS_MOTION_CLASSES = [
    'aura-status-button--enter',
    'aura-status-button--loading',
    'aura-status-button--running',
    'aura-status-button--enter-shake',
    'aura-status-button--enter-pop',
    'aura-status-button--enter-bounce',
    'aura-status-button--enter-rise',
    'aura-status-button--enter-fade',
    'aura-status-button--morph',
    'aura-status-button--shine',
    'aura-status-button--ready'
  ];

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
        '@keyframes auraDraw{to{stroke-dashoffset:0}}' +
        '@keyframes auraButtonEnter{0%{opacity:0;transform:translate3d(-10px,6px,0) scale(.78);filter:blur(2px)}55%{opacity:1;transform:translate3d(1px,-1px,0) scale(1.035);filter:blur(0)}100%{opacity:1;transform:none;filter:none}}' +
        '@keyframes auraButtonEnterSoft{0%{opacity:0;transform:translate3d(-8px,5px,0) scale(.9);filter:blur(1px)}100%{opacity:1;transform:none;filter:none}}' +
        '@keyframes auraIconEnter{0%{opacity:0;transform:scale(.64) rotate(-10deg)}72%{opacity:1;transform:scale(1.08) rotate(3deg)}100%{opacity:1;transform:scale(1) rotate(0)}}' +
        '@keyframes auraTextEnter{0%{opacity:0;transform:translate3d(-5px,0,0)}100%{opacity:1;transform:none}}' +
        '@keyframes auraBtnShine{0%,100%{opacity:0;transform:translateX(-120%)}18%{opacity:.48}42%{opacity:0;transform:translateX(120%)}}' +
        '@keyframes auraStatusBreath{0%,100%{opacity:.22;transform:scale(.985)}50%{opacity:.48;transform:scale(1.01)}}' +
        '@keyframes auraStarSpin{0%{transform:rotate(0deg) scale(.9)}50%{transform:rotate(180deg) scale(1)}100%{transform:rotate(360deg) scale(.9)}}' +
        '@keyframes auraIconSwapOut{to{opacity:0;transform:scale(.78) rotate(-20deg);filter:blur(2px)}}' +
        '@keyframes auraIconSwapIn{0%{opacity:0;transform:scale(.8) rotate(20deg);filter:blur(2px)}100%{opacity:1;transform:none;filter:blur(0)}}' +
        '@keyframes auraCopySwapOut{to{opacity:0;transform:translateY(8px);filter:blur(2px)}}' +
        '@keyframes auraCopySwapIn{from{opacity:0;transform:translateY(-7px);filter:blur(2px)}to{opacity:1;transform:none;filter:none}}' +
        '@keyframes auraArrowIn{from{opacity:0;transform:translateX(-4px)}to{opacity:1;transform:none}}' +
        '@keyframes auraArrowOut{to{opacity:0;transform:translateX(4px)}}' +
        '.aura-status-button{box-sizing:border-box;appearance:none;-webkit-appearance:none;position:relative;isolation:isolate;overflow:hidden;display:inline-flex;align-items:center;gap:9px;min-height:40px;max-width:100%;padding:6px 11px 6px 7px;margin:0;border:1px solid var(--aura-status-border,rgba(148,163,184,.22));border-radius:14px;background:var(--aura-status-surface,#fff);color:#263247;box-shadow:var(--aura-status-shadow,0 4px 16px rgba(15,23,42,.08),0 1px 3px rgba(15,23,42,.04));font:600 13px/1.4 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;letter-spacing:.1px;text-align:left;cursor:pointer;touch-action:manipulation;-webkit-tap-highlight-color:transparent;transition:background-color .45s ease,border-color .45s ease,box-shadow .45s ease,color .45s ease,width .52s cubic-bezier(.22,1,.36,1),transform .25s ease;animation:none;}' +
        '.aura-status-button::before{content:"";position:absolute;inset:1px;border-radius:13px;background:radial-gradient(circle at 8% 50%,var(--aura-status-tint),transparent 62%);opacity:.22;pointer-events:none;z-index:0;transform-origin:8% 50%;animation:auraStatusBreath 4.2s ease-in-out infinite;transition:opacity .45s ease;}' +
        '.aura-status-button::after{content:"";position:absolute;top:0;bottom:0;left:0;width:36%;background:linear-gradient(105deg,transparent,rgba(255,255,255,.66),transparent);pointer-events:none;z-index:0;opacity:0;transform:translateX(-120%);}' +
        '.aura-status-button--enter::after,.aura-status-button--shine::after{animation:auraBtnShine 3.2s ease-out .18s both;}' +
        '.aura-status-icon,.aura-status-copy,.aura-status-text,.aura-status-score,.aura-status-arrow{position:relative;z-index:1;}' +
        '.aura-status-icon{display:flex;align-items:center;justify-content:center;width:28px;height:28px;flex:0 0 28px;border-radius:9px;background:var(--aura-status-tint);color:var(--aura-status-color);overflow:hidden;transition:background-color .45s ease,color .45s ease;}' +
        '.aura-status-icon svg{display:block;width:18px;height:18px;transform-origin:center;}' +
        '.aura-status-icon-layer{display:flex;align-items:center;justify-content:center;width:28px;height:28px;}' +
        '.aura-status-icon-layer svg{display:block;width:18px;height:18px;transform-origin:center;}' +
        '.aura-status-icon-layer--out{position:absolute;inset:0;animation:auraIconSwapOut .24s ease both;}' +
        '.aura-status-icon-layer--in{animation:auraIconSwapIn .48s cubic-bezier(.22,1,.36,1) both;}' +
        '.aura-status-copy{display:flex;align-items:center;min-width:0;overflow:hidden;}' +
        '.aura-status-copy-inner{display:inline-flex;align-items:center;gap:9px;white-space:nowrap;}' +
        '.aura-status-copy-inner--out{position:absolute;left:0;top:0;bottom:0;animation:auraCopySwapOut .32s ease both;pointer-events:none;}' +
        '.aura-status-copy-inner--in{animation:auraCopySwapIn .4s cubic-bezier(.22,.7,.2,1) .04s both;}' +
        '.aura-status-text{min-width:0;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;}' +
        '.aura-status-score{display:inline-flex;align-items:baseline;gap:2px;flex:none;padding-left:10px;border-left:1px solid var(--aura-status-border);color:var(--aura-status-color);font-variant-numeric:tabular-nums;transition:color .45s ease;}' +
        '.aura-status-score b{font-size:17px;font-weight:700;line-height:1;}' +
        '.aura-status-score small{font-size:10px;font-weight:500;}' +
        '.aura-status-arrow{display:block;width:14px;height:14px;flex:0 0 14px;color:#94a3b8;}' +
        '.aura-status-arrow--in{animation:auraArrowIn .32s ease .08s both;}' +
        '.aura-status-arrow--out{animation:auraArrowOut .24s ease both;}' +
        '.aura-status-button:focus-visible{outline:2px solid var(--aura-status-color);outline-offset:3px;}' +
        '.aura-status-button:active{transform:scale(.98);}' +
        '.aura-status-button--morph{pointer-events:none;}' +
        '.aura-status-button[aria-disabled="true"]{cursor:default;}' +
        '.aura-status-button[aria-disabled="true"]:not(.aura-status-button--loading){color:var(--aura-status-color);}' +
        '.aura-status-button[aria-disabled="true"]:not(.aura-status-button--loading)::before{animation:none;opacity:.12;}' +
        '.aura-status-button--enter{animation:auraButtonEnter .62s cubic-bezier(.22,.7,.2,1) both;}' +
        '.aura-status-button--enter .aura-status-icon{opacity:0;transform:scale(.76);animation:auraIconEnter .38s cubic-bezier(.22,.7,.2,1) .36s both;}' +
        '.aura-status-button--enter .aura-status-copy,.aura-status-button--enter .aura-status-text{opacity:0;transform:translate3d(-5px,0,0);animation:auraTextEnter .34s ease-out .53s both;}' +
        '.aura-status-button--enter .aura-status-score{opacity:0;transform:translate3d(-4px,0,0);animation:auraTextEnter .34s ease-out .6s both;}' +
        '.aura-status-button--enter .aura-status-arrow{opacity:0;transform:translate3d(-3px,0,0);animation:auraTextEnter .34s ease-out .66s both;}' +
        '.aura-status-button--enter.aura-status-button--loading,.aura-status-button--enter.aura-status-button--running{animation:auraButtonEnterSoft .4s cubic-bezier(.22,.7,.2,1) both;}' +
        '.aura-status-button--enter.aura-status-button--loading .aura-status-icon,.aura-status-button--enter.aura-status-button--running .aura-status-icon,.aura-status-button--enter.aura-status-button--loading .aura-status-copy,.aura-status-button--enter.aura-status-button--running .aura-status-copy,.aura-status-button--enter.aura-status-button--loading .aura-status-text,.aura-status-button--enter.aura-status-button--running .aura-status-text{opacity:1;transform:none;animation:none;}' +
        '.aura-status-button--enter.aura-status-button--enter-shake .aura-status-icon svg>*,.aura-status-button--enter.aura-status-button--enter-pop .aura-status-icon svg>*,.aura-status-button--enter.aura-status-button--enter-bounce .aura-status-icon svg>*,.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>*,.aura-status-button--enter.aura-status-button--enter-fade .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-shake .aura-status-icon-layer--in svg>*,.aura-status-button--morph.aura-status-button--enter-pop .aura-status-icon-layer--in svg>*,.aura-status-button--morph.aura-status-button--enter-bounce .aura-status-icon-layer--in svg>*,.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>*,.aura-status-button--morph.aura-status-button--enter-fade .aura-status-icon-layer--in svg>*{stroke-dasharray:1;stroke-dashoffset:1}' +
        '.aura-status-button--loading .aura-status-icon>svg,.aura-status-button--loading .aura-status-icon-layer--in svg{animation:auraBtnSpin .9s linear infinite}' +
        '.aura-status-button--running .aura-status-icon>svg .aura-running-star,.aura-status-button--running .aura-status-icon-layer--in .aura-running-star{transform-box:fill-box;transform-origin:center;animation:auraStarSpin 3.6s cubic-bezier(.45,0,.55,1) infinite}' +
        '.aura-status-button--enter.aura-status-button--enter-shake .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-shake .aura-status-icon-layer--in svg>*{animation:auraDraw .34s cubic-bezier(.22,.7,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-shake .aura-status-icon svg>:nth-child(2),.aura-status-button--morph.aura-status-button--enter-shake .aura-status-icon-layer--in svg>:nth-child(2){animation-delay:.22s}' +
        '.aura-status-button--enter.aura-status-button--enter-pop .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-pop .aura-status-icon-layer--in svg>*{animation:auraDraw .48s cubic-bezier(.22,.72,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-bounce .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-bounce .aura-status-icon-layer--in svg>*{animation:auraDraw .36s cubic-bezier(.22,.7,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-bounce .aura-status-icon svg>:nth-child(2),.aura-status-button--morph.aura-status-button--enter-bounce .aura-status-icon-layer--in svg>:nth-child(2){animation-delay:.22s}' +
        '.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>*{animation:auraDraw .32s cubic-bezier(.22,.7,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>:nth-child(2),.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>:nth-child(2){animation-delay:.18s}' +
        '.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>:nth-child(3),.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>:nth-child(3){animation-delay:.26s}' +
        '.aura-status-button--enter.aura-status-button--enter-rise .aura-status-icon svg>:nth-child(4),.aura-status-button--morph.aura-status-button--enter-rise .aura-status-icon-layer--in svg>:nth-child(4){animation-delay:.34s}' +
        '.aura-status-button--enter.aura-status-button--enter-fade .aura-status-icon svg>*,.aura-status-button--morph.aura-status-button--enter-fade .aura-status-icon-layer--in svg>*{animation:auraDraw .4s cubic-bezier(.22,.7,.2,1) .08s forwards}' +
        '.aura-status-button--enter.aura-status-button--enter-fade .aura-status-icon svg>:nth-child(2),.aura-status-button--morph.aura-status-button--enter-fade .aura-status-icon-layer--in svg>:nth-child(2){animation-delay:.2s}' +
        '.aura-status-sr{position:absolute;width:1px;height:1px;padding:0;margin:-1px;overflow:hidden;clip:rect(0,0,0,0);white-space:nowrap;border:0;}' +
        '.aura-status-group{display:inline-flex;flex-direction:column;max-width:100%;align-items:stretch;gap:8px;padding:4px 0;}' +
        '.aura-status-group .aura-status-button{width:100%;}' +
        '#auraMobileEmbedFloatContainer{position:fixed;max-width:calc(100vw - 32px);z-index:9999;touch-action:none;user-select:none;-webkit-user-select:none;-webkit-touch-callout:none;cursor:grab;}' +
        '#auraMobileEmbedFloatContainer.aura-float--bottom-left{bottom:16px;bottom:calc(16px + env(safe-area-inset-bottom,0px));left:16px;left:calc(16px + env(safe-area-inset-left,0px));}' +
        '#auraMobileEmbedFloatContainer.aura-float--top-right{top:88px;top:calc(88px + env(safe-area-inset-top,0px));right:24px;right:calc(24px + env(safe-area-inset-right,0px));}' +
        '#auraMobileEmbedFloatContainer.aura-is-dragging{cursor:grabbing!important;opacity:.92;transition:none!important;}' +
        '#auraMobileEmbedFloatContainer.aura-is-dragging .aura-status-button{transform:scale(.96)!important;box-shadow:0 10px 28px rgba(15,23,42,.22)!important;}' +
        '@media screen and (max-width:768px){' +
        '#auraMobileEmbedFloatContainer.aura-float--top-right{top:116px;top:calc(116px + env(safe-area-inset-top,0px));right:14px;right:calc(14px + env(safe-area-inset-right,0px));}' +
        '#auraMobileEmbedFloatContainer.aura-float--bottom-left{bottom:20px;bottom:calc(20px + env(safe-area-inset-bottom,0px));left:14px;left:calc(14px + env(safe-area-inset-left,0px));}' +
        '}' +
        '.aura-status-button .aura-status-text,.aura-status-button .aura-status-arrow,.aura-is-compact .aura-status-text,.aura-is-compact .aura-status-arrow{display:none!important;}' +
        '.aura-status-button,.aura-is-compact.aura-status-button{min-height:44px!important;padding:7px 12px 7px 7px!important;gap:6px!important;border-radius:16px!important;}' +
        '.aura-status-score,.aura-is-compact .aura-status-score{padding-left:6px!important;border-left-width:1px!important;}' +
        '.aura-status-score b,.aura-is-compact .aura-status-score b{font-size:15px!important;}' +
        '.aura-status-score small,.aura-is-compact .aura-status-score small{font-size:9px!important;}' +
        '.aura-status-button--icon-only,.aura-is-compact.aura-status-button--icon-only{padding:5px!important;gap:0!important;min-width:44px;width:44px;justify-content:center!important;}' +
        '.aura-status-button--icon-only .aura-status-copy,.aura-is-compact.aura-status-button--icon-only .aura-status-copy{display:none!important;}' +
        '.aura-embed-tooltip{position:fixed;z-index:10001;pointer-events:none;max-width:280px;padding:6px 12px;border-radius:10px;background:var(--aura-status-surface,#fff);color:var(--aura-status-color,#1e293b);border:1px solid var(--aura-status-border,rgba(148,163,184,.3));font:600 12px/1.4 -apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC",sans-serif;letter-spacing:.2px;white-space:nowrap;box-shadow:0 6px 20px rgba(15,23,42,.08);backdrop-filter:blur(8px);-webkit-backdrop-filter:blur(8px);opacity:0;transform:scale(.92);transition:opacity .18s ease,transform .18s cubic-bezier(.22,.7,.2,1);}' +
        '.aura-status-summary-label{display:inline-block;color:var(--aura-status-color);font-weight:600;white-space:nowrap;}' +
        '.aura-embed-tooltip--visible{opacity:1;transform:scale(1);}' +
        '@media(hover:hover){.aura-status-button:not([aria-disabled="true"]):not(.aura-status-button--morph):hover{transform:translateY(-1px);}}' +
        '@media(pointer:coarse){.aura-status-button{min-height:44px;}}' +
        ".aura-status-button { box-shadow: 0 4px 12px rgba(15, 23, 42, .07), 0 1px 2px rgba(15, 23, 42, .04); }" +
        ".aura-status-button::before { animation: none; opacity: .16; }" +
        ".aura-status-button--ready {" +
        "box-shadow: 0 0 0 1px var(--aura-halo-soft), 0 0 10px 1px var(--aura-halo-soft), 0 4px 12px rgba(15, 23, 42, .07);" +
        "animation: auraResultHalo 3.4s ease-in-out .52s infinite;" +
        "}" +
        ".aura-status-button--ready.aura-status-button--enter {" +
        "animation: auraButtonEnter .62s cubic-bezier(.22, .7, .2, 1) both, auraResultHalo 3.4s ease-in-out .62s infinite;" +
        "}" +
        "@keyframes auraResultHalo {" +
        "0%, 100% { box-shadow: 0 0 0 1px var(--aura-halo-soft), 0 0 8px 1px var(--aura-halo-soft), 0 4px 12px rgba(15, 23, 42, .07); }" +
        "50% { box-shadow: 0 0 0 1px var(--aura-halo-strong), 0 0 13px 2px var(--aura-halo-strong), 0 4px 12px rgba(15, 23, 42, .07); }" +
        "}" +
        ".aura-embed-dialog-overlay{position:fixed;inset:0;z-index:2147483646;display:flex;align-items:center;justify-content:center;padding:12px;box-sizing:border-box;background:rgba(28,31,38,.24);}" +
        ".aura-embed-dialog-panel{box-sizing:border-box;width:min(740px,94vw);height:86vh;height:86dvh;max-height:880px;background:#f8fafc;border-radius:22px;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 24px 64px -12px rgba(15,23,42,.28);}" +
        ".aura-embed-dialog-header{display:flex;align-items:center;justify-content:space-between;gap:12px;padding:12px 20px;background:#fff;border-bottom:1px solid #f1f5f9;flex-shrink:0;}" +
        ".aura-embed-dialog-heading{display:flex;align-items:center;gap:8px;min-width:0;flex-wrap:wrap;}" +
        ".aura-embed-dialog-title{font:700 16px/1.4 -apple-system,BlinkMacSystemFont,\"Segoe UI\",\"PingFang SC\",sans-serif;color:#0f172a;white-space:nowrap;}" +
        ".aura-embed-dialog-tag{padding:2px 8px;border-radius:999px;font:500 11px/1.5 sans-serif;background:#f1f5f9;color:#64748b;white-space:nowrap;}" +
        ".aura-embed-dialog-close{appearance:none;cursor:pointer;flex-shrink:0;width:40px;height:40px;border-radius:50%;border:0;background:#f5f6f8;color:#64748b;display:flex;align-items:center;justify-content:center;padding:0;}" +
        ".aura-embed-dialog-close:focus-visible{outline:2px solid #64748b;outline-offset:2px;}" +
        ".aura-embed-dialog-frame{width:100%;flex:1;min-height:0;border:0;display:block;background:#f8fafc;}" +
        ".aura-embed-genie-layer{position:fixed;inset:0;z-index:2147483647;pointer-events:none;overflow:hidden;}" +
        ".aura-embed-dialog-overlay--mobile{padding:10px 8px;padding:max(10px,env(safe-area-inset-top)) 8px max(10px,env(safe-area-inset-bottom));}" +
        ".aura-embed-dialog-overlay--mobile .aura-embed-dialog-panel{width:100%;height:100%;max-height:none;border-radius:20px;}" +
        ".aura-embed-dialog-overlay--mobile .aura-embed-dialog-header{padding:10px 14px;}" +
        '@media(prefers-reduced-motion:reduce){.aura-status-button{transition:none;animation:none!important;}.aura-status-button::before,.aura-status-button::after,.aura-status-icon,.aura-status-copy,.aura-status-text,.aura-status-score,.aura-status-arrow,.aura-status-icon-layer,.aura-status-copy-inner{animation:none!important;opacity:1!important;transform:none!important;filter:none!important;}.aura-status-icon svg,.aura-status-icon svg>*{animation:none!important;stroke-dashoffset:0!important;}}';
      document.head.appendChild(style);
    }
  } catch (e) {}

  var statusConfig = {
    gray: { color: '#475569', text: 'AI审核详情', bg: '#eef2f7', surface: '#f8fafc', border: '#cbd5e1', dot: '#94a3b8', shadow: '0 2px 8px rgba(15,23,42,.06)' },
    green: { color: '#15803d', text: '审核通过', bg: '#e5f7ed', surface: '#f1fbf5', border: '#86efac', dot: '#16a36a', shadow: '0 3px 12px rgba(22,163,106,.13)' },
    yellow: { color: '#a15c00', text: '建议关注', bg: '#fff0d2', surface: '#fff9ee', border: '#f2c078', dot: '#d98200', shadow: '0 3px 12px rgba(217,130,0,.14)' },
    red: { color: '#c62828', text: '建议退回', bg: '#ffe8e6', surface: '#fff3f2', border: '#ef8a84', dot: '#d92d20', shadow: '0 4px 14px rgba(198,40,40,.16)' },
    disabled: { color: '#2dd4bf', text: '暂不可用', bg: '#ccfbf1', surface: '#f4fffc', border: '#99f6e4', dot: '#5eead4', shadow: '0 2px 10px rgba(45,212,191,.16)' },
    error: { color: '#b42318', text: '加载失败', bg: '#ffe1df', surface: '#fff1f0', border: '#e97870', dot: '#d92d20', shadow: '0 4px 16px rgba(180,35,24,.2)' },
    loading: { color: '#1d4ed8', text: '加载中...', bg: '#dbeafe', surface: '#eef5ff', border: '#60a5fa', dot: '#2563eb', shadow: '0 3px 12px rgba(37,99,235,.15)' }
  };

  // 单功能、双功能及电脑端共用按钮结构：首次进入播放入场，后续在同一颗胶囊上变形。
  function escapeButtonText(value) {
    return String(value || '').replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
  }

  function prefersReducedMotion() {
    try {
      return !!(window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches);
    } catch (e) {
      return false;
    }
  }

  function isBusyStatus(status, isLoading) {
    return !!isLoading || status === 'loading';
  }

  function parseButtonLabel(displayText) {
    var text = String(displayText || '');
    var scoreMatch = text.match(/ [(]([0-9]+)([^()]*)[)]$/);
    return {
      label: scoreMatch ? text.slice(0, scoreMatch.index) : text,
      scoreMatch: scoreMatch
    };
  }

  function buttonTheme(status, isLoading) {
    var fetching = !!isLoading;
    var running = !fetching && status === 'loading';
    return (fetching || running) ? statusConfig.loading : (statusConfig[status] || statusConfig.gray);
  }

  function iconMarkup(feature, status, isLoading) {
    var fetching = !!isLoading;
    var running = !fetching && status === 'loading';
    var iconPath = '<path pathLength="1" d="m12 3 2.4 6.6L21 12l-6.6 2.4L12 21l-2.4-6.6L3 12l6.6-2.4Z"/>';
    if (feature === 'summary') iconPath = '<path pathLength="1" d="M14 3H6v18h12V7Z" stroke-width="2.4"/><path pathLength="1" d="M14 3v5h4" stroke-width="2.4"/><path pathLength="1" d="M9 12h6" stroke-width="2.4"/><path pathLength="1" d="M9 16h4" stroke-width="2.4"/>';
    if (status === 'green' && feature === 'audit') iconPath = '<path pathLength="1" d="m6 12 4 4 8-8" stroke-width="2.8"/>';
    if (status === 'red' || status === 'error') iconPath = '<path pathLength="1" d="M8 8l8 8" stroke-width="2.7"/><path pathLength="1" d="M16 8l-8 8" stroke-width="2.7"/>';
    if (status === 'yellow') iconPath = '<path pathLength="1" d="M12 4.1v9.4" stroke-width="3.6"/><path pathLength="1" d="M12 18.25v.02" stroke-width="4.4"/>';
    if (status === 'disabled') iconPath = '<path pathLength="1" d="M8 10V7a4 4 0 0 1 8 0v3" stroke-width="2.4"/><rect pathLength="1" x="5" y="10" width="14" height="11" rx="3" stroke-width="2.4"/>';
    if (running) {
      iconPath = '<g class="aura-running-star">';
      for (var ray = 0; ray < 12; ray++) {
        iconPath += '<path d="M12 4v4" transform="rotate(' + (ray * 30) + ' 12 12)" stroke-width="2.2" opacity="' + (.5 + (ray % 3) * .22) + '"/>';
      }
      iconPath += '</g>';
    }
    if (fetching) iconPath = '<circle cx="12" cy="12" r="8.1" fill="none" stroke="currentColor" stroke-width="2.7" opacity="0.22"/><path d="M12 3.9a8.1 8.1 0 0 1 0 16.2" fill="none" stroke="currentColor" stroke-width="2.8" stroke-linecap="round"/>';
    return iconPath;
  }

  function iconSvgMarkup(feature, status, isLoading) {
    return '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">' + iconMarkup(feature, status, isLoading) + '</svg>';
  }

  function copyInnerMarkup(label, scoreMatch, feature) {
    return '<span class="aura-status-text">' + escapeButtonText(label) + '</span>' +
      (feature === 'summary' ? '<span class="aura-status-summary-label">总结</span>' :
        (scoreMatch ? '<span class="aura-status-score"><b>' + escapeButtonText(scoreMatch[1]) + '</b></span>' : ''));
  }

  function arrowMarkup() {
    return '<svg class="aura-status-arrow" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m6 4 4 4-4 4"/></svg>';
  }

  function statusMotionClass(status, isLoading, isEnter) {
    var prefix = isEnter ? ' aura-status-button--enter' : '';
    if (!isLoading && ['green', 'yellow', 'red'].indexOf(status) !== -1) prefix += ' aura-status-button--ready';
    if (isLoading) return prefix + ' aura-status-button--loading';
    if (status === 'loading') return prefix + ' aura-status-button--running';
    if (status === 'red' || status === 'error') return prefix + ' aura-status-button--enter-shake';
    if (status === 'green') return prefix + ' aura-status-button--enter-pop';
    if (status === 'yellow') return prefix + ' aura-status-button--enter-bounce';
    if (status === 'disabled') return prefix + ' aura-status-button--enter-fade';
    return prefix + ' aura-status-button--enter-rise';
  }

  function haloColor(theme, opacity) {
    var hex = theme.color.slice(1);
    return 'rgba(' + parseInt(hex.slice(0, 2), 16) + ',' + parseInt(hex.slice(2, 4), 16) + ',' + parseInt(hex.slice(4, 6), 16) + ',' + opacity + ')';
  }

  function applyButtonTheme(btn, theme) {
    btn.style.setProperty('--aura-halo-soft', haloColor(theme, .14));
    btn.style.setProperty('--aura-halo-strong', haloColor(theme, .24));
    btn.style.setProperty('--aura-status-color', theme.color);
    btn.style.setProperty('--aura-status-tint', theme.bg);
    btn.style.setProperty('--aura-status-surface', theme.surface);
    btn.style.setProperty('--aura-status-border', theme.border);
    btn.style.setProperty('--aura-status-shadow', theme.shadow);
  }

  function syncMotionClass(btn, status, isLoading, isEnter) {
    var i;
    for (i = 0; i < STATUS_MOTION_CLASSES.length; i++) {
      btn.classList.remove(STATUS_MOTION_CLASSES[i]);
    }
    var parts = statusMotionClass(status, isLoading, isEnter).replace(/^\s+/, '').split(/\s+/);
    for (i = 0; i < parts.length; i++) {
      if (parts[i]) btn.classList.add(parts[i]);
    }
  }

  function buttonInnerHtml(feature, status, displayText, isLoading, labelClass) {
    var parsed = parseButtonLabel(displayText);
    var unavailable = status === 'disabled' || status === 'error';
    return '<span class="aura-status-icon" aria-hidden="true">' + iconSvgMarkup(feature, status, isLoading) + '</span>' +
      '<span class="aura-status-copy" aria-hidden="true"><span class="aura-status-copy-inner">' + copyInnerMarkup(parsed.label, parsed.scoreMatch, feature) + '</span></span>' +
      (!unavailable && !isLoading ? arrowMarkup() : '') +
      '<span class="aura-status-sr ' + labelClass + '">' + escapeButtonText(displayText) + '</span>';
  }

  function buildStatusButton(btnId, feature, status, displayText, isLoading, labelClass, isEnter) {
    var theme = buttonTheme(status, isLoading);
    var unavailable = status === 'disabled' || status === 'error';
    var enter = isEnter !== false;
    var parsed = parseButtonLabel(displayText);
    var compactClass = ' aura-is-compact';
    var iconOnlyClass = feature !== 'summary' && !parsed.scoreMatch ? ' aura-status-button--icon-only' : '';
    return '<button id="' + btnId + '" type="button" class="aura-status-button' + compactClass + iconOnlyClass + statusMotionClass(status, isLoading, enter) + '" aria-disabled="' + (unavailable || isLoading ? 'true' : 'false') + '" aria-haspopup="dialog" style="--aura-halo-soft:' + haloColor(theme, .14) + ';--aura-halo-strong:' + haloColor(theme, .24) + ';--aura-status-color:' + theme.color + ';--aura-status-tint:' + theme.bg + ';--aura-status-surface:' + theme.surface + ';--aura-status-border:' + theme.border + ';--aura-status-shadow:' + theme.shadow + ';">' +
      buttonInnerHtml(feature, status, displayText, isLoading, labelClass) +
    '</button>';
  }

  var dragSession = {
    isDown: false,
    hasMoved: false,
    startX: 0,
    startY: 0,
    initialLeft: 0,
    initialTop: 0,
    longPressTimer: null,
    longPressTriggered: false,
    targetBtn: null,
    ignoreNextClick: false
  };

  var activeTooltipEl = null;
  function hideTooltip() {
    if (activeTooltipEl && activeTooltipEl.parentNode) {
      activeTooltipEl.classList.remove('aura-embed-tooltip--visible');
      var el = activeTooltipEl;
      activeTooltipEl = null;
      setTimeout(function () {
        if (el.parentNode) el.parentNode.removeChild(el);
      }, 200);
    }
  }

  function showTooltipForButton(btn, text) {
    hideTooltip();
    if (!btn || !text || typeof document === 'undefined') return;
    var tip = document.createElement('div');
    tip.className = 'aura-embed-tooltip';
    tip.textContent = text;

    try {
      if (btn.style && btn.style.getPropertyValue) {
        var c = btn.style.getPropertyValue('--aura-status-color');
        var tint = btn.style.getPropertyValue('--aura-status-tint');
        var surface = btn.style.getPropertyValue('--aura-status-surface');
        var border = btn.style.getPropertyValue('--aura-status-border');
        var shadow = btn.style.getPropertyValue('--aura-status-shadow');
        if (c) tip.style.setProperty('--aura-status-color', c);
        if (tint) tip.style.setProperty('--aura-status-tint', tint);
        if (surface) tip.style.setProperty('--aura-status-surface', surface);
        if (border) tip.style.setProperty('--aura-status-border', border);
        if (shadow) tip.style.setProperty('--aura-status-shadow', shadow);
      }
    } catch (e) {}

    document.body.appendChild(tip);
    activeTooltipEl = tip;

    try {
      var rect = btn.getBoundingClientRect ? btn.getBoundingClientRect() : { left: 0, top: 0, width: 44, height: 40 };
      var tipRect = tip.getBoundingClientRect ? tip.getBoundingClientRect() : { width: 140, height: 32 };
      var top = rect.top - (tipRect.height || 32) - 8;
      if (top < 12) {
        top = rect.bottom + 8;
      }
      var winWidth = (typeof window !== 'undefined' && window.innerWidth) || (document.documentElement && document.documentElement.clientWidth) || 360;
      var tipWidth = tipRect.width || 140;
      var left = rect.left + ((rect.width || 44) - tipWidth) / 2;
      var maxLeft = winWidth - tipWidth - 12;
      if (left < 12) left = 12;
      if (left > maxLeft) left = maxLeft;

      tip.style.top = top + 'px';
      tip.style.left = left + 'px';
      void tip.offsetWidth;
      tip.classList.add('aura-embed-tooltip--visible');
    } catch (e) {}
  }

  function setupDraggableFloatContainer(containerEl) {
    if (!containerEl || typeof containerEl.getAttribute !== 'function' || containerEl.getAttribute('data-aura-drag-bound') === '1') return;
    containerEl.setAttribute('data-aura-drag-bound', '1');

    function onMouseEnter(e) {
      if (dragSession.hasMoved || dragSession.isDown) return;
      var target = e.target;
      while (target && target !== containerEl && !(target.classList && target.classList.contains && target.classList.contains('aura-status-button'))) {
        target = target.parentNode;
      }
      var btn = (target && target.classList && target.classList.contains && target.classList.contains('aura-status-button')) ? target : null;
      if (!btn) return;
      var tipText = (btn.getAttribute && btn.getAttribute('data-aura-copy'))
        || (btn.querySelector && btn.querySelector('.aura-status-text') && btn.querySelector('.aura-status-text').textContent)
        || 'AI 详情';
      var isSummary = btn.id && btn.id.indexOf('Summary') !== -1;
      showTooltipForButton(btn, (isSummary ? 'AI 流程总结 · ' : 'AI 流程审核 · ') + tipText);
    }

    function onMouseLeave() {
      if (!dragSession.isDown) {
        hideTooltip();
      }
    }

    function onPointerDown(e) {
      if (e.type === 'mousedown' && e.button !== 0) return;
      var clientX = e.touches ? e.touches[0].clientX : e.clientX;
      var clientY = e.touches ? e.touches[0].clientY : e.clientY;

      var target = e.target;
      while (target && target !== containerEl && !(target.classList && target.classList.contains && target.classList.contains('aura-status-button'))) {
        target = target.parentNode;
      }
      var btn = (target && target.classList && target.classList.contains && target.classList.contains('aura-status-button')) ? target : null;

      var rect = containerEl.getBoundingClientRect ? containerEl.getBoundingClientRect() : { left: 0, top: 0 };
      dragSession.isDown = true;
      dragSession.hasMoved = false;
      dragSession.longPressTriggered = false;
      dragSession.startX = clientX;
      dragSession.startY = clientY;
      dragSession.initialLeft = rect.left;
      dragSession.initialTop = rect.top;
      dragSession.targetBtn = btn;

      if (dragSession.longPressTimer) {
        clearTimeout(dragSession.longPressTimer);
        dragSession.longPressTimer = null;
      }
      dragSession.longPressTimer = setTimeout(function () {
        if (dragSession.isDown && !dragSession.hasMoved && dragSession.targetBtn) {
          dragSession.longPressTriggered = true;
          dragSession.ignoreNextClick = true;
          var tipText = (dragSession.targetBtn.getAttribute && dragSession.targetBtn.getAttribute('data-aura-copy'))
            || (dragSession.targetBtn.querySelector && dragSession.targetBtn.querySelector('.aura-status-text') && dragSession.targetBtn.querySelector('.aura-status-text').textContent)
            || 'AI 详情';
          var isSummary = dragSession.targetBtn.id && dragSession.targetBtn.id.indexOf('Summary') !== -1;
          showTooltipForButton(dragSession.targetBtn, (isSummary ? 'AI 流程总结 · ' : 'AI 流程审核 · ') + tipText);
          if (typeof navigator !== 'undefined' && navigator.vibrate) {
            try { navigator.vibrate(20); } catch (err) {}
          }
        }
      }, 450);
    }

    function onPointerMove(e) {
      if (!dragSession.isDown) return;
      var clientX = e.touches ? e.touches[0].clientX : e.clientX;
      var clientY = e.touches ? e.touches[0].clientY : e.clientY;
      var dx = clientX - dragSession.startX;
      var dy = clientY - dragSession.startY;

      if (!dragSession.hasMoved && Math.hypot(dx, dy) > 5) {
        dragSession.hasMoved = true;
        dragSession.ignoreNextClick = true;
        if (dragSession.longPressTimer) {
          clearTimeout(dragSession.longPressTimer);
          dragSession.longPressTimer = null;
        }
        hideTooltip();
        if (containerEl.classList && containerEl.classList.add) {
          containerEl.classList.add('aura-is-dragging');
          containerEl.classList.remove('aura-float--bottom-left');
          containerEl.classList.remove('aura-float--top-right');
        }
        if (containerEl.style) {
          containerEl.style.bottom = 'auto';
          containerEl.style.right = 'auto';
        }
      }

      if (dragSession.hasMoved) {
        if (e.cancelable && typeof e.preventDefault === 'function') e.preventDefault();
        var winWidth = (typeof window !== 'undefined' && window.innerWidth) || (document.documentElement && document.documentElement.clientWidth) || 360;
        var winHeight = (typeof window !== 'undefined' && window.innerHeight) || (document.documentElement && document.documentElement.clientHeight) || 640;
        var cWidth = containerEl.offsetWidth || 120;
        var cHeight = containerEl.offsetHeight || 44;

        var minLeft = 8;
        var maxLeft = Math.max(minLeft, winWidth - cWidth - 8);
        var minTop = 8;
        var maxTop = Math.max(minTop, winHeight - cHeight - 8);

        var newLeft = dragSession.initialLeft + dx;
        var newTop = dragSession.initialTop + dy;

        newLeft = Math.max(minLeft, Math.min(maxLeft, newLeft));
        newTop = Math.max(minTop, Math.min(maxTop, newTop));

        if (containerEl.style) {
          containerEl.style.left = newLeft + 'px';
          containerEl.style.top = newTop + 'px';
        }
      }
    }

    function onPointerUp() {
      if (!dragSession.isDown) return;
      dragSession.isDown = false;
      if (dragSession.longPressTimer) {
        clearTimeout(dragSession.longPressTimer);
        dragSession.longPressTimer = null;
      }
      if (dragSession.hasMoved) {
        if (containerEl.classList && containerEl.classList.remove) containerEl.classList.remove('aura-is-dragging');
        setTimeout(function () {
          dragSession.hasMoved = false;
          dragSession.ignoreNextClick = false;
        }, 120);
      } else if (dragSession.longPressTriggered) {
        setTimeout(function () {
          hideTooltip();
          dragSession.longPressTriggered = false;
          dragSession.ignoreNextClick = false;
        }, 1200);
      }
    }

    if (containerEl.addEventListener) {
      containerEl.addEventListener('touchstart', onPointerDown, { passive: true });
      containerEl.addEventListener('mousedown', onPointerDown);
      containerEl.addEventListener('mouseover', onMouseEnter);
      containerEl.addEventListener('mouseleave', onMouseLeave);
    }
    if (typeof window !== 'undefined' && window.addEventListener) {
      window.addEventListener('touchmove', onPointerMove, { passive: false });
      window.addEventListener('touchend', onPointerUp, { passive: true });
      window.addEventListener('touchcancel', onPointerUp, { passive: true });
      window.addEventListener('mousemove', onPointerMove);
      window.addEventListener('mouseup', onPointerUp);
    }
  }

  function createButtonPainter(paint) {
    var timer = null;
    var next = null;
    var shownAt = 0;
    var busy = false;
    function flush() {
      timer = null;
      var payload = next;
      next = null;
      if (!payload) return;
      paint(payload);
      busy = isBusyStatus(payload.status, payload.isLoading);
      if (busy) shownAt = Date.now();
    }
    return function (payload) {
      next = payload;
      if (timer) return;
      var wait = 0;
      if (busy && !prefersReducedMotion()) {
        wait = Math.max(0, BUTTON_BUSY_MIN_MS - (Date.now() - shownAt));
      }
      if (wait === 0) {
        flush();
        return;
      }
      timer = setTimeout(flush, wait);
    };
  }

  function floatAnchorClass() {
    var pos = mobileClient ? FLOAT_POSITION_MOBILE : FLOAT_POSITION_DESKTOP;
    return pos === 'top-right' ? 'aura-float--top-right' : 'aura-float--bottom-left';
  }

  function ensureStatusGroup() {
    var $container = jQuery('#' + BUTTON_CONTAINER_ID);
    var isFloat = false;
    if (!$container.length) {
      if (!jQuery('#auraMobileEmbedFloatContainer').length) {
        jQuery('body').append('<div id="auraMobileEmbedFloatContainer" class="' + floatAnchorClass() + '"></div>');
      }
      $container = jQuery('#auraMobileEmbedFloatContainer');
      isFloat = true;
    }
    var group = $container.children('.aura-status-group')[0];
    if (!group) {
      $container.html('<div class="aura-status-group"></div>');
      group = $container.children('.aura-status-group')[0];
    }
    if (isFloat && $container[0]) {
      setupDraggableFloatContainer($container[0]);
    }
    return group;
  }

  function finishButtonMorph(btn) {
    var icon = btn.querySelector('.aura-status-icon');
    var copy = btn.querySelector('.aura-status-copy');
    var inIcon = icon && icon.querySelector('.aura-status-icon-layer--in');
    var inCopy = copy && copy.querySelector('.aura-status-copy-inner--in');
    if (inIcon) icon.innerHTML = inIcon.innerHTML;
    if (inCopy) copy.innerHTML = '<span class="aura-status-copy-inner">' + inCopy.innerHTML + '</span>';
    var outArrow = btn.querySelector('.aura-status-arrow--out');
    if (outArrow && outArrow.parentNode) outArrow.parentNode.removeChild(outArrow);
    var inArrow = btn.querySelector('.aura-status-arrow--in');
    if (inArrow) inArrow.classList.remove('aura-status-arrow--in');
    btn.classList.remove('aura-status-button--morph');
    btn.style.width = '';
    btn.style.transition = '';
    btn._auraMorphTimer = null;
  }

  function morphStatusButton(btn, feature, status, displayText, isLoading, labelClass) {
    var theme = buttonTheme(status, isLoading);
    var unavailable = status === 'disabled' || status === 'error';
    var parsed = parseButtonLabel(displayText);
    var nextIcon = iconSvgMarkup(feature, status, isLoading);
    var nextCopyInner = copyInnerMarkup(parsed.label, parsed.scoreMatch, feature);
    var showArrow = !unavailable && !isLoading;
    var iconKey = feature + '|' + status + '|' + (isLoading ? '1' : '0');
    var fromWidth = btn.getBoundingClientRect().width;

    if (btn._auraMorphTimer) {
      clearTimeout(btn._auraMorphTimer);
      finishButtonMorph(btn);
      fromWidth = btn.getBoundingClientRect().width;
    }

    applyButtonTheme(btn, theme);
    btn.setAttribute('aria-disabled', (unavailable || isLoading) ? 'true' : 'false');
    btn.classList.add('aura-is-compact');
    if (feature !== 'summary' && !parsed.scoreMatch) {
      btn.classList.add('aura-status-button--icon-only');
    } else {
      btn.classList.remove('aura-status-button--icon-only');
    }
    var sr = btn.querySelector('.aura-status-sr');
    if (sr) {
      sr.className = 'aura-status-sr ' + labelClass;
      sr.textContent = displayText;
    }

    if (prefersReducedMotion()) {
      syncMotionClass(btn, status, isLoading, false);
      btn.innerHTML = buttonInnerHtml(feature, status, displayText, isLoading, labelClass);
      btn.setAttribute('data-aura-icon', iconKey);
      btn.setAttribute('data-aura-copy', displayText);
      return;
    }

    var icon = btn.querySelector('.aura-status-icon');
    var copy = btn.querySelector('.aura-status-copy');
    if (!icon || !copy) {
      syncMotionClass(btn, status, isLoading, true);
      btn.innerHTML = buttonInnerHtml(feature, status, displayText, isLoading, labelClass);
      btn.setAttribute('data-aura-icon', iconKey);
      btn.setAttribute('data-aura-copy', displayText);
      return;
    }

    var iconChanged = btn.getAttribute('data-aura-icon') !== iconKey;
    var copyChanged = btn.getAttribute('data-aura-copy') !== displayText;
    btn.setAttribute('data-aura-icon', iconKey);
    btn.setAttribute('data-aura-copy', displayText);

    if (iconChanged) {
      icon.innerHTML = '<span class="aura-status-icon-layer aura-status-icon-layer--out">' + icon.innerHTML + '</span>' +
        '<span class="aura-status-icon-layer aura-status-icon-layer--in">' + nextIcon + '</span>';
    }
    if (copyChanged) {
      var oldCopy = copy.querySelector('.aura-status-copy-inner');
      var oldCopyHtml = oldCopy ? oldCopy.innerHTML : copy.innerHTML;
      copy.innerHTML = '<span class="aura-status-copy-inner aura-status-copy-inner--out">' + oldCopyHtml + '</span>' +
        '<span class="aura-status-copy-inner aura-status-copy-inner--in">' + nextCopyInner + '</span>';
    }

    var arrow = btn.querySelector('.aura-status-arrow:not(.aura-status-arrow--out)');
    if (showArrow && !arrow) {
      var holder = document.createElement('div');
      holder.innerHTML = arrowMarkup();
      var newArrow = holder.firstChild;
      newArrow.classList.add('aura-status-arrow--in');
      btn.insertBefore(newArrow, sr || null);
    } else if (!showArrow && arrow) {
      arrow.classList.add('aura-status-arrow--out');
    }

    syncMotionClass(btn, status, isLoading, false);
    btn.classList.add('aura-status-button--morph');
    btn.classList.remove('aura-status-button--shine');
    void btn.offsetWidth;
    btn.classList.add('aura-status-button--shine');

    btn.style.transition = 'background-color .45s ease,border-color .45s ease,box-shadow .45s ease,color .22s ease';
    btn.style.width = fromWidth + 'px';
    void btn.offsetWidth;
    btn.style.width = 'auto';
    var toWidth = btn.getBoundingClientRect().width;
    btn.style.width = fromWidth + 'px';
    void btn.offsetWidth;
    btn.style.transition = '';
    if (Math.abs(toWidth - fromWidth) >= 0.5) {
      btn.style.width = toWidth + 'px';
    } else {
      btn.style.width = '';
    }

    btn._auraMorphTimer = setTimeout(function () {
      finishButtonMorph(btn);
    }, 520);
  }

  function upsertStatusButton(group, btnId, feature, status, displayText, isLoading, labelClass) {
    var btn = document.getElementById(btnId);
    if (!btn || btn.parentNode !== group) {
      var wrap = document.createElement('div');
      wrap.innerHTML = buildStatusButton(btnId, feature, status, displayText, isLoading, labelClass, true);
      btn = wrap.firstChild;
      btn.setAttribute('data-aura-icon', feature + '|' + status + '|' + (isLoading ? '1' : '0'));
      btn.setAttribute('data-aura-copy', displayText);
      if (btnId === 'auraMobileEmbedAuditBtn' && group.firstChild) {
        group.insertBefore(btn, group.firstChild);
      } else {
        group.appendChild(btn);
      }
      return btn;
    }
    morphStatusButton(btn, feature, status, displayText, isLoading, labelClass);
    return btn;
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

  /* 沿用 credit-copilot 的扫描带卷曲：相邻带共用边界，连续收拢到触发按钮。 */
  function createEmbedGenie(panel, anchor) {
    var box = panel.getBoundingClientRect();
    var count = Math.max(24, Math.min(48, Math.ceil(box.height / 18)));
    var bandHeight = box.height / count;
    var host = document.createElement('div');
    host.className = 'aura-embed-genie-layer';
    host.setAttribute('aria-hidden', 'true');
    host.inert = true;
    var slices = [];
    var appearance = getComputedStyle(panel);
    // 只绘制外层标题与按钮轮廓；跨域 iframe 不读取、不复制，也不重复加载。
    var shapes = [];
    var styleCache = new Map();
    panel.querySelectorAll('.aura-embed-dialog-title, .aura-embed-dialog-tag, button').forEach(function (node) {
      if (shapes.length >= 28) return;
      var r = node.getBoundingClientRect();
      var left = Math.max(r.left, box.left), top = Math.max(r.top, box.top);
      var right = Math.min(r.right, box.right), bottom = Math.min(r.bottom, box.bottom);
      for (var parent = node.parentElement; parent && parent !== panel; parent = parent.parentElement) {
        var style = styleCache.get(parent);
        if (!style) { style = getComputedStyle(parent); styleCache.set(parent, style); }
        if (style.overflowY !== 'visible') {
          var bounds = parent.getBoundingClientRect();
          top = Math.max(top, bounds.top); bottom = Math.min(bottom, bounds.bottom);
        }
      }
      if (right <= left || bottom <= top) return;
      var text = node.tagName !== 'BUTTON';
      var height = text ? Math.min(5, bottom - top) : bottom - top;
      shapes.push('<rect x="' + (left - box.left) + '" y="' + (top - box.top + (text ? (bottom - top - height) / 2 : 0)) + '" width="' + (text ? Math.min(right - left, box.width * .4) : right - left) + '" height="' + height + '" rx="' + Math.min(10, height / 2) + '" fill="' + (text ? '#c2c9d3' : '#edf1f6') + '"/>');
    });
    var impression = 'url("data:image/svg+xml,' + encodeURIComponent('<svg xmlns="http://www.w3.org/2000/svg" width="' + box.width + '" height="' + box.height + '">' + shapes.join('') + '</svg>') + '")';
    for (var i = 0; i < count; i++) {
      var slice = document.createElement('div');
      Object.assign(slice.style, { position: 'absolute', left: '0', top: '0', width: box.width + 'px', height: (bandHeight + (i === count - 1 ? 0 : .5)) + 'px', overflow: 'hidden', transformOrigin: '0 0', willChange: 'transform' });
      var shell = document.createElement('div');
      Object.assign(shell.style, { width: box.width + 'px', height: box.height + 'px', backgroundColor: appearance.backgroundColor, backgroundImage: impression, borderRadius: appearance.borderRadius, transform: 'translateY(' + (-i * bandHeight) + 'px)' });
      slice.appendChild(shell);
      host.appendChild(slice);
      slices.push(slice);
    }
    document.body.appendChild(host);
    var clamp = function (n, lo, hi) { return Math.max(lo, Math.min(hi, n)); };
    var mix = function (a, b, t) { return a + (b - a) * t; };
    var cx = anchor.left + anchor.width / 2, cy = anchor.top + anchor.height / 2;
    var mouth = clamp(Math.round((cy - box.top) / box.height * count - .5), 0, count - 1);
    var maxDistance = Math.max(mouth, count - 1 - mouth, 1);
    return {
      destroy: function () { host.remove(); },
      apply: function (progress, closing, opacity) {
        var amounts = [], heights = [], tops = [], centers = [];
        for (var j = 0; j < count; j++) {
          var t = clamp((progress - Math.abs(j - mouth) / maxDistance * .16) / .84, 0, 1);
          var amount = t * t * (3 - 2 * t);
          amounts.push(amount);
          heights.push(mix(bandHeight, anchor.height / count, amount));
        }
        tops[mouth] = mix(box.top + (mouth + .5) * bandHeight, cy, amounts[mouth]) - heights[mouth] / 2;
        for (var j = mouth - 1; j >= 0; j--) tops[j] = tops[j + 1] - heights[j];
        for (var j = mouth + 1; j < count; j++) tops[j] = tops[j - 1] + heights[j - 1];
        var shift = anchor.height / count * (mouth + .5 - count / 2) * progress;
        for (var j = 0; j < count; j++) {
          var width = mix(box.width, clamp(anchor.width / box.width, .04, 1) * box.width, amounts[j]);
          var x = mix(box.left + box.width / 2, cx, amounts[j]);
          centers.push({ left: x - width / 2, right: x + width / 2, y: tops[j] + shift + heights[j] / 2 });
        }
        var edges = [{ left: centers[0].left, right: centers[0].right, y: centers[0].y - heights[0] / 2 }];
        for (var j = 1; j < count; j++) edges.push({ left: (centers[j - 1].left + centers[j].left) / 2, right: (centers[j - 1].right + centers[j].right) / 2, y: centers[j].y - heights[j] / 2 });
        edges.push({ left: centers[count - 1].left, right: centers[count - 1].right, y: centers[count - 1].y + heights[count - 1] / 2 });
        slices.forEach(function (slice, j) {
          var top = edges[j], bottom = edges[j + 1];
          var width = top.right - top.left;
          var ratio = width / Math.max(.001, bottom.right - bottom.left);
          var matrix = [width / box.width, 0, 0, 0, (bottom.left * ratio - top.left) / bandHeight, (bottom.y * ratio - top.y) / bandHeight, 0, (ratio - 1) / bandHeight, 0, 0, 1, 0, top.left, top.y, 0, 1];
          slice.style.transform = 'matrix3d(' + matrix.join(',') + ')';
        });
        var tail = progress > .88 ? 1 - Math.pow((progress - .88) / .12, 2) : 1;
        host.style.opacity = opacity * tail;
      }
    };
  }

  // 对三次贝塞尔的时间轴求解，展开快速响应后缓慢落定，收起平稳加速后减速。
  function dialogEase(time, closing) {
    var x1 = closing ? .35 : .22, y1 = closing ? 0 : .65;
    var x2 = closing ? .5 : .2, y2 = 1;
    function sample(t, a, b) { return 3 * (1 - t) * (1 - t) * t * a + 3 * (1 - t) * t * t * b + t * t * t; }
    var low = 0, high = 1, t = time;
    for (var i = 0; i < 14; i++) {
      t = (low + high) / 2;
      if (sample(t, x1, x2) < time) low = t; else high = t;
    }
    return time === 0 || time === 1 ? time : sample(t, y1, y2);
  }

  var activeEmbedSurface = null;

  // 保存整个按钮组的可见状态；双按钮同时让位，关闭后还原到最新业务状态。
  function concealEmbedButtons(trigger) {
    hideTooltip();
    var node = document.getElementById('auraMobileEmbedFloatContainer') || (trigger && trigger.parentNode);
    if (!node) return { paint: function () {}, restore: function () {} };
    var saved = { opacity: node.style.opacity, transform: node.style.transform, pointerEvents: node.style.pointerEvents, inert: node.inert };
    node.style.pointerEvents = 'none';
    node.inert = true;
    return {
      paint: function (amount) {
        node.style.opacity = String(amount);
        node.style.transform = 'scale(' + (.84 + .16 * amount) + ')';
      },
      restore: function () {
        node.style.opacity = saved.opacity;
        node.style.transform = saved.transform;
        node.style.pointerEvents = saved.pointerEvents;
        node.inert = saved.inert;
      }
    };
  }

  // 单功能和双功能共用详情入口；原生模式只负责隐藏/恢复按钮，由 OA 管理弹窗。
  function openAnimatedEmbedDialog(type, targetUrl, trigger, onClosed) {
    if (activeEmbedSurface) return;
    var titleText = type === 'summary' ? 'AI 流程总结' : 'AI 流程审核';
    if (mobileClient && USE_NATIVE_MOBILE_DIALOG && window.weaJs && typeof window.weaJs.showDialog === 'function') {
      var nativeButtons = concealEmbedButtons(trigger);
      var nativeSurface = {};
      activeEmbedSurface = nativeSurface;
      nativeButtons.paint(0);
      function finishNative(refresh) {
        if (activeEmbedSurface !== nativeSurface) return;
        nativeButtons.restore();
        activeEmbedSurface = null;
        window.removeEventListener('pagehide', onNativePageHide);
        if (refresh) onClosed();
      }
      function onNativePageHide() { finishNative(false); }
      window.addEventListener('pagehide', onNativePageHide);
      try {
        window.weaJs.showDialog(targetUrl, {
          title: titleText, moduleName: 'workflow', style: { width: '100%', height: '100%' },
          callback: function () { finishNative(true); }
        });
        return;
      } catch (error) {
        finishNative(false);
        // 原生弹窗不可用时回退到脚本弹层，确保入口不会永久隐藏。
        console.warn('[aura-embed-mobile] OA 原生弹窗打开失败，已使用内嵌弹层');
      }
    }

    var previousFocus = document.activeElement;
    var overlay = document.createElement('div');
    overlay.id = 'auraDesktopEmbedDialog';
    overlay.className = 'aura-embed-dialog-overlay' + (mobileClient ? ' aura-embed-dialog-overlay--mobile' : '');
    var panel = document.createElement('div');
    panel.className = 'aura-embed-dialog-panel';
    panel.setAttribute('role', 'dialog');
    panel.setAttribute('aria-modal', 'true');
    panel.setAttribute('aria-labelledby', 'auraDesktopEmbedTitle');
    panel.tabIndex = -1;
    var header = document.createElement('div');
    header.className = 'aura-embed-dialog-header';
    var heading = document.createElement('div');
    heading.className = 'aura-embed-dialog-heading';
    var title = document.createElement('span');
    title.id = 'auraDesktopEmbedTitle';
    title.className = 'aura-embed-dialog-title';
    title.textContent = titleText;
    var tag = document.createElement('span');
    tag.className = 'aura-embed-dialog-tag';
    tag.textContent = type === 'summary' ? '智能总结 · 仅供参考' : '智审辅助 · 仅供参考';
    heading.appendChild(title);
    heading.appendChild(tag);
    var close = document.createElement('button');
    close.type = 'button';
    close.className = 'aura-embed-dialog-close';
    close.setAttribute('aria-label', '关闭详情');
    close.title = '关闭 (Esc)';
    close.innerHTML = '<svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" aria-hidden="true"><path d="M18 6L6 18M6 6l12 12"/></svg>';
    var frame = document.createElement('iframe');
    frame.className = 'aura-embed-dialog-frame';
    frame.title = titleText;
    frame.src = targetUrl;
    header.appendChild(heading);
    header.appendChild(close);
    panel.appendChild(header);
    panel.appendChild(frame);
    overlay.appendChild(panel);
    document.body.appendChild(overlay);

    var opening = true, disposed = false, progress = 1, raf = 0, layer = null;
    var anchor = trigger && trigger.getBoundingClientRect();
    var buttons = concealEmbedButtons(trigger);
    var previousOverflow = document.body.style.overflow;
    var previousRootOverflow = document.documentElement.style.overflow;
    var background = [];
    activeEmbedSurface = { dismiss: dismiss };
    if (trigger) trigger.setAttribute('aria-expanded', 'true');
    document.body.style.overflow = 'hidden';
    document.documentElement.style.overflow = 'hidden';
    // 只隔离当前文档；不接触跨域 iframe，关闭和离开页面时恢复 OA 的交互状态。
    Array.prototype.forEach.call(document.body.children, function (node) {
      if (node === overlay) return;
      background.push({ node: node, inert: node.inert });
      node.inert = true;
    });
    close.focus();

    function dispose(refresh) {
      if (disposed) return;
      disposed = true;
      if (raf) window.cancelAnimationFrame(raf);
      if (layer) { layer.destroy(); layer = null; }
      document.removeEventListener('keydown', onKeyDown);
      window.removeEventListener('resize', onResize);
      window.removeEventListener('pagehide', onPageHide);
      window.removeEventListener('popstate', onNavigation);
      window.removeEventListener('hashchange', onNavigation);
      overlay.remove();
      background.forEach(function (item) { item.node.inert = item.inert; });
      buttons.restore();
      document.body.style.overflow = previousOverflow;
      document.documentElement.style.overflow = previousRootOverflow;
      activeEmbedSurface = null;
      if (trigger) trigger.setAttribute('aria-expanded', 'false');
      var focus = trigger || previousFocus;
      if (refresh && focus && focus.isConnected && focus.focus) focus.focus({ preventScroll: true });
      if (refresh) onClosed();
    }
    function settle() {
      if (raf) window.cancelAnimationFrame(raf);
      raf = 0;
      if (layer) { layer.destroy(); layer = null; }
      if (!opening) { dispose(true); return; }
      progress = 0;
      panel.style.opacity = '';
      panel.style.visibility = '';
      panel.style.pointerEvents = '';
      overlay.style.backgroundColor = '';
      buttons.paint(0);
      close.focus({ preventScroll: true });
    }
    function paint(closing) {
      var blend = Math.max(0, Math.min(1, 1 - progress / .12));
      blend = blend * blend * (3 - 2 * blend);
      panel.style.visibility = blend > 0 ? 'visible' : 'hidden';
      panel.style.opacity = String(blend);
      layer.apply(progress, closing, 1 - blend);
      overlay.style.backgroundColor = 'rgba(28,31,38,' + (.24 * (1 - progress)) + ')';
      buttons.paint(Math.max(0, Math.min(1, (progress - .86) / .14)));
    }
    function animate(target) {
      if (raf) window.cancelAnimationFrame(raf);
      // 低版本容器和减少动态效果设置直接展示详情，业务功能不依赖动画 API。
      if (prefersReducedMotion() || !window.requestAnimationFrame || !anchor || !anchor.width || !anchor.height) { settle(); return; }
      try {
        if (!layer) layer = createEmbedGenie(panel, anchor);
        paint(target === 1);
      } catch (error) { settle(); return; }
      panel.style.pointerEvents = 'none';
      var from = progress, start = null;
      var duration = Math.max(80, (target === 1 ? 520 : 640) * Math.abs(target - from));
      function tick(now) {
        if (disposed) return;
        if (start === null) start = now;
        var time = Math.min(1, (now - start) / duration);
        progress = from + (target - from) * dialogEase(time, target === 1);
        paint(target === 1);
        if (time < 1) raf = window.requestAnimationFrame(tick);
        else settle();
      }
      raf = window.requestAnimationFrame(tick);
    }
    function dismiss() {
      if (!opening || disposed) return;
      opening = false;
      // 用真实按钮当前位置收回；隐藏仅改变按钮组，测量时临时撤销视觉缩放。
      if (!layer && trigger && trigger.isConnected) {
        buttons.paint(1);
        anchor = trigger.getBoundingClientRect();
        buttons.paint(0);
      }
      animate(1);
    }
    function onKeyDown(event) {
      if (event.key === 'Escape') { event.preventDefault(); dismiss(); }
      if (event.key === 'Tab' && layer) { event.preventDefault(); close.focus(); }
      else if (event.key === 'Tab' && event.shiftKey && document.activeElement === close) { event.preventDefault(); frame.focus(); }
    }
    function onResize() { if (layer) settle(); }
    function onPageHide() { dispose(false); }
    function onNavigation() { dispose(false); }
    close.onclick = dismiss;
    overlay.onclick = function (event) { if (event.target === overlay) dismiss(); };
    document.addEventListener('keydown', onKeyDown);
    window.addEventListener('resize', onResize);
    window.addEventListener('pagehide', onPageHide);
    window.addEventListener('popstate', onNavigation);
    window.addEventListener('hashchange', onNavigation);
    animate(0);
  }

  // =========================================================================
  // 单模式（audit 或 summary）
  // =========================================================================
  if (EMBED_TYPE !== 'all') {
    var statusWatchTimer = null;
    var statusWatchUntil = 0;
    var lastStatusSignature = '';
    var resumeRefreshTimer = null;
    var singleBtnRuntime = { isLoading: false, isDisabled: false, errorMessage: '', onClick: null };
    var paintSingleButton = createButtonPainter(function (payload) {
      renderButton(payload.status, payload.textOverride, payload.errorMessage, payload.isLoading, payload.onClick);
    });

    function openEmbedDialog(requestId, userId) {
      var path = EMBED_TYPE === 'summary' ? '/embed/summary' : '/embed/audit';
      var targetUrl = AURA_EMBED_ORIGIN + path
        + '?requestid=' + encodeURIComponent(requestId)
        + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
        + '&oa_user_id=' + encodeURIComponent(userId);

      console.log('[aura-embed-mobile] 调起嵌入弹窗');
      startStatusWatch(STATUS_WATCH_MS);

      openAnimatedEmbedDialog(EMBED_TYPE, targetUrl, document.getElementById('auraMobileEmbedBtn'), function () {
        refreshCurrentStatus(true);
      });
    }

    function renderButton(status, textOverride, errorMessage, isLoading, onClick) {
      var currentStatus = isLoading ? statusConfig.loading : (statusConfig[status] || statusConfig.gray);
      var displayText = textOverride || currentStatus.text;
      singleBtnRuntime.isLoading = !!isLoading;
      singleBtnRuntime.isDisabled = status === 'disabled' || status === 'error';
      singleBtnRuntime.errorMessage = errorMessage || '';
      singleBtnRuntime.onClick = onClick;

      var group = ensureStatusGroup();
      upsertStatusButton(group, 'auraMobileEmbedBtn', EMBED_TYPE, status, displayText, isLoading, 'aura-mobile-label');
      if (group.getAttribute('data-aura-bound') === '1') return;
      group.setAttribute('data-aura-bound', '1');
      group.addEventListener('click', function (event) {
        if (dragSession.ignoreNextClick || dragSession.hasMoved || dragSession.longPressTriggered) {
          if (event && event.preventDefault) event.preventDefault();
          if (event && event.stopPropagation) event.stopPropagation();
          return;
        }
        var target = event.target;
        while (target && target !== group && !(target.classList && target.classList.contains('aura-status-button'))) {
          target = target.parentNode;
        }
        if (!target || target === group) return;
        if (singleBtnRuntime.isLoading) {
          if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
            WfForm.showMessage('数据加载中，请稍候...', 2, 2);
          }
          return;
        }
        if (singleBtnRuntime.isDisabled) {
          if (typeof WfForm !== 'undefined' && WfForm.showMessage) {
            WfForm.showMessage(singleBtnRuntime.errorMessage || '当前节点暂不可用', 2, 3);
          }
          return;
        }
        if (typeof singleBtnRuntime.onClick === 'function') {
          singleBtnRuntime.onClick();
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
      paintSingleButton({
        status: status,
        textOverride: textOverride,
        errorMessage: errorMessage,
        isLoading: isLoading,
        onClick: onClick
      });
    }

    function applyResultToButton(isSummary, result, hasResult, shouldAutoRun, runningJobId) {
      var name = featureName();
      var openDetails = openCurrentDetails;
      if (runningJobId) {
        renderStatusButton('loading', name + '分析中，查看进度', '', false, openDetails);
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
        renderStatusButton('loading', name + '分析中，查看进度', '', false, openDetails);
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
        renderStatusButton('disabled', EMBED_TYPE === 'summary' ? '保存查看AI总结' : '保存查看AI智审', '流程保存并生成编号后即可查看 AI ' + featureName(), false);
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

  function openDualEmbedDialog(type, requestId, userId) {
    var path = type === 'summary' ? '/embed/summary' : '/embed/audit';
    var targetUrl = AURA_EMBED_ORIGIN + path
      + '?requestid=' + encodeURIComponent(requestId)
      + '&embed_token=' + encodeURIComponent(EMBED_ACCESS_TOKEN)
      + '&oa_user_id=' + encodeURIComponent(userId);

    console.log('[aura-embed-mobile] 调起嵌入弹窗: ' + type);
    startDualStatusWatch(STATUS_WATCH_MS);

    var trigger = document.getElementById(type === 'summary' ? 'auraMobileEmbedSummaryBtn' : 'auraMobileEmbedAuditBtn');
    openAnimatedEmbedDialog(type, targetUrl, trigger, function () { refreshDualStatus(type, true); });
  }

  function bindDualGroupClicks(group) {
    if (group.getAttribute('data-aura-bound') === '1') return;
    group.setAttribute('data-aura-bound', '1');
    group.addEventListener('click', function (event) {
      if (dragSession.ignoreNextClick || dragSession.hasMoved || dragSession.longPressTriggered) {
        if (event && event.preventDefault) event.preventDefault();
        if (event && event.stopPropagation) event.stopPropagation();
        return;
      }
      var target = event.target;
      while (target && target !== group && !(target.classList && target.classList.contains('aura-status-button'))) {
        target = target.parentNode;
      }
      if (!target || target === group) return;
      var featType = target.id === 'auraMobileEmbedSummaryBtn' ? 'summary' : 'audit';
      var item = dualState[featType];
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

  function renderDualFeature(featType) {
    var item = dualState[featType];
    var currentStatus = item.isLoading ? statusConfig.loading : (statusConfig[item.status] || statusConfig.gray);
    var displayText = item.text || currentStatus.text;
    var btnId = featType === 'summary' ? 'auraMobileEmbedSummaryBtn' : 'auraMobileEmbedAuditBtn';
    var group = ensureStatusGroup();
    upsertStatusButton(group, btnId, featType, item.status, displayText, item.isLoading, 'aura-mobile-label-' + featType);
    bindDualGroupClicks(group);
  }

  var paintDualFeature = {
    audit: createButtonPainter(function () { renderDualFeature('audit'); }),
    summary: createButtonPainter(function () { renderDualFeature('summary'); })
  };

  function setDualFeatureState(featType, status, text, errorMessage, isLoading) {
    var sig = status + '|' + text + '|' + (isLoading ? '1' : '0');
    if (dualState[featType].lastSig === sig) return;
    dualState[featType].lastSig = sig;
    dualState[featType].status = status;
    dualState[featType].text = text;
    dualState[featType].errorMessage = errorMessage || '';
    dualState[featType].isLoading = !!isLoading;
    paintDualFeature[featType]({ status: status, isLoading: isLoading });
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
      setDualFeatureState(featType, 'loading', name + '分析中...', '', false);
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
      setDualFeatureState(featType, 'loading', name + '分析中...', '', false);
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
        setDualFeatureState(featType, 'loading', (featType === 'summary' ? '总结' : '审核') + '分析中...', '', false);
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
      setDualFeatureState('audit', 'disabled', '保存查看AI智审', '流程保存并生成编号后即可查看 AI 审核', false);
      setDualFeatureState('summary', 'disabled', '保存查看AI总结', '流程保存并生成编号后即可查看 AI 总结', false);
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
