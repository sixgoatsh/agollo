package balancer

import (
	"errors"
	"time"

	"github.com/songxuexian/agollo/core/client"
	"github.com/songxuexian/agollo/core/config"
	"github.com/songxuexian/agollo/core/util"
	"github.com/songxuexian/agollo/pkg/log"
)

var (
	defaultRefreshIntervalInSecond = time.Second * 60
	ErrNoConfigServerAvailable     = errors.New("no config server availbale")
)

type Balancer interface {
	Select() (string, error)
	Stop()
}

func NewBalancer(conf config.Config, enableSLB bool, refreshIntervalInSecond time.Duration, log log.Logger, client client.IMetaServerClient) (Balancer, error) {
	var b Balancer
	configServerURLs := util.GetConfigServers(conf.ConfigServerUrl)
	if enableSLB || len(configServerURLs) == 0 {
		var err error
		b, err = NewAutoFetchBalancer(conf, client, refreshIntervalInSecond, log)
		if err != nil {
			return nil, err
		}
	} else {
		b = NewRoundRobin(configServerURLs)
	}

	return b, nil
}
