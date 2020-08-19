package cons

import "time"

var (
	ConfigFilePath = "app.properties"
	ConfigType     = "properties"
	MetaURL        = "http://apollo.meta"
	NotificationID = -1
	WatchTimeout   = 500 * time.Millisecond
	Cluster                    = "default"
	Namespace                  = "application"
)

var (
	BackupFile                 = ".goApollo"
	AutoFetchOnCacheMiss       = false
	FailTolerantOnBackupExists = false
	EnableSLB                  = false
	LongPollInterval           = 1 * time.Second
)