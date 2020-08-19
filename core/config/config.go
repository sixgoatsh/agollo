package config

import (
	"encoding/json"

	"github.com/sixgoatsh/agollo/core/cons"
	"github.com/sixgoatsh/agollo/core/util"
)

type Config struct {
	ConfigServerUrl string         `json:"config_server_url"` // appId: "http://192.168.0.1:9898",
	AppID           string         `json:"appId"`             // appId: "AppTest",
	ClusterName     string         `json:"clusterName"`       // cluster: "default",
	NamespaceName   string         `json:"namespaceName"`     // namespaceName: "TEST.Namespace1",
	Configurations  Configurations `json:"configurations"`    // configurations: {Name: "Foo"},
	IP              string         `json:"ip"`                // releaseKey: "192.168.0.1"
	ReleaseKey      string         `json:"releaseKey"`        // releaseKey: "20181017110222-5ce3b2da895720e8"
	Notifications   Notifications  `json:"notifications"`
	AccessKey       string         `json:"accessKey"`
	ConfigType      string         `json:"configType"`
}

type Notifications []Notification

func (n Notifications) String() string {
	bytes, _ := json.Marshal(n)
	return string(bytes)
}

type Notification struct {
	NamespaceName  string `json:"namespaceName"`  // namespaceName: "application",
	NotificationID int    `json:"notificationId"` // notificationId: 107
}

type Option func(*Config)

func WithClientIP(ip string) Option {
	return func(a *Config) {
		a.IP = ip
	}
}

func WithClientAccessKey(accessKey string) Option {
	return func(a *Config) {
		a.AccessKey = accessKey
	}
}

func WithClientAppID(appID string) Option {
	return func(a *Config) {
		a.AppID = appID
	}
}

func WithClientClusterName(clusterName string) Option {
	return func(a *Config) {
		a.ClusterName = clusterName
	}
}

func WithClientNamespaceName(namespaceName string) Option {
	return func(a *Config) {
		a.NamespaceName = namespaceName
	}
}

func WithClientReleaseKey(releaseKey string) Option {
	return func(a *Config) {
		a.ReleaseKey = releaseKey
	}
}

func WithClientConfigServerUrl(configServerUrl string) Option {
	return func(a *Config) {
		a.ConfigServerUrl = configServerUrl
	}
}

func WithClientConfigurations(configurations Configurations) Option {
	return func(a *Config) {
		a.Configurations = configurations
	}
}

func WithClientNotifications(notifications Notifications) Option {
	return func(a *Config) {
		a.Notifications = notifications
	}
}

func (c *Config) Apply(opts ...Option) {
	for _, opt := range opts {
		opt(c)
	}
}

type IClientConfigurator interface {
	Apply(opts ...Option)
}

func DefaultConfig(configServerURL, appID string) Config {
	return Config{
		ConfigServerUrl: configServerURL,
		AppID:           appID,
		ClusterName:     cons.Cluster,
		IP:              util.GetLocalIP(),
	}
}
