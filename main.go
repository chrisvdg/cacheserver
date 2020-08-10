package main

import (
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

	c := &server.Config{
		ListenAddr:    *listAddr,
		TLSListenAddr: *tlsListAddr,
		TLSOnly:       *tlsOnly,
		TLS: &server.TLSConfig{
			KeyFile:  *tlsKey,
			CertFile: *tlsCert,
		},
		ProxyTarget: *target,
		Verbose:     *verbose,
	}

	s, err := server.New(c)
	if err != nil {
		log.Fatal(err)
	}

	s.ListenAndServe()
}
