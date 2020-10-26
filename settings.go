package crawler

// Settings 爬虫配置
type Settings struct {
	MaxConcurrentRequests     int32
	RequestDelay              int
	RequestTimeout            int
	MaxConcurrentProcessItems int
	MaxRetryTimes             int
}

// DefaultSettings 创建默认Setting
func DefaultSettings() *Settings {
	return &Settings{
		MaxConcurrentRequests:     100,
		RequestDelay:              0,
		RequestTimeout:            20000,
		MaxConcurrentProcessItems: 6,
		MaxRetryTimes: 3,
	}
}
