// GT Desktop Client 前端主控制逻辑

// 全局状态管理
let currentConfig = null;
let currentStatus = {
    isRunning: false,
    serverAddr: "",
    clientId: "",
    pingMs: 0,
    speedUp: "0 KB/s",
    speedDown: "0 KB/s",
    activeSvc: []
};
let logsLockScroll = false;
let logsCached = [];

// 账号状态（B1）
let accountData = null; // GetAccountData 返回：{email, plan, credentials, prefixes, nodes}

// DOM 元素引用
const dom = {
    // 菜单导航
    menuItems: document.querySelectorAll('.menu-item'),
    panels: document.querySelectorAll('.tab-panel'),
    
    // Overview 元素
    btnToggleTunnel: document.getElementById('btn-toggle-tunnel'),
    globalStatusDot: document.getElementById('global-status-dot'),
    globalStatusText: document.getElementById('global-status-text'),
    globalNodeInfo: document.getElementById('global-node-info'),
    statPing: document.getElementById('stat-ping'),
    statPingDesc: document.getElementById('stat-ping-desc'),
    statSpeed: document.getElementById('stat-speed'),
    statSpeedDetail: document.getElementById('stat-speed-detail'),
    statState: document.getElementById('stat-state'),
    statConns: document.getElementById('stat-conns'),
    statClientId: document.getElementById('stat-client-id'),
    noActiveTunnels: document.getElementById('no-active-tunnels'),
    activeTunnelList: document.getElementById('active-tunnel-list'),

    // Tunnels 元素
    tunnelsList: document.getElementById('tunnels-list'),
    btnAddTunnel: document.getElementById('btn-add-tunnel'),
    drawerOverlay: document.getElementById('drawer-overlay'),
    tunnelDrawer: document.getElementById('tunnel-drawer'),
    btnCloseDrawer: document.getElementById('btn-close-drawer'),
    btnCancelTunnel: document.getElementById('btn-cancel-tunnel'),
    tunnelForm: document.getElementById('tunnel-form'),
    formEditIndex: document.getElementById('form-edit-index'),
    drawerTitleText: document.getElementById('drawer-title-text'),
    tunnelLocal: document.getElementById('tunnel-local'),
    tunnelPrefix: document.getElementById('tunnel-prefix'),
    tunnelRemotePort: document.getElementById('tunnel-remote-port'),
    tunnelRandomPort: document.getElementById('tunnel-random-port'),
    tunnelUseHost: document.getElementById('tunnel-use-host'),

    // Server Settings 元素
    serverForm: document.getElementById('server-config-form'),
    srvRemote: document.getElementById('srv-remote'),
    srvStun: document.getElementById('srv-stun'),
    srvId: document.getElementById('srv-id'),
    srvSecret: document.getElementById('srv-secret'),
    btnToggleSecret: document.getElementById('btn-toggle-secret-visibility'),
    srvMaxConn: document.getElementById('srv-max-conn'),
    srvIdleConn: document.getElementById('srv-idle-conn'),
    srvBbr: document.getElementById('srv-bbr'),
    srvInsecure: document.getElementById('srv-insecure'),
    btnTestServer: document.getElementById('btn-test-server'),
    btnImportFile: document.getElementById('btn-import-file'),
    btnImportClipboard: document.getElementById('btn-import-clipboard'),
    btnImportText: document.getElementById('btn-import-text'),
    importConfigText: document.getElementById('import-config-text'),
    importConfigFile: document.getElementById('import-config-file'),

    // Logs 元素
    logLevelFilter: document.getElementById('log-level-filter'),
    btnLockScroll: document.getElementById('btn-lock-scroll'),
    btnClearLogs: document.getElementById('btn-clear-logs'),
    logTerminal: document.getElementById('log-terminal-output'),

    // 账号元素（B1）
    acctArea: document.getElementById('acct-area'),
    btnSidebarLogin: document.getElementById('btn-sidebar-login'),
    acctCard: document.getElementById('acct-card'),
    acctLoginCard: document.getElementById('acct-login-card'),
    acctLoggedCard: document.getElementById('acct-logged-card'),
    acctLoginForm: document.getElementById('acct-login-form'),
    acctEmail: document.getElementById('acct-email'),
    acctPassword: document.getElementById('acct-password'),
    btnAcctLogin: document.getElementById('btn-acct-login'),
    acctEmailDisplay: document.getElementById('acct-email-display'),
    acctPlanDisplay: document.getElementById('acct-plan-display'),
    btnAcctLogout: document.getElementById('btn-acct-logout'),
    acctNodeSelect: document.getElementById('acct-node-select'),
    btnApplyNode: document.getElementById('btn-apply-node'),
    acctNodeApplied: document.getElementById('acct-node-applied'),
    acctCredDisplay: document.getElementById('acct-cred-display'),
    btnIssueCred: document.getElementById('btn-issue-cred'),
    acctSecretBox: document.getElementById('acct-secret-box'),
    acctSecretText: document.getElementById('acct-secret-text'),
    acctPrefixList: document.getElementById('acct-prefix-list'),
    claimPrefixInput: document.getElementById('claim-prefix-input'),
    btnClaimPrefix: document.getElementById('btn-claim-prefix'),
    prefixOptions: document.getElementById('prefix-options'),
    linkConsoleSignup: document.getElementById('link-console-signup'),

    // 提示框
    toast: document.getElementById('toast-notify')
};

// 辅助方法：显示 Toast 提示
function showToast(message, duration = 3000) {
    dom.toast.innerText = message;
    dom.toast.classList.add('show');
    setTimeout(() => {
        dom.toast.classList.remove('show');
    }, duration);
}

function getGTApp() {
    return window.go && window.go.main && window.go.main.GTApp ? window.go.main.GTApp : null;
}

function escapeHTML(value) {
    return String(value ?? "")
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;")
        .replaceAll("'", "&#39;");
}

function createButton(label, className, onClick) {
    const button = document.createElement('button');
    button.type = 'button';
    button.className = className;
    button.innerText = label;
    button.addEventListener('click', onClick);
    return button;
}

// ==========================================
// 1. 页签切换状态机 (Tabs Switching Router)
// ==========================================
function initTabs() {
    dom.menuItems.forEach(item => {
        item.addEventListener('click', () => {
            const targetTab = item.getAttribute('data-tab');
            
            // 切换菜单项激活状态
            dom.menuItems.forEach(i => i.classList.remove('active'));
            item.classList.add('active');
            
            // 切换面板显示
            dom.panels.forEach(panel => {
                panel.classList.remove('active');
                if (panel.id === `tab-${targetTab}`) {
                    panel.classList.add('active');
                }
            });

            // 针对特定页签的初始化操作
            if (targetTab === 'logs') {
                loadHistoryLogs();
                scrollToBottomLogs();
            }
        });
    });
}

// ==========================================
// 2. 状态轮询与仪表盘呈现 (Dashboard Telemetry)
// ==========================================
async function pollStatus() {
    const gtApp = getGTApp();
    if (!gtApp) return;

    try {
        const info = await gtApp.GetStatus();
        currentStatus = info;
        updateDashboardUI();
    } catch (err) {
        console.error("轮询状态出错:", err);
    }
}

function updateDashboardUI() {
    const isRunning = currentStatus.isRunning;

    // 1. 一级视觉中心：状态条与控制按钮
    if (isRunning) {
        dom.globalStatusDot.className = "status-dot online";
        dom.globalStatusText.innerText = "正在穿透中";
        dom.globalNodeInfo.innerText = `(${currentStatus.serverAddr || '未知中转服务器'})`;
        dom.btnToggleTunnel.innerText = "关闭穿透";
        dom.btnToggleTunnel.className = "btn btn-connect online";
    } else {
        dom.globalStatusDot.className = "status-dot offline";
        dom.globalStatusText.innerText = "已断开连接";
        dom.globalNodeInfo.innerText = "";
        dom.btnToggleTunnel.innerText = "开启穿透";
        dom.btnToggleTunnel.className = "btn btn-connect";
    }

    // 2. 二级视觉中心：核心遥测卡片 (延迟 & Bandwidth)
    if (isRunning) {
        dom.statPing.innerText = `${currentStatus.pingMs} ms`;
        dom.statPingDesc.innerText = "与中转服务器延迟 - 优秀";
        dom.statSpeed.innerText = `${currentStatus.speedDown} Down`;
        dom.statSpeedDetail.innerText = `↑ 上行速率: ${currentStatus.speedUp}`;
    } else {
        dom.statPing.innerText = "--";
        dom.statPingDesc.innerText = "等待连接...";
        dom.statSpeed.innerText = "--";
        dom.statSpeedDetail.innerText = "--";
    }

    // 3. 三级视觉中心：累计会话指标 (无框降噪)
    dom.statState.innerText = isRunning ? "ACTIVE" : "OFFLINE";
    dom.statConns.innerText = isRunning ? (currentStatus.activeSvc.length || "0") : "0";
    dom.statClientId.innerText = currentStatus.clientId || "--";

    // 4. 活动隧道映射列表渲染
    if (isRunning && currentStatus.activeSvc && currentStatus.activeSvc.length > 0) {
        dom.noActiveTunnels.style.display = "none";
        dom.activeTunnelList.style.display = "flex";
        dom.activeTunnelList.innerHTML = "";

        currentStatus.activeSvc.forEach(svc => {
            // 解析 local -> remote 串，如 127.0.0.1:8080 -> app.gt.com
            const parts = svc.split(" -> ");
            const local = parts[0] || "未知本地服务";
            const remote = parts[1] || "未分配公网域名";

            const card = document.createElement('div');
            card.className = "tunnel-item-card";
            
            const isHTTP = remote.includes("http://") || remote.includes("https://") || local.includes(":80") || local.includes(":8080");

            const meta = document.createElement('div');
            meta.className = 'tunnel-meta';

            const row = document.createElement('div');
            row.className = 'tunnel-mapping-row';

            const localSpan = document.createElement('span');
            localSpan.className = 'tunnel-local';
            localSpan.innerText = local;

            const arrow = document.createElement('span');
            arrow.className = 'tunnel-mapping-arrow';
            arrow.innerText = '⚡';

            const remoteSpan = document.createElement('span');
            remoteSpan.className = 'tunnel-remote';
            remoteSpan.innerText = remote;

            row.append(localSpan, arrow, remoteSpan);

            const details = document.createElement('div');
            details.className = 'tunnel-details';
            details.innerText = `类型: ${isHTTP ? 'HTTP 网页映射' : 'TCP 端口代理'}`;

            meta.append(row, details);

            const actions = document.createElement('div');
            actions.className = 'tunnel-actions';
            actions.appendChild(createButton('复制链接', 'btn btn-secondary btn-sm', () => copyToClipboard(remote)));
            if (isHTTP) {
                actions.appendChild(createButton('浏览器打开', 'btn btn-primary btn-sm', () => openInBrowser(remote)));
            }

            card.append(meta, actions);
            dom.activeTunnelList.appendChild(card);
        });
    } else {
        dom.noActiveTunnels.style.display = "block";
        dom.activeTunnelList.style.display = "none";
    }
}

// 供活动列表卡片调用的全局方法
window.copyToClipboard = function(text) {
    navigator.clipboard.writeText(text).then(() => {
        showToast("已成功复制公网链接至剪贴板！");
    }).catch(err => {
        console.error("复制失败:", err);
    });
};

window.openInBrowser = function(url) {
    const gtApp = getGTApp();
    if (gtApp && gtApp.OpenExternalURL) {
        gtApp.OpenExternalURL(url).catch(err => showToast(`打开浏览器失败: ${err}`, 4000));
        return;
    }
    window.open(url, "_blank", "noopener,noreferrer");
};

// ==========================================
// 3. 开启/停止内网穿透
// ==========================================
async function toggleTunnel() {
    const gtApp = getGTApp();
    if (!gtApp) {
        showToast("Wails Go 底层未就绪");
        return;
    }

    const isRunning = currentStatus.isRunning;
    
    // 改变按钮为连接中状态
    dom.btnToggleTunnel.disabled = true;
    dom.btnToggleTunnel.innerText = isRunning ? "正在停止..." : "正在开启...";
    dom.globalStatusDot.className = "status-dot connecting";
    dom.globalStatusText.innerText = isRunning ? "正在优雅释放连接池..." : "正在握手鉴权中...";

    try {
        if (isRunning) {
            await gtApp.StopTunnel();
            showToast("内网穿透已成功关闭");
        } else {
            // 先加载配置，确保有配置项存在
            if (!currentConfig || !currentConfig.Remote || currentConfig.Remote.length === 0) {
                showToast("中转服务器节点未配置，请先在 Server Settings 页面完成配置", 4000);
                dom.btnToggleTunnel.disabled = false;
                updateDashboardUI();
                return;
            }
            await gtApp.StartTunnel();
            showToast("一键穿透开启成功！隧道已全部建立");
        }
        await pollStatus();
    } catch (err) {
        showToast(`穿透操作失败: ${err}`, 4500);
    } finally {
        dom.btnToggleTunnel.disabled = false;
    }
}

// ==========================================
// 4. 隧道穿透配置逻辑 (Tunnels Config & Cards)
// ==========================================
function renderTunnels() {
    dom.tunnelsList.innerHTML = "";
    
    if (!currentConfig || !currentConfig.Services || currentConfig.Services.length === 0) {
        dom.tunnelsList.innerHTML = `
            <div class="no-tunnels-msg" style="grid-column: 1 / -1;">
                暂无配置的本地隧道服务规则。请点击右上方按钮添加。
            </div>
        `;
        return;
    }

    currentConfig.Services.forEach((svc, index) => {
        const localURL = svc.LocalURL && svc.LocalURL.URL ? svc.LocalURL.URL : "未知地址";
        const prefix = svc.HostPrefix || "--";
        const randomPort = svc.RemoteTCPRandom === null || svc.RemoteTCPRandom === undefined ? true : svc.RemoteTCPRandom;
        const remotePort = svc.RemoteTCPPort || "--";
        const useHost = svc.UseLocalAsHTTPHost || false;

        const isHTTP = localURL.startsWith("http://") || localURL.startsWith("https://") || localURL.includes(":80") || localURL.includes(":8080");

        const card = document.createElement('div');
        card.className = `tunnel-config-card ${currentStatus.isRunning ? 'active' : ''}`;
        
        card.innerHTML = `
            <div class="card-header-row">
                <div class="tunnel-title">隧道 #${index + 1}</div>
                <span class="tunnel-protocol-badge">${isHTTP ? 'http/https' : 'tcp'}</span>
            </div>
            <div class="tunnel-body">
                <div class="detail-row">
                    <span class="detail-lbl">本地映射服务:</span>
                    <span class="detail-val">${escapeHTML(localURL)}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-lbl">主机前缀:</span>
                    <span class="detail-val">${escapeHTML(prefix)}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-lbl">外网 TCP 端口:</span>
                    <span class="detail-val">${escapeHTML(randomPort ? '随机分配' : remotePort)}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-lbl">校验 Host 头:</span>
                    <span class="detail-val">${useHost ? '开启' : '关闭'}</span>
                </div>
            </div>
            <div class="card-actions-row">
                <div class="op-buttons">
                    <button class="btn-icon" title="编辑" onclick="openDrawer(${index})">✏️</button>
                    <button class="btn-icon danger" title="删除" onclick="deleteTunnel(${index})">🗑️</button>
                </div>
                <!-- 独立开启/关闭本条隧道的 flat 样式 toggle 开关 -->
                <label class="switch">
                    <input type="checkbox" ${currentStatus.isRunning ? 'checked' : ''} disabled>
                    <span class="slider"></span>
                </label>
            </div>
        `;
        dom.tunnelsList.appendChild(card);
    });
}

// 侧滑抽屉展示控制
window.openDrawer = function(index = -1) {
    dom.drawerOverlay.style.display = "block";
    dom.tunnelDrawer.classList.add('open');

    if (index >= 0) {
        // 编辑现有配置
        dom.drawerTitleText.innerText = "编辑隧道映射";
        dom.formEditIndex.value = index;
        
        const svc = currentConfig.Services[index];
        const localURL = svc.LocalURL && svc.LocalURL.URL ? svc.LocalURL.URL : "";
        dom.tunnelLocal.value = localURL;
        dom.tunnelPrefix.value = svc.HostPrefix || "";
        dom.tunnelRemotePort.value = svc.RemoteTCPPort || "";
        dom.tunnelRandomPort.checked = svc.RemoteTCPRandom === null || svc.RemoteTCPRandom === undefined ? true : svc.RemoteTCPRandom;
        dom.tunnelUseHost.checked = svc.UseLocalAsHTTPHost || false;
    } else {
        // 添加全新配置
        dom.drawerTitleText.innerText = "添加隧道映射";
        dom.formEditIndex.value = -1;
        
        dom.tunnelLocal.value = "";
        dom.tunnelPrefix.value = "";
        dom.tunnelRemotePort.value = "";
        dom.tunnelRandomPort.checked = true;
        dom.tunnelUseHost.checked = false;
    }
};

function closeDrawer() {
    dom.drawerOverlay.style.display = "none";
    dom.tunnelDrawer.classList.remove('open');
}

async function saveTunnel(e) {
    e.preventDefault();
    if (!currentConfig) return;

    const index = parseInt(dom.formEditIndex.value);
    
    // 构建 service 对象
    const svc = {
        HostPrefix: dom.tunnelPrefix.value,
        RemoteTCPPort: dom.tunnelRemotePort.value ? parseInt(dom.tunnelRemotePort.value) : 0,
        RemoteTCPRandom: dom.tunnelRandomPort.checked,
        LocalURL: {
            URL: dom.tunnelLocal.value
        },
        UseLocalAsHTTPHost: dom.tunnelUseHost.checked
    };

    if (index >= 0) {
        // 覆盖已存的
        currentConfig.Services[index] = svc;
        showToast("隧道配置修改已保存");
    } else {
        // 追加新配置
        if (!currentConfig.Services) currentConfig.Services = [];
        currentConfig.Services.push(svc);
        showToast("新隧道穿透规则已添加");
    }

    try {
        const gtApp = getGTApp();
        if (!gtApp) throw new Error("Wails Go 底层未就绪");
        await gtApp.SaveConfig(currentConfig);
        closeDrawer();
        renderTunnels();
        updateDashboardUI();
    } catch (err) {
        showToast(`保存配置文件失败: ${err}`);
    }
}

window.deleteTunnel = async function(index) {
    if (!currentConfig || index < 0) return;
    
    if (confirm(`确认要删除此条内网穿透规则吗？`)) {
        currentConfig.Services.splice(index, 1);
        try {
            const gtApp = getGTApp();
            if (!gtApp) throw new Error("Wails Go 底层未就绪");
            await gtApp.SaveConfig(currentConfig);
            showToast("穿透规则已成功移除");
            renderTunnels();
            updateDashboardUI();
        } catch (err) {
            showToast(`删除失败: ${err}`);
        }
    }
};

// ==========================================
// 5. 中转节点配置逻辑 (Server Node)
// ==========================================
function renderServerConfig() {
    if (!currentConfig) return;

    // 基本配置
    if (currentConfig.Remote && currentConfig.Remote.length > 0) {
        dom.srvRemote.value = currentConfig.Remote[0];
    } else {
        dom.srvRemote.value = "";
    }
    if (currentConfig.RemoteSTUN && currentConfig.RemoteSTUN.length > 0) {
        dom.srvStun.value = currentConfig.RemoteSTUN[0];
    } else {
        dom.srvStun.value = "";
    }
    dom.srvId.value = currentConfig.ID || "";
    dom.srvSecret.value = currentConfig.Secret || "";

    // 高级连接池设置
    dom.srvMaxConn.value = currentConfig.RemoteConnections || 3;
    dom.srvIdleConn.value = currentConfig.RemoteIdleConnections || 1;
    dom.srvBbr.checked = currentConfig.OpenBBR || false;
    dom.srvInsecure.checked = currentConfig.RemoteCertInsecure || false;
}

async function saveServerConfig(e) {
    e.preventDefault();
    if (!currentConfig) return;

    // 封包表单参数
    currentConfig.Remote = [dom.srvRemote.value];
    currentConfig.RemoteSTUN = dom.srvStun.value ? [dom.srvStun.value] : [];
    currentConfig.ID = dom.srvId.value;
    currentConfig.Secret = dom.srvSecret.value;

    currentConfig.RemoteConnections = parseInt(dom.srvMaxConn.value) || 3;
    currentConfig.RemoteIdleConnections = parseInt(dom.srvIdleConn.value) || 1;
    currentConfig.OpenBBR = dom.srvBbr.checked;
    currentConfig.RemoteCertInsecure = dom.srvInsecure.checked;

    try {
        const gtApp = getGTApp();
        if (!gtApp) throw new Error("Wails Go 底层未就绪");
        await gtApp.SaveConfig(currentConfig);
        showToast("服务器节点配置已成功持久化保存！");
        if (currentStatus.isRunning) {
            showToast("配置已修改！请关闭并重新开启穿透以使新配置生效", 5000);
        }
    } catch (err) {
        showToast(`保存失败: ${err}`);
    }
}

async function testServerConnection() {
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.TestServerConnection) {
        showToast("Wails Go 底层未就绪");
        return;
    }

    showToast("正在测试中转服务器 TCP 连通性...", 2000);
    try {
        const message = await gtApp.TestServerConnection();
        showToast(message, 4000);
    } catch (err) {
        showToast(`连接测试失败: ${err}`, 5000);
    }
}

async function importConfigPayload(payload) {
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.ImportConfig) {
        showToast("Wails Go 底层未就绪");
        return;
    }

    if (!payload || !payload.trim()) {
        showToast("导入内容为空");
        return;
    }

    if (!confirm("导入会覆盖当前 GT 桌面客户端配置，确认继续吗？")) {
        return;
    }

    try {
        currentConfig = await gtApp.ImportConfig(payload);
        renderTunnels();
        renderServerConfig();
        updateDashboardUI();
        dom.importConfigText.value = "";
        showToast(currentStatus.isRunning ? "配置已导入，请重启穿透使配置生效" : "配置导入成功", 5000);
    } catch (err) {
        showToast(`导入失败: ${err}`, 6000);
    }
}

function importConfigFromFile(file) {
    if (!file) return;

    const reader = new FileReader();
    reader.onload = () => {
        importConfigPayload(String(reader.result || ""));
        dom.importConfigFile.value = "";
    };
    reader.onerror = () => {
        showToast("读取配置文件失败");
        dom.importConfigFile.value = "";
    };
    reader.readAsText(file);
}

async function importConfigFromClipboard() {
    if (!navigator.clipboard || !navigator.clipboard.readText) {
        showToast("当前环境不支持读取剪贴板，请粘贴到文本框后导入");
        return;
    }

    try {
        const text = await navigator.clipboard.readText();
        dom.importConfigText.value = text;
        await importConfigPayload(text);
    } catch (err) {
        showToast(`读取剪贴板失败: ${err}`, 5000);
    }
}

// ==========================================
// 6. 实时网络日志 (Logs Terminal)
// ==========================================
function appendLogLine(line) {
    logsCached.push(line);
    if (logsCached.length > 1000) {
        logsCached.shift();
        renderFilteredLogs();
        return;
    }

    if (!shouldDisplayLogLine(line)) {
        return;
    }
    appendLogElement(line);
    if (!logsLockScroll) {
        scrollToBottomLogs();
    }
}

function renderFilteredLogs() {
    const filter = dom.logLevelFilter.value;
    dom.logTerminal.innerHTML = "";

    logsCached.filter(shouldDisplayLogLine).forEach(appendLogElement);

    if (!logsLockScroll) {
        scrollToBottomLogs();
    }
}

function shouldDisplayLogLine(line) {
    const filter = dom.logLevelFilter.value;
    if (filter === "ALL") return true;
    return line.includes(`[${filter}]`) || line.includes(`level=${filter.toLowerCase()}`);
}

function appendLogElement(line) {
    const span = document.createElement('span');
    let textClass = 'log-info';

    if (line.includes('[WARN]') || line.includes('level=warn')) {
        textClass = 'log-warn';
    } else if (line.includes('[ERROR]') || line.includes('level=error') || line.includes('level=fatal') || line.includes('level=panic')) {
        textClass = 'log-error';
    }

    span.className = textClass;
    span.innerText = line;
    dom.logTerminal.appendChild(span);
}

async function loadHistoryLogs() {
    const gtApp = getGTApp();
    if (!gtApp) return;

    try {
        const history = await gtApp.GetLogs();
        logsCached = history;
        renderFilteredLogs();
    } catch (err) {
        console.error("加载历史日志出错:", err);
    }
}

function scrollToBottomLogs() {
    dom.logTerminal.scrollTop = dom.logTerminal.scrollHeight;
}

// ==========================================
// 7. 账号集成 (Account / B1)
// ==========================================
async function initAccount() {
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.AccountStatus) return;
    try {
        const st = await gtApp.AccountStatus();
        if (st && st.loggedIn) {
            await refreshAccountData();
        } else {
            renderLoggedOut();
        }
    } catch (err) {
        console.warn("账号状态检查失败:", err);
        renderLoggedOut();
    }
}

function renderLoggedOut() {
    accountData = null;

    // 侧栏
    dom.acctArea.innerHTML = "";
    const loginBtn = document.createElement('button');
    loginBtn.className = 'btn btn-acct-login';
    loginBtn.id = 'btn-sidebar-login';
    loginBtn.innerText = '登录账号';
    loginBtn.addEventListener('click', jumpToServerTab);
    dom.acctArea.appendChild(loginBtn);

    // 连接页
    dom.acctLoginCard.classList.remove('hidden');
    dom.acctLoggedCard.classList.add('hidden');
}

function renderAccountUI() {
    if (!accountData) { renderLoggedOut(); return; }

    // 侧栏
    dom.acctArea.innerHTML = `
        <div class="acct-email-row">
            <span class="email">${escapeHTML(accountData.email)}</span>
            <button id="btn-sidebar-logout">退出登录</button>
        </div>`;
    document.getElementById('btn-sidebar-logout').addEventListener('click', doLogout);

    // 连接页：登录卡切换
    dom.acctLoginCard.classList.add('hidden');
    dom.acctLoggedCard.classList.remove('hidden');
    dom.acctEmailDisplay.innerText = accountData.email;

    const plan = accountData.plan;
    dom.acctPlanDisplay.innerText = plan
        ? `${plan.name || "Free"} · ${plan.host_number ?? "-"} 前缀 · ${plan.tcp_number ?? "-"} TCP`
        : "未分配套餐（默认 Free）";

    // 节点下拉
    dom.acctNodeSelect.innerHTML = "";
    (accountData.nodes || []).forEach(n => {
        const opt = document.createElement('option');
        opt.value = n.addr;
        const dot = n.status === 'online' ? '●' : (n.status === 'maintenance' ? '◐' : '○');
        opt.innerText = `${dot} ${n.name}（${n.region || n.addr}）`;
        dom.acctNodeSelect.appendChild(opt);
    });
    if (!accountData.nodes || accountData.nodes.length === 0) {
        const opt = document.createElement('option');
        opt.value = "";
        opt.innerText = "暂无可用节点";
        dom.acctNodeSelect.appendChild(opt);
    }

    // 当前凭证（配置中的 id 高亮）
    const credRows = (accountData.credentials || []).map(c => {
        const current = currentConfig && currentConfig.ID === c.gt_id;
        return `<div class="prefix-row">${current ? '★' : '·'} <b>${escapeHTML(c.gt_id)}</b>${c.enabled ? '' : '（已停用）'}</div>`;
    });
    dom.acctCredDisplay.innerHTML = credRows.length
        ? credRows.join("")
        : `<div>尚无凭证（当前配置使用的是手动/导入的凭证）</div>`;

    // 前缀列表 + datalist
    const prefixRows = (accountData.prefixes || []).map(p =>
        `<div class="prefix-row"><b>${escapeHTML(p.prefix)}</b>.app.gtunnel.dev</div>`);
    dom.acctPrefixList.innerHTML = prefixRows.length ? prefixRows.join("") : "<div>尚未申领前缀</div>";

    dom.prefixOptions.innerHTML = "";
    (accountData.prefixes || []).forEach(p => {
        const opt = document.createElement('option');
        opt.value = p.prefix;
        dom.prefixOptions.appendChild(opt);
    });
}

async function refreshAccountData() {
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.GetAccountData) return;
    try {
        accountData = await gtApp.GetAccountData();
        renderAccountUI();
    } catch (err) {
        if (String(err).includes("未登录")) {
            renderLoggedOut();
        } else {
            showToast(`获取账号信息失败: ${err}`, 5000);
        }
    }
}

async function doLogin(e) {
    e.preventDefault();
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.Login) { showToast("Wails Go 底层未就绪"); return; }

    dom.btnAcctLogin.disabled = true;
    dom.btnAcctLogin.innerText = "登录中...";
    try {
        await gtApp.Login(dom.acctEmail.value.trim(), dom.acctPassword.value);
        dom.acctPassword.value = "";
        showToast("登录成功");
        await refreshAccountData();
    } catch (err) {
        showToast(`登录失败: ${err}`, 5000);
    } finally {
        dom.btnAcctLogin.disabled = false;
        dom.btnAcctLogin.innerText = "登录";
    }
}

async function doLogout() {
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.Logout) return;
    try {
        await gtApp.Logout();
        renderLoggedOut();
        showToast("已退出登录");
    } catch (err) {
        showToast(`退出失败: ${err}`);
    }
}

async function issueCredFlow() {
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.IssueCredential) return;

    dom.btnIssueCred.disabled = true;
    dom.btnIssueCred.innerText = "签发中...";
    dom.acctSecretBox.classList.add('hidden');
    try {
        const issued = await gtApp.IssueCredential();
        // 凭证已由 Go 侧写入配置，前端同步刷新
        currentConfig = await gtApp.LoadConfig();
        renderServerConfig();
        renderTunnels();

        dom.acctSecretText.textContent = `id:     ${issued.gt_id}\nsecret: ${issued.secret}`;
        dom.acctSecretBox.classList.remove('hidden');
        showToast("新凭证已签发并写入配置（secret 仅此一次展示）");
        await refreshAccountData();
    } catch (err) {
        showToast(`签发失败: ${err}`, 5000);
    } finally {
        dom.btnIssueCred.disabled = false;
        dom.btnIssueCred.innerText = "签发新凭证并自动配置";
    }
}

async function applyNodeFlow() {
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.ApplyNode) return;

    const addr = dom.acctNodeSelect.value;
    if (!addr) { showToast("暂无可用节点"); return; }

    try {
        await gtApp.ApplyNode(addr);
        currentConfig = await gtApp.LoadConfig();
        renderServerConfig();
        dom.acctNodeApplied.innerText = `已应用：${currentConfig.Remote[0]}`;
        showToast(`节点已写入配置（${currentConfig.Remote[0]}）`);
        if (currentStatus.isRunning) {
            showToast("节点已切换，请重启穿透使配置生效", 5000);
        }
    } catch (err) {
        showToast(`应用节点失败: ${err}`, 5000);
    }
}

async function claimPrefixFlow() {
    const gtApp = getGTApp();
    if (!gtApp || !gtApp.ClaimPrefix) return;

    const prefix = dom.claimPrefixInput.value.trim().toLowerCase();
    if (!prefix) { showToast("请输入前缀"); return; }

    try {
        await gtApp.ClaimPrefix(prefix);
        dom.claimPrefixInput.value = "";
        showToast(`前缀 ${prefix} 申领成功`);
        await refreshAccountData();
    } catch (err) {
        const msg = String(err);
        if (msg.includes("already taken")) showToast("该前缀已被占用", 4000);
        else if (msg.includes("plan allows")) showToast("已达套餐前缀上限", 4000);
        else showToast(`申领失败: ${err}`, 5000);
    }
}

function jumpToServerTab() {
    const target = document.querySelector('.menu-item[data-tab="server"]');
    if (target) target.click();
}

// ==========================================
// 8. 初始化程序入口 (Main Initialize)
// ==========================================
async function initApp() {
    // 1. 初始化 Tab 页路由
    initTabs();

    // 2. 检查 Wails Go 对象就绪性并拉取初始配置
    const gtApp = getGTApp();
    if (gtApp) {
        try {
            currentConfig = await gtApp.LoadConfig();

            // 渲染数据
            renderTunnels();
            renderServerConfig();

            // 获取初始状态
            await pollStatus();
        } catch (err) {
            console.error("获取基础配置失败:", err);
            showToast("加载配置文件失败");
        }

        // 账号状态恢复（B1）
        await initAccount();

        // 3. 启动 Wails v3 原生事件总线监听 (监听来自 Go 的实时日志流)
        if (window.wails && window.wails.Events) {
            window.wails.Events.On("gt:log", (event) => {
                const logLine = event && event.data ? event.data : event;
                if (logLine) {
                    appendLogLine(logLine);
                }
            });
        }
    } else {
        console.warn("Wails Go bindings 未加载，处于离线设计预览模式");
    }

    // 4. 定时轮询遥测指标
    setInterval(pollStatus, 1500);
}

// 绑定所有的 DOM 动作事件监听器
document.addEventListener('DOMContentLoaded', () => {
    // 核心开启/关闭
    dom.btnToggleTunnel.addEventListener('click', toggleTunnel);

    // 侧滑抽屉表单事件
    dom.btnAddTunnel.addEventListener('click', () => openDrawer(-1));
    dom.btnCloseDrawer.addEventListener('click', closeDrawer);
    dom.btnCancelTunnel.addEventListener('click', closeDrawer);
    dom.drawerOverlay.addEventListener('click', closeDrawer);
    dom.tunnelForm.addEventListener('submit', saveTunnel);

    // 服务器配置表单事件
    dom.serverForm.addEventListener('submit', saveServerConfig);
    dom.btnTestServer.addEventListener('click', testServerConnection);
    dom.btnImportFile.addEventListener('click', () => dom.importConfigFile.click());
    dom.importConfigFile.addEventListener('change', () => importConfigFromFile(dom.importConfigFile.files[0]));
    dom.btnImportClipboard.addEventListener('click', importConfigFromClipboard);
    dom.btnImportText.addEventListener('click', () => importConfigPayload(dom.importConfigText.value));
    
    // 一键隐藏/显示密码
    dom.btnToggleSecret.addEventListener('click', () => {
        const show = dom.srvSecret.type === 'password';
        dom.srvSecret.type = show ? 'text' : 'password';
        dom.btnToggleSecret.innerText = show ? "隐藏" : "显示";
    });

    // 账号事件（B1）
    dom.acctLoginForm.addEventListener('submit', doLogin);
    dom.btnAcctLogout.addEventListener('click', doLogout);
    dom.btnIssueCred.addEventListener('click', issueCredFlow);
    dom.btnApplyNode.addEventListener('click', applyNodeFlow);
    dom.btnClaimPrefix.addEventListener('click', claimPrefixFlow);
    if (dom.linkConsoleSignup) {
        dom.linkConsoleSignup.addEventListener('click', (e) => {
            e.preventDefault();
            window.openInBrowser('https://console.gtunnel.dev/auth');
        });
    }

    // 日志控制台事件
    dom.logLevelFilter.addEventListener('change', renderFilteredLogs);
    dom.btnLockScroll.addEventListener('click', () => {
        logsLockScroll = !logsLockScroll;
        dom.btnLockScroll.innerText = logsLockScroll ? "锁定滚动" : "自由滚动";
        dom.btnLockScroll.classList.toggle('btn-primary', logsLockScroll);
    });
    dom.btnClearLogs.addEventListener('click', () => {
        logsCached = [];
        dom.logTerminal.innerHTML = "";
        const gtApp = getGTApp();
        if (gtApp) {
            gtApp.ClearLogs();
        }
    });

    // 启动主初始化
    initApp();
});
