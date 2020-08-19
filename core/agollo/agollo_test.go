package agollo

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sixgoatsh/agollo/core/client"
	"github.com/sixgoatsh/agollo/core/client/balancer"
	"github.com/sixgoatsh/agollo/core/config"
	"github.com/sixgoatsh/agollo/core/mock"
	"github.com/sixgoatsh/agollo/core/options"
	"github.com/sixgoatsh/agollo/pkg/log"
)

type testCase struct {
	Name string
	Test func(configs map[string]*client.NonCacheResp)
}

func TestAgollo(t *testing.T) {
	configServerURL := "http://localhost:8080"
	appid := "test"
	cluster := "default"
	newConfigs := func() map[string]*client.NonCacheResp {
		return map[string]*client.NonCacheResp{
			"application": {
				AppID:         appid,
				Cluster:       cluster,
				NamespaceName: "application",
				Configurations: map[string]interface{}{
					"timeout": "100",
				},
				ReleaseKey: "111",
			},
			"test.json": {
				AppID:         appid,
				Cluster:       cluster,
				NamespaceName: "test.json",
				Configurations: map[string]interface{}{
					"content": `{"name":"foo","age":18}`,
				},
				ReleaseKey: "121",
			},
		}
	}

	rand.Seed(time.Now().Unix())

	newMetaClient := &mock.MetaServerClient{
		ConfigServers: func(conf config.Config) (int, []client.ConfigServerResp, error) {
			return 200, []client.ConfigServerResp{
				{HomePageURL: conf.ConfigServerUrl},
			}, nil
		},
	}
	newNonCacheClient := func(configs map[string]*client.NonCacheResp) client.INonCacheClient {
		return &mock.NonCacheClient{
			ConfigsFromNonCache: func(c config.Config, opts ...client.NotificationsOption) (int, *client.NonCacheResp, error) {
				var notificationsOptions client.NotificationsOptions
				for _, opt := range opts {
					opt(&notificationsOptions)
				}

				conf, ok := configs[c.NamespaceName]

				if !ok {
					return 404, nil, nil
				}

				if conf.ReleaseKey == notificationsOptions.ReleaseKey {
					return 304, nil, nil
				}

				return 200, conf, nil
			},
		}
	}
	newCacheClient := &mock.CacheClient{
		ConfigsFromCache: func(config.Config) (conf *config.Configurations, err error) {
			return nil, nil
		},
	}

	newNotificationClient := func(configs map[string]*client.NonCacheResp) client.INotificationClient {
		return &mock.NotificationsClient{
			Notifications: func(conf config.Config, result []config.Notification) (status int, err error) {
				rk, _ := strconv.Atoi(configs["application"].ReleaseKey)
				n := rand.Intn(2)
				if n%2 == 0 {
					rk++
					configs["application"].ReleaseKey = fmt.Sprint(rk)
				}
				result = []config.Notification{
					{
						NamespaceName:  "application",
						NotificationID: rk,
					},
				}

				return 200, nil
			},
		}
	}

	badMetaClient := &mock.MetaServerClient{
		ConfigServers: func(conf config.Config, ) (int, []client.ConfigServerResp, error) {
			return 500, nil, nil
		},
	}

	badNonCacheClient := func(configs map[string]*client.NonCacheResp) client.INonCacheClient {
		return &mock.NonCacheClient{
			ConfigsFromNonCache: func(conf config.Config, opts ...client.NotificationsOption) (int, *client.NonCacheResp, error) {
				return 500, nil, nil
			},
		}
	}
	badCacheClient := &mock.CacheClient{
		ConfigsFromCache: func(config.Config) (conf *config.Configurations, err error) {
			return nil, nil
		},
	}

	badNotificationClient := func(configs map[string]*client.NonCacheResp) client.INotificationClient {
		return &mock.NotificationsClient{
			Notifications: func(conf config.Config, result []config.Notification) (status int, err error) {
				return 500, nil
			},
		}
	}

	var tests = []testCase{
		{
			Name: "测试：预加载的namespace应该正常可获取，非预加载的namespace无法获取配置",
			Test: func(configs map[string]*client.NonCacheResp) {
				backupFile, err := ioutil.TempFile("", "backup")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(backupFile.Name())
				ba, _ := defaultBalance(configServerURL, appid, newMetaClient)
				a, err := NewGoApollo(configServerURL, appid,
					client.NewApolloClient(newMetaClient, newNonCacheClient(configs), newCacheClient, newNotificationClient(configs)),
					ba,
					options.PreloadNamespaces("test.json"),
					options.BackupFile(backupFile.Name()),
				)
				assert.Nil(t, err)
				assert.NotNil(t, a)

				for namespace, conf := range configs {
					for key, expected := range conf.Configurations {
						if namespace == "test.json" {
							actual := a.Get(key, options.WithNamespace(namespace))
							assert.Equal(t, expected, actual)
						} else {
							actual := a.Get(key, options.WithNamespace(namespace))
							assert.Empty(t, actual)
						}
					}
				}
			},
		},
		{
			Name: "测试：自动获取非预加载namespace时，正常读取配置配置项",
			Test: func(configs map[string]*client.NonCacheResp) {
				backupFile, err := ioutil.TempFile("", "backup")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(backupFile.Name())
				ba, _ := defaultBalance(configServerURL, appid, newMetaClient)
				a, err := NewGoApollo(configServerURL, appid,
					client.NewApolloClient(newMetaClient, newNonCacheClient(configs), newCacheClient, newNotificationClient(configs)),
					ba,
					options.AutoFetchOnCacheMiss(),
					options.WithLogger(log.NewLogger(log.LoggerWriter(os.Stdout))),
					options.BackupFile(backupFile.Name()),
				)
				assert.Nil(t, err)
				assert.NotNil(t, a)

				for namespace, conf := range configs {
					for key, expected := range conf.Configurations {
						actual := a.Get(key, options.WithNamespace(namespace))
						assert.Equal(t, expected, actual,
							"configs: %v, goApollo: %v, Namespace: %s, Key: %s",
							configs, a.GetNameSpace(namespace), namespace, key)
					}
				}

				// 测试无WithNamespace配置项时读取application的配置
				key := "timeout"
				expected := configs["application"].Configurations[key]
				actual := a.Get(key)
				assert.Equal(t, expected, actual)
			},
		},
		{
			Name: "测试：初始化后 start 监听配置的情况",
			Test: func(configs map[string]*client.NonCacheResp) {
				backupFile, err := ioutil.TempFile("", "backup")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(backupFile.Name())
				ba, _ := defaultBalance(configServerURL, appid, newMetaClient)
				a, err := NewGoApollo(configServerURL, appid,
					client.NewApolloClient(newMetaClient, newNonCacheClient(configs), newCacheClient, newNotificationClient(configs)),
					ba,
					options.AutoFetchOnCacheMiss(),
					options.BackupFile(backupFile.Name()),
				)
				assert.Nil(t, err)
				assert.NotNil(t, a)

				a.Start()
				defer a.Stop()

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()

					for i := 0; i < 3; i++ {
						for namespace, config := range configs {
							for key, expected := range config.Configurations {
								actual := a.Get(key, options.WithNamespace(namespace))
								assert.Equal(t, expected, actual)
							}
						}
						time.Sleep(time.Second / 2)
					}
				}()

				wg.Wait()
			},
		},
		{
			Name: "测试：容灾配置项",
			Test: func(configs map[string]*client.NonCacheResp) {
				backupFile, err := ioutil.TempFile("", "backup")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(backupFile.Name())

				enc := json.NewEncoder(backupFile)

				backup := map[string]config.Configurations{}
				for _, conf := range configs {
					backup[conf.NamespaceName] = conf.Configurations
				}

				err = enc.Encode(backup)
				if err != nil {
					t.Fatal(err)
				}
				ba, _ := defaultBalance(configServerURL, appid, badMetaClient)
				a, err := NewGoApollo(configServerURL, appid,
					client.NewApolloClient(badMetaClient, badNonCacheClient(configs), badCacheClient, badNotificationClient(configs)),
					ba,
					options.AutoFetchOnCacheMiss(),
					options.FailTolerantOnBackupExists(),
					options.BackupFile(backupFile.Name()),
				)
				assert.Nil(t, err)
				assert.NotNil(t, a)

				a.Start()
				defer a.Stop()

				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()

					for i := 0; i < 3; i++ {
						for namespace, config := range configs {
							for key, expected := range config.Configurations {
								actual := a.Get(key, options.WithNamespace(namespace))
								assert.Equal(t, expected, actual, "%v %s", a.GetNameSpace(namespace), namespace)
							}
						}
						time.Sleep(time.Second / 2)
					}
				}()

				wg.Wait()
			},
		},
	}

	var wg sync.WaitGroup
	wg.Add(len(tests))
	for _, test := range tests {
		go func(test testCase) {
			defer wg.Done()
			t.Log("Test case:", test.Name)
			configs := newConfigs()
			test.Test(configs)
		}(test)
	}

	wg.Wait()
}

func defaultBalance(configServerURL, appID string, serverClient client.IMetaServerClient) (balancer.Balancer, error) {
	return balancer.NewBalancer(config.DefaultConfig(configServerURL, appID), false, 0, nil, serverClient)
}
