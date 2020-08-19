package options

import "time"

var (
	defaultCluster                    = "default"
	defaultNamespace                  = "application"
	defaultBackupFile                 = ".goApollo"
	defaultAutoFetchOnCacheMiss       = false
	defaultFailTolerantOnBackupExists = false
	defaultEnableSLB                  = false
	defaultLongPollInterval           = 1 * time.Second
)
