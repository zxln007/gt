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

    // 快速开始（B2）
    quickstartCard: document.getElementById('quickstart-card'),
    quickstartForm: document.getElementById('quickstart-form'),
    qsPort: document.getElementById('qs-port'),
    qsPrefix: document.getElementById('qs-prefix'),
    btnQuickstart: document.getElementById('btn-quickstart'),
    qsResult: document.getElementById('qs-result'),
    qsUrl: document.getElementById('qs-url'),
    qsNote: document.getElementById('qs-note'),
    btnQsCopy: document.getElementById('btn-qs-copy'),
    btnQsOpen: document.getElementById('btn-qs-open'),

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
    // Wails v3 alpha 只暴露 window.wails.Call.ByName("<pkg>.<Service>.<Method>")，
    // 这里用 Proxy 动态转发，保持 gtApp.Method(...) 的调用风格
    if (window.__gtAppProxy) return window.__gtAppProxy;
    if (!(window.wails && window.wails.Call && window.wails.Call.ByName)) return null;
    window.__gtAppProxy = new Proxy({}, {
        get: (_, prop) => {
            if (typeof prop !== 'string') return undefined;
            return (...args) => window.wails.Call.ByName(`main.GTApp.${prop}`, ...args);
        }
    });
    return window.__gtAppProxy;
}

function ensureWailsRuntime(onReady) {
    if (window.wails && window.wails.Call) return onReady();
    const s = document.createElement('script');
    s.type = 'module';
    s.src = '/wails/runtime.js';
    s.onload = () => onReady();
    s.onerror = () => console.warn("Wails runtime 加载失败");
    document.head.appendChild(s);
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

    // 快速开始（B2）
    dom.quickstartForm.addEventListener('submit', quickStartFlow);
    dom.btnQsCopy.addEventListener('click', () => {
        if (qsPublicUrl) window.copyToClipboard(qsPublicUrl);
    });
    dom.btnQsOpen.addEventListener('click', () => {
        if (qsPublicUrl) window.openInBrowser(qsPublicUrl);
    });
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

    // 启动主初始化（先确保 Wails runtime 就绪）
    ensureWailsRuntime(initApp);

});
