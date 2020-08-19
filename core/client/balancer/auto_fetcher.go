package balancer

import (
	"sync"
	"time"

	"github.com/songxuexian/agollo/core/client"
	"github.com/songxuexian/agollo/core/config"
	"github.com/songxuexian/agollo/core/util"
	"github.com/songxuexian/agollo/pkg/log"
	"github.com/songxuexian/agollo/pkg/util/uri"
)

type autoFetchBalancer struct {
	conf              config.Config
	metaServerClient  client.IMetaServerClient
	metaServerAddress string

	logger log.Logger

	mu sync.RWMutex
	b  Balancer

	stopCh chan struct{}
}

func NewAutoFetchBalancer(conf config.Config, metaServerClient client.IMetaServerClient,
	refreshIntervalInSecond time.Duration, logger log.Logger) (Balancer, error) {

	if refreshIntervalInSecond <= time.Duration(0) {
		refreshIntervalInSecond = defaultRefreshIntervalInSecond
	}

	b := &autoFetchBalancer{
		conf:              conf,
		metaServerClient:  metaServerClient,
		metaServerAddress: util.GetMetaServerAddress(conf.ConfigServerUrl), // Meta Server只是一个逻辑角色，在部署时和Config Service是在一个JVM进程中的，所以IP、端口和Config Service一致
		logger:            logger,
		stopCh:            make(chan struct{}),
		b:                 NewRoundRobin([]string{conf.ConfigServerUrl}),
	}

	err := b.updateConfigServices()
	if err != nil {
		return nil, err
	}

	go func() {
		ticker := time.NewTicker(refreshIntervalInSecond)
		defer ticker.Stop()

		for {
			select {
			case <-b.stopCh:
				return
			case <-ticker.C:
				_ = b.updateConfigServices()
			}
		}
	}()

	return b, nil
}

func (b *autoFetchBalancer) updateConfigServices() error {
	css, err := b.getConfigServices()
	if err != nil {
		return err
	}

	var urls []string
	for _, url := range css {
		// check whether /services/config is accessible
		conf := b.conf
		conf.ConfigServerUrl = url
		status, _, err := b.metaServerClient.GetConfigServers(conf)
		if err != nil {
			continue
		}

		// select the first available meta server
		// https://github.com/ctripcorp/apollo/blob/7545bd3cd7d4b996d7cda50f53cd4aa8b045a2bb/apollo-core/src/main/java/com/ctrip/framework/apollo/core/MetaDomainConsts.java#L166
		// 这里这段逻辑是参考java客户端，直接选了第一个可用的meta server
		if 200 <= status && status <= 399 {
			urls = append(urls, url)
			break
		}
	}

	if len(urls) == 0 {
		return nil
	}

	b.mu.Lock()
	b.b = NewRoundRobin(css)
	b.mu.Unlock()

	return nil
}

func (b *autoFetchBalancer) getConfigServices() ([]string, error) {
	conf := b.conf
	conf.ConfigServerUrl = b.metaServerAddress
	_, css, err := b.metaServerClient.GetConfigServers(conf)
	if err != nil {
		b.logger.Log(
			"[GoApollo]", "",
			"AppID", conf.AppID,
			"MetaServerAddress", b.metaServerAddress,
			"Error", err,
		)
		return nil, err
	}

	var urls []string
	for _, cs := range css {
		urls = append(urls, uri.NormalizeURL(cs.HomePageURL))
	}

	return urls, nil
}

func (b *autoFetchBalancer) Select() (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.b.Select()
}

func (b *autoFetchBalancer) Stop() {
	close(b.stopCh)
}
