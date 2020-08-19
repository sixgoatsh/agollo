package options

import (
	"time"

	"github.com/songxuexian/agollo/core/config"
	"github.com/songxuexian/agollo/pkg/log"
	"github.com/songxuexian/agollo/pkg/util/str"
)

type Options struct {
	Conf                       config.Config
	PreloadNamespaces          []string        // 预加载命名空间，默认：为空
	Logger                     log.Logger      // 日志实现类，可以设置自定义实现或者通过NewLogger()创建并设置有效的io.Writer，默认: ioutil.Discard
	AutoFetchOnCacheMiss       bool            // 自动获取非预设以外的Namespace的配置，默认：false
	LongPollerInterval         time.Duration   // 轮训间隔时间，默认：1s
	BackupFile                 string          // 备份文件存放地址，默认：.goApollo
	FailTolerantOnBackupExists bool            // 服务器连接失败时允许读取备份，默认：false
	EnableSLB                  bool            // 启用ConfigServer负载均衡
	RefreshIntervalInSecond    time.Duration   // ConfigServer刷新间隔
	ClientOptions              []config.Option // 设置apollo HTTP api的配置项
}

func NewOptions(configServerURL, appID string, opts ...Option) (Options, error) {
	conf := config.DefaultConfig(configServerURL, appID)
	var options = Options{
		Conf:                       conf,
		Logger:                     log.NewLogger(),
		AutoFetchOnCacheMiss:       defaultAutoFetchOnCacheMiss,
		LongPollerInterval:         defaultLongPollInterval,
		BackupFile:                 defaultBackupFile,
		FailTolerantOnBackupExists: defaultFailTolerantOnBackupExists,
		EnableSLB:                  defaultEnableSLB,
	}
	for _, opt := range opts {
		opt(&options)
	}

	options.Conf.Apply(options.ClientOptions...)

	if options.Conf.NamespaceName != "" && !str.StringInSlice(options.Conf.NamespaceName, options.PreloadNamespaces) {
		options.PreloadNamespaces = append(options.PreloadNamespaces, options.Conf.NamespaceName)
	}

	return options, nil
}

type Option func(*Options)

func Cluster(cluster string) Option {
	return func(o *Options) {
		o.Conf.ClusterName = cluster
	}
}

func DefaultNamespace(defaultNamespace string) Option {
	return func(o *Options) {
		o.Conf.NamespaceName = defaultNamespace
	}
}

func ConfigServerUrl(ConfigServerUrl string) Option {
	return func(o *Options) {
		o.Conf.ConfigServerUrl = ConfigServerUrl
	}
}
func PreloadNamespaces(namespaces ...string) Option {
	return func(o *Options) {
		o.PreloadNamespaces = append(o.PreloadNamespaces, namespaces...)
	}
}

func WithLogger(l log.Logger) Option {
	return func(o *Options) {
		o.Logger = l
	}
}

func AutoFetchOnCacheMiss() Option {
	return func(o *Options) {
		o.AutoFetchOnCacheMiss = true
	}
}

func LongPollerInterval(i time.Duration) Option {
	return func(o *Options) {
		o.LongPollerInterval = i
	}
}

func BackupFile(backupFile string) Option {
	return func(o *Options) {
		o.BackupFile = backupFile
	}
}

func FailTolerantOnBackupExists() Option {
	return func(o *Options) {
		o.FailTolerantOnBackupExists = true
	}
}

func EnableSLB(b bool) Option {
	return func(o *Options) {
		o.EnableSLB = b
	}
}

func ConfigServerRefreshIntervalInSecond(refreshIntervalInSecond time.Duration) Option {
	return func(o *Options) {
		o.RefreshIntervalInSecond = refreshIntervalInSecond
	}
}

func AccessKey(accessKey string) Option {
	return func(o *Options) {
		o.ClientOptions = append(o.ClientOptions, config.WithClientAccessKey(accessKey))
	}
}

func WithClientOptions(opts ...config.Option) Option {
	return func(o *Options) {
		o.ClientOptions = append(o.ClientOptions, opts...)
	}
}

type GetOptions struct {
	// Get时，如果key不存在将返回此值
	DefaultValue string

	// Get时，显示的指定需要获取那个Namespace中的key。非空情况下，优先级顺序为：
	// GetOptions.Namespace > Options.DefaultNamespace > "application"
	Namespace string
}

func (o Options) NewGetOptions(opts ...GetOption) GetOptions {
	var getOpts GetOptions
	for _, opt := range opts {
		opt(&getOpts)
	}

	if getOpts.Namespace == "" {
		getOpts.Namespace = str.NonEmptyString(defaultNamespace, o.Conf.NamespaceName)
	}

	return getOpts
}

type GetOption func(*GetOptions)

func WithDefault(defVal string) GetOption {
	return func(o *GetOptions) {
		o.DefaultValue = defVal
	}
}

func WithNamespace(namespace string) GetOption {
	return func(o *GetOptions) {
		o.Namespace = namespace
	}
}
