package main

import (
	"context"
	"errors"
	"sync"
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
