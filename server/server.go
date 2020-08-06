package server

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

// New creates a new server instance
func New(c *Config) (*Server, error) {

	return &Server{
		c: c,
	}, nil
}

// Server represents a server instance
type Server struct {
	c *Config
}

// ListenAndServe listens for new requests and serves them
func (s *Server) ListenAndServe() {
	r := mux.NewRouter()
	h := newHandlers()

	r.HandleFunc("/", h.CacheHandler).Methods("GET")
	r.HandleFunc("/", h.ProxyHandler)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tlsEnabled := s.c.TLS.CertFile != "" && s.c.TLS.KeyFile != ""
	if !s.c.TLSOnly {
		go listenAndServe(ctx, cancel, s.c.ListenAddr, r)
	}

	if tlsEnabled {
		go listenAndServeTLS(ctx, cancel, s.c.TLSListenAddr, s.c.TLS, r)
	}

	<-ctx.Done()
}

// listenAndServe serves a plain http webserver
func listenAndServe(ctx context.Context, cancel func(), addr string, handler http.Handler) {
	defer cancel()
	log.Infof("http server listening on: localhost%s\n", addr)
	log.Print(http.ListenAndServe(addr, handler))
}

// listenAndServeTLS serves a tls webserver
func listenAndServeTLS(ctx context.Context, cancel func(), addr string, tls *TLSConfig, handler http.Handler) {
	defer cancel()
	log.Infof("https server listening on: localhost%s\n", addr)
	log.Print(http.ListenAndServeTLS(addr, tls.CertFile, tls.KeyFile, handler))
}
