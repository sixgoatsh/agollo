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

type ICacheClient interface {
	GetConfigsFromCache(config.Config) (conf *config.Configurations, err error)
}

type CacheClient struct {
}

func (c *CacheClient) GetConfigsFromCache(clientConf config.Config) (conf *config.Configurations, err error) {
	requestURI := fmt.Sprintf("/configfiles/json/%s/%s/%s?ip=%s",
		url.QueryEscape(clientConf.AppID),
		url.QueryEscape(clientConf.ClusterName),
		url.QueryEscape(util.GetNamespace(clientConf.ConfigType, clientConf.NamespaceName)),
		clientConf.IP,
	)
	apiURL := fmt.Sprintf("%s%s", uri.NormalizeURL(clientConf.ConfigServerUrl), requestURI)
	headers := auth.HttpHeader(clientConf.AccessKey, clientConf.AppID, requestURI)
	conf = new(config.Configurations)
	_, err = rest.Do("GET", apiURL, headers, conf)
	return
}
