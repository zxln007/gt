package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type GTProcessRuntime struct {
	mu            sync.Mutex
	configManager *ConfigManager
	logWriter     *WailsLogWriter
	httpClient    *http.Client

	cmd        *exec.Cmd
	apiBaseURL string
	token      string
	activeCfg  *DesktopConfig
}

type gtLaunchSpec struct {
	command string
	args    []string
	dir     string
	label   string
}

func NewGTProcessRuntime(cm *ConfigManager, lw *WailsLogWriter) *GTProcessRuntime {
	return &GTProcessRuntime{
		configManager: cm,
		logWriter:     lw,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

func (r *GTProcessRuntime) Start(cfg *DesktopConfig) error {
	r.mu.Lock()
	if r.cmd != nil {
		r.mu.Unlock()
		return errors.New("gt worker process is already running")
	}
	r.mu.Unlock()

	launchSpec, err := r.resolveGTLaunch()
	if err != nil {
		return err
	}

	controlPort, err := reserveLoopbackPort()
	if err != nil {
		return err
	}

	runtimeCfg := RuntimeConfig{
		DesktopConfig: cloneDesktopConfig(*cfg),
		WebAddr:       fmt.Sprintf("127.0.0.1:%d", controlPort),
		SigningKey:    randomASCII(32),
		Admin:         randomASCII(12),
		Password:      randomASCII(18),
	}
	if err := r.configManager.SaveRuntimeConfig(&runtimeCfg); err != nil {
		return err
	}

	cmd := exec.Command(launchSpec.command, append(launchSpec.args, r.configManager.GetRuntimeConfigPath())...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	cmd.Dir = launchSpec.dir

	if err := cmd.Start(); err != nil {
		return err
	}

	baseURL := "http://" + runtimeCfg.WebAddr
	r.mu.Lock()
	r.cmd = cmd
	r.apiBaseURL = baseURL
	r.activeCfg = &runtimeCfg.DesktopConfig
	r.mu.Unlock()

	go r.scanLogs(stdout, "stdout")
	go r.scanLogs(stderr, "stderr")
	go r.waitProcess(cmd)

	if err := r.waitForReady(baseURL, 20*time.Second); err != nil {
		_ = r.forceStop(cmd)
		return err
	}

	token, err := r.login(baseURL, runtimeCfg.Admin, runtimeCfg.Password)
	if err != nil {
		_ = r.forceStop(cmd)
		return err
	}

	r.mu.Lock()
	if r.cmd == cmd {
		r.token = token
	}
	r.mu.Unlock()

	r.logWriter.AppendLine("desktop: gt subprocess started via " + launchSpec.label)
	return nil
}

func (r *GTProcessRuntime) Stop() error {
	r.mu.Lock()
	cmd := r.cmd
	baseURL := r.apiBaseURL
	token := r.token
	r.mu.Unlock()

	if cmd == nil {
		return nil
	}

	if baseURL != "" && token != "" {
		if err := r.apiRequest(http.MethodPut, baseURL+"/api/server/stop", token, nil, nil); err != nil {
			r.logWriter.AppendLine("desktop: graceful stop failed, falling back to kill: " + err.Error())
		}
	}

	if r.waitForExit(5 * time.Second) {
		return nil
	}
	return r.forceStop(cmd)
}

func (r *GTProcessRuntime) Close() error {
	return r.Stop()
}

func (r *GTProcessRuntime) Status() (*StatusInfo, error) {
	r.mu.Lock()
	cfg := r.activeCfg
	running := r.cmd != nil
	baseURL := r.apiBaseURL
	r.mu.Unlock()

	if !running || cfg == nil {
		return &StatusInfo{IsRunning: false}, nil
	}

	pingMs := 0
	if baseURL != "" {
		start := time.Now()
		resp, err := r.httpClient.Get(baseURL + "/api/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				pingMs = int(time.Since(start).Milliseconds())
			}
		}
	}

	info := &StatusInfo{
		IsRunning:  true,
		ClientID:   cfg.ID,
		PingMs:     pingMs,
		SpeedUp:    "--",
		SpeedDown:  "--",
		ActiveSvc:  buildActiveServices(cfg),
		ServerAddr: firstRemote(cfg.Remote),
	}
	return info, nil
}

func (r *GTProcessRuntime) resolveGTLaunch() (*gtLaunchSpec, error) {
	if custom := os.Getenv("GT_DESKTOP_GT_BINARY"); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return &gtLaunchSpec{
				command: custom,
				args:    []string{"client", "--config"},
				dir:     filepath.Dir(custom),
				label:   custom,
			}, nil
		}
	}

	names := []string{"gt"}
	if runtime.GOOS == "windows" {
		names = []string{"gt.exe", "gt"}
	}
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return &gtLaunchSpec{
				command: path,
				args:    []string{"client", "--config"},
				dir:     filepath.Dir(path),
				label:   path,
			}, nil
		}
	}

	exeDir, _ := os.Executable()
	exeDir = filepath.Dir(exeDir)
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(exeDir, binaryName("gt")),
		filepath.Join(exeDir, "..", "target", "debug", binaryName("gt")),
		filepath.Join(exeDir, "..", "target", "release", binaryName("gt")),
		filepath.Join(exeDir, "..", "bin", "gt-win-x86_64.exe"),
		filepath.Join(exeDir, "..", "bin", binaryName("gt")),
		filepath.Join(cwd, "..", "target", "debug", binaryName("gt")),
		filepath.Join(cwd, "..", "target", "release", binaryName("gt")),
		filepath.Join(cwd, "..", "bin", "gt-win-x86_64.exe"),
		filepath.Join(cwd, "..", "bin", binaryName("gt")),
		filepath.Join(cwd, "..", "release", "gt-win-x86_64.exe"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			abs, _ := filepath.Abs(candidate)
			return &gtLaunchSpec{
				command: abs,
				args:    []string{"client", "--config"},
				dir:     filepath.Dir(abs),
				label:   abs,
			}, nil
		}
	}

	repoRoot := detectRepoRoot(exeDir, cwd)
	if repoRoot != "" {
		if cargoPath, err := exec.LookPath("cargo"); err == nil {
			return &gtLaunchSpec{
				command: cargoPath,
				args:    []string{"run", "--package", "gt", "--", "client", "--config"},
				dir:     repoRoot,
				label:   "cargo run -p gt",
			}, nil
		}
	}

	return nil, errors.New("未找到 gt 可执行文件，且无法使用 cargo run 启动，请先构建 gt 二进制，或通过 GT_DESKTOP_GT_BINARY 指定路径")
}

func (r *GTProcessRuntime) waitForReady(baseURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r.isStopped() {
			return errors.New("gt subprocess exited before control api became ready")
		}
		resp, err := r.httpClient.Get(baseURL + "/api/health")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(300 * time.Millisecond)
	}
	return errors.New("等待 gt 本地控制接口超时")
}

func (r *GTProcessRuntime) login(baseURL, username, password string) (string, error) {
	payload := map[string]string{
		"username": username,
		"password": password,
	}
	var respData struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
		Msg string `json:"msg"`
	}
	if err := r.apiRequest(http.MethodPost, baseURL+"/api/login", "", payload, &respData); err != nil {
		return "", err
	}
	if respData.Code != 200 || respData.Data.Token == "" {
		return "", fmt.Errorf("login failed: %s", respData.Msg)
	}
	return respData.Data.Token, nil
}

func (r *GTProcessRuntime) apiRequest(method, url, token string, payload interface{}, target interface{}) error {
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("x-token", token)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if target != nil {
		if err := json.Unmarshal(data, target); err != nil {
			return err
		}
	}
	return nil
}

func (r *GTProcessRuntime) scanLogs(pipe io.Reader, stream string) {
	scanner := bufio.NewScanner(pipe)
	scanner.Buffer(make([]byte, 0, 4096), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		r.logWriter.AppendLine(line)
	}
	if err := scanner.Err(); err != nil {
		r.logWriter.AppendLine("desktop: failed to read gt " + stream + ": " + err.Error())
	}
}

func (r *GTProcessRuntime) waitProcess(cmd *exec.Cmd) {
	err := cmd.Wait()
	if err != nil {
		r.logWriter.AppendLine("desktop: gt subprocess exited with error: " + err.Error())
	} else {
		r.logWriter.AppendLine("desktop: gt subprocess exited")
	}

	r.mu.Lock()
	if r.cmd == cmd {
		r.cmd = nil
		r.apiBaseURL = ""
		r.token = ""
		r.activeCfg = nil
	}
	r.mu.Unlock()

	_ = os.Remove(r.configManager.GetRuntimeConfigPath())
}

func (r *GTProcessRuntime) waitForExit(timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if r.isStopped() {
			return true
		}
		time.Sleep(150 * time.Millisecond)
	}
	return r.isStopped()
}

func (r *GTProcessRuntime) forceStop(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := cmd.Process.Kill(); err != nil && !strings.Contains(err.Error(), "process already finished") {
		return err
	}
	_ = r.waitForExit(2 * time.Second)
	return nil
}

func (r *GTProcessRuntime) isStopped() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd == nil
}

func buildActiveServices(cfg *DesktopConfig) []string {
	result := make([]string, 0, len(cfg.Services))
	for _, svc := range cfg.Services {
		local := svc.LocalURL.URL
		if local == "" {
			local = "未知本地服务"
		}

		remote := svc.HostPrefix
		if remote == "" {
			random := svc.RemoteTCPRandom == nil || *svc.RemoteTCPRandom
			switch {
			case !random && svc.RemoteTCPPort > 0:
				remote = fmt.Sprintf("remote tcp port %d", svc.RemoteTCPPort)
			case random:
				remote = "random remote tcp port"
			default:
				remote = "configured remote endpoint"
			}
		}

		result = append(result, fmt.Sprintf("%s -> %s", local, remote))
	}
	return result
}

func firstRemote(remotes []string) string {
	if len(remotes) == 0 {
		return ""
	}
	return remotes[0]
}

func cloneDesktopConfig(cfg DesktopConfig) DesktopConfig {
	out := cfg
	if cfg.Remote != nil {
		out.Remote = append([]string(nil), cfg.Remote...)
	}
	if cfg.RemoteSTUN != nil {
		out.RemoteSTUN = append([]string(nil), cfg.RemoteSTUN...)
	}
	if cfg.Services != nil {
		out.Services = make([]DesktopService, len(cfg.Services))
		copy(out.Services, cfg.Services)
	}
	if cfg.Extras != nil {
		out.Extras = make(map[string]interface{}, len(cfg.Extras))
		for k, v := range cfg.Extras {
			out.Extras[k] = v
		}
	}
	return out
}

func reserveLoopbackPort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()

	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		return 0, errors.New("failed to allocate loopback port")
	}
	return addr.Port, nil
}

func binaryName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func detectRepoRoot(paths ...string) string {
	for _, base := range paths {
		if base == "" {
			continue
		}
		candidates := []string{
			base,
			filepath.Join(base, ".."),
			filepath.Join(base, "..", ".."),
		}
		for _, candidate := range candidates {
			abs, err := filepath.Abs(candidate)
			if err != nil {
				continue
			}
			if _, err := os.Stat(filepath.Join(abs, "Cargo.toml")); err == nil {
				if _, err := os.Stat(filepath.Join(abs, "cli", "Cargo.toml")); err == nil {
					return abs
				}
			}
		}
	}
	return ""
}

func randomASCII(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	buf := make([]byte, length)
	_, _ = rand.Read(buf)
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf)
}
