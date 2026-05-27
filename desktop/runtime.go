package main

type StatusInfo struct {
	IsRunning  bool     `json:"isRunning"`
	ServerAddr string   `json:"serverAddr"`
	ClientID   string   `json:"clientId"`
	PingMs     int      `json:"pingMs"`
	SpeedUp    string   `json:"speedUp"`
	SpeedDown  string   `json:"speedDown"`
	ActiveSvc  []string `json:"activeSvc"`
}

type TunnelRuntime interface {
	Start(cfg *DesktopConfig) error
	Stop() error
	Status() (*StatusInfo, error)
	Close() error
}
