package server

import (
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func newHandlers(target string) *handlers {
	return &handlers{
		proxyBaseURL: target,
		http:         &http.Client{},
	}
}

type handlers struct {
	proxyBaseURL string
	http         *http.Client
}

func (h *handlers) CacheHandler(res http.ResponseWriter, req *http.Request) {
	log.Errorf("Cache Handler")
}

func (h *handlers) ProxyHandler(res http.ResponseWriter, req *http.Request) {
	log.Debug("Proxy Handler")

	targetURL, err := h.getProxyURL(req.URL.Path)
	if err != nil {
		h.handleError(res, req, err)
		return
	}
	targetReq, err := http.NewRequest(req.Method, targetURL, req.Body)
	if err != nil {
		h.handleError(res, req, err)
		return
	}

	for name, values := range req.Header {
		for _, v := range values {
			targetReq.Header.Add(name, v)
		}
	}

	targetResp, err := h.http.Do(targetReq)
	if err != nil {
		h.handleError(res, req, errors.Wrap(err, "target request failed"))
	}
	defer targetResp.Body.Close()

	for name, values := range targetResp.Header {
		for _, v := range values {
			res.Header().Add(name, v)
		}
	}
	res.WriteHeader(targetResp.StatusCode)

	io.Copy(res, targetResp.Body)
}

func (h *handlers) handleError(res http.ResponseWriter, req *http.Request, err error) {
	log.Error(err)
	res.WriteHeader(http.StatusInternalServerError)
}

func (h *handlers) getProxyURL(reqPath string) (string, error) {
	u, err := url.Parse(h.proxyBaseURL)
	if err != nil {
		return "", errors.Wrap(err, "Failed to fetch path from request")
	}
	u.Path = path.Join(u.Path, reqPath)
	return u.String(), nil
}
