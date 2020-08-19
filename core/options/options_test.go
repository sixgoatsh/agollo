package options

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/sixgoatsh/agollo/core/config"
)

func TestOptions(t *testing.T) {
	var (
		configServerURL = "localhost:8080"
		appID           = "SampleApp"
	)
	clientConf := config.Config{
		ConfigServerUrl: configServerURL,
		AppID:           appID,
		ClusterName:     defaultCluster,
		NamespaceName:   defaultNamespace,
	}
	var tests = []struct {
		Options []Option
		Check   func(Options)
	}{
		{
			[]Option{},
			func(opts Options) {
				assert.Equal(t, clientConf, opts.Conf)
				assert.Equal(t, defaultAutoFetchOnCacheMiss, opts.AutoFetchOnCacheMiss)
				assert.Equal(t, defaultLongPollInterval, opts.LongPollerInterval)
				assert.Equal(t, defaultBackupFile, opts.BackupFile)
				assert.Equal(t, defaultFailTolerantOnBackupExists, opts.FailTolerantOnBackupExists)
				assert.Equal(t, defaultEnableSLB, opts.EnableSLB)
				assert.NotNil(t, opts.Logger)
				assert.Equal(t, clientConf.NamespaceName, opts.PreloadNamespaces[0])
				getOpts := opts.NewGetOptions()
				assert.Equal(t, "application", getOpts.Namespace)
				getOpts = opts.NewGetOptions(WithNamespace("customize_namespace"))
				assert.Equal(t, "customize_namespace", getOpts.Namespace)
				assert.Empty(t, opts.Conf.AccessKey)
			},
		},
		{
			[]Option{
				Cluster("test_cluster"),
				DefaultNamespace("default_namespace"),
				PreloadNamespaces("preload_namespace"),
				AutoFetchOnCacheMiss(),
				LongPollerInterval(time.Second * 30),
				BackupFile("test_backup"),
				FailTolerantOnBackupExists(),
				AccessKey("test_access_key"),
			},
			func(opts Options) {
				assert.Equal(t, "test_cluster", opts.Conf.ClusterName)
				assert.Equal(t, []string{"preload_namespace", "default_namespace"}, opts.PreloadNamespaces)
				assert.Equal(t, "default_namespace", opts.Conf.NamespaceName)
				getOpts := opts.NewGetOptions()
				assert.Equal(t, "default_namespace", getOpts.Namespace)
				getOpts = opts.NewGetOptions(WithNamespace("customize_namespace"))
				assert.Equal(t, "customize_namespace", getOpts.Namespace)
				assert.Equal(t, true, opts.AutoFetchOnCacheMiss)
				assert.Equal(t, time.Second*30, opts.LongPollerInterval)
				assert.Equal(t, "test_backup", opts.BackupFile)
				assert.Equal(t, true, opts.FailTolerantOnBackupExists)
				assert.Equal(t, "test_access_key", opts.Conf.AccessKey)
			},
		},
		{
			[]Option{
				EnableSLB(true),
			},
			func(opts Options) {
				assert.Equal(t, true, opts.EnableSLB)
			},
		},
	}

	for _, test := range tests {
		opts, err := NewOptions(configServerURL, appID, test.Options...)
		if err != nil {
			assert.Nil(t, err)
		}
		test.Check(opts)
	}
}
