package main

import "gopkg.in/yaml.v3"

type DesktopConfig struct {
	ConfigType            string                 `yaml:"type,omitempty" json:"ConfigType,omitempty"`
	Services              []DesktopService       `yaml:"services,omitempty" json:"Services,omitempty"`
	ID                    string                 `yaml:"id,omitempty" json:"ID,omitempty"`
	Secret                string                 `yaml:"secret,omitempty" json:"Secret,omitempty"`
	ReconnectDelay        string                 `yaml:"reconnectDelay,omitempty" json:"ReconnectDelay,omitempty"`
	Remote                []string               `yaml:"remote,omitempty" json:"Remote,omitempty"`
	RemoteSTUN            []string               `yaml:"remoteSTUN,omitempty" json:"RemoteSTUN,omitempty"`
	RemoteAPI             string                 `yaml:"remoteAPI,omitempty" json:"RemoteAPI,omitempty"`
	RemoteCert            string                 `yaml:"remoteCert,omitempty" json:"RemoteCert,omitempty"`
	RemoteCertInsecure    bool                   `yaml:"remoteCertInsecure,omitempty" json:"RemoteCertInsecure,omitempty"`
	RemoteConnections     uint                   `yaml:"remoteConnections,omitempty" json:"RemoteConnections,omitempty"`
	RemoteIdleConnections uint                   `yaml:"remoteIdleConnections,omitempty" json:"RemoteIdleConnections,omitempty"`
	RemoteTimeout         string                 `yaml:"remoteTimeout,omitempty" json:"RemoteTimeout,omitempty"`
	WebRTCLogLevel        string                 `yaml:"webrtcLogLevel,omitempty" json:"WebRTCLogLevel,omitempty"`
	LogFile               string                 `yaml:"logFile,omitempty" json:"LogFile,omitempty"`
	LogFileMaxSize        int64                  `yaml:"logFileMaxSize,omitempty" json:"LogFileMaxSize,omitempty"`
	LogFileMaxCount       uint                   `yaml:"logFileMaxCount,omitempty" json:"LogFileMaxCount,omitempty"`
	LogLevel              string                 `yaml:"logLevel,omitempty" json:"LogLevel,omitempty"`
	OpenBBR               bool                   `yaml:"bbr,omitempty" json:"OpenBBR,omitempty"`
	Extras                map[string]interface{} `yaml:",inline" json:"Extras,omitempty"`
}

type DesktopService struct {
	HostPrefix         string                 `yaml:"hostPrefix,omitempty" json:"HostPrefix,omitempty"`
	RemoteTCPPort      uint16                 `yaml:"remoteTCPPort,omitempty" json:"RemoteTCPPort,omitempty"`
	RemoteTCPRandom    *bool                  `yaml:"remoteTCPRandom,omitempty" json:"RemoteTCPRandom,omitempty"`
	LocalURL           DesktopURL             `yaml:"local,omitempty" json:"LocalURL,omitempty"`
	LocalTimeout       string                 `yaml:"localTimeout,omitempty" json:"LocalTimeout,omitempty"`
	UseLocalAsHTTPHost bool                   `yaml:"useLocalAsHTTPHost,omitempty" json:"UseLocalAsHTTPHost,omitempty"`
	Extras             map[string]interface{} `yaml:",inline" json:"Extras,omitempty"`
}

type DesktopURL struct {
	URL string `json:"URL,omitempty"`
}

func (u *DesktopURL) UnmarshalYAML(value *yaml.Node) error {
	u.URL = value.Value
	return nil
}

func (u DesktopURL) MarshalYAML() (interface{}, error) {
	if u.URL == "" {
		return nil, nil
	}
	return u.URL, nil
}

type RuntimeConfig struct {
	DesktopConfig `yaml:",inline"`
	WebAddr       string `yaml:"webAddr,omitempty"`
	SigningKey    string `yaml:"signingKey,omitempty"`
	Admin         string `yaml:"admin,omitempty"`
	Password      string `yaml:"password,omitempty"`
}
