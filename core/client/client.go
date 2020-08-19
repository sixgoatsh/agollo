package client

import (
	"github.com/sixgoatsh/agollo/core/config"
)

type IApolloClient interface {
	GetNotifications(conf config.Config, result []config.Notification) (status int, err error)

	// 该接口会直接从数据库中获取配置，可以配合配置推送通知实现实时更新配置。
	GetConfigsFromNonCache(conf config.Config, opts ...NotificationsOption) (int, *NonCacheResp, error)
	// 该接口会从缓存中获取配置，适合频率较高的配置拉取请求，如简单的每30秒轮询一次配置。
	GetConfigsFromCache(config.Config) (conf *config.Configurations, err error)

	// 该接口从MetaServer获取ConfigServer列表
	GetConfigServers(config.Config) (int, []ConfigServerResp, error)
}

type ApolloClient struct {
	IMetaServerClient
	INonCacheClient
	ICacheClient
	INotificationClient
}

func NewApolloClient(metaServerClient IMetaServerClient, nonCacheClient INonCacheClient, cacheClient ICacheClient, notificationClient INotificationClient) IApolloClient {
	return &ApolloClient{
		IMetaServerClient:   metaServerClient,
		INonCacheClient:     nonCacheClient,
		ICacheClient:        cacheClient,
		INotificationClient: notificationClient,
	}
}

func New() IApolloClient {
	return &ApolloClient{
		IMetaServerClient:   &MetaServerClient{},
		INonCacheClient:     &NonCacheClient{},
		ICacheClient:        &CacheClient{},
		INotificationClient: &NotificationClient{},
	}
}
