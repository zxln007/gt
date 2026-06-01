package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pkg/browser"
	"gopkg.in/yaml.v3"
)

type GTApp struct {
	ctx           context.Context
	runtime       TunnelRuntime
	runtimeMtx    sync.Mutex
	configManager *ConfigManager
	logWriter     *WailsLogWriter
}

func NewGTApp(cm *ConfigManager, lw *WailsLogWriter, runtime TunnelRuntime) *GTApp {
	return &GTApp{
		runtime:       runtime,
		configManager: cm,
		logWriter:     lw,
	}
}

func (a *GTApp) OnStartup(ctx context.Context) {
	a.ctx = ctx
}

func (a *GTApp) OnShutdown(ctx context.Context) {
	_ = a.runtime.Close()
}

func (a *GTApp) LoadConfig() (*DesktopConfig, error) {
	return a.configManager.Load()
}

func (a *GTApp) SaveConfig(cfg *DesktopConfig) error {
	return a.configManager.Save(cfg)
}

func (a *GTApp) ImportConfig(payload string) (*DesktopConfig, error) {
	data, err := decodeImportPayload(payload)
	if err != nil {
		return nil, err
	}

	var cfg DesktopConfig
	if err := yaml.Unmarshal([]byte(data), &cfg); err != nil {
		return nil, fmt.Errorf("配置解析失败: %w", err)
	}
	if len(cfg.Remote) == 0 || strings.TrimSpace(cfg.Remote[0]) == "" {
		return nil, errors.New("导入配置缺少 remote 中转服务器地址")
	}
	if strings.TrimSpace(cfg.ID) == "" {
		return nil, errors.New("导入配置缺少 id 客户端鉴权 ID")
	}
	if strings.TrimSpace(cfg.Secret) == "" {
		return nil, errors.New("导入配置缺少 secret 鉴权密钥")
	}

	deleteRuntimeOnlyKeys(&cfg)
	if err := a.configManager.Save(&cfg); err != nil {
		return nil, err
	}
	return a.configManager.Load()
}

func (a *GTApp) TestServerConnection() (string, error) {
	cfg, err := a.configManager.Load()
	if err != nil {
		return "", err
	}
	if len(cfg.Remote) == 0 || strings.TrimSpace(cfg.Remote[0]) == "" {
		return "", errors.New("中转服务器节点未配置")
	}

	address, err := remoteDialAddress(cfg.Remote[0])
	if err != nil {
		return "", err
	}

	start := time.Now()
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("连接 %s 失败: %w", address, err)
	}
	_ = conn.Close()

	return fmt.Sprintf("连通成功，TCP 握手耗时 %dms", time.Since(start).Milliseconds()), nil
}

func (a *GTApp) OpenExternalURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("仅支持打开 http/https 链接")
	}
	return browser.OpenURL(parsed.String())
}

func (a *GTApp) StartTunnel() error {
	a.runtimeMtx.Lock()
	defer a.runtimeMtx.Unlock()

	status, err := a.runtime.Status()
	if err == nil && status != nil && status.IsRunning {
		return errors.New("内网穿透隧道已经运行")
	}

	cfg, err := a.configManager.Load()
	if err != nil {
		return err
	}

	a.logWriter.Clear()
	return a.runtime.Start(cfg)
}

func (a *GTApp) StopTunnel() error {
	a.runtimeMtx.Lock()
	defer a.runtimeMtx.Unlock()

	status, err := a.runtime.Status()
	if err != nil {
		return err
	}
	if status == nil || !status.IsRunning {
		return errors.New("内网穿透隧道未在运行")
	}

	return a.runtime.Stop()
}

func (a *GTApp) GetStatus() (*StatusInfo, error) {
	a.runtimeMtx.Lock()
	defer a.runtimeMtx.Unlock()

	info, err := a.runtime.Status()
	if err != nil {
		return &StatusInfo{IsRunning: false}, err
	}
	if info == nil {
		return &StatusInfo{IsRunning: false}, nil
	}
	return info, nil
}

func (a *GTApp) GetLogs() []string {
	return a.logWriter.GetLogs()
}

func (a *GTApp) ClearLogs() {
	a.logWriter.Clear()
}

func remoteDialAddress(rawRemote string) (string, error) {
	rawRemote = strings.TrimSpace(rawRemote)
	parsed, err := url.Parse(rawRemote)
	if err == nil && parsed.Host != "" {
		if _, _, err := net.SplitHostPort(parsed.Host); err != nil {
			return "", fmt.Errorf("中转服务器地址必须包含端口: %s", rawRemote)
		}
		return parsed.Host, nil
	}
	if strings.Contains(rawRemote, "://") {
		return "", fmt.Errorf("无法解析中转服务器地址: %s", rawRemote)
	}
	if _, _, err := net.SplitHostPort(rawRemote); err != nil {
		return "", fmt.Errorf("中转服务器地址必须包含端口: %s", rawRemote)
	}
	return rawRemote, nil
}

func decodeImportPayload(payload string) (string, error) {
	payload = strings.TrimSpace(payload)
	if payload == "" {
		return "", errors.New("导入内容为空")
	}
	if !strings.HasPrefix(payload, "gt://") {
		return payload, nil
	}

	parsed, err := url.Parse(payload)
	if err != nil {
		return "", err
	}
	encoded := parsed.Query().Get("profile")
	if encoded == "" {
		encoded = parsed.Query().Get("config")
	}
	if encoded == "" {
		return "", errors.New("导入链接缺少 profile 参数")
	}

	decoded, err := decodeBase64URL(encoded)
	if err != nil {
		return "", fmt.Errorf("导入链接 profile 解码失败: %w", err)
	}
	return string(decoded), nil
}

func decodeBase64URL(encoded string) ([]byte, error) {
	if decoded, err := base64.RawURLEncoding.DecodeString(encoded); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.URLEncoding.DecodeString(encoded); err == nil {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(encoded); err == nil {
		return decoded, nil
	}
	return base64.StdEncoding.DecodeString(encoded)
}

func deleteRuntimeOnlyKeys(cfg *DesktopConfig) {
	for _, key := range []string{"webAddr", "signingKey", "admin", "password"} {
		delete(cfg.Extras, key)
	}
}
