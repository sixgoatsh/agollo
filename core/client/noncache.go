package client

import (
	"fmt"
	"net/url"

	"github.com/sixgoatsh/agollo/core/auth"
	"github.com/sixgoatsh/agollo/core/config"
	"github.com/sixgoatsh/agollo/core/util"
	"github.com/sixgoatsh/agollo/pkg/rest"
	"github.com/sixgoatsh/agollo/pkg/util/uri"
)

type INonCacheClient interface {
	GetConfigsFromNonCache(conf config.Config, opts ...NotificationsOption) (int, *NonCacheResp, error)
}

type NonCacheClient struct {
}

type NonCacheResp struct {
	AppID          string                `json:"appId"`          // appId: "AppTest",
	Cluster        string                `json:"cluster"`        // cluster: "default",
	NamespaceName  string                `json:"namespaceName"`  // namespaceName: "TEST.Namespace1",
	Configurations config.Configurations `json:"configurations"` // configurations: {Name: "Foo"},
	ReleaseKey     string                `json:"releaseKey"`     // releaseKey: "20181017110222-5ce3b2da895720e8"
}

func (c *NonCacheClient) GetConfigsFromNonCache(conf config.Config, opts ...NotificationsOption) (status int, resp *NonCacheResp, err error) {
	var options = NotificationsOptions{}
	for _, opt := range opts {
		opt(&options)
	}

	requestURI := fmt.Sprintf("/configs/%s/%s/%s?releaseKey=%s&ip=%s",
		url.QueryEscape(conf.AppID),
		url.QueryEscape(conf.ClusterName),
		url.QueryEscape(util.GetNamespace(conf.ConfigType, conf.NamespaceName)),
		options.ReleaseKey,
		conf.IP,
	)
	apiURL := fmt.Sprintf("%s%s", uri.NormalizeURL(conf.ConfigServerUrl), requestURI)
	headers := auth.HttpHeader(conf.AccessKey, conf.AppID, requestURI)
	resp = new(NonCacheResp)
	status, err = rest.Do("GET", apiURL, headers, resp)
	return

}
