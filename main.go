package main

import (
	"time"

	"github.com/chrisvdg/cacheserver/server"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

func main() {
	listAddr := pflag.StringP("listenaddr", "l", ":8080", "http listen address")
	tlsListAddr := pflag.StringP("tlsaddr", "t", "8443", "https listen address")
	tlsKey := pflag.StringP("tlskey", "k", "", "TLS private key file path")
	tlsCert := pflag.StringP("tlscert", "c", "", "TLS certificate file path")
	tlsOnly := pflag.BoolP("tlsonly", "s", false, "Only serve TLS")
	target := pflag.StringP("proxytarget", "p", "", "Target server to proxy")
	backendFile := pflag.StringP("backendfile", "f", "./cachebackend.data", "backend metadata file")
	cacheDir := pflag.StringP("cachedir", "d", "./cachebackend", "directory where cached downloads will be stored")
	cacheExpiration := pflag.StringP("cacheexpiration", "e", "1d", "amount of time a cache entry is valid. eg: -e 1d2m (1 day and 2 minutes). Or provide 0 to disable")
	cacheCleanInterval := pflag.StringP("chachecleanint", "i", "12h", "amount of time where in between the cache will be cleaned up.  eg: -e 4h (4 hours). Or provide 0 to disable")
	verbose := pflag.BoolP("verbose", "v", false, "Verbose output")
	pflag.Parse()

	if *verbose {
		log.SetLevel(log.DebugLevel)
	} else {
		log.SetLevel(log.InfoLevel)
	}
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05 02/01/2006",
	})

	cacheExp, err := time.ParseDuration(*cacheExpiration)
	if err != nil {
		log.Fatalf("Failed to parse cache expiration: %s", err)
	}
	cacheInt, err := time.ParseDuration(*cacheCleanInterval)
	if err != nil {
		log.Fatalf("Failed to parse cache cleaning interval: %s", err)
	}

	c := &server.Config{
		ListenAddr:    *listAddr,
		TLSListenAddr: *tlsListAddr,
		TLSOnly:       *tlsOnly,
		TLS: &server.TLSConfig{
			KeyFile:  *tlsKey,
			CertFile: *tlsCert,
		},
		ProxyTarget:          *target,
		BackendFile:          *backendFile,
		CacheDir:             *cacheDir,
		Verbose:              *verbose,
		CacheExpiration:      cacheExp,
		CacheCleanupInterval: cacheInt,
	}

	s, err := server.New(c)
	if err != nil {
		log.Fatal(err)
	}

	s.ListenAndServe()
}
