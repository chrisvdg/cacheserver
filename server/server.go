package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/chrisvdg/cacheserver/cache"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// New creates a new server instance
func New(c *Config) (*Server, error) {
	if c.ProxyTarget == "" {
		return nil, errors.New("No proxy target provided")
	}

	err := testTarget(c.ProxyTarget)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to proxy target")
	}
	cache, err := cache.New(c.BackendFile, c.CacheDir, c.ProxyTarget, 0)
	if err != nil {
		return nil, err
	}

	return &Server{
		c:     c,
		cache: cache,
	}, nil
}

// Server represents a server instance
type Server struct {
	c     *Config
	cache *cache.Cache
}

// ListenAndServe listens for new requests and serves them
func (s *Server) ListenAndServe() {
	r := mux.NewRouter()
	h := newHandlers(s.c.ProxyTarget, s.cache)

	r.PathPrefix("/").HandlerFunc(h.CacheHandler).Methods("GET")
	r.PathPrefix("/").HandlerFunc(h.ProxyHandler)

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
	addrStr := getAddrString(addr)
	log.Infof("http server listening on: http://%s\n", addrStr)
	log.Error(http.ListenAndServe(addr, handler))
}

// listenAndServeTLS serves a tls webserver
func listenAndServeTLS(ctx context.Context, cancel func(), addr string, tls *TLSConfig, handler http.Handler) {
	defer cancel()
	addrStr := getAddrString(addr)
	log.Infof("https server listening on: http://%s\n", addrStr)
	log.Error(http.ListenAndServeTLS(addr, tls.CertFile, tls.KeyFile, handler))
}

func testTarget(url string) error {
	_, err := http.Get(url)
	return err
}

func getAddrString(addr string) string {
	if strings.HasPrefix(addr, ":") {
		addr = fmt.Sprintf("0.0.0.0%s", addr)
	}
	return addr
}
