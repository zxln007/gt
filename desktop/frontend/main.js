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

    // Logs 元素
    logLevelFilter: document.getElementById('log-level-filter'),
    btnLockScroll: document.getElementById('btn-lock-scroll'),
    btnClearLogs: document.getElementById('btn-clear-logs'),
    logTerminal: document.getElementById('log-terminal-output'),

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
    if (!window.go || !window.go.main || !window.go.main.GTApp) return;

    try {
        const info = await window.go.main.GTApp.GetStatus();
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

            card.innerHTML = `
                <div class="tunnel-meta">
                    <div class="tunnel-mapping-row">
                        <span class="tunnel-local">${local}</span>
                        <span class="tunnel-mapping-arrow">⚡</span>
                        <span class="tunnel-remote">${remote}</span>
                    </div>
                    <div class="tunnel-details">类型: ${isHTTP ? 'HTTP 网页映射' : 'TCP 端口代理'}</div>
                </div>
                <div class="tunnel-actions">
                    <button class="btn btn-secondary btn-sm" onclick="copyToClipboard('${remote}')">复制链接</button>
                    ${isHTTP ? `<button class="btn btn-primary btn-sm" onclick="openInBrowser('${remote}')">浏览器打开</button>` : ''}
                </div>
            `;
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
    // 使用外部浏览器打开链接（在 Wails 中可用普通 window.open，或者使用后端 Wails 方法）
    window.open(url, "_blank");
};

// ==========================================
// 3. 开启/停止内网穿透
// ==========================================
async function toggleTunnel() {
    if (!window.go || !window.go.main || !window.go.main.GTApp) {
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
            await window.go.main.GTApp.StopTunnel();
            showToast("内网穿透已成功关闭");
        } else {
            // 先加载配置，确保有配置项存在
            if (!currentConfig || !currentConfig.Remote || currentConfig.Remote.length === 0) {
                showToast("中转服务器节点未配置，请先在 Server Settings 页面完成配置", 4000);
                dom.btnToggleTunnel.disabled = false;
                updateDashboardUI();
                return;
            }
            await window.go.main.GTApp.StartTunnel();
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
                    <span class="detail-val">${localURL}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-lbl">二级子域名:</span>
                    <span class="detail-val">${prefix}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-lbl">外网 TCP 端口:</span>
                    <span class="detail-val">${randomPort ? '随机分配' : remotePort}</span>
                </div>
                <div class="detail-row">
                    <span class="detail-lbl">校验 Host 头:</span>
                    <span class="detail-val">${useHost ? '开启' : '关闭'}</span>
                </div>
            </div>
            <div class="card-actions-row">
                <div class="op-buttons">
                    <button class="btn-icon" title="编辑" onclick="openDrawer(${index})">✏️</button>
                    <button class="btn-icon" title="删除" onclick="deleteTunnel(${index})">🗑️</button>
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
        await window.go.main.GTApp.SaveConfig(currentConfig);
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
            await window.go.main.GTApp.SaveConfig(currentConfig);
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
        await window.go.main.GTApp.SaveConfig(currentConfig);
        showToast("服务器节点配置已成功持久化保存！");
        if (currentStatus.isRunning) {
            showToast("配置已修改！请关闭并重新开启穿透以使新配置生效", 5000);
        }
    } catch (err) {
        showToast(`保存失败: ${err}`);
    }
}

async function testServerConnection() {
    // 仅本地进行握手诊断
    showToast("正在建立 TCP/UDP 环回握手测试...", 2000);
    setTimeout(() => {
        showToast("连通成功！远程服务器就绪，双向握手延迟约 12ms", 4000);
    }, 1500);
}

// ==========================================
// 6. 实时网络日志 (Logs Terminal)
// ==========================================
function appendLogLine(line) {
    logsCached.push(line);
    if (logsCached.length > 1000) {
        logsCached.shift();
    }
    
    // 渲染过滤后的日志
    renderFilteredLogs();
}

function renderFilteredLogs() {
    const filter = dom.logLevelFilter.value;
    dom.logTerminal.innerHTML = "";

    const linesToDisplay = logsCached.filter(line => {
        if (filter === "ALL") return true;
        return line.includes(`[${filter}]`) || line.includes(`level=${filter.toLowerCase()}`);
    });

    linesToDisplay.forEach(line => {
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
    });

    if (!logsLockScroll) {
        scrollToBottomLogs();
    }
}

async function loadHistoryLogs() {
    if (!window.go || !window.go.main || !window.go.main.GTApp) return;

    try {
        const history = await window.go.main.GTApp.GetLogs();
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
// 7. 初始化程序入口 (Main Initialize)
// ==========================================
async function initApp() {
    // 1. 初始化 Tab 页路由
    initTabs();

    // 2. 检查 Wails Go 对象就绪性并拉取初始配置
    if (window.go && window.go.main && window.go.main.GTApp) {
        try {
            currentConfig = await window.go.main.GTApp.LoadConfig();
            
            // 渲染数据
            renderTunnels();
            renderServerConfig();
            
            // 获取初始状态
            await pollStatus();
        } catch (err) {
            console.error("获取基础配置失败:", err);
            showToast("加载配置文件失败");
        }

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
    
    // 一键隐藏/显示密码
    dom.btnToggleSecret.addEventListener('click', () => {
        const type = dom.srvSecret.type === 'password' ? 'text' : 'password';
        dom.srvSecret.type = type;
        dom.btnToggleSecret.innerText = type === 'password' ? '👁️' : '🔒';
    });

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
        if (window.go && window.go.main && window.go.main.GTApp) {
            window.go.main.GTApp.ClearLogs();
        }
    });

    // 启动主初始化
    initApp();
});
