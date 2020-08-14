package server

// Config represents a server config
type Config struct {
	ListenAddr    string
	TLSListenAddr string
	TLSOnly       bool
	TLS           *TLSConfig
	Verbose       bool
	BackendFile   string
	CacheDir      string
	ProxyTarget   string
}

// TLSConfig represents a TLS configuration
type TLSConfig struct {
	KeyFile  string
	CertFile string
}
