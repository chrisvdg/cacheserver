package server

import (
	"net/http"

	log "github.com/sirupsen/logrus"
)

func newHandlers() *handlers {
	return &handlers{}
}

type handlers struct {
}

func (h *handlers) CacheHandler(res http.ResponseWriter, req *http.Request) {
	log.Errorf("Cache Handler")
}

func (h *handlers) ProxyHandler(res http.ResponseWriter, req *http.Request) {
	log.Errorf("Proxy Handler")
}
