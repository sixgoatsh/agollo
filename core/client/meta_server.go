package client

import (
	"fmt"

	"github.com/songxuexian/agollo/core/auth"
	"github.com/songxuexian/agollo/core/config"
	"github.com/songxuexian/agollo/pkg/rest"
	"github.com/songxuexian/agollo/pkg/util/uri"
)

type IMetaServerClient interface {
	GetConfigServers(conf config.Config) (int, []ConfigServerResp, error)
}

type MetaServerClient struct {
}

type ConfigServerResp struct {
	AppName     string `json:"appName"`
	InstanceID  string `json:"instanceId"`
	HomePageURL string `json:"homepageUrl"`
}

func (c *MetaServerClient) GetConfigServers(conf config.Config) (int, []ConfigServerResp, error) {
	requestURI := fmt.Sprintf("/services/config?id=%s&appId=%s", conf.IP, conf.AppID)
	apiURL := fmt.Sprintf("%s%s", uri.NormalizeURL(conf.ConfigServerUrl), requestURI)
	headers := auth.HttpHeader(conf.AccessKey, conf.AppID, requestURI)
	var cfs []ConfigServerResp
	status, err := rest.Do("GET", apiURL, headers, &cfs)
	return status, cfs, err
}
