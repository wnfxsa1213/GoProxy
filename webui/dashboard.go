package webui

const dashboardHTML = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>GoProxy — 智能代理池</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=Share+Tech+Mono&display=swap" rel="stylesheet">
<style>
*{box-sizing:border-box;margin:0;padding:0}
:root{
  --bg:#0a0a0a;
  --bg-elevated:#111;
  --bg-card:#0d0d0d;
  --fg:#00ff41;
  --fg-dim:#00cc33;
  --fg-text:#0f0;
  --border:#1a3a1a;
  --border-heavy:#00ff41;
  --gray-1:#0d0d0d;
  --gray-2:#151515;
  --gray-3:#1a1a1a;
  --gray-4:#2a4a2a;
  --gray-5:#00aa2a;
  --gray-6:#00dd38;
  --green:#00ff41;
  --yellow:#ffff00;
  --orange:#ff8800;
  --red:#ff0033;
  --mono:JetBrains Mono,Share Tech Mono,monospace;
  --sans:JetBrains Mono,monospace;
}
body{background:var(--bg);color:var(--fg);font-family:var(--mono);font-size:14px;line-height:1.5;-webkit-font-smoothing:antialiased;position:relative}

/* CRT 扫描线效果 */
body::before{content:'';position:fixed;top:0;left:0;width:100%;height:100%;background:repeating-linear-gradient(0deg,rgba(0,255,65,0.03) 0px,transparent 1px,transparent 2px,rgba(0,255,65,0.03) 3px);pointer-events:none;z-index:9999}

/* 荧光光晕效果 */
body::after{content:'';position:fixed;top:0;left:0;width:100%;height:100%;background:radial-gradient(ellipse at center,rgba(0,255,65,0.05) 0%,transparent 70%);pointer-events:none;z-index:9998}

.layout{max-width:1800px;margin:0 auto;padding:0 32px}

/* 双列布局 */
.content-grid{display:grid;grid-template-columns:1fr 420px;gap:32px;align-items:start}
.main-content{min-width:0;position:relative}
.sidebar{position:sticky;top:32px}

/* 控制面板 */
.control-panel{background:var(--bg-card);border:1px solid var(--border-heavy);padding:20px;margin-bottom:20px;box-shadow:0 0 20px rgba(0,255,65,0.15)}
.control-header{display:flex;align-items:center;justify-content:center;margin-bottom:16px;padding-bottom:12px;border-bottom:1px solid var(--border)}
.control-title{font-size:14px;font-weight:700;letter-spacing:0.12em;font-family:var(--mono);text-transform:uppercase;color:var(--fg);text-shadow:0 0 10px var(--fg)}
.control-ops{display:flex;flex-direction:column;gap:8px}
.ctrl-btn-primary{width:100%;padding:10px;font-size:10px;font-weight:600;cursor:pointer;border:1px solid var(--border-heavy);background:var(--bg-card);color:var(--fg);font-family:var(--mono);text-transform:uppercase;letter-spacing:0.08em;transition:all 0.2s}
.ctrl-btn-primary:hover{background:var(--border);box-shadow:0 0 15px var(--border-heavy);color:var(--fg);text-shadow:0 0 5px var(--fg)}
.ctrl-btn-secondary{width:100%;padding:8px;font-size:9px;font-weight:600;cursor:pointer;border:1px solid var(--border);background:var(--bg-card);color:var(--fg-dim);font-family:var(--mono);text-transform:uppercase;letter-spacing:0.08em;transition:all 0.2s}
.ctrl-btn-secondary:hover{background:var(--border);color:var(--fg);box-shadow:0 0 8px var(--border)}

/* 代理列表区域 */
.proxy-section{display:block}
.proxy-header{position:sticky;top:0;z-index:100;background:var(--bg);padding:20px 0 16px;border-bottom:1px solid var(--border-heavy);display:flex;align-items:center;justify-content:space-between;gap:24px;backdrop-filter:blur(8px);box-shadow:0 2px 0 0 rgba(0,255,65,0.2)}
.proxy-logo-area{display:flex;align-items:baseline;gap:12px;flex-shrink:0}
.proxy-logo{font-size:28px;font-weight:900;letter-spacing:0.2em;font-family:var(--mono);text-transform:uppercase;color:var(--fg);text-shadow:0 0 15px var(--fg),0 0 30px var(--fg);animation:glow 2s ease-in-out infinite alternate}
@keyframes glow{0%{text-shadow:0 0 15px var(--fg),0 0 30px var(--fg)}100%{text-shadow:0 0 20px var(--fg),0 0 40px var(--fg),0 0 60px var(--fg)}}
.user-badge{font-size:10px;color:var(--fg-dim);font-family:var(--mono);letter-spacing:0.08em;opacity:0.6}
.proxy-content{}
.header-actions{display:flex;gap:8px;align-items:center;flex-shrink:0}

/* 响应式：屏幕小于1200px时变为单列 */
@media (max-width: 1200px) {
  .content-grid{grid-template-columns:1fr;height:auto}
  .sidebar{overflow-y:visible;padding-right:0}
  .main-content{overflow:visible}
  .proxy-section{height:auto;overflow:visible}
  .proxy-content{overflow-y:visible}
}

/* Health Grid - 侧边栏紧凑布局 */
.health-grid{display:grid;grid-template-columns:repeat(2,1fr);gap:2px;background:var(--bg);border:1px solid var(--border);margin-bottom:16px;box-shadow:0 0 20px rgba(0,255,65,0.1)}
.health-card{background:var(--bg-card);padding:16px;position:relative;border:1px solid var(--border)}
.health-label{font-size:8px;text-transform:uppercase;letter-spacing:0.15em;color:var(--fg-dim);margin-bottom:8px;font-weight:600;font-family:var(--mono)}
.health-value{font-size:24px;font-weight:700;font-family:var(--mono);line-height:1;letter-spacing:0.05em;color:var(--fg);text-shadow:0 0 10px var(--fg)}
.health-status{position:absolute;top:16px;right:16px;width:8px;height:8px;border-radius:50%}
.health-status.healthy{background:var(--green);box-shadow:0 0 8px var(--green)}
.health-status.warning{background:var(--orange);box-shadow:0 0 8px var(--orange)}
.health-status.critical{background:var(--red);box-shadow:0 0 8px var(--red)}
.health-status.emergency{background:var(--red);box-shadow:0 0 15px var(--red),0 0 0 3px rgba(255,0,51,0.3);animation:pulse 1s infinite}
.health-meta{font-size:9px;color:var(--gray-5);margin-top:6px;font-family:var(--mono)}

@keyframes pulse{0%,100%{opacity:1}50%{opacity:0.6}}

/* Tabs/按钮样式 */
.tab{padding:8px 16px;min-height:36px;font-size:10px;font-weight:600;cursor:pointer;border:1px solid var(--border);background:var(--bg-card);color:var(--fg-dim);font-family:var(--mono);transition:all 0.2s;text-transform:uppercase;letter-spacing:0.05em;display:inline-flex;align-items:center;justify-content:center;text-decoration:none;box-sizing:border-box}
.tab:hover{background:var(--border);color:var(--fg);box-shadow:0 0 8px var(--border)}

/* 筛选下拉框 */
.filter-select{padding:8px 16px;min-height:36px;font-size:10px;font-weight:600;cursor:pointer;border:1px solid var(--border);background:var(--bg-card);color:var(--fg-dim);font-family:var(--mono);text-transform:uppercase;letter-spacing:0.05em;transition:all 0.2s;outline:none;appearance:none;background-image:url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%2300ff41' d='M6 9L1 4h10z'/%3E%3C/svg%3E");background-repeat:no-repeat;background-position:right 8px center;padding-right:32px}
.filter-select:hover{background-color:var(--border);color:var(--fg);box-shadow:0 0 8px var(--border)}
.filter-select option{background:var(--bg-card);color:var(--fg-dim)}

/* Quality Bar - 侧边栏紧凑布局 */
.quality-bar{background:var(--bg-card);border:1px solid var(--border);padding:16px;margin-bottom:16px;box-shadow:0 0 15px rgba(0,255,65,0.08)}
.quality-bar-title{font-size:8px;text-transform:uppercase;letter-spacing:0.15em;color:var(--fg-dim);margin-bottom:10px;font-weight:600}
.quality-visual{display:flex;height:20px;border:1px solid var(--border);overflow:hidden;box-shadow:inset 0 0 10px rgba(0,255,65,0.1)}
.quality-segment{display:flex;align-items:center;justify-content:center;font-size:9px;font-weight:700;font-family:var(--mono);color:#000;transition:width 0.3s;text-shadow:none}
.quality-s{background:var(--green);box-shadow:0 0 10px var(--green)}
.quality-a{background:var(--yellow);box-shadow:0 0 10px var(--yellow)}
.quality-b{background:var(--orange);box-shadow:0 0 10px var(--orange)}
.quality-c{background:var(--red);box-shadow:0 0 10px var(--red)}
.quality-legend{display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-top:10px}
.quality-legend-item{font-size:9px;font-family:var(--mono);color:var(--fg-dim)}
.quality-legend-dot{display:inline-block;width:6px;height:6px;margin-right:5px;box-shadow:0 0 4px currentColor}

/* 操作按钮样式 */
.btn-danger{border:1px solid var(--red);color:var(--red);padding:5px 10px;font-size:9px;text-transform:uppercase;letter-spacing:0.08em;background:var(--bg-card);cursor:pointer;transition:all 0.2s}
.btn-danger:hover{background:var(--red);color:#000;box-shadow:0 0 10px var(--red)}
.btn-action{border:1px solid var(--border);color:var(--fg-dim);padding:5px 10px;font-size:9px;text-transform:uppercase;letter-spacing:0.08em;background:var(--bg-card);margin-left:6px;cursor:pointer;transition:all 0.2s}
.btn-action:hover{background:var(--border);color:var(--fg);box-shadow:0 0 8px var(--border)}

/* Table */
table{width:100%;border-collapse:collapse;font-size:11px;font-family:var(--mono);border:1px solid var(--border);background:var(--bg-card)}
thead{position:sticky;top:78px;z-index:50;border-bottom:1px solid var(--border-heavy);background:var(--bg-elevated);box-shadow:0 2px 8px rgba(0,0,0,0.3)}
th{padding:10px 12px;text-align:left;font-size:9px;text-transform:uppercase;letter-spacing:0.12em;color:var(--fg-dim);font-weight:600}
td{padding:12px;border-bottom:1px solid var(--border);color:var(--fg-dim)}
tr:last-child td{border-bottom:none}
tr:hover{background:var(--gray-2);box-shadow:inset 0 0 20px rgba(0,255,65,0.05)}
.cell-mono{font-family:var(--mono);font-size:10px}
.cell-grade{font-weight:700;font-size:14px}
.cell-clickable{cursor:pointer;transition:all 0.2s}
.cell-clickable:hover{background:var(--border)!important;color:var(--fg)!important;box-shadow:0 0 8px var(--border)!important}
.cell-clickable:active{background:var(--border-heavy)!important}
.grade-s{color:var(--green);text-shadow:0 0 8px var(--green)}
.grade-a{color:var(--yellow);text-shadow:0 0 8px var(--yellow)}
.grade-b{color:var(--orange);text-shadow:0 0 8px var(--orange)}
.grade-c{color:var(--red);text-shadow:0 0 8px var(--red)}
.badge{display:inline-block;padding:3px 8px;font-size:9px;font-weight:600;text-transform:uppercase;letter-spacing:0.08em;border:1px solid;font-family:var(--mono)}
.badge-http{border-color:var(--fg-dim);color:var(--fg-dim);background:transparent}
.badge-socks5{background:var(--fg-dim);color:#000;border-color:var(--fg-dim);box-shadow:0 0 6px var(--fg-dim)}
.latency{font-weight:600}
.latency-excellent{color:var(--green)}
.latency-good{color:#333}
.latency-fair{color:#666}
.latency-poor{color:var(--red)}

/* Modal */
.modal-overlay{display:none;position:fixed;inset:0;background:rgba(0,0,0,0.95);backdrop-filter:blur(10px);z-index:100;align-items:center;justify-content:center}
.modal-overlay.show{display:flex}
.modal{background:var(--bg-elevated);border:1px solid var(--border-heavy);padding:40px;width:700px;box-shadow:0 0 40px rgba(0,255,65,0.3);max-height:90vh;overflow-y:auto}
.modal-title{font-size:20px;font-weight:700;margin-bottom:28px;letter-spacing:0.08em;text-transform:uppercase;color:var(--fg);text-shadow:0 0 10px var(--fg)}
.form-section{margin-bottom:28px}
.form-section-title{font-size:9px;text-transform:uppercase;letter-spacing:0.12em;color:var(--fg-dim);margin-bottom:12px;font-weight:600;padding-bottom:8px;border-bottom:1px solid var(--border)}
.form-grid{display:grid;grid-template-columns:1fr 1fr;gap:16px}
.form-group{display:flex;flex-direction:column}
.form-group label{font-size:9px;text-transform:uppercase;letter-spacing:0.08em;color:var(--fg-dim);margin-bottom:6px;font-weight:600}
.form-group input{padding:10px;background:var(--bg-card);border:1px solid var(--border);font-size:12px;font-family:var(--mono);color:var(--fg);outline:none;transition:all 0.2s}
.form-group input:focus{border-color:var(--border-heavy);background:var(--bg-elevated);box-shadow:0 0 10px var(--border-heavy)}
.form-help{font-size:9px;color:var(--gray-5);margin-top:4px;font-family:var(--mono)}
.modal-actions{display:flex;gap:12px;margin-top:28px;padding-top:28px;border-top:1px solid var(--border)}
.modal-actions .btn{flex:1;padding:12px 24px;font-size:11px;font-weight:600;cursor:pointer;border:1px solid var(--border-heavy);background:var(--bg-card);color:var(--fg);font-family:var(--mono);text-transform:uppercase;letter-spacing:0.08em;transition:all 0.2s}
.modal-actions .btn:hover{background:var(--border);box-shadow:0 0 15px var(--border-heavy);color:var(--fg);text-shadow:0 0 5px var(--fg)}
.modal-actions .btn-secondary{border:1px solid var(--border);background:var(--bg-card);color:var(--fg-dim)}
.modal-actions .btn-secondary:hover{background:var(--gray-2);color:var(--fg);box-shadow:0 0 10px var(--border)}

/* Log - 适配侧边栏布局 */
.log-box{padding:12px;background:var(--bg);border:1px solid var(--border);font-family:var(--mono);font-size:10px;color:var(--fg-dim);height:350px;overflow-y:auto;line-height:1.8;box-shadow:inset 0 0 20px rgba(0,255,65,0.05)}
.log-box::-webkit-scrollbar{width:4px}
.log-box::-webkit-scrollbar-track{background:var(--bg)}
.log-box::-webkit-scrollbar-thumb{background:var(--border);border-radius:2px}
.log-box::-webkit-scrollbar-thumb:hover{background:var(--border-heavy)}
.log-line{padding:3px 0;opacity:0.85}
.log-line.error{color:var(--red);font-weight:600;text-shadow:0 0 5px var(--red)}
.log-line.success{color:var(--green);text-shadow:0 0 5px var(--green)}

/* 侧边栏样式 */
.sidebar>*:not(:last-child){margin-bottom:16px}
.sidebar .section{margin-bottom:0;border:1px solid var(--border);background:var(--bg-card);padding:16px;box-shadow:0 0 15px rgba(0,255,65,0.1)}
.sidebar .section-header{padding-bottom:10px;margin-bottom:12px;border-bottom:1px solid var(--border)}
.sidebar .section-title{font-size:12px;letter-spacing:0.12em}

/* 响应式布局 */
@media (max-width: 1200px) {
  .content-grid{grid-template-columns:1fr}
  .sidebar{position:static}
  .health-grid{grid-template-columns:repeat(4,1fr)}
  .health-card{padding:20px}
  .health-value{font-size:32px}
  .log-box{height:400px}
  .sidebar .section{border:1px solid var(--border)}
}

.empty{padding:48px;text-align:center;color:var(--gray-4);font-size:12px;font-family:var(--mono);text-transform:uppercase;letter-spacing:0.08em}

/* 权限控制 - 默认隐藏管理员功能 */
.admin-only{display:none}

/* Toast 提示 */
.toast{position:fixed;bottom:32px;left:50%;transform:translateX(-50%) translateY(100px);background:var(--fg);color:#000;padding:12px 24px;font-size:11px;font-weight:600;font-family:var(--mono);opacity:0;transition:all 0.3s;z-index:1000;pointer-events:none;box-shadow:0 0 20px var(--fg);text-transform:uppercase;letter-spacing:0.05em}
.toast.show{transform:translateX(-50%) translateY(0);opacity:1}
</style>
</head>
<body>
<div class="layout">
  <div class="content-grid">
    <div class="main-content">
      <div class="proxy-section">
        <div class="proxy-header">
          <div class="proxy-logo-area">
            <div class="proxy-logo">[ GoProxy ]</div>
            <span id="user-mode" class="user-badge">guest</span>
          </div>
          <div class="header-actions">
            <select class="filter-select" id="protocol-filter" onchange="setProtocolFilter(this.value)">
              <option value="" id="protocol-filter-label">协议</option>
              <option value="http">HTTP</option>
              <option value="socks5">SOCKS5</option>
            </select>
            <select class="filter-select" id="country-filter" onchange="setCountryFilter(this.value)">
              <option value="" id="country-filter-label">出口国家</option>
            </select>
            <button class="tab" onclick="toggleLang()" id="lang-btn">[ EN ]</button>
            <a href="https://github.com/isboyjc/ProxyGo" target="_blank" class="tab" title="GitHub">
              <svg width="14" height="14" viewBox="0 0 16 16" fill="currentColor" style="vertical-align: middle;">
                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z"/>
              </svg>
            </a>
            <a href="/login" class="tab" id="login-link" style="display: none;" data-i18n="nav.login">登录</a>
            <a href="/logout" class="tab admin-only" data-i18n="nav.logout">退出</a>
          </div>
        </div>
        <div class="proxy-content">
          <div id="proxy-table-wrap"><div class="empty" data-i18n="proxy.loading">加载中...</div></div>
        </div>
      </div>
    </div>

    <aside class="sidebar">
      <div class="control-panel admin-only">
        <div class="control-header">
          <div class="control-title">[ CONTROL_PANEL ]</div>
        </div>
        <div class="control-ops">
          <button class="ctrl-btn-primary" onclick="triggerFetch()" data-i18n="actions.fetch">抓取代理</button>
          <button class="ctrl-btn-secondary" onclick="refreshLatency()" data-i18n="actions.refresh">刷新延迟</button>
          <button class="ctrl-btn-secondary" onclick="openSettings()" data-i18n="actions.config">配置池子</button>
        </div>
      </div>

      <div class="health-grid">
        <div class="health-card">
          <div class="health-label" data-i18n="health.status">池子状态</div>
          <div class="health-value" id="pool-state" style="font-size:18px;text-transform:uppercase">—</div>
          <div class="health-status" id="pool-status-dot"></div>
        </div>
        <div class="health-card">
          <div class="health-label" data-i18n="health.total">总代理数</div>
          <div class="health-value" id="stat-total">0</div>
          <div class="health-meta"><span id="stat-capacity">0</span> <span data-i18n="health.capacity">容量</span></div>
        </div>
        <div class="health-card">
          <div class="health-label">HTTP</div>
          <div class="health-value" id="stat-http">0</div>
          <div class="health-meta"><span id="http-slots">0</span> <span data-i18n="health.slots">槽位</span> · <span id="http-avg">—</span>ms <span data-i18n="health.avg">平均</span></div>
        </div>
        <div class="health-card">
          <div class="health-label">SOCKS5</div>
          <div class="health-value" id="stat-socks5">0</div>
          <div class="health-meta"><span id="socks5-slots">0</span> <span data-i18n="health.slots">槽位</span> · <span id="socks5-avg">—</span>ms <span data-i18n="health.avg">平均</span></div>
        </div>
      </div>

      <div class="quality-bar">
        <div class="quality-bar-title" data-i18n="quality.title">质量分布</div>
        <div class="quality-visual" id="quality-visual">
          <div class="quality-segment quality-s" style="width:0%"></div>
          <div class="quality-segment quality-a" style="width:0%"></div>
          <div class="quality-segment quality-b" style="width:0%"></div>
          <div class="quality-segment quality-c" style="width:0%"></div>
        </div>
        <div class="quality-legend">
          <div class="quality-legend-item"><span class="quality-legend-dot" style="background:#22c55e"></span><span data-i18n="quality.grade_s">S级</span> (<span id="grade-s-count">0</span>)</div>
          <div class="quality-legend-item"><span class="quality-legend-dot" style="background:#eab308"></span><span data-i18n="quality.grade_a">A级</span> (<span id="grade-a-count">0</span>)</div>
          <div class="quality-legend-item"><span class="quality-legend-dot" style="background:#f97316"></span><span data-i18n="quality.grade_b">B级</span> (<span id="grade-b-count">0</span>)</div>
          <div class="quality-legend-item"><span class="quality-legend-dot" style="background:#ef4444"></span><span data-i18n="quality.grade_c">C级</span> (<span id="grade-c-count">0</span>)</div>
        </div>
      </div>

      <div class="section">
        <div class="section-header">
          <h2 class="section-title" data-i18n="log.title">系统日志</h2>
        </div>
        <div class="log-box" id="logs-box"><span data-i18n="log.loading">加载中...</span></div>
        <div style="font-size:10px;color:var(--gray-5);font-family:var(--mono);margin-top:8px;text-align:center">
          <span data-i18n="log.auto_refresh_label">自动刷新</span>: <span id="log-countdown" style="color:var(--fg-dim);font-weight:600">5</span>s
        </div>
      </div>
    </aside>
  </div>
</div>

<div class="modal-overlay" id="settings-modal" onclick="if(event.target===this) closeSettings()">
  <div class="modal">
    <div class="modal-title" data-i18n="config.title">池子配置</div>
    
    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_capacity">池子容量</div>
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="config.max_size">最大容量</label>
          <input type="number" id="cfg-pool-size" min="10" max="500">
          <div class="form-help" data-i18n="config.max_size_help">代理池总槽位数</div>
        </div>
        <div class="form-group">
          <label data-i18n="config.http_ratio">HTTP占比</label>
          <input type="number" id="cfg-http-ratio" min="0" max="1" step="0.05">
          <div class="form-help" data-i18n="config.http_ratio_help">0.5 = 50% HTTP, 50% SOCKS5</div>
        </div>
        <div class="form-group">
          <label data-i18n="config.min_per_protocol">每协议最小数</label>
          <input type="number" id="cfg-min-per-protocol" min="1" max="50">
          <div class="form-help" data-i18n="config.min_per_protocol_help">最小保证数量</div>
        </div>
      </div>
    </div>

    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_latency">延迟标准 (ms)</div>
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="config.latency_standard">标准模式</label>
          <input type="number" id="cfg-max-latency" min="500" max="5000" step="100">
        </div>
        <div class="form-group">
          <label data-i18n="config.latency_healthy">健康模式</label>
          <input type="number" id="cfg-max-latency-healthy" min="500" max="3000" step="100">
        </div>
        <div class="form-group">
          <label data-i18n="config.latency_emergency">紧急模式</label>
          <input type="number" id="cfg-max-latency-emergency" min="1000" max="5000" step="100">
        </div>
      </div>
    </div>

    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_validation">验证与健康检查</div>
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="config.validate_concurrency">验证并发数</label>
          <input type="number" id="cfg-concurrency" min="50" max="500" step="50">
        </div>
        <div class="form-group">
          <label data-i18n="config.validate_timeout">验证超时(秒)</label>
          <input type="number" id="cfg-timeout" min="3" max="15">
        </div>
        <div class="form-group">
          <label data-i18n="config.health_interval">检查间隔(分钟)</label>
          <input type="number" id="cfg-health-interval" min="1" max="60">
        </div>
        <div class="form-group">
          <label data-i18n="config.health_batch">每批数量</label>
          <input type="number" id="cfg-health-batch" min="10" max="100" step="10">
        </div>
      </div>
    </div>

    <div class="form-section">
      <div class="form-section-title" data-i18n="config.section_optimization">优化设置</div>
      <div class="form-grid">
        <div class="form-group">
          <label data-i18n="config.optimize_interval">优化间隔(分钟)</label>
          <input type="number" id="cfg-optimize-interval" min="10" max="120" step="10">
        </div>
        <div class="form-group">
          <label data-i18n="config.replace_threshold">替换阈值</label>
          <input type="number" id="cfg-replace-threshold" min="0.5" max="0.9" step="0.05">
          <div class="form-help" data-i18n="config.replace_threshold_help">新代理需快30%</div>
        </div>
      </div>
    </div>

    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="closeSettings()" data-i18n="config.cancel">取消</button>
      <button class="btn" onclick="saveConfig()" data-i18n="config.save">保存配置</button>
    </div>
  </div>
</div>

<script>
// 国际化翻译
const i18n = {
  zh: {
    'nav.config': '配置',
    'nav.login': '登录',
    'nav.logout': '退出',
    'health.status': '池子状态',
    'health.total': '总代理数',
    'health.capacity': '容量',
    'health.slots': '槽位',
    'health.avg': '平均',
    'health.state.healthy': '健康',
    'health.state.warning': '警告',
    'health.state.critical': '危急',
    'health.state.emergency': '紧急',
    'quality.title': '质量分布',
    'quality.grade_s': 'S级',
    'quality.grade_a': 'A级',
    'quality.grade_b': 'B级',
    'quality.grade_c': 'C级',
    'actions.fetch': '抓取代理',
    'actions.refresh': '刷新延迟',
    'actions.config': '配置池子',
    'proxy.title': '代理列表',
    'proxy.tab_all': '全部',
    'proxy.filter_protocol': '协议',
    'proxy.filter_country': '出口国家',
    'proxy.loading': '加载中...',
    'proxy.empty': '暂无代理',
    'proxy.th_grade': '等级',
    'proxy.th_score': '分数',
    'proxy.th_protocol': '协议',
    'proxy.th_address': '地址',
    'proxy.th_exit_ip': '出口IP',
    'proxy.th_location': '位置',
    'proxy.th_latency': '延迟',
    'proxy.th_usage': '使用统计',
    'proxy.th_action': '操作',
    'proxy.btn_delete': '删除',
    'proxy.btn_refresh': '刷新',
    'proxy.copy_success': '已复制',
    'proxy.refresh_started': '刷新已启动',
    'log.title': '系统日志',
    'log.auto_refresh_label': '自动刷新',
    'log.loading': '加载中...',
    'log.empty': '暂无日志',
    'config.title': '池子配置',
    'config.section_capacity': '池子容量',
    'config.max_size': '最大容量',
    'config.max_size_help': '代理池总槽位数',
    'config.http_ratio': 'HTTP占比',
    'config.http_ratio_help': '0.5 = 50% HTTP, 50% SOCKS5',
    'config.min_per_protocol': '每协议最小数',
    'config.min_per_protocol_help': '最小保证数量',
    'config.section_latency': '延迟标准 (ms)',
    'config.latency_standard': '标准模式',
    'config.latency_healthy': '健康模式',
    'config.latency_emergency': '紧急模式',
    'config.section_validation': '验证与健康检查',
    'config.validate_concurrency': '验证并发数',
    'config.validate_timeout': '验证超时(秒)',
    'config.health_interval': '检查间隔(分钟)',
    'config.health_batch': '每批数量',
    'config.section_optimization': '优化设置',
    'config.optimize_interval': '优化间隔(分钟)',
    'config.replace_threshold': '替换阈值',
    'config.replace_threshold_help': '新代理需快30%',
    'config.cancel': '取消',
    'config.save': '保存配置',
    'msg.fetch_confirm': '确定开始抓取代理吗？',
    'msg.fetch_started': '抓取已在后台启动',
    'msg.refresh_confirm': '确定刷新所有代理的延迟吗？这可能需要一些时间。',
    'msg.refresh_started': '延迟刷新已启动',
    'msg.delete_confirm': '确定删除代理',
    'msg.config_saved': '配置保存成功',
    'msg.config_failed': '配置保存失败',
  },
  en: {
    'nav.config': 'Config',
    'nav.login': 'Login',
    'nav.logout': 'Logout',
    'health.status': 'Pool Status',
    'health.total': 'Total Proxies',
    'health.capacity': 'capacity',
    'health.slots': 'slots',
    'health.avg': 'avg',
    'health.state.healthy': 'Healthy',
    'health.state.warning': 'Warning',
    'health.state.critical': 'Critical',
    'health.state.emergency': 'Emergency',
    'quality.title': 'Quality Distribution',
    'quality.grade_s': 'S Grade',
    'quality.grade_a': 'A Grade',
    'quality.grade_b': 'B Grade',
    'quality.grade_c': 'C Grade',
    'actions.fetch': 'Fetch Proxies',
    'actions.refresh': 'Refresh Latency',
    'actions.config': 'Configure Pool',
    'proxy.title': 'Proxy Registry',
    'proxy.tab_all': 'All',
    'proxy.filter_protocol': 'Protocol',
    'proxy.filter_country': 'Exit Country',
    'proxy.loading': 'Loading...',
    'proxy.empty': 'No proxies available',
    'proxy.th_grade': 'Grade',
    'proxy.th_score': 'Score',
    'proxy.th_protocol': 'Protocol',
    'proxy.th_address': 'Address',
    'proxy.th_exit_ip': 'Exit IP',
    'proxy.th_location': 'Location',
    'proxy.th_latency': 'Latency',
    'proxy.th_usage': 'Usage',
    'proxy.th_action': 'Action',
    'proxy.btn_delete': 'DEL',
    'proxy.btn_refresh': 'Refresh',
    'proxy.copy_success': 'Copied',
    'proxy.refresh_started': 'Refresh started',
    'log.title': 'System Log',
    'log.auto_refresh_label': 'Auto Refresh',
    'log.loading': 'Loading...',
    'log.empty': 'No logs',
    'config.title': 'Pool Configuration',
    'config.section_capacity': 'Pool Capacity',
    'config.max_size': 'Max Size',
    'config.max_size_help': 'Total proxy slots',
    'config.http_ratio': 'HTTP Ratio',
    'config.http_ratio_help': '0.5 = 50% HTTP, 50% SOCKS5',
    'config.min_per_protocol': 'Min Per Protocol',
    'config.min_per_protocol_help': 'Minimum guarantee',
    'config.section_latency': 'Latency Standards (ms)',
    'config.latency_standard': 'Standard',
    'config.latency_healthy': 'Healthy',
    'config.latency_emergency': 'Emergency',
    'config.section_validation': 'Validation & Health Check',
    'config.validate_concurrency': 'Validate Concurrency',
    'config.validate_timeout': 'Validate Timeout (s)',
    'config.health_interval': 'Health Check Interval (min)',
    'config.health_batch': 'Batch Size',
    'config.section_optimization': 'Optimization',
    'config.optimize_interval': 'Optimize Interval (min)',
    'config.replace_threshold': 'Replace Threshold',
    'config.replace_threshold_help': 'New proxy must be 30% faster',
    'config.cancel': 'Cancel',
    'config.save': 'Save Configuration',
    'msg.fetch_confirm': 'Start proxy fetch?',
    'msg.fetch_started': 'Fetch started in background',
    'msg.refresh_confirm': 'Refresh latency for all proxies? This may take a while.',
    'msg.refresh_started': 'Latency refresh started',
    'msg.delete_confirm': 'Delete proxy',
    'msg.config_saved': 'Configuration saved successfully',
    'msg.config_failed': 'Failed to save configuration',
  }
};

let currentLang = 'zh';
let logCountdown = 5;

function t(key) {
  return i18n[currentLang][key] || key;
}

function updateLogCountdown() {
  const el = document.getElementById('log-countdown');
  if (el) el.textContent = logCountdown;
}

function updateI18n() {
  document.querySelectorAll('[data-i18n]').forEach(el => {
    const key = el.getAttribute('data-i18n');
    el.textContent = t(key);
  });
  document.getElementById('lang-btn').textContent = currentLang === 'zh' ? 'EN' : '中';
  document.title = currentLang === 'zh' ? 'GoProxy — 智能代理池' : 'GoProxy — Intelligent Pool';
  
  // 更新筛选下拉框标签
  const protocolLabel = document.getElementById('protocol-filter-label');
  if (protocolLabel) protocolLabel.textContent = t('proxy.filter_protocol');
  const countryLabel = document.getElementById('country-filter-label');
  if (countryLabel) countryLabel.textContent = t('proxy.filter_country');
}

function toggleLang() {
  currentLang = currentLang === 'zh' ? 'en' : 'zh';
  document.getElementById('lang-btn').textContent = currentLang === 'zh' ? '[ EN ]' : '[ 中文 ]';
  localStorage.setItem('lang', currentLang);
  updateI18n();
  if (allProxies.length > 0) {
    filterAndRender();
  }
}

// 页面加载时恢复语言设置
const savedLang = localStorage.getItem('lang');
if (savedLang) {
  currentLang = savedLang;
  updateI18n();
}

let currentProtocol = '';
let currentCountry = '';
let allProxies = [];
let isAdmin = false; // 是否为管理员

async function api(path, opts) {
  const r = await fetch(path, opts);
  if (r.status === 401) { location.href = '/login'; return null; }
  return r.json();
}

// 检查当前用户权限
async function checkAuth() {
  try {
    const auth = await fetch('/api/auth/check').then(r => r.json());
    isAdmin = auth.isAdmin || false;
    updateUIByRole();
  } catch (e) {
    isAdmin = false;
    updateUIByRole();
  }
}

// 根据角色更新 UI
function updateUIByRole() {
  // 显示/隐藏管理员专属元素
  document.querySelectorAll('.admin-only').forEach(el => {
    if (isAdmin) {
      el.style.display = 'block';
    } else {
      el.style.display = 'none';
    }
  });
  
  // 显示/隐藏登录链接（访客模式下显示）
  const loginLink = document.getElementById('login-link');
  if (loginLink) {
    if (isAdmin) {
      loginLink.style.display = 'none';
    } else {
      loginLink.style.display = 'inline-flex';
    }
  }
  
  // 更新用户模式标识
  const modeEl = document.getElementById('user-mode');
  if (modeEl) {
    if (isAdmin) {
      modeEl.textContent = 'admin';
    } else {
      modeEl.textContent = 'guest';
    }
  }
  
  // 重新渲染代理列表（更新操作列）
  if (allProxies.length > 0) {
    filterAndRender();
  }
}

function getCountryFlag(countryCode) {
  if (!countryCode || countryCode === 'UNKNOWN') return '';
  const offset = 127397;
  return countryCode.toUpperCase().split('').map(c => String.fromCodePoint(c.charCodeAt(0) + offset)).join('');
}

function showToast(message) {
  const toast = document.getElementById('toast');
  toast.textContent = message;
  toast.classList.add('show');
  setTimeout(() => toast.classList.remove('show'), 2000);
}

function copyToClipboard(text) {
  navigator.clipboard.writeText(text).then(() => {
    showToast(t('proxy.copy_success') + ': ' + text);
  }).catch(err => {
    console.error('Copy failed:', err);
  });
}

async function refreshProxy(address) {
  const res = await api('/api/proxy/refresh', { address });
  if (res) {
    showToast(t('proxy.refresh_started'));
    setTimeout(() => loadProxies(currentFilter), 2000);
  }
}

async function loadPoolStatus() {
  const status = await api('/api/pool/status');
  if (!status) return;

  document.getElementById('stat-total').textContent = status.Total;
  document.getElementById('stat-capacity').textContent = status.HTTPSlots + status.SOCKS5Slots;
  document.getElementById('stat-http').textContent = status.HTTP;
  document.getElementById('stat-socks5').textContent = status.SOCKS5;
  document.getElementById('http-slots').textContent = status.HTTPSlots;
  document.getElementById('socks5-slots').textContent = status.SOCKS5Slots;
  document.getElementById('http-avg').textContent = status.AvgLatencyHTTP || '—';
  document.getElementById('socks5-avg').textContent = status.AvgLatencySocks5 || '—';
  
  const stateEl = document.getElementById('pool-state');
  const dotEl = document.getElementById('pool-status-dot');
  const stateText = t('health.state.' + status.State.toLowerCase());
  stateEl.textContent = stateText.toUpperCase();
  dotEl.className = 'health-status ' + status.State.toLowerCase();
}

async function loadQualityDistribution() {
  const dist = await api('/api/pool/quality');
  if (!dist) return;

  const total = (dist.S || 0) + (dist.A || 0) + (dist.B || 0) + (dist.C || 0);
  
  document.getElementById('grade-s-count').textContent = dist.S || 0;
  document.getElementById('grade-a-count').textContent = dist.A || 0;
  document.getElementById('grade-b-count').textContent = dist.B || 0;
  document.getElementById('grade-c-count').textContent = dist.C || 0;

  if (total > 0) {
    const visual = document.getElementById('quality-visual');
    visual.innerHTML = '';
    if (dist.S) visual.innerHTML += '<div class="quality-segment quality-s" style="width:' + (dist.S/total*100) + '%">' + (dist.S/total*100 >= 10 ? 'S' : '') + '</div>';
    if (dist.A) visual.innerHTML += '<div class="quality-segment quality-a" style="width:' + (dist.A/total*100) + '%">' + (dist.A/total*100 >= 10 ? 'A' : '') + '</div>';
    if (dist.B) visual.innerHTML += '<div class="quality-segment quality-b" style="width:' + (dist.B/total*100) + '%">' + (dist.B/total*100 >= 10 ? 'B' : '') + '</div>';
    if (dist.C) visual.innerHTML += '<div class="quality-segment quality-c" style="width:' + (dist.C/total*100) + '%">' + (dist.C/total*100 >= 10 ? 'C' : '') + '</div>';
  }
}

async function loadProxies() {
  const path = currentProtocol ? '/api/proxies?protocol=' + currentProtocol : '/api/proxies';
  const proxies = await api(path);
  if (!proxies) return;
  
  allProxies = proxies;
  updateCountryOptions();
  filterAndRender();
}

function updateCountryOptions() {
  const countries = new Set();
  allProxies.forEach(p => {
    if (p.exit_location) {
      const countryCode = p.exit_location.split(' ')[0];
      if (countryCode) countries.add(countryCode);
    }
  });
  
  const select = document.getElementById('country-filter');
  const currentValue = select.value;
  select.innerHTML = '<option value="" id="country-filter-label">' + t('proxy.filter_country') + '</option>';
  Array.from(countries).sort().forEach(code => {
    const flag = getCountryFlag(code);
    select.innerHTML += '<option value="' + code + '">' + flag + ' ' + code + '</option>';
  });
  if (currentValue && countries.has(currentValue)) {
    select.value = currentValue;
  }
}

function filterAndRender() {
  let filtered = allProxies;
  if (currentCountry) {
    filtered = filtered.filter(p => p.exit_location && p.exit_location.startsWith(currentCountry + ' '));
  }
  renderProxies(filtered);
}

function setProtocolFilter(protocol) {
  currentProtocol = protocol;
  loadProxies();
}

function setCountryFilter(country) {
  currentCountry = country;
  filterAndRender();
}

function renderProxies(proxies) {
  let html = '';
  if (proxies.length === 0) {
    html = '<div class="empty" data-i18n="proxy.empty">' + t('proxy.empty') + '</div>';
  } else {
    html = '<table><thead><tr>';
    html += '<th data-i18n="proxy.th_grade">' + t('proxy.th_grade') + '</th>';
    html += '<th data-i18n="proxy.th_score">' + t('proxy.th_score') + '</th>';
    html += '<th data-i18n="proxy.th_protocol">' + t('proxy.th_protocol') + '</th>';
    html += '<th data-i18n="proxy.th_address">' + t('proxy.th_address') + '</th>';
    html += '<th data-i18n="proxy.th_exit_ip">' + t('proxy.th_exit_ip') + '</th>';
    html += '<th data-i18n="proxy.th_location">' + t('proxy.th_location') + '</th>';
    html += '<th data-i18n="proxy.th_latency">' + t('proxy.th_latency') + '</th>';
    html += '<th data-i18n="proxy.th_usage">' + t('proxy.th_usage') + '</th>';
    if (isAdmin) {
      html += '<th data-i18n="proxy.th_action">' + t('proxy.th_action') + '</th>';
    }
    html += '</tr></thead><tbody>';

    proxies.forEach(p => {
      const flag = p.exit_location ? getCountryFlag(p.exit_location.split(' ')[0]) : '';
      const grade = (p.quality_grade || 'C').toLowerCase();
      const latencyClass = 'grade-' + grade;
      
      html += '<tr>';
      html += '<td class="cell-grade grade-' + grade + '">' + (p.quality_grade || 'C') + '</td>';
      html += '<td class="cell-mono">' + ((p.quality_score || p.quality_score === 0) ? p.quality_score : '—') + '</td>';
      html += '<td><span class="badge badge-' + p.protocol + '">' + p.protocol.toUpperCase() + '</span></td>';
      html += '<td class="cell-mono cell-clickable" onclick="copyToClipboard(\'' + p.address + '\')" title="点击复制">' + p.address + '</td>';
      html += '<td class="cell-mono">' + (p.exit_ip || '—') + '</td>';
      html += '<td>' + flag + ' ' + (p.exit_location || '—') + '</td>';
      html += '<td class="cell-mono ' + latencyClass + '">' + (p.latency > 0 ? p.latency + 'ms' : '—') + '</td>';
      html += '<td class="cell-mono">' + (p.use_count || 0) + ' / ' + (p.success_count || 0) + '</td>';
      
      if (isAdmin) {
        html += '<td>';
        html += '<button class="btn-action" onclick="refreshProxy(\'' + p.address + '\')" data-i18n="proxy.btn_refresh">' + t('proxy.btn_refresh') + '</button>';
        html += '<button class="btn-danger" onclick="deleteProxy(\'' + p.address + '\')" data-i18n="proxy.btn_delete">' + t('proxy.btn_delete') + '</button>';
        html += '</td>';
      }
      
      html += '</tr>';
    });

    html += '</tbody></table>';
  }

  document.getElementById('proxy-table-wrap').innerHTML = html;
}

async function triggerFetch() {
  if (!confirm(t('msg.fetch_confirm'))) return;
  await api('/api/fetch', {method: 'POST'});
  alert(t('msg.fetch_started'));
  setTimeout(loadAll, 2000);
}

async function refreshLatency() {
  if (!confirm(t('msg.refresh_confirm'))) return;
  await api('/api/refresh-latency', {method: 'POST'});
  alert(t('msg.refresh_started'));
  setTimeout(loadAll, 2000);
}

async function deleteProxy(addr) {
  if (!confirm(t('msg.delete_confirm') + ' ' + addr + '?')) return;
  await api('/api/proxy/delete', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify({address: addr})
  });
  loadProxies();
}

async function loadLogs() {
  const data = await api('/api/logs');
  if (!data) return;
  
  const box = document.getElementById('logs-box');
  if (!data.lines || data.lines.length === 0) {
    box.innerHTML = '<div class="empty" data-i18n="log.empty">' + t('log.empty') + '</div>';
    return;
  }

  let html = '';
  data.lines.forEach(line => {
    let cls = '';
    if (line.includes('error') || line.includes('failed') || line.includes('❌') || line.includes('失败')) cls = 'error';
    if (line.includes('success') || line.includes('✅') || line.includes('completed') || line.includes('成功')) cls = 'success';
    html += '<div class="log-line ' + cls + '">' + line + '</div>';
  });
  box.innerHTML = html;
  box.scrollTop = box.scrollHeight;
  
  // 重置倒计时
  logCountdown = 5;
  
  // 同时刷新代理列表
  loadProxies();
}

async function openSettings() {
  const cfg = await api('/api/config');
  if (!cfg) return;

  document.getElementById('cfg-pool-size').value = cfg.pool_max_size;
  document.getElementById('cfg-http-ratio').value = cfg.pool_http_ratio;
  document.getElementById('cfg-min-per-protocol').value = cfg.pool_min_per_protocol;
  document.getElementById('cfg-max-latency').value = cfg.max_latency_ms;
  document.getElementById('cfg-max-latency-healthy').value = cfg.max_latency_healthy;
  document.getElementById('cfg-max-latency-emergency').value = cfg.max_latency_emergency;
  document.getElementById('cfg-concurrency').value = cfg.validate_concurrency;
  document.getElementById('cfg-timeout').value = cfg.validate_timeout;
  document.getElementById('cfg-health-interval').value = cfg.health_check_interval;
  document.getElementById('cfg-health-batch').value = cfg.health_check_batch_size;
  document.getElementById('cfg-optimize-interval').value = cfg.optimize_interval;
  document.getElementById('cfg-replace-threshold').value = cfg.replace_threshold;

  document.getElementById('settings-modal').classList.add('show');
}

function closeSettings() {
  document.getElementById('settings-modal').classList.remove('show');
}

async function saveConfig() {
  const cfg = {
    pool_max_size: parseInt(document.getElementById('cfg-pool-size').value),
    pool_http_ratio: parseFloat(document.getElementById('cfg-http-ratio').value),
    pool_min_per_protocol: parseInt(document.getElementById('cfg-min-per-protocol').value),
    max_latency_ms: parseInt(document.getElementById('cfg-max-latency').value),
    max_latency_healthy: parseInt(document.getElementById('cfg-max-latency-healthy').value),
    max_latency_emergency: parseInt(document.getElementById('cfg-max-latency-emergency').value),
    validate_concurrency: parseInt(document.getElementById('cfg-concurrency').value),
    validate_timeout: parseInt(document.getElementById('cfg-timeout').value),
    health_check_interval: parseInt(document.getElementById('cfg-health-interval').value),
    health_check_batch_size: parseInt(document.getElementById('cfg-health-batch').value),
    optimize_interval: parseInt(document.getElementById('cfg-optimize-interval').value),
    replace_threshold: parseFloat(document.getElementById('cfg-replace-threshold').value),
  };

  const result = await api('/api/config/save', {
    method: 'POST',
    headers: {'Content-Type': 'application/json'},
    body: JSON.stringify(cfg)
  });

  if (result && result.status === 'saved') {
    alert(t('msg.config_saved'));
    closeSettings();
    loadAll();
  } else {
    alert(t('msg.config_failed'));
  }
}

async function loadAll() {
  await checkAuth(); // 先检查权限
  loadPoolStatus();
  loadQualityDistribution();
  loadProxies();
  loadLogs();
}

loadAll();
setInterval(loadPoolStatus, 5000);
setInterval(loadQualityDistribution, 10000);
setInterval(loadLogs, 5000);

// 日志倒计时
setInterval(() => {
  logCountdown--;
  if (logCountdown < 0) logCountdown = 5;
  updateLogCountdown();
}, 1000);
</script>

<div id="toast" class="toast"></div>

</body>
</html>`
