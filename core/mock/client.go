package mock

import (
	"github.com/songxuexian/agollo/core/client"
	"github.com/songxuexian/agollo/core/config"
)

type CacheClient struct {
	ConfigsFromCache func(config.Config) (conf *config.Configurations, err error)
}

func (c *CacheClient) GetConfigsFromCache(clientConf config.Config) (conf *config.Configurations, err error) {
	if c.ConfigsFromCache == nil {
		return nil, nil
	}
	return c.ConfigsFromCache(clientConf)
}

type NonCacheClient struct {
	ConfigsFromNonCache func(conf config.Config, opts ...client.NotificationsOption) (int, *client.NonCacheResp, error)
}

func (c *NonCacheClient) GetConfigsFromNonCache(conf config.Config, opts ...client.NotificationsOption) (int, *client.NonCacheResp, error) {
	if c.ConfigsFromNonCache == nil {
		return 404, nil, nil
	}
	return c.ConfigsFromNonCache(conf, opts...)
}

type NotificationsClient struct {
	Notifications func(conf config.Config, result []config.Notification) (status int, err error)
}

func (c *NotificationsClient) GetNotifications(conf config.Config, result []config.Notification) (status int, err error) {
	if c.Notifications == nil {
		return 404, nil
	}
	return c.Notifications(conf, result)
}

type MetaServerClient struct {
	ConfigServers func(config.Config) (int, []client.ConfigServerResp, error)
}

func (c *MetaServerClient) GetConfigServers(conf config.Config) (int, []client.ConfigServerResp, error) {
	if c.ConfigServers == nil {
		return 404, nil, nil
	}
	return c.ConfigServers(conf)
}
