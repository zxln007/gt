package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"strings"
)

type ConfigManager struct {
	configDir  string
	configPath string
}

func NewConfigManager() *ConfigManager {
	homeDir, err := os.UserHomeDir()
	configDir := "."
	if err == nil {
		configDir = filepath.Join(homeDir, ".gt-desktop")
	}

	_ = os.MkdirAll(configDir, 0o700)

	return &ConfigManager{
		configDir:  configDir,
		configPath: filepath.Join(configDir, "config.yaml"),
	}
}

func (cm *ConfigManager) GetConfigPath() string {
	return cm.configPath
}

func (cm *ConfigManager) GetRuntimeConfigPath() string {
	return filepath.Join(cm.configDir, "runtime-client.yaml")
}

func (cm *ConfigManager) Load() (*DesktopConfig, error) {
	cfg := defaultDesktopConfig()

	if _, err := os.Stat(cm.configPath); os.IsNotExist(err) {
		if err := cm.Save(&cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	applyDesktopDefaults(&cfg)
	return &cfg, nil
}

func (cm *ConfigManager) Save(cfg *DesktopConfig) error {
	cfg.ConfigType = "client"
	applyDesktopDefaults(cfg)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(cm.configPath, data, 0o600)
}

// gt 客户端（v2.3+）要求的配置格式是 version/services/options 包裹结构，
// 顶层扁平的 id/secret 会被静默丢弃（表现为 "agent id ” is invalid"）。
// 本地 config.yaml 仍保留扁平格式便于桌面端读写，仅运行时输出转换为原生格式。
type gtRuntimeOptions struct {
	DesktopConfig `yaml:",inline"`
	WebAddr       string `yaml:"webAddr,omitempty"`
	SigningKey    string `yaml:"signingKey,omitempty"`
	Admin         string `yaml:"admin,omitempty"`
	Password      string `yaml:"password,omitempty"`
}

type gtRuntimeConfig struct {
	Version  string           `yaml:"version"`
	Services []DesktopService `yaml:"services,omitempty"`
	Options  gtRuntimeOptions `yaml:"options"`
}

func (cm *ConfigManager) SaveRuntimeConfig(cfg *RuntimeConfig) error {
	opts := gtRuntimeOptions{
		DesktopConfig: cfg.DesktopConfig,
		WebAddr:       cfg.WebAddr,
		SigningKey:    cfg.SigningKey,
		Admin:         cfg.Admin,
		Password:      cfg.Password,
	}
	opts.ConfigType = "" // type 不属于 options 节
	native := gtRuntimeConfig{
		Version:  "1.0",
		Services: cfg.Services,
		Options:  opts,
	}
	data, err := yaml.Marshal(&native)
	if err != nil {
		return err
	}
	return os.WriteFile(cm.GetRuntimeConfigPath(), data, 0o600)
}

func defaultDesktopConfig() DesktopConfig {
	randomPort := true
	return DesktopConfig{
		ConfigType:            "client",
		ID:                    "your-custom-id",
		Secret:                "your-custom-secret",
		Remote:                []string{"tcp://your-server-ip:7001"},
		RemoteConnections:     3,
		RemoteIdleConnections: 1,
		LogFileMaxCount:       7,
		LogFileMaxSize:        512 * 1024 * 1024,
		LogLevel:              "info",
		WebRTCLogLevel:        "warning",
		Services: []DesktopService{
			{
				LocalURL:        DesktopURL{URL: "http://127.0.0.1:8080"},
				RemoteTCPRandom: &randomPort,
			},
		},
	}
}

func applyDesktopDefaults(cfg *DesktopConfig) {
	if cfg.ConfigType == "" {
		cfg.ConfigType = "client"
	}
	if cfg.RemoteConnections == 0 {
		cfg.RemoteConnections = 3
	}
	if cfg.RemoteIdleConnections == 0 {
		cfg.RemoteIdleConnections = 1
	}
	if cfg.LogFileMaxCount == 0 {
		cfg.LogFileMaxCount = 7
	}
	if cfg.LogFileMaxSize == 0 {
		cfg.LogFileMaxSize = 512 * 1024 * 1024
	}
	if cfg.LogLevel == "" {
		cfg.LogLevel = "info"
	}
	if cfg.WebRTCLogLevel == "" {
		cfg.WebRTCLogLevel = "warning"
	}
}

// placeholder defaults from defaultDesktopConfig() must count as UNCONFIGURED,
// otherwise QuickStart skips issuing credentials and the worker starts with a
// literal "your-custom-id".
func isPlaceholder(v string) bool {
	return v == "" || strings.Contains(v, "your-custom-") || strings.Contains(v, "your-server-ip")
}

func credentialConfigured(cfg *DesktopConfig) bool {
	return !isPlaceholder(cfg.ID) && !isPlaceholder(cfg.Secret)
}

func remoteConfigured(cfg *DesktopConfig) bool {
	return len(cfg.Remote) > 0 && !isPlaceholder(cfg.Remote[0])
}
