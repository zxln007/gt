package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/zalando/go-keyring"
)

// Account integration with the gt-control control plane (B1).
//
// All calls run on the Go side — the webview never talks to the API directly,
// so CORS is irrelevant and the auth token never leaves the process except
// into the OS credential store (Windows Credential Manager / macOS Keychain).

const (
	keyringService = "gt-desktop"
	keyringUser    = "gt-control-token"
)

var errNotLoggedIn = errors.New("未登录")

func apiBase() string {
	if base := strings.TrimSpace(os.Getenv("GT_DESKTOP_API")); base != "" {
		return strings.TrimRight(base, "/")
	}
	return "https://api.gtunnel.dev:8443"
}

var accountClient = &http.Client{Timeout: 15 * time.Second}

// ── token storage ────────────────────────────────────

func loadToken() (string, error) {
	return keyring.Get(keyringService, keyringUser)
}

func saveToken(token string) error {
	return keyring.Set(keyringService, keyringUser, token)
}

func clearToken() {
	_ = keyring.Delete(keyringService, keyringUser)
}

// ── generic API helpers ──────────────────────────────

func apiCall(method, path string, body any, out any) error {
	token, err := loadToken()
	if err != nil || token == "" {
		return errNotLoggedIn
	}
	return apiCallWithToken(method, path, body, out, token)
}

// apiCallAuthed tries the request, and on 401 attempts a single token
// refresh before giving up (and clearing the stale token).
func apiCallAuthed(method, path string, body any, out any) error {
	token, err := loadToken()
	if err != nil || token == "" {
		return errNotLoggedIn
	}
	err = apiCallWithToken(method, path, body, out, token)
	if err == nil || !errors.Is(err, errUnauthorized) {
		return err
	}
	newToken, refreshErr := refreshToken(token)
	if refreshErr != nil {
		clearToken()
		return errNotLoggedIn
	}
	if err := saveToken(newToken); err != nil {
		return err
	}
	return apiCallWithToken(method, path, body, out, newToken)
}

var errUnauthorized = errors.New("unauthorized")

func apiCallWithToken(method, path string, body any, out any, token string) error {
	var payload io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		payload = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, apiBase()+path, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := accountClient.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接控制台服务: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode == http.StatusUnauthorized {
		return errUnauthorized
	}
	if resp.StatusCode >= 400 {
		msg := string(raw)
		var errBody struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if json.Unmarshal(raw, &errBody) == nil {
			if errBody.Message != "" {
				msg = errBody.Message
			} else if errBody.Error != "" {
				msg = errBody.Error
			}
		}
		return fmt.Errorf("请求失败 (%d): %s", resp.StatusCode, msg)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("响应解析失败: %w", err)
		}
	}
	return nil
}

func refreshToken(oldToken string) (string, error) {
	req, err := http.NewRequest("POST", apiBase()+"/api/collections/users/auth-refresh", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", oldToken)
	resp, err := accountClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errUnauthorized
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", errUnauthorized
	}
	return out.Token, nil
}

// ── wire types ───────────────────────────────────────

type AccountStatus struct {
	LoggedIn bool   `json:"loggedIn"`
	Email    string `json:"email"`
}

type AccountCredential struct {
	ID      string `json:"id"`
	GtID    string `json:"gt_id"`
	Enabled bool   `json:"enabled"`
}

type AccountPrefix struct {
	ID     string `json:"id"`
	Prefix string `json:"prefix"`
	Status string `json:"status"`
}

type AccountNode struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Region string `json:"region"`
	Addr   string `json:"addr"`
	Status string `json:"status"`
}

type AccountData struct {
	LoggedIn    bool                 `json:"loggedIn"`
	Email       string               `json:"email"`
	Plan        map[string]any       `json:"plan,omitempty"`
	Credentials []AccountCredential  `json:"credentials"`
	Prefixes    []AccountPrefix      `json:"prefixes"`
	Nodes       []AccountNode        `json:"nodes"`
}

type issuedCredential struct {
	GtID   string `json:"gt_id"`
	Secret string `json:"secret"`
}

// ── bound methods (delegated from GTApp) ────────────

func (a *GTApp) AccountStatus() (*AccountStatus, error) {
	var me struct {
		Email string `json:"email"`
	}
	if err := apiCallAuthed("GET", "/api/console/me", nil, &me); err != nil {
		if errors.Is(err, errNotLoggedIn) {
			return &AccountStatus{LoggedIn: false}, nil
		}
		return nil, err
	}
	return &AccountStatus{LoggedIn: true, Email: me.Email}, nil
}

func (a *GTApp) Login(email, password string) error {
	body := map[string]string{"identity": email, "password": password}
	var out struct {
		Token string `json:"token"`
	}
	req, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := accountClient.Post(apiBase()+"/api/collections/users/auth-with-password", "application/json", bytes.NewReader(req))
	if err != nil {
		return fmt.Errorf("无法连接控制台服务: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		var errBody struct {
			Message string `json:"message"`
		}
		msg := string(raw)
		if json.Unmarshal(raw, &errBody) == nil && errBody.Message != "" {
			msg = errBody.Message
		}
		if strings.Contains(msg, "Invalid email or password") {
			msg = "邮箱或密码错误"
		}
		return fmt.Errorf("登录失败: %s", msg)
	}
	if err := json.Unmarshal(raw, &out); err != nil || out.Token == "" {
		return errors.New("登录响应异常")
	}
	return saveToken(out.Token)
}

func (a *GTApp) Logout() error {
	clearToken()
	return nil
}

func (a *GTApp) GetAccountData() (*AccountData, error) {
	var me struct {
		Email string         `json:"email"`
		Plan  map[string]any `json:"plan"`
	}
	if err := apiCallAuthed("GET", "/api/console/me", nil, &me); err != nil {
		return nil, err
	}

	var creds struct {
		Credentials []AccountCredential `json:"credentials"`
	}
	var prefixes struct {
		HostPrefixes []AccountPrefix `json:"host_prefixes"`
	}
	var nodes struct {
		Items []AccountNode `json:"items"`
	}

	// nodes are public; the other two are per-user and must not fail the
	// whole call individually
	errCreds := apiCallAuthed("GET", "/api/console/credentials", nil, &creds)
	errPrefixes := apiCallAuthed("GET", "/api/console/host-prefixes", nil, &prefixes)
	_ = apiCall("GET", "/api/collections/nodes/records?perPage=50&sort=region", nil, &nodes)

	if errCreds != nil && !errors.Is(errCreds, errNotLoggedIn) {
		return nil, errCreds
	}
	if errPrefixes != nil && !errors.Is(errPrefixes, errNotLoggedIn) {
		return nil, errPrefixes
	}

	return &AccountData{
		LoggedIn:    true,
		Email:       me.Email,
		Plan:        me.Plan,
		Credentials:  creds.Credentials,
		Prefixes:    prefixes.HostPrefixes,
		Nodes:       nodes.Items,
	}, nil
}

// IssueCredential issues a new relay credential and writes it straight into
// the local config, so the user never copies id/secret by hand. The secret is
// returned once for the UI to display.
func (a *GTApp) IssueCredential() (*issuedCredential, error) {
	var issued issuedCredential
	if err := apiCallAuthed("POST", "/api/console/credentials", nil, &issued); err != nil {
		return nil, err
	}

	cfg, err := a.configManager.Load()
	if err != nil {
		return &issued, fmt.Errorf("凭证已签发但写入配置失败: %w", err)
	}
	cfg.ID = issued.GtID
	cfg.Secret = issued.Secret
	if err := a.configManager.Save(cfg); err != nil {
		return &issued, fmt.Errorf("凭证已签发但写入配置失败: %w", err)
	}
	return &issued, nil
}

// ApplyNode writes a node address (tcp:// for :80, tls:// for :443) into the
// config so the account UI's node picker wires straight into options.remote.
func (a *GTApp) ApplyNode(addr string) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return errors.New("节点地址为空")
	}
	port := addr
	if idx := strings.LastIndex(addr, ":"); idx >= 0 {
		port = addr[idx+1:]
	}
	scheme := "tcp"
	if port == "443" {
		scheme = "tls"
	}

	cfg, err := a.configManager.Load()
	if err != nil {
		return err
	}
	cfg.Remote = []string{scheme + "://" + addr}
	return a.configManager.Save(cfg)
}

func (a *GTApp) ClaimPrefix(prefix string) error {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return errors.New("前缀不能为空")
	}
	return apiCallAuthed("POST", "/api/console/host-prefixes",
		map[string]string{"prefix": prefix}, nil)
}

// ── B2: zero-config quick start ──────────────────────

type QuickStartResult struct {
	PublicURL string `json:"publicUrl"`
	Prefix    string `json:"prefix"`
	Node      string `json:"node"`
	GtID      string `json:"gtId"`
	Issued    bool   `json:"issued"` // whether a new credential was issued
}

// QuickStart takes a local port (and an optional prefix) and does everything:
// ensure credential → ensure node → ensure prefix → write a single HTTP
// service → start the tunnel. The user only ever typed a port number.
//
// It REPLACES the service list — that is the one-click contract; anything
// more elaborate belongs to the Tunnels tab.
func (a *GTApp) QuickStart(localPort uint16, prefix string) (*QuickStartResult, error) {
	if localPort == 0 {
		return nil, errors.New("本地端口无效")
	}
	userPrefix := strings.TrimSpace(prefix)
	if _, err := loadToken(); err != nil {
		return nil, errNotLoggedIn
	}

	// 1. credential: issue one only when the config has none
	cfg, err := a.configManager.Load()
	if err != nil {
		return nil, err
	}
	issued := false
	if strings.TrimSpace(cfg.ID) == "" || strings.TrimSpace(cfg.Secret) == "" {
		var cred issuedCredential
		if err := apiCallAuthed("POST", "/api/console/credentials", nil, &cred); err != nil {
			return nil, fmt.Errorf("签发凭证失败: %w", err)
		}
		cfg.ID = cred.GtID
		cfg.Secret = cred.Secret
		issued = true
	}

	// 2. node: keep the configured remote, otherwise pick the first online node
	nodeAddr := ""
	if len(cfg.Remote) > 0 && strings.TrimSpace(cfg.Remote[0]) != "" {
		nodeAddr = strings.TrimPrefix(strings.TrimPrefix(cfg.Remote[0], "tcp://"), "tls://")
	} else {
		nodeAddr, err = pickOnlineNode()
		if err != nil {
			return nil, err
		}
		scheme := "tcp"
		if strings.HasSuffix(nodeAddr, ":443") {
			scheme = "tls"
		}
		cfg.Remote = []string{scheme + "://" + nodeAddr}
	}

	// 3. prefix: use the given one, else generate; claim if not owned yet
	prefix = userPrefix
	if prefix == "" {
		prefix = randomPrefix()
	} else if !validPrefix(prefix) {
		return nil, errors.New("前缀格式无效（小写字母/数字/连字符，3-30 字符）")
	}
	owned := false
	var ownedList struct {
		HostPrefixes []AccountPrefix `json:"host_prefixes"`
	}
	if err := apiCallAuthed("GET", "/api/console/host-prefixes", nil, &ownedList); err == nil {
		for _, p := range ownedList.HostPrefixes {
			if p.Prefix == prefix {
				owned = true
				break
			}
		}
	}
	for attempt := 0; !owned; attempt++ {
		err := a.ClaimPrefix(prefix)
		if err == nil {
			owned = true
			break
		}
		if userPrefix != "" || attempt >= 2 || !strings.Contains(err.Error(), "already taken") {
			return nil, fmt.Errorf("申领前缀 %s 失败: %w", prefix, err)
		}
		// random guess collided — try another one
		prefix = randomPrefix()
	}

	// 4. write a single HTTP service replacing whatever was there
	cfg.Services = []DesktopService{{
		LocalURL:   DesktopURL{URL: fmt.Sprintf("http://127.0.0.1:%d", localPort)},
		HostPrefix: prefix,
	}}
	if err := a.configManager.Save(cfg); err != nil {
		return nil, fmt.Errorf("保存配置失败: %w", err)
	}

	// 5. start
	if err := a.StartTunnel(); err != nil {
		return nil, err
	}

	return &QuickStartResult{
		PublicURL: fmt.Sprintf("https://%s.app.gtunnel.dev", prefix),
		Prefix:    prefix,
		Node:      nodeAddr,
		GtID:      cfg.ID,
		Issued:    issued,
	}, nil
}

func pickOnlineNode() (string, error) {
	var nodes struct {
		Items []AccountNode `json:"items"`
	}
	if err := apiCall("GET", "/api/collections/nodes/records?perPage=50&sort=region", nil, &nodes); err != nil {
		return "", fmt.Errorf("获取节点列表失败: %w", err)
	}
	for _, n := range nodes.Items {
		if n.Status == "online" && strings.TrimSpace(n.Addr) != "" {
			return n.Addr, nil
		}
	}
	if len(nodes.Items) > 0 {
		return nodes.Items[0].Addr, nil // all offline — still try the first one
	}
	return "", errors.New("暂无可用节点，请在「连接」页手动配置或稍后再试")
}

const prefixLetters = "abcdefghjkmnpqrstuvwxyz"
const prefixChars = "abcdefghjkmnpqrstuvwxyz23456789"

func randomPrefix() string {
	raw := make([]byte, 8)
	if _, err := cryptorand.Read(raw); err != nil {
		return "app" + fmt.Sprint(time.Now().UnixNano()%10000)
	}
	out := make([]byte, 0, 8)
	out = append(out, prefixLetters[int(raw[0])%len(prefixLetters)])
	for _, b := range raw[1:] {
		out = append(out, prefixChars[int(b)%len(prefixChars)])
	}
	return string(out)
}

func validPrefix(p string) bool {
	if len(p) < 3 || len(p) > 30 {
		return false
	}
	for i, r := range p {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			continue
		case r == '-' && i > 0 && i < len(p)-1:
			continue
		default:
			return false
		}
	}
	return true
}
