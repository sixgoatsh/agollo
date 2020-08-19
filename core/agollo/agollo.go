package agollo

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/sixgoatsh/agollo/core/client"
	"github.com/sixgoatsh/agollo/core/client/balancer"
	"github.com/sixgoatsh/agollo/core/config"
	"github.com/sixgoatsh/agollo/core/options"
	"github.com/sixgoatsh/agollo/pkg/util/str"
)

var (
	defaultGoApollo GoApollo
)

type GoApollo interface {
	Start() <-chan *LongPollerError
	Stop()
	Get(key string, opts ...options.GetOption) string
	GetNameSpace(namespace string) config.Configurations
	Watch() <-chan *ApolloResponse
	WatchNamespace(namespace string, stop chan bool) <-chan *ApolloResponse
	Options() options.Options
}

type ApolloResponse struct {
	Namespace string
	OldValue  config.Configurations
	NewValue  config.Configurations
	Changes   config.Changes
	Error     error
}

type LongPollerError struct {
	ConfigServerURL string
	AppID           string
	Cluster         string
	Notifications   []config.Notification
	Namespace       string // 服务响应200后去非缓存接口拉取时的namespace
	Err             error
}

type goApollo struct {
	opts            options.Options
	apolloClient    client.IApolloClient
	balance         balancer.Balancer
	notificationMap sync.Map // key: namespace value: notificationId
	releaseKeyMap   sync.Map // key: namespace value: releaseKey
	cache           sync.Map // key: namespace value: Configurations
	initialized     sync.Map // key: namespace value: bool

	watchCh             chan *ApolloResponse // watch all namespace
	watchNamespaceChMap sync.Map             // key: namespace value: chan *ApolloResponse

	errorsCh chan *LongPollerError

	runOnce  sync.Once
	stop     bool
	stopCh   chan struct{}
	stopLock sync.Mutex
}

func NewWithConfigFile(configFilePath string, opts ...options.Option) (GoApollo, error) {
	f, err := os.Open(configFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var conf struct {
		AppID          string   `json:"appId,omitempty"`
		Cluster        string   `json:"cluster,omitempty"`
		NamespaceNames []string `json:"namespaceNames,omitempty"`
		IP             string   `json:"ip,omitempty"`
		AccessKey      string   `json:"accessKey,omitempty"`
	}
	if err := json.NewDecoder(f).Decode(&conf); err != nil {
		return nil, err
	}

	return NewGoApollo(
		conf.IP,
		conf.AppID,
		nil,
		nil,
		append(
			[]options.Option{
				options.Cluster(conf.Cluster),
				options.PreloadNamespaces(conf.NamespaceNames...),
				options.AccessKey(conf.AccessKey),
			},
			opts...,
		)...,
	)
}

func NewGoApollo(configServerURL, appID string, apolloC client.IApolloClient, ba balancer.Balancer, opts ...options.Option) (GoApollo, error) {
	a := &goApollo{
		stopCh:       make(chan struct{}),
		errorsCh:     make(chan *LongPollerError),
		apolloClient: apolloC,
		balance:      ba,
	}
	var err error
	a.opts, err = options.NewOptions(configServerURL, appID, opts...)
	if err != nil {
		return nil, err
	}

	return a, a.initNamespace(a.opts.PreloadNamespaces...)
}

func (a *goApollo) initNamespace(namespaces ...string) error {
	var errs []error
	for _, namespace := range namespaces {
		_, found := a.initialized.LoadOrStore(namespace, true)
		if !found {
			// (1)读取配置 (2)设置初始化notificationMap
			status, _, err := a.reloadNamespace(a.balance, a.apolloClient, namespace)

			// 这里没法光凭靠error==nil来判断namespace是否存在，即使http请求失败，如果开启 容错，会导致error丢失
			// 从而可能将一个不存在的namespace拿去调用getRemoteNotifications导致被hold
			a.setNotificationIDFromRemote(namespace, status == http.StatusOK)

			// 即使存在异常也需要继续初始化下去，有一些使用者会拂掠初始化时的错误
			// 期望在未来某个时间点apollo的服务器恢复过来
			if err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (a *goApollo) setNotificationIDFromRemote(namespace string, exists bool) {
	if !exists {
		// 不能正常获取notificationID的设置为默认notificationID
		// 为之后longPoll提供localNoticationID参数
		a.notificationMap.Store(namespace, defaultNotificationID)
		return
	}

	localNotifications := []config.Notification{
		{
			NotificationID: defaultNotificationID,
			NamespaceName:  namespace,
		},
	}
	// 由于apollo去getRemoteNotifications获取一个不存在的namespace的notificationID时会hold请求90秒
	// (1) 为防止意外传入一个不存在的namespace而发生上述情况，仅将成功获取配置在apollo存在的namespace,去初始化notificationID
	// (2) 此处忽略error返回，在容灾逻辑下配置能正确读取而去获取notificationid可能会返回http请求失败，防止服务不能正常容灾启动
	remoteNotifications, _ := a.getRemoteNotifications(localNotifications)
	if len(remoteNotifications) > 0 {
		for _, notification := range remoteNotifications {
			// 设置namespace初始化的notificationID
			a.notificationMap.Store(notification.NamespaceName, notification.NotificationID)
		}
	} else {
		// 不能正常获取notificationID的设置为默认notificationID
		a.notificationMap.Store(namespace, defaultNotificationID)
	}
}

func (a *goApollo) reloadNamespace(balance balancer.Balancer, nonCacheClient client.IApolloClient, namespace string) (status int, conf config.Configurations, err error) {
	clientConf := a.opts.Conf
	clientConf.ConfigServerUrl, err = balance.Select()
	clientConf.NamespaceName = namespace
	if err != nil {
		a.log("Action", "BalancerSelect", "Error", err)
		return
	}

	var (
		serverConf          *client.NonCacheResp
		cachedReleaseKey, _ = a.releaseKeyMap.LoadOrStore(namespace, "")
	)

	status, serverConf, err = nonCacheClient.GetConfigsFromNonCache(
		clientConf,
		client.ReleaseKey(cachedReleaseKey.(string)),
	)
	if err != nil {
		a.log("ConfigServerUrl", clientConf.ConfigServerUrl, "Namespace", namespace,
			"Action", "GetConfigsFromNonCache", "ServerResponseStatus", status,
			"Error", err)
	}

	switch status {
	case http.StatusOK: // 正常响应
		a.cache.Store(namespace, serverConf.Configurations)     // 覆盖旧缓存
		a.releaseKeyMap.Store(namespace, serverConf.ReleaseKey) // 存储最新的release_key
		conf = serverConf.Configurations

		// 备份配置
		if err = a.backup(); err != nil {
			a.log("BackupFile", a.opts.BackupFile, "Namespace", namespace,
				"Action", "Backup", "Error", err)
			return
		}
	case http.StatusNotModified: // 服务端未修改配置情况下返回304
		conf = a.getNameSpace(namespace)
	default:
		conf = config.Configurations{}

		// 异常状况下，如果开启容灾，则读取备份
		if a.opts.FailTolerantOnBackupExists {
			backupConfig, err := a.loadBackup(namespace)
			if err != nil {
				a.log("BackupFile", a.opts.BackupFile, "Namespace", namespace,
					"Action", "LoadBackup", "Error", err)
				return status, nil, err
			}

			a.cache.Store(namespace, backupConfig)
			return status, backupConfig, nil
		}
	}

	return
}

func (a *goApollo) Get(key string, opts ...options.GetOption) string {
	getOpts := a.opts.NewGetOptions(opts...)

	val, found := a.GetNameSpace(getOpts.Namespace)[key]
	if !found {
		return getOpts.DefaultValue
	}

	v, _ := str.ToStringE(val)
	return v
}

func (a *goApollo) GetNameSpace(namespace string) config.Configurations {
	conf, found := a.cache.LoadOrStore(namespace, config.Configurations{})
	if !found && a.opts.AutoFetchOnCacheMiss {
		err := a.initNamespace(namespace)
		if err != nil {
			a.log("Action", "InitNamespace", "Error", err)
		}
		return a.getNameSpace(namespace)
	}

	return conf.(config.Configurations)
}

func (a *goApollo) getNameSpace(namespace string) config.Configurations {
	v, ok := a.cache.Load(namespace)
	if !ok {
		return config.Configurations{}
	}
	return v.(config.Configurations)
}

func (a *goApollo) Options() options.Options {
	return a.opts
}

// 启动goroutine去轮训apollo通知接口
func (a *goApollo) Start() <-chan *LongPollerError {
	a.runOnce.Do(func() {
		go func() {
			timer := time.NewTimer(a.opts.LongPollerInterval)
			defer timer.Stop()

			for !a.shouldStop() {
				select {
				case <-timer.C:
					a.longPoll()
					timer.Reset(a.opts.LongPollerInterval)
				case <-a.stopCh:
					return
				}
			}
		}()
	})

	return a.errorsCh
}

func (a *goApollo) shouldStop() bool {
	select {
	case <-a.stopCh:
		return true
	default:
		return false
	}
}

func (a *goApollo) longPoll() {
	localNotifications := a.getLocalNotifications()

	// 这里有个问题是非预加载的namespace，如果在Start开启监听后才被initNamespace
	// 需要等待90秒后的下一次轮训才能收到事件通知
	notifications, err := a.getRemoteNotifications(localNotifications)
	if err != nil {
		a.sendErrorsCh("", nil, "", err)
		return
	}

	// HTTP Status: 200时，正常返回notifications数据，数组含有需要更新namespace和notificationID
	// HTTP Status: 304时，上报的namespace没有更新的修改，返回notifications为空数组，遍历空数组跳过
	for _, notification := range notifications {
		// 读取旧缓存用来给监听队列
		oldValue := a.getNameSpace(notification.NamespaceName)

		// 更新namespace
		_, newValue, err := a.reloadNamespace(a.balance, a.apolloClient, notification.NamespaceName)
		if err == nil {
			// 发送到监听channel
			a.sendWatchCh(notification.NamespaceName, oldValue, newValue)

			// 仅在无异常的情况下更新NotificationID，
			// 极端情况下，提前设置notificationID，reloadNamespace还未更新配置并将配置备份，
			// 访问apollo失败导致notificationid已是最新，而配置不是最新
			a.notificationMap.Store(notification.NamespaceName, notification.NotificationID)
		} else {
			a.sendErrorsCh("", notifications, notification.NamespaceName, err)
		}
	}
}

func (a *goApollo) Stop() {
	a.stopLock.Lock()
	defer a.stopLock.Unlock()
	if a.stop {
		return
	}

	if a.balance != nil {
		a.balance.Stop()
	}

	a.stop = true
	close(a.stopCh)
}

func (a *goApollo) Watch() <-chan *ApolloResponse {
	if a.watchCh == nil {
		a.watchCh = make(chan *ApolloResponse)
	}

	return a.watchCh
}

func (a *goApollo) WatchNamespace(namespace string, stop chan bool) <-chan *ApolloResponse {
	watchNamespace := fixWatchNamespace(namespace)
	watchCh, exists := a.watchNamespaceChMap.LoadOrStore(watchNamespace, make(chan *ApolloResponse))
	if !exists {
		go func() {
			// 非预加载以外的namespace,初始化基础meta信息,否则没有longpoll
			err := a.initNamespace(namespace)
			if err != nil {
				watchCh.(chan *ApolloResponse) <- &ApolloResponse{
					Namespace: namespace,
					Error:     err,
				}
			}

			if stop != nil {
				<-stop
				a.watchNamespaceChMap.Delete(watchNamespace)
			}
		}()
	}

	return watchCh.(chan *ApolloResponse)
}

func fixWatchNamespace(namespace string) string {
	// fix: 传给apollo类似test.properties这种namespace
	// 通知回来的NamespaceName却没有.properties后缀，追加.properties后缀来修正此问题
	ext := path.Ext(namespace)
	if ext == "" {
		namespace = namespace + "." + defaultConfigType
	}
	return namespace
}

func (a *goApollo) sendWatchCh(namespace string, oldVal, newVal config.Configurations) {
	changes := oldVal.Different(newVal)
	if len(changes) == 0 {
		return
	}

	resp := &ApolloResponse{
		Namespace: namespace,
		OldValue:  oldVal,
		NewValue:  newVal,
		Changes:   changes,
	}

	timer := time.NewTimer(defaultWatchTimeout)
	for _, watchCh := range a.getWatchChs(namespace) {
		select {
		case watchCh <- resp:

		case <-timer.C: // 防止创建全局监听或者某个namespace监听却不消费死锁问题
			timer.Reset(defaultWatchTimeout)
		}
	}
}

func (a *goApollo) getWatchChs(namespace string) []chan *ApolloResponse {
	var chs []chan *ApolloResponse
	if a.watchCh != nil {
		chs = append(chs, a.watchCh)
	}

	watchNamespace := fixWatchNamespace(namespace)
	if watchNamespaceCh, found := a.watchNamespaceChMap.Load(watchNamespace); found {
		chs = append(chs, watchNamespaceCh.(chan *ApolloResponse))
	}

	return chs
}

// sendErrorsCh 发送轮训时发生的错误信息channel，如果使用者不监听消费channel，错误会被丢弃
// 改成负载均衡机制后，不太好获取每个api使用的configServerURL有点蛋疼
func (a *goApollo) sendErrorsCh(configServerURL string, notifications []config.Notification, namespace string, err error) {
	longPollerError := &LongPollerError{
		ConfigServerURL: configServerURL,
		AppID:           a.opts.Conf.AppID,
		Cluster:         a.opts.Conf.ClusterName,
		Notifications:   notifications,
		Namespace:       namespace,
		Err:             err,
	}
	select {
	case a.errorsCh <- longPollerError:

	default:

	}
}

func (a *goApollo) log(kvs ...interface{}) {
	a.opts.Logger.Log(
		append([]interface{}{
			"[GoApollo]", "",
			"AppID", a.opts.Conf.AppID,
			"Cluster", a.opts.Conf.ClusterName,
		},
			kvs...,
		)...,
	)
}

func (a *goApollo) backup() error {
	backup := map[string]config.Configurations{}
	a.cache.Range(func(key, val interface{}) bool {
		k, _ := key.(string)
		conf, _ := val.(config.Configurations)
		backup[k] = conf
		return true
	})

	data, err := json.Marshal(backup)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(a.opts.BackupFile), 0777)
	if err != nil && !os.IsExist(err) {
		return err
	}

	return ioutil.WriteFile(a.opts.BackupFile, data, 0666)
}

func (a *goApollo) loadBackup(specifyNamespace string) (config.Configurations, error) {
	if _, err := os.Stat(a.opts.BackupFile); err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(a.opts.BackupFile)
	if err != nil {
		return nil, err
	}

	backup := map[string]config.Configurations{}
	err = json.Unmarshal(data, &backup)
	if err != nil {
		return nil, err
	}

	for namespace, configs := range backup {
		if namespace == specifyNamespace {
			return configs, nil
		}
	}

	return nil, nil
}

// getRemoteNotifications
// 立即返回的情况：
// 1. 请求中的namespace任意一个在apollo服务器中有更新的ID会立即返回结果
// 请求被hold 90秒的情况:
// 1. 请求的notificationID和apollo服务器中的ID相等
// 2. 请求的namespace都是在apollo中不存在的
func (a *goApollo) getRemoteNotifications(req []config.Notification) (notifies []config.Notification, err error) {
	clientConf := a.opts.Conf
	clientConf.ConfigServerUrl, err = a.balance.Select()
	clientConf.Notifications = req
	if err != nil {
		a.log("ConfigServerUrl", clientConf.ConfigServerUrl, "Error", err, "Action", "Balancer.Select")
		return
	}

	status, err := a.apolloClient.GetNotifications(clientConf, notifies)
	if err != nil {
		a.log("ConfigServerUrl", clientConf.ConfigServerUrl,
			"GetNotifications", req, "ServerResponseStatus", status,
			"Error", err, "Action", "LongPoll")
		return nil, err
	}

	return
}

func (a *goApollo) getLocalNotifications() []config.Notification {
	var notifications []config.Notification

	a.notificationMap.Range(func(key, val interface{}) bool {
		k, _ := key.(string)
		v, _ := val.(int)
		notifications = append(notifications, config.Notification{
			NamespaceName:  k,
			NotificationID: v,
		})

		return true
	})
	return notifications
}

func Init(configServerURL, appID string, apolloC client.IApolloClient, ba balancer.Balancer, opts ...options.Option) (err error) {
	defaultGoApollo, err = NewGoApollo(configServerURL, appID, apolloC, ba, opts...)
	return
}

func InitWithConfigFile(configFilePath string, opts ...options.Option) (err error) {
	defaultGoApollo, err = NewWithConfigFile(configFilePath, opts...)
	return
}

func InitWithDefaultConfigFile(opts ...options.Option) error {
	return InitWithConfigFile(defaultConfigFilePath, opts...)
}

func Start() <-chan *LongPollerError {
	return defaultGoApollo.Start()
}

func Stop() {
	defaultGoApollo.Stop()
}

func Get(key string, opts ...options.GetOption) string {
	return defaultGoApollo.Get(key, opts...)
}

func GetNameSpace(namespace string) config.Configurations {
	return defaultGoApollo.GetNameSpace(namespace)
}

func Watch() <-chan *ApolloResponse {
	return defaultGoApollo.Watch()
}

func WatchNamespace(namespace string, stop chan bool) <-chan *ApolloResponse {
	return defaultGoApollo.WatchNamespace(namespace, stop)
}

func GetAgollo() GoApollo {
	return defaultGoApollo
}
