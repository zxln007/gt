package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
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

func (cm *ConfigManager) SaveRuntimeConfig(cfg *RuntimeConfig) error {
	data, err := yaml.Marshal(cfg)
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
