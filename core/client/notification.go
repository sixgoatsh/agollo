package client

import (
	"fmt"
	"net/url"

	"github.com/sixgoatsh/agollo/core/auth"
	"github.com/sixgoatsh/agollo/core/config"
	"github.com/sixgoatsh/agollo/pkg/rest"
	"github.com/sixgoatsh/agollo/pkg/util/uri"
)

type NotificationsOptions struct {
	ReleaseKey string
}

type NotificationsOption func(*NotificationsOptions)

func ReleaseKey(releaseKey string) NotificationsOption {
	return func(o *NotificationsOptions) {
		o.ReleaseKey = releaseKey
	}
}

type INotificationClient interface {
	GetNotifications(conf config.Config,result []config.Notification) (status int, err error)
}

type NotificationClient struct {
}

func (c *NotificationClient) GetNotifications(conf config.Config, result []config.Notification) (status int, err error) {
	requestURI := fmt.Sprintf("/notifications/v2?appId=%s&cluster=%s&notifications=%s",
		url.QueryEscape(conf.AppID),
		url.QueryEscape(conf.ClusterName),
		url.QueryEscape(conf.Notifications.String()),
	)
	apiURL := fmt.Sprintf("%s%s", uri.NormalizeURL(conf.ConfigServerUrl), requestURI)

	headers := auth.HttpHeader(conf.AccessKey, conf.AppID, requestURI)
	status, err = rest.Do("GET", apiURL, headers, &result)
	return
}
