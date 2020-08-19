package util

import (
	"math/rand"
	"net"
	"os"

	"github.com/sixgoatsh/agollo/core/cons"
	"github.com/sixgoatsh/agollo/pkg/util/uri"
)

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}

		}
	}
	return ""
}

/*
参考了java客户端实现
目前实现方式:
 0. 客户端显式传入ConfigServerURL
 2. Get from OS environment variable

未实现:
 1. Get from System Property
 3. Get from server.properties
https://github.com/ctripcorp/apollo/blob/master/apollo-client/src/main/java/com/ctrip/framework/apollo/internals/ConfigServiceLocator.java#L74
*/
func GetConfigServers(configServerURL string) []string {
	var urls []string
	for _, url := range []string{
		configServerURL,
		os.Getenv("APOLLO_CONFIGSERVICE"),
	} {
		if url != "" {
			urls = uri.SplitCommaSeparatedURL(url)
			break
		}
	}

	return urls
}

/*
参考了java客户端实现
目前实现方式:
0. 客户端显式传入ConfigServerURL
1. 读取APOLLO_META环境变量
2. 默认如果没有提供meta服务地址默认使用(http://apollo.meta)

未实现:
读取properties的逻辑
https://github.com/ctripcorp/apollo/blob/7545bd3cd7d4b996d7cda50f53cd4aa8b045a2bb/apollo-core/src/main/java/com/ctrip/framework/apollo/core/MetaDomainConsts.java#L27
*/
func GetMetaServerAddress(configServerURL string) string {
	var urls []string
	for _, url := range []string{
		configServerURL,
		os.Getenv("APOLLO_META"),
	} {
		if url != "" {
			urls = uri.SplitCommaSeparatedURL(url)
			break
		}
	}

	if len(urls) > 0 {
		return uri.NormalizeURL(urls[rand.Intn(len(urls))])
	}

	return cons.MetaURL
}
