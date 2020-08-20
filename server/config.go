package server

import "time"

// Config represents a server config
type Config struct {
	ListenAddr           string
	TLSListenAddr        string
	TLSOnly              bool
	TLS                  *TLSConfig
	Verbose              bool
	BackendFile          string
	CacheDir             string
	ProxyTarget          string
	CacheExpiration      time.Duration
	CacheCleanupInterval time.Duration
}

// TLSConfig represents a TLS configuration
type TLSConfig struct {
	KeyFile  string
	CertFile string
}
