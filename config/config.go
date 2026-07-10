package config

var (
	Host = "0.0.0.0"
	Port = 8080

	// MaxKeys caps how many keys the store retains before evicting.
	// 0 means unlimited.
	MaxKeys = 0
)
